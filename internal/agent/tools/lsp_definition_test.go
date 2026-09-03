package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
	"github.com/stretchr/testify/require"
)

func TestFormatDefinitionsIncludesContextAndMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	content := "package sample\n\nfunc helper() {}\n\nfunc target() {\n\thelper()\n}\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	text, meta := formatDefinitions([]protocol.Location{{
		URI: protocol.URIFromPath(path),
		Range: protocol.Range{
			Start: protocol.Position{Line: 4, Character: 0},
			End:   protocol.Position{Line: 6, Character: 1},
		},
	}})

	require.Contains(t, text, "Found 1 definition(s):")
	require.Contains(t, text, path+":5")
	require.Contains(t, text, ">    5 | func target() {")
	require.NotNil(t, meta)
	require.Equal(t, path, meta.FilePath)
	require.Equal(t, 4, meta.Line)
	require.Contains(t, meta.Content, "func target() {")
}
