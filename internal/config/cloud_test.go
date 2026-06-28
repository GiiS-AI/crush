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

type mockCloudClient struct {
	models []catwalk.Model
	err    error
}

func (m *mockCloudClient) GetModels(context.Context) ([]catwalk.Model, error) {
	return m.models, m.err
}

func TestCloudSync_Get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/models", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": "gpt-4o", "name": "GPT-4o"}},
		})
	}))
	t.Cleanup(ts.Close)
	t.Setenv("GIIS_CLOUD_URL", ts.URL+"/api/v1")

	syncer := &cloudSync{}
	syncer.Init(&mockCloudClient{models: []catwalk.Model{{ID: "gpt-4o", Name: "GPT-4o"}}}, t.TempDir()+"/giis-cloud.json", true)

	provider, err := syncer.Get(t.Context())
	require.NoError(t, err)
	require.Equal(t, "giis-cloud", string(provider.ID))
	require.NotEmpty(t, provider.Models)
	require.Equal(t, "gpt-4o", provider.Models[0].ID)
}
