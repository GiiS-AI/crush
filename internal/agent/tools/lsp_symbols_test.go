package tools

import (
	"testing"

	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
	"github.com/stretchr/testify/require"
)

func TestFormatSymbolsRendersNestedOutline(t *testing.T) {
	t.Parallel()

	symbols := []protocol.DocumentSymbolResult{
		&protocol.DocumentSymbol{
			Name: "Thing",
			Kind: protocol.Struct,
			Range: protocol.Range{
				Start: protocol.Position{Line: 1},
			},
			Children: []protocol.DocumentSymbol{{
				Name: "Run",
				Kind: protocol.Method,
				Range: protocol.Range{
					Start: protocol.Position{Line: 3},
				},
			}},
		},
	}

	out := formatSymbols(symbols, 0)
	require.Equal(t, "Struct Thing (line 2)\n  Method Run (line 4)\n", out)
}
