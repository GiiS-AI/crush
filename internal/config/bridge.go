package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"charm.land/catwalk/pkg/catwalk"
)

const defaultGiiSBridgeURL = "http://127.0.0.1:8787/v1"

type bridgeClient interface {
	GetModels(context.Context) ([]catwalk.Model, error)
}

var _ syncer[catwalk.Provider] = (*bridgeSync)(nil)

type bridgeSync struct {
	once       sync.Once
	result     catwalk.Provider
	cache      cache[catwalk.Provider]
	client     bridgeClient
	autoupdate bool
	init       atomic.Bool
}

func (s *bridgeSync) Init(client bridgeClient, path string, autoupdate bool) {
	s.client = client
	s.cache = newCache[catwalk.Provider](path)
	s.autoupdate = autoupdate
	s.init.Store(true)
}

func (s *bridgeSync) Get(ctx context.Context) (catwalk.Provider, error) {
	if !s.init.Load() {
		panic("called Get before Init")
	}

	var throwErr error
	s.once.Do(func() {
		cached, _, cachedErr := s.cache.Get()
		if !s.autoupdate {
			slog.Info("Using cached GiiS Bridge provider")
			if cachedErr == nil && cached.ID != "" {
				s.result = cached
			}
			return
		}

		if cached.ID == "" || cachedErr != nil {
			cached = catwalk.Provider{
				ID:          catwalk.InferenceProvider("giis-bridge"),
				Name:        "GiiS Bridge",
				Type:        catwalk.TypeOpenAICompat,
				APIEndpoint: bridgeBaseURL(),
			}
		}

		models, err := s.client.GetModels(ctx)
		if err != nil {
			if cached.ID != "" {
				s.result = cached
				return
			}
			throwErr = err
			return
		}
		if len(models) == 0 {
			if cached.ID != "" {
				s.result = cached
				return
			}
			throwErr = errors.New("empty model list from giis bridge")
			return
		}

		s.result = catwalk.Provider{
			Name:        "GiiS Bridge",
			ID:          catwalk.InferenceProvider("giis-bridge"),
			APIEndpoint: bridgeBaseURL(),
			Type:        catwalk.TypeOpenAICompat,
			Models:      models,
		}
		throwErr = s.cache.Store(s.result)
	})
	return s.result, throwErr
}

type realBridgeClient struct{}

func (realBridgeClient) GetModels(ctx context.Context) ([]catwalk.Model, error) {
	var resp struct {
		Models []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := getJSONBridge(ctx, bridgeBaseURL()+"/models", &resp); err != nil {
		return nil, fmt.Errorf("failed to fetch models from GiiS Bridge: %w", err)
	}
	models := make([]catwalk.Model, 0, len(resp.Models))
	for _, m := range resp.Models {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		models = append(models, catwalk.Model{ID: m.ID, Name: name})
	}
	return models, nil
}

func bridgeBaseURL() string {
	if v := os.Getenv("GIIS_BRIDGE_URL"); v != "" {
		return v
	}
	return defaultGiiSBridgeURL
}

func bridgeCachePath() string {
	return cachePathFor("giis-bridge")
}

func getJSONBridge(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
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
