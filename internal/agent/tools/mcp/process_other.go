//go:build windows

package mcp

import "os/exec"

// configureStdioProcess is a no-op on platforms without POSIX process groups.
func configureStdioProcess(cmd *exec.Cmd) {}
