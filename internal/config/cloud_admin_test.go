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

func TestSyncBridgeProvider_SendsProviderUpsert(t *testing.T) {
	var gotAuth string
	var gotReq cloudLLMProviderUpsertRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		require.Equal(t, "/api/admin/llm/provider", r.URL.Path)
		gotAuth = r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotReq))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ts.Close)
	t.Setenv("GIIS_CLOUD_URL", ts.URL+"/api")

	err := SyncBridgeProvider(context.Background(), "bridge-token", "http://127.0.0.1:8787/v1", []catwalk.Model{{ID: "lmstudio/gemma", Name: "Gemma"}})
	require.NoError(t, err)
	require.Equal(t, "Bearer bridge-token", gotAuth)
	require.Equal(t, "openai-compatible", gotReq.Provider)
	require.Equal(t, "GiiS Bridge", *gotReq.Name)
	require.Equal(t, "http://127.0.0.1:8787/v1", *gotReq.APIBase)
	require.Len(t, gotReq.ModelConfigurations, 1)
	require.Equal(t, "lmstudio/gemma", gotReq.ModelConfigurations[0].Name)
	require.True(t, gotReq.ModelConfigurations[0].IsVisible)
	require.NotNil(t, gotReq.ModelConfigurations[0].DisplayName)
	require.Equal(t, "Gemma", *gotReq.ModelConfigurations[0].DisplayName)
}

func TestLoginCloudAndCreatePAT(t *testing.T) {
	var gotLogin bool
	var gotTokenReq cloudCreateTokenRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/login":
			require.Equal(t, http.MethodPost, r.Method)
			require.NoError(t, r.ParseForm())
			require.Equal(t, "password", r.Form.Get("grant_type"))
			require.Equal(t, "user@example.com", r.Form.Get("username"))
			require.Equal(t, "secret", r.Form.Get("password"))
			gotLogin = true
			http.SetCookie(w, &http.Cookie{Name: "fastapiusersauth", Value: "session-cookie", Path: "/"})
			w.WriteHeader(http.StatusOK)
		case "/api/user/pats":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "session-cookie", cookieValue(r, "fastapiusersauth"))
			require.NoError(t, json.NewDecoder(r.Body).Decode(&gotTokenReq))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(cloudCreatedPATResponse{
				cloudPATResponse: cloudPATResponse{
					ID:           42,
					Name:         gotTokenReq.Name,
					TokenDisplay: "giis-code-42",
					CreatedAt:    "2026-01-01T00:00:00Z",
				},
				Token: "created-pat",
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(ts.Close)
	t.Setenv("GIIS_CLOUD_URL", ts.URL+"/api/v1")

	tok, err := LoginCloudAndCreatePAT(context.Background(), "user@example.com", "secret", "giis-code", nil)
	require.NoError(t, err)
	require.True(t, gotLogin)
	require.Equal(t, "giis-code", gotTokenReq.Name)
	require.Equal(t, "created-pat", tok.Token)
	require.Equal(t, 42, tok.ID)
}

func cookieValue(r *http.Request, name string) string {
	for _, c := range r.Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}
