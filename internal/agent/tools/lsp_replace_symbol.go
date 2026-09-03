package tools

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"charm.land/fantasy"
	"github.com/GiiS-AI/GiiS-Code/internal/filetracker"
	"github.com/GiiS-AI/GiiS-Code/internal/fsext"
	"github.com/GiiS-AI/GiiS-Code/internal/history"
	"github.com/GiiS-AI/GiiS-Code/internal/lsp"
	"github.com/GiiS-AI/GiiS-Code/internal/permission"
	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
)

type ReplaceSymbolParams struct {
	Symbol      string `json:"symbol" description:"The symbol name to target (e.g., function name, method name, type name)"`
	FilePath    string `json:"file_path" description:"The path to the file containing the symbol"`
	Replacement string `json:"replacement,omitempty" description:"The replacement text. Required for 'replace', 'add_before', and 'add_after'."`
	Action      string `json:"action,omitempty" description:"Operation to perform: 'replace' (default), 'add_before', 'add_after', or 'delete'."`
}

const ReplaceSymbolToolName = "lsp_replace_symbol"

//go:embed lsp_replace_symbol.md
var replaceSymbolDescription string

// ReplaceSymbolResponseMetadata carries diff data for the renderer.
type ReplaceSymbolResponseMetadata struct {
	FilePath   string `json:"file_path"`
	OldContent string `json:"old_content"`
	NewContent string `json:"new_content"`
	Action     string `json:"action"`
}

// ReplaceSymbolPermissionsParams carries diff data for the permission dialog.
type ReplaceSymbolPermissionsParams struct {
	FilePath   string `json:"file_path"`
	OldContent string `json:"old_content"`
	NewContent string `json:"new_content"`
}

func NewReplaceSymbolTool(
	lspManager *lsp.Manager,
	permissions permission.Service,
	files history.Service,
	filetracker filetracker.Service,
) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		ReplaceSymbolToolName,
		replaceSymbolDescription,
		func(ctx context.Context, params ReplaceSymbolParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Symbol == "" {
				return fantasy.NewTextErrorResponse("symbol is required"), nil
			}
			if params.FilePath == "" {
				return fantasy.NewTextErrorResponse("file_path is required"), nil
			}

			action := params.Action
			if action == "" {
				action = "replace"
			}
			switch action {
			case "replace", "add_before", "add_after", "delete":
			default:
				return fantasy.NewTextErrorResponse(fmt.Sprintf("invalid action %q: must be replace, add_before, add_after, or delete", action)), nil
			}
			if action != "delete" && params.Replacement == "" {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("replacement is required for action %q", action)), nil
			}

			filePath, err := filepath.Abs(params.FilePath)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to resolve file path: %s", err)), nil
			}

			lspManager.Start(ctx, filePath)
			client := findLSPClient(lspManager, filePath)
			if client == nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("no LSP client handles file: %s", filePath)), nil
			}

			symbols, err := client.DocumentSymbols(ctx, filePath)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to get document symbols: %s", err)), nil
			}

			target := findSymbolByName(symbols, params.Symbol)
			if target == nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("symbol '%s' not found in %s", params.Symbol, filePath)), nil
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
			}

			newContent, lineRange, err := applySymbolAction(string(content), target.GetRange(), action, params.Replacement)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			sessionID := GetSessionFromContext(ctx)
			if sessionID != "" && permissions != nil {
				granted, err := permissions.Request(ctx, permission.CreatePermissionRequest{
					SessionID:   sessionID,
					ToolCallID:  call.ID,
					Path:        fsext.PathOrPrefix(filePath, filepath.Dir(filePath)),
					ToolName:    ReplaceSymbolToolName,
					Action:      "edit",
					Description: fmt.Sprintf("%s symbol '%s' in %s", action, params.Symbol, filePath),
					Params: ReplaceSymbolPermissionsParams{
						FilePath:   filePath,
						OldContent: string(content),
						NewContent: newContent,
					},
				})
				if err != nil {
					return fantasy.ToolResponse{}, fmt.Errorf("permission request failed: %w", err)
				}
				if !granted {
					return NewPermissionDeniedResponse(), nil
				}
			}

			if files != nil && sessionID != "" {
				if _, err := files.CreateVersion(ctx, sessionID, filePath, string(content)); err != nil {
					slog.Warn("Failed to create file version before replace", "path", filePath, "error", err)
				}
			}

			if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
			}

			if filetracker != nil && sessionID != "" {
				filetracker.RecordRead(ctx, sessionID, filePath)
			}

			notifyLSPs(ctx, lspManager, filePath)

			summary := formatReplaceSymbolSummary(action, params.Symbol, filePath, lineRange[0], lineRange[1])
			resp := fantasy.NewTextResponse(summary + "\n" + getDiagnostics(filePath, lspManager))
			resp = fantasy.WithResponseMetadata(resp, ReplaceSymbolResponseMetadata{
				FilePath:   filePath,
				OldContent: string(content),
				NewContent: newContent,
				Action:     action,
			})
			return resp, nil
		},
	)
}

func formatReplaceSymbolSummary(action, symbol, filePath string, startLine, endLine int) string {
	switch action {
	case "replace":
		return fmt.Sprintf("Replaced symbol '%s' in %s (lines %d-%d)", symbol, filePath, startLine+1, endLine+1)
	case "add_before":
		return fmt.Sprintf("Inserted before symbol '%s' in %s (before line %d)", symbol, filePath, startLine+1)
	case "add_after":
		return fmt.Sprintf("Inserted after symbol '%s' in %s (after line %d)", symbol, filePath, endLine+1)
	case "delete":
		return fmt.Sprintf("Deleted symbol '%s' from %s (lines %d-%d)", symbol, filePath, startLine+1, endLine+1)
	default:
		return fmt.Sprintf("Updated symbol '%s' in %s", symbol, filePath)
	}
}

func applySymbolAction(content string, rng protocol.Range, action, replacement string) (string, [2]int, error) {
	lines := strings.Split(content, "\n")
	startLine := int(rng.Start.Line)
	endLine := int(rng.End.Line)
	if startLine < 0 || endLine < startLine || startLine >= len(lines) || endLine >= len(lines) {
		return "", [2]int{}, fmt.Errorf("symbol range exceeds file length")
	}

	var newLines []string
	switch action {
	case "replace":
		newLines = make([]string, 0, len(lines))
		newLines = append(newLines, lines[:startLine]...)
		newLines = append(newLines, strings.Split(replacement, "\n")...)
		newLines = append(newLines, lines[endLine+1:]...)
	case "add_before":
		newLines = make([]string, 0, len(lines)+strings.Count(replacement, "\n")+1)
		newLines = append(newLines, lines[:startLine]...)
		newLines = append(newLines, strings.Split(replacement, "\n")...)
		newLines = append(newLines, lines[startLine:]...)
	case "add_after":
		newLines = make([]string, 0, len(lines)+strings.Count(replacement, "\n")+1)
		newLines = append(newLines, lines[:endLine+1]...)
		newLines = append(newLines, strings.Split(replacement, "\n")...)
		newLines = append(newLines, lines[endLine+1:]...)
	case "delete":
		newLines = make([]string, 0, len(lines))
		newLines = append(newLines, lines[:startLine]...)
		newLines = append(newLines, lines[endLine+1:]...)
	default:
		return "", [2]int{}, fmt.Errorf("invalid action %q", action)
	}

	return strings.Join(newLines, "\n"), [2]int{startLine, endLine}, nil
}

// findSymbolByName searches for a symbol by name in the document symbol tree.
func findSymbolByName(symbols []protocol.DocumentSymbolResult, name string) protocol.DocumentSymbolResult {
	for _, sym := range symbols {
		if sym.GetName() == name {
			return sym
		}
		if ds, ok := sym.(*protocol.DocumentSymbol); ok && len(ds.Children) > 0 {
			children := make([]protocol.DocumentSymbolResult, len(ds.Children))
			for i := range ds.Children {
				children[i] = &ds.Children[i]
			}
			if found := findSymbolByName(children, name); found != nil {
				return found
			}
		}
	}
	return nil
}
