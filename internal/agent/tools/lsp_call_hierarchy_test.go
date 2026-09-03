package tools

import (
	"testing"

	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
	"github.com/stretchr/testify/require"
)

func TestFormatIncomingCallHierarchy(t *testing.T) {
	t.Parallel()

	item := protocol.CallHierarchyItem{Name: "target"}
	calls := []protocol.CallHierarchyIncomingCall{{
		From: protocol.CallHierarchyItem{
			Name: "caller",
			URI:  protocol.URIFromPath("/tmp/in.go"),
			Range: protocol.Range{
				Start: protocol.Position{Line: 9},
			},
		},
	}}

	out := formatIncomingCallHierarchy(item, calls)
	require.Contains(t, out, "Call hierarchy for 'target':")
	require.Contains(t, out, "1 caller(s):")
	require.Contains(t, out, "/tmp/in.go:10 - caller")
}

func TestFormatOutgoingCallHierarchyEmpty(t *testing.T) {
	t.Parallel()

	out := formatOutgoingCallHierarchy(protocol.CallHierarchyItem{Name: "target"}, nil)
	require.Contains(t, out, "No outgoing calls found.")
}
