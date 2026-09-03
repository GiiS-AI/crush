//go:build live_lsp

package lsp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/GiiS-AI/GiiS-Code/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLiveGoplsCodeIntelligenceAPIs(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skipf("gopls not available: %v", err)
	}

	dir := t.TempDir()
	t.Setenv("HOME", filepath.Join(dir, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "xdg-data"))

	workspace := filepath.Join(dir, "workspace")
	require.NoError(t, os.MkdirAll(workspace, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/live-lsp\n\ngo 1.23\n"), 0o644))

	sourcePath := filepath.Join(workspace, "sample.go")
	require.NoError(t, os.WriteFile(sourcePath, []byte(`package sample

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
`), 0o644))

	store, err := config.Load(workspace, filepath.Join(dir, "data"), false)
	require.NoError(t, err)

	manager := NewManager(store)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	manager.Start(ctx, sourcePath)
	defer func() {
		for client := range manager.Clients().Seq() {
			_ = client.Close(context.Background())
		}
	}()

	var client *Client
	require.Eventually(t, func() bool {
		for candidate := range manager.Clients().Seq() {
			if candidate.HandlesFile(sourcePath) {
				client = candidate
				return candidate.GetServerState() == StateReady
			}
		}
		return false
	}, 30*time.Second, 200*time.Millisecond)

	symbols, err := client.DocumentSymbols(ctx, sourcePath)
	require.NoError(t, err)
	require.NotEmpty(t, symbols)

	locations, err := client.Definition(ctx, sourcePath, 6, 9)
	require.NoError(t, err)
	require.NotEmpty(t, locations)

	defPath, err := locations[0].URI.Path()
	require.NoError(t, err)
	require.Equal(t, sourcePath, defPath)
	require.Equal(t, uint32(8), locations[0].Range.Start.Line)
}
