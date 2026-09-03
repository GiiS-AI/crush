package cmd

import (
	"context"
	"strings"
	"testing"
)

func TestNewPassthroughCmd_MissingBinary(t *testing.T) {
	t.Parallel()

	cmd := newPassthroughCmd(
		"fake",
		"fake passthrough",
		"definitely-not-a-real-binary-xyz",
		"install hint",
	)
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found on PATH") {
		t.Fatalf("expected error to mention PATH lookup failure, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "definitely-not-a-real-binary-xyz") {
		t.Fatalf("expected error to mention the missing binary, got %q", err.Error())
	}
}

func TestPassthroughCmds_DisableFlagParsing(t *testing.T) {
	t.Parallel()

	if !claudeCmd.DisableFlagParsing {
		t.Fatal("expected claudeCmd to disable flag parsing")
	}
	if !codexCmd.DisableFlagParsing {
		t.Fatal("expected codexCmd to disable flag parsing")
	}
}
