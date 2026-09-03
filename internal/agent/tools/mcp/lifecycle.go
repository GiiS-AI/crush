package mcp

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"sync"

	"github.com/GiiS-AI/GiiS-Code/internal/config"
)

type reinitAction int

const (
	reinitDisable reinitAction = iota + 1
	reinitRemove
	reinitStart
)

// reconcile diffs the current config against the running MCP state and decides
// which servers need to be removed, disabled, or (re)started.
func reconcile(current config.MCPs, running map[string]ClientInfo) map[string]reinitAction {
	actions := map[string]reinitAction{}

	for name := range running {
		if _, exists := current[name]; !exists {
			actions[name] = reinitRemove
		}
	}

	for name, m := range current {
		info, exists := running[name]
		if m.Disabled {
			if exists && info.State != StateDisabled {
				actions[name] = reinitDisable
			}
			continue
		}

		if exists {
			switch info.State {
			case StateStarting:
				if info.PendingConfig != nil && mcpConfigEqual(*info.PendingConfig, m) {
					continue
				}
			case StateConnected:
				if mcpConfigEqual(info.Config, m) {
					continue
				}
			}
		}
		actions[name] = reinitStart
	}

	return actions
}

var (
	reinitMu      sync.Mutex
	reinitRunning bool
	reinitDirty   bool
)

// Reinitialize reconciles process-global MCP state against the current config.
func Reinitialize(ctx context.Context, cfg *config.ConfigStore) {
	reinitMu.Lock()
	if reinitRunning {
		reinitDirty = true
		reinitMu.Unlock()
		return
	}
	reinitRunning = true
	reinitMu.Unlock()

	for {
		reconcileOnce(ctx, cfg)

		reinitMu.Lock()
		if !reinitDirty {
			reinitRunning = false
			reinitMu.Unlock()
			return
		}
		reinitDirty = false
		reinitMu.Unlock()
	}
}

func reconcileOnce(ctx context.Context, cfg *config.ConfigStore) {
	current := cfg.Config().MCP
	actions := reconcile(current, states.Copy())
	for name, action := range actions {
		switch action {
		case reinitRemove:
			slog.Info("Removing MCP server no longer in config", "name", name)
			removeServer(name)
		case reinitDisable:
			slog.Info("Disabling MCP server", "name", name)
			_ = DisableSingle(cfg, name)
		case reinitStart:
			m := current[name]
			if _, exists := states.Get(name); exists {
				slog.Info("Re-initializing MCP server after config change", "name", name)
			} else {
				slog.Info("Initializing new MCP server after config change", "name", name)
			}
			teardown(name)
			updateState(name, StateStarting, nil, nil, Counts{}, withPending(m))
			goInitClient(ctx, cfg, name, m, nil)
		}
	}
}

func removeServer(name string) {
	teardown(name)
	states.Del(name)
	gens.Del(name)
}

func mcpConfigEqual(a, b config.MCPConfig) bool {
	return a.Command == b.Command &&
		maps.Equal(a.Env, b.Env) &&
		slices.Equal(a.Args, b.Args) &&
		a.Type == b.Type &&
		a.URL == b.URL &&
		a.Disabled == b.Disabled &&
		slices.Equal(a.DisabledTools, b.DisabledTools) &&
		slices.Equal(a.EnabledTools, b.EnabledTools) &&
		a.Timeout == b.Timeout &&
		maps.Equal(a.Headers, b.Headers)
}
