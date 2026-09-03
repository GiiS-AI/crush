//go:build !windows

package mcp

import (
	"testing"

	"github.com/GiiS-AI/GiiS-Code/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestCreateTransport_StdioProcessGroup(t *testing.T) {
	t.Parallel()

	m := config.MCPConfig{Type: config.MCPStdio, Command: "echo", Args: []string{"hi"}}
	tr, err := createTransport(t.Context(), m, shellResolverWithPath(t, nil))
	require.NoError(t, err)

	ct, ok := tr.(*mcp.CommandTransport)
	require.True(t, ok)
	require.NotNil(t, ct.Command.SysProcAttr)
	require.True(t, ct.Command.SysProcAttr.Setpgid)
	require.NotNil(t, ct.Command.Cancel)
}
