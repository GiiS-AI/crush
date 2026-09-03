//go:build !windows

package mcp

import (
	"os/exec"
	"syscall"
	"time"
)

// configureStdioProcess puts a stdio MCP child in its own process group so
// cancellation can reap the whole process tree instead of only the direct
// child.
func configureStdioProcess(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 5 * time.Second
}
