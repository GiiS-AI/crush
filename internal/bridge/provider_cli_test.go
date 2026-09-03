package bridge

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildProviderCLICommand(t *testing.T) {
	dir := t.TempDir()
	binary := "fake-provider-cli"
	path := filepath.Join(dir, binary)
	content := "#!/bin/sh\nexit 0\n"
	if runtime.GOOS == "windows" {
		path += ".bat"
		content = "@echo off\r\nexit /b 0\r\n"
	}
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cmd, err := BuildProviderCLICommand(context.Background(), binary, []string{"--flag", "value"})
	require.NoError(t, err)
	require.True(t, strings.Contains(filepath.Base(cmd.Path), binary), "path %q should contain binary name", cmd.Path)
	require.Equal(t, []string{cmd.Path, "--flag", "value"}, cmd.Args)
}

func TestBuildProviderCLICommandMissingBinary(t *testing.T) {
	t.Parallel()

	_, err := BuildProviderCLICommand(context.Background(), "definitely-not-a-real-binary-xyz", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "definitely-not-a-real-binary-xyz CLI not found on PATH")
}
