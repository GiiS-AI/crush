package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectLMStudio(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/models", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data":   []map[string]any{{"id": "gemma"}},
		})
	}))
	t.Cleanup(ts.Close)

	models, err := detectLMStudio(context.Background(), ts.URL+"/v1")
	require.NoError(t, err)
	require.Len(t, models, 1)
	require.Equal(t, "lmstudio/gemma", models[0].ID)
}

func TestDetectOllama(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/tags", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{{"name": "llama3"}},
		})
	}))
	t.Cleanup(ts.Close)

	models, err := detectOllama(context.Background(), ts.URL)
	require.NoError(t, err)
	require.Len(t, models, 1)
	require.Equal(t, "ollama/llama3", models[0].ID)
}

func TestDetectIgnoresOfflineProviders(t *testing.T) {
	oldRunner := commandRunner
	oldLM := lmStudioBaseURL
	oldOllama := ollamaBaseURL
	oldShim := giisShimBaseURL
	commandRunner = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "exit 1")
	}
	lmStudioBaseURL = "http://127.0.0.1:1/v1"
	ollamaBaseURL = "http://127.0.0.1:1"
	giisShimBaseURL = "http://127.0.0.1:1/v1"
	t.Cleanup(func() {
		commandRunner = oldRunner
		lmStudioBaseURL = oldLM
		ollamaBaseURL = oldOllama
		giisShimBaseURL = oldShim
	})

	models, err := Detect(context.Background())
	require.NoError(t, err)
	require.Empty(t, models)
}
