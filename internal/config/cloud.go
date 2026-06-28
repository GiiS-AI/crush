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

const defaultGiiSCloudURL = "https://chat.giis.ai/api/v1"

type cloudClient interface {
	GetModels(context.Context) ([]catwalk.Model, error)
}

var _ syncer[catwalk.Provider] = (*cloudSync)(nil)

type cloudSync struct {
	once       sync.Once
	result     catwalk.Provider
	cache      cache[catwalk.Provider]
	client     cloudClient
	autoupdate bool
	init       atomic.Bool
}

func (s *cloudSync) Init(client cloudClient, path string, autoupdate bool) {
	s.client = client
	s.cache = newCache[catwalk.Provider](path)
	s.autoupdate = autoupdate
	s.init.Store(true)
}

func (s *cloudSync) Get(ctx context.Context) (catwalk.Provider, error) {
	if !s.init.Load() {
		panic("called Get before Init")
	}

	var throwErr error
	s.once.Do(func() {
		if !s.autoupdate {
			slog.Info("Using cached GiiS Cloud provider")
			cached, _, err := s.cache.Get()
			if err == nil && cached.ID != "" {
				s.result = cached
			}
			return
		}

		cached, _, cachedErr := s.cache.Get()
		if cached.ID == "" || cachedErr != nil {
			cached = catwalk.Provider{
				ID:          catwalk.InferenceProvider("giis-cloud"),
				Name:        "GiiS Cloud",
				Type:        catwalk.TypeOpenAICompat,
				APIEndpoint: cloudBaseURL(),
			}
		}

		models, err := s.client.GetModels(ctx)
		if err != nil {
			s.result = cached
			return
		}
		if len(models) == 0 {
			s.result = cached
			throwErr = errors.New("empty model list from giis cloud")
			return
		}

		s.result = catwalk.Provider{
			Name:        "GiiS Cloud",
			ID:          catwalk.InferenceProvider("giis-cloud"),
			APIEndpoint: cloudBaseURL(),
			Type:        catwalk.TypeOpenAICompat,
			Models:      models,
		}
		throwErr = s.cache.Store(s.result)
	})
	return s.result, throwErr
}

type realCloudClient struct{}

func (realCloudClient) GetModels(ctx context.Context) ([]catwalk.Model, error) {
	var resp struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := getJSONCloud(ctx, cloudBaseURL()+"/models", &resp); err != nil {
		return nil, fmt.Errorf("failed to fetch models from GiiS Cloud: %w", err)
	}
	models := make([]catwalk.Model, 0, len(resp.Data))
	for _, m := range resp.Data {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		models = append(models, catwalk.Model{ID: m.ID, Name: name})
	}
	return models, nil
}

func cloudBaseURL() string {
	if v := os.Getenv("GIIS_CLOUD_URL"); v != "" {
		return v
	}
	return defaultGiiSCloudURL
}

func cloudCachePath() string {
	return cachePathFor("giis-cloud")
}

func getJSONCloud(ctx context.Context, url string, out any) error {
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
