package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/GiiS-AI/GiiS-Code/internal/bridge"
	"github.com/spf13/cobra"
)

func newPassthroughCmd(name, short, binary, installHint string) *cobra.Command {
	return &cobra.Command{
		Use:                name + " [args...]",
		Short:              short,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := bridge.BuildProviderCLICommand(cmd.Context(), binary, args)
			if err != nil {
				return fmt.Errorf("%w.\n%s", err, installHint)
			}

			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Env = os.Environ()

			if err := c.Run(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					os.Exit(exitErr.ExitCode())
				}
				return err
			}
			return nil
		},
	}
}

var claudeCmd = newPassthroughCmd(
	"claude",
	"Launch the installed Claude Code CLI",
	bridge.ClaudeCodeCLIBinary,
	"Install Claude Code first (for example: brew install anthropic/cli/claude-code), ensure `claude` is on PATH, then run `c0d3r claude` again.",
)

var codexCmd = newPassthroughCmd(
	"codex",
	"Launch the installed Codex CLI",
	bridge.CodexCLIBinary,
	"Install the Codex CLI and ensure `codex` is on PATH, then run `c0d3r codex` again.",
)

func init() {
	rootCmd.AddCommand(claudeCmd, codexCmd)
}
