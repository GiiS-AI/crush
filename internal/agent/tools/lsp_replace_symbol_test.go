package tools

import (
	"testing"

	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
	"github.com/stretchr/testify/require"
)

func TestApplySymbolAction(t *testing.T) {
	t.Parallel()

	content := "one\ntwo\nthree\nfour"
	rng := protocol.Range{
		Start: protocol.Position{Line: 1},
		End:   protocol.Position{Line: 2},
	}

	tests := []struct {
		name        string
		action      string
		replacement string
		want        string
	}{
		{"replace", "replace", "TWO\nTHREE", "one\nTWO\nTHREE\nfour"},
		{"add before", "add_before", "zero", "one\nzero\ntwo\nthree\nfour"},
		{"add after", "add_after", "tail", "one\ntwo\nthree\ntail\nfour"},
		{"delete", "delete", "", "one\nfour"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, lines, err := applySymbolAction(content, rng, tt.action, tt.replacement)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.Equal(t, [2]int{1, 2}, lines)
		})
	}
}

func TestFindSymbolByName(t *testing.T) {
	t.Parallel()

	symbols := []protocol.DocumentSymbolResult{
		&protocol.DocumentSymbol{
			Name: "Thing",
			Children: []protocol.DocumentSymbol{{
				Name: "Run",
				Range: protocol.Range{
					Start: protocol.Position{Line: 4},
				},
			}},
		},
	}

	found := findSymbolByName(symbols, "Run")
	require.NotNil(t, found)
	require.Equal(t, "Run", found.GetName())
}
