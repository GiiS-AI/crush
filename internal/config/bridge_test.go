package config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/require"
)

type mockBridgeClient struct {
	models []catwalk.Model
	err    error
}

func (m *mockBridgeClient) GetModels(context.Context) ([]catwalk.Model, error) {
	return m.models, m.err
}

func TestBridgeSync_Get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/models", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{{"id": "lmstudio/gemma", "name": "Gemma"}},
		})
	}))
	t.Cleanup(ts.Close)
	t.Setenv("GIIS_BRIDGE_URL", ts.URL+"/v1")

	syncer := &bridgeSync{}
	syncer.Init(&mockBridgeClient{models: []catwalk.Model{{ID: "lmstudio/gemma", Name: "Gemma"}}}, t.TempDir()+"/giis-bridge.json", true)

	provider, err := syncer.Get(t.Context())
	require.NoError(t, err)
	require.Equal(t, "giis-bridge", string(provider.ID))
	require.NotEmpty(t, provider.Models)
	require.Equal(t, "lmstudio/gemma", provider.Models[0].ID)
}
