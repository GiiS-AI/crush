package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDaemonModelsEndpoint(t *testing.T) {
	t.Parallel()

	d := NewDaemon("127.0.0.1:0", time.Second)
	d.mu.Lock()
	d.models = []LocalModel{{Provider: "lmstudio", ID: "lmstudio/gemma", Name: "gemma", Available: true, Source: "http://127.0.0.1:1234/v1"}}
	d.mu.Unlock()

	ts := httptest.NewServer(http.HandlerFunc(d.handleModels))
	t.Cleanup(ts.Close)

	rsp, err := http.Get(ts.URL)
	require.NoError(t, err)
	defer rsp.Body.Close()

	var payload struct {
		Models []LocalModel `json:"models"`
	}
	require.NoError(t, json.NewDecoder(rsp.Body).Decode(&payload))
	require.Len(t, payload.Models, 1)
	require.Equal(t, "lmstudio/gemma", payload.Models[0].ID)
}
