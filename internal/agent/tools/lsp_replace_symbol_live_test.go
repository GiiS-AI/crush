//go:build live_lsp

package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/GiiS-AI/GiiS-Code/internal/config"
	"github.com/GiiS-AI/GiiS-Code/internal/lsp"
	"github.com/stretchr/testify/require"
)

func TestLiveReplaceSymbolToolAppliesEdit(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skipf("gopls not available: %v", err)
	}

	dir := t.TempDir()
	t.Setenv("HOME", filepath.Join(dir, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "xdg-data"))

	workspace := filepath.Join(dir, "workspace")
	require.NoError(t, os.MkdirAll(workspace, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/live-replace-symbol-tool\n\ngo 1.23\n"), 0o644))

	sourcePath := filepath.Join(workspace, "sample.go")
	original := `package sample

type Greeter struct{}

func (Greeter) Hello() string {
	return helper()
}

func helper() string {
	return "hi"
}

func Use() string {
	return Greeter{}.Hello()
}
`
	require.NoError(t, os.WriteFile(sourcePath, []byte(original), 0o644))

	store, err := config.Load(workspace, filepath.Join(dir, "data"), false)
	require.NoError(t, err)

	manager := lsp.NewManager(store)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	manager.Start(ctx, sourcePath)
	defer func() {
		for client := range manager.Clients().Seq() {
			_ = client.Close(context.Background())
		}
	}()

	require.Eventually(t, func() bool {
		for candidate := range manager.Clients().Seq() {
			if candidate.HandlesFile(sourcePath) {
				return candidate.GetServerState() == lsp.StateReady
			}
		}
		return false
	}, 30*time.Second, 200*time.Millisecond)

	tool := NewReplaceSymbolTool(manager, nil, nil, nil)

	input, err := json.Marshal(ReplaceSymbolParams{
		Symbol:      "helper",
		FilePath:    sourcePath,
		Replacement: "func helper() string {\n\treturn \"bye\"\n}",
		Action:      "replace",
	})
	require.NoError(t, err)

	resp, err := tool.Run(ctx, fantasy.ToolCall{ID: "test-call", Name: ReplaceSymbolToolName, Input: string(input)})
	require.NoError(t, err)
	require.False(t, resp.IsError, "tool returned an error response: %s", resp.Content)
	t.Logf("tool response: %s", resp.Content)

	updated, err := os.ReadFile(sourcePath)
	require.NoError(t, err)
	content := string(updated)

	require.Contains(t, content, `return "bye"`)
	require.NotContains(t, content, `return "hi"`)

	// Unrelated code must be untouched.
	require.Contains(t, content, "type Greeter struct{}")
	require.Contains(t, content, "return helper()")
	require.Contains(t, content, "return Greeter{}.Hello()")
}
