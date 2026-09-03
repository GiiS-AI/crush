package bridge

import (
	"context"
	"fmt"
	"os/exec"
)

const (
	CodexCLIBinary      = "codex"
	ClaudeCodeCLIBinary = "claude"
)

// BuildProviderCLICommand resolves a provider CLI binary and returns a command
// ready for the caller to run directly or hand to Bubble Tea.
func BuildProviderCLICommand(ctx context.Context, binary string, args []string) (*exec.Cmd, error) {
	path, err := exec.LookPath(binary)
	if err != nil {
		return nil, fmt.Errorf("%s CLI not found on PATH", binary)
	}
	return exec.CommandContext(ctx, path, args...), nil
}
