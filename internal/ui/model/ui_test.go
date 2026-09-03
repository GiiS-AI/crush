package model

import (
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/GiiS-AI/GiiS-Code/internal/config"
	"github.com/GiiS-AI/GiiS-Code/internal/csync"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/common"
	"github.com/GiiS-AI/GiiS-Code/internal/workspace"
	"github.com/stretchr/testify/require"
)

func TestCurrentModelSupportsImages(t *testing.T) {
	t.Parallel()

	t.Run("returns false when config is nil", func(t *testing.T) {
		t.Parallel()

		ui := newTestUIWithConfig(t, nil)
		require.False(t, ui.currentModelSupportsImages())
	})

	t.Run("returns false when coder agent is missing", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			Providers: csync.NewMap[string, config.ProviderConfig](),
			Agents:    map[string]config.Agent{},
		}
		ui := newTestUIWithConfig(t, cfg)
		require.False(t, ui.currentModelSupportsImages())
	})

	t.Run("returns false when model is not found", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			Providers: csync.NewMap[string, config.ProviderConfig](),
			Agents: map[string]config.Agent{
				config.AgentCoder: {Model: config.SelectedModelTypeLarge},
			},
		}
		ui := newTestUIWithConfig(t, cfg)
		require.False(t, ui.currentModelSupportsImages())
	})

	t.Run("returns true when current model supports images", func(t *testing.T) {
		t.Parallel()

		providers := csync.NewMap[string, config.ProviderConfig]()
		providers.Set("test-provider", config.ProviderConfig{
			ID: "test-provider",
			Models: []catwalk.Model{
				{ID: "test-model", SupportsImages: true},
			},
		})

		cfg := &config.Config{
			Models: map[config.SelectedModelType]config.SelectedModel{
				config.SelectedModelTypeLarge: {
					Provider: "test-provider",
					Model:    "test-model",
				},
			},
			Providers: providers,
			Agents: map[string]config.Agent{
				config.AgentCoder: {Model: config.SelectedModelTypeLarge},
			},
		}

		ui := newTestUIWithConfig(t, cfg)
		require.True(t, ui.currentModelSupportsImages())
	})
}

func TestProviderCLIForSelectedModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		model      config.SelectedModel
		wantBinary string
		wantLabel  string
		wantOK     bool
	}{
		{
			name: "codex persisted model",
			model: config.SelectedModel{
				Provider: "giis-local",
				Model:    "codex",
			},
			wantBinary: "codex",
			wantLabel:  "Codex",
			wantOK:     true,
		},
		{
			name: "real persisted claude-code config",
			model: config.SelectedModel{
				Provider: "giis-local",
				Model:    "claude-code",
			},
			wantBinary: "claude",
			wantLabel:  "Claude Code",
			wantOK:     true,
		},
		{
			name: "composite model id safety net",
			model: config.SelectedModel{
				Provider: "giis-local",
				Model:    "giis-local/codex",
			},
			wantBinary: "codex",
			wantLabel:  "Codex",
			wantOK:     true,
		},
		{
			name: "non bridge provider",
			model: config.SelectedModel{
				Provider: "anthropic",
				Model:    "claude-sonnet",
			},
		},
		{
			name: "unknown bridge model",
			model: config.SelectedModel{
				Provider: "giis-local",
				Model:    "ollama/gemma2:latest",
			},
		},
		{
			name: "umbrella provider group is not persisted cli selection",
			model: config.SelectedModel{
				Provider: "giis-bridge",
				Model:    "giis-local/codex",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			binary, label, ok := providerCLIForSelectedModel(tt.model)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantBinary, binary)
			require.Equal(t, tt.wantLabel, label)
		})
	}
}

func newTestUIWithConfig(t *testing.T, cfg *config.Config) *UI {
	t.Helper()

	return &UI{
		com: &common.Common{
			Workspace: &testWorkspace{cfg: cfg},
		},
	}
}

// testWorkspace is a minimal [workspace.Workspace] stub for unit tests.
type testWorkspace struct {
	workspace.Workspace
	cfg *config.Config
}

func (w *testWorkspace) Config() *config.Config {
	return w.cfg
}
