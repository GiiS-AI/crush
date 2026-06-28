package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"sort"
	"time"
)

const (
	lmStudioDefaultBaseURL = "http://127.0.0.1:1234/v1"
	ollamaDefaultBaseURL   = "http://127.0.0.1:11434"
	giisShimDefaultBaseURL = "http://127.0.0.1:8765/v1"
)

var (
	lmStudioBaseURL = lmStudioDefaultBaseURL
	ollamaBaseURL   = ollamaDefaultBaseURL
	giisShimBaseURL = giisShimDefaultBaseURL
	commandRunner   = exec.CommandContext
)

type LocalModel struct {
	Provider  string `json:"provider"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Source    string `json:"source"`
}

func Detect(ctx context.Context) ([]LocalModel, error) {
	var models []LocalModel
	for _, detector := range []func(context.Context) ([]LocalModel, error){
		func(ctx context.Context) ([]LocalModel, error) { return detectLMStudio(ctx, lmStudioBaseURL) },
		func(ctx context.Context) ([]LocalModel, error) { return detectOllama(ctx, ollamaBaseURL) },
		func(ctx context.Context) ([]LocalModel, error) { return detectGiiSShim(ctx, giisShimBaseURL) },
		detectCodexCLI,
		detectClaudeCLI,
	} {
		found, err := detector(ctx)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			continue
		}
		models = append(models, found...)
	}

	models = dedupe(models)
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

func detectLMStudio(ctx context.Context, baseURL string) ([]LocalModel, error) {
	var resp struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := getJSON(ctx, baseURL+"/models", &resp); err != nil {
		return nil, err
	}
	models := make([]LocalModel, 0, len(resp.Data))
	for _, m := range resp.Data {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		models = append(models, LocalModel{
			Provider:  "lmstudio",
			ID:        "lmstudio/" + m.ID,
			Name:      name,
			Available: true,
			Source:    baseURL,
		})
	}
	return models, nil
}

func detectOllama(ctx context.Context, baseURL string) ([]LocalModel, error) {
	var resp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := getJSON(ctx, baseURL+"/api/tags", &resp); err != nil {
		return nil, err
	}
	models := make([]LocalModel, 0, len(resp.Models))
	for _, m := range resp.Models {
		if m.Name == "" {
			continue
		}
		models = append(models, LocalModel{
			Provider:  "ollama",
			ID:        "ollama/" + m.Name,
			Name:      m.Name,
			Available: true,
			Source:    baseURL,
		})
	}
	return models, nil
}

func detectGiiSShim(ctx context.Context, baseURL string) ([]LocalModel, error) {
	var resp struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := getJSON(ctx, baseURL+"/models", &resp); err != nil {
		return nil, err
	}
	models := make([]LocalModel, 0, len(resp.Data))
	for _, m := range resp.Data {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		models = append(models, LocalModel{
			Provider:  "giis-local",
			ID:        "giis-local/" + m.ID,
			Name:      name,
			Available: true,
			Source:    baseURL,
		})
	}
	return models, nil
}

func detectCodexCLI(ctx context.Context) ([]LocalModel, error) {
	if _, err := commandRunner(ctx, "codex", "--version").Output(); err != nil {
		return nil, err
	}
	return []LocalModel{{
		Provider:  "codex",
		ID:        "giis-local/codex",
		Name:      "Codex",
		Available: true,
		Source:    "codex",
	}}, nil
}

func detectClaudeCLI(ctx context.Context) ([]LocalModel, error) {
	if _, err := commandRunner(ctx, "claude", "--version").Output(); err != nil {
		return nil, err
	}
	return []LocalModel{{
		Provider:  "claude-code",
		ID:        "giis-local/claude-code",
		Name:      "Claude Code",
		Available: true,
		Source:    "claude",
	}}, nil
}

func getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func dedupe(models []LocalModel) []LocalModel {
	seen := make(map[string]struct{}, len(models))
	out := make([]LocalModel, 0, len(models))
	for _, m := range models {
		if m.ID == "" {
			continue
		}
		if _, ok := seen[m.ID]; ok {
			continue
		}
		seen[m.ID] = struct{}{}
		out = append(out, m)
	}
	return out
}
