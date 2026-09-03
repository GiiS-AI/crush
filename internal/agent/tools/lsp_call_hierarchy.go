package tools

import (
	"cmp"
	"context"
	_ "embed"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/GiiS-AI/GiiS-Code/internal/lsp"
	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
)

type CallHierarchyParams struct {
	Symbol    string `json:"symbol" description:"The symbol name to show call hierarchy for"`
	Direction string `json:"direction" description:"Either 'incoming' (who calls this) or 'outgoing' (what does this call)"`
	Path      string `json:"path,omitempty" description:"The directory to search in. Defaults to the current working directory."`
}

const CallHierarchyToolName = "lsp_call_hierarchy"

//go:embed lsp_call_hierarchy.md
var callHierarchyDescription string

func NewCallHierarchyTool(lspManager *lsp.Manager) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		CallHierarchyToolName,
		callHierarchyDescription,
		func(ctx context.Context, params CallHierarchyParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Symbol == "" {
				return fantasy.NewTextErrorResponse("symbol is required"), nil
			}
			if params.Direction != "incoming" && params.Direction != "outgoing" {
				return fantasy.NewTextErrorResponse("direction must be 'incoming' or 'outgoing'"), nil
			}

			workingDir := cmp.Or(params.Path, ".")
			resolved, err := resolveSymbol(ctx, lspManager, params.Symbol, workingDir)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("Symbol '%s' not found", params.Symbol)), nil
			}

			items, err := resolved.client.PrepareCallHierarchy(ctx, resolved.path, resolved.line, resolved.char)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to prepare call hierarchy: %s", err)), nil
			}
			if len(items) == 0 {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("No call hierarchy information for '%s'", params.Symbol)), nil
			}

			var (
				item = items[0]
				text string
			)
			switch params.Direction {
			case "incoming":
				calls, err := resolved.client.IncomingCalls(ctx, item)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to get incoming calls: %s", err)), nil
				}
				text = formatIncomingCallHierarchy(item, calls)
			case "outgoing":
				calls, err := resolved.client.OutgoingCalls(ctx, item)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to get outgoing calls: %s", err)), nil
				}
				text = formatOutgoingCallHierarchy(item, calls)
			}

			return fantasy.NewTextResponse(text), nil
		},
	)
}

func formatIncomingCallHierarchy(item protocol.CallHierarchyItem, calls []protocol.CallHierarchyIncomingCall) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Call hierarchy for '%s':\n\n", item.Name)
	if len(calls) == 0 {
		b.WriteString("No incoming calls found.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "%d caller(s):\n\n", len(calls))
	for _, call := range calls {
		path, _ := call.From.URI.Path()
		line := call.From.Range.Start.Line + 1
		fmt.Fprintf(&b, "  %s:%d - %s\n", path, line, call.From.Name)
	}
	return b.String()
}

func formatOutgoingCallHierarchy(item protocol.CallHierarchyItem, calls []protocol.CallHierarchyOutgoingCall) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Call hierarchy for '%s':\n\n", item.Name)
	if len(calls) == 0 {
		b.WriteString("No outgoing calls found.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "%d callee(s):\n\n", len(calls))
	for _, call := range calls {
		path, _ := call.To.URI.Path()
		line := call.To.Range.Start.Line + 1
		fmt.Fprintf(&b, "  %s:%d - %s\n", path, line, call.To.Name)
	}
	return b.String()
}
