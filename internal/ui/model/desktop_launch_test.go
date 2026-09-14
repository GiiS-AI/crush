package model

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLaunchDesktopApp_NotInstalled(t *testing.T) {
	// Ensure the binary genuinely cannot be found on PATH.
	t.Setenv("PATH", t.TempDir())

	ui := &UI{}
	err := ui.launchDesktopApp()
	require.Error(t, err, "expected a clear error when giis-desktop is not on PATH")
}

func TestLaunchDesktopApp_DetachedAndNonBlocking(t *testing.T) {
	// Fake a slow "giis-desktop" binary on PATH and confirm launchDesktopApp
	// returns quickly instead of waiting for it to exit.
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, desktopAppBinary)
	script := "#!/bin/sh\nsleep 5\n"
	require.NoError(t, os.WriteFile(fakeBin, []byte(script), 0o755))
	t.Setenv("PATH", dir)

	ui := &UI{}
	start := time.Now()
	err := ui.launchDesktopApp()
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Less(t, elapsed, 2*time.Second, "launchDesktopApp must not block waiting for the child process")
}
