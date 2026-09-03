package tools

import (
	"testing"

	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
	"github.com/stretchr/testify/require"
)

func TestGetSymbolOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		symbol string
		want   int
	}{
		{"bare symbol", "Bar", 0},
		{"dot qualified", "foo.Bar", 4},
		{"double colon qualified", "Class::method", 7},
		{"backslash qualified", `ns\Func`, 3},
		{"nested dots", "a.b.C", 4},
		{"empty", "", 0},
		{"single char", "x", 0},
		{"dot at end", "foo.", 4},
		{"colon at end", "foo::", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, getSymbolOffset(tt.symbol))
		})
	}
}

func TestGetSymbolOffsetDoesNotOvershoot(t *testing.T) {
	t.Parallel()

	cases := []struct {
		symbol   string
		expected string
	}{
		{"foo.Bar", "Bar"},
		{"Class::method", "method"},
		{`ns\Func`, "Func"},
		{"a.b.c.D", "D"},
		{"Bar", "Bar"},
	}

	for _, tc := range cases {
		offset := getSymbolOffset(tc.symbol)
		require.LessOrEqual(t, offset, len(tc.symbol))
		require.Equal(t, tc.expected, tc.symbol[offset:])
	}
}

func TestCollectAffectedFiles(t *testing.T) {
	t.Parallel()

	first := protocol.URIFromPath("/tmp/a.go")
	second := protocol.URIFromPath("/tmp/b.go")
	third := protocol.URIFromPath("/tmp/c.go")

	edit := &protocol.WorkspaceEdit{
		Changes: map[protocol.DocumentURI][]protocol.TextEdit{
			first: nil,
		},
		DocumentChanges: []protocol.DocumentChange{
			{TextDocumentEdit: &protocol.TextDocumentEdit{
				TextDocument: protocol.OptionalVersionedTextDocumentIdentifier{TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: second}},
			}},
			{RenameFile: &protocol.RenameFile{OldURI: second, NewURI: third}},
			{DeleteFile: &protocol.DeleteFile{URI: first}},
		},
	}

	require.ElementsMatch(t, []string{"/tmp/a.go", "/tmp/b.go", "/tmp/c.go"}, collectAffectedFiles(edit))
}
