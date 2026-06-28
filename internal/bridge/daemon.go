package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync"
	"time"
)

const defaultDaemonAddr = "127.0.0.1:8787"

type Daemon struct {
	Addr     string
	Interval time.Duration

	mu     sync.RWMutex
	models []LocalModel
	server *http.Server
}

func NewDaemon(addr string, interval time.Duration) *Daemon {
	if addr == "" {
		addr = defaultDaemonAddr
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Daemon{Addr: addr, Interval: interval}
}

func (d *Daemon) Run(ctx context.Context) error {
	if err := d.refresh(ctx); err != nil {
		slog.Warn("Initial bridge detection failed", "error", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", d.handleHealth)
	mux.HandleFunc("GET /v1/models", d.handleModels)
	mux.HandleFunc("GET /v1/bridge/models", d.handleModels)

	d.server = &http.Server{Addr: d.Addr, Handler: mux}

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.server.ListenAndServe()
	}()

	ticker := time.NewTicker(d.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return d.server.Shutdown(shutdownCtx)
		case <-ticker.C:
			if err := d.refresh(ctx); err != nil {
				slog.Warn("Bridge refresh failed", "error", err)
			}
		case err := <-errCh:
			if err == nil || err == http.ErrServerClosed {
				return nil
			}
			return fmt.Errorf("bridge server error: %w", err)
		}
	}
}

func (d *Daemon) refresh(ctx context.Context) error {
	models, err := Detect(ctx)
	if err != nil {
		return err
	}

	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })

	d.mu.Lock()
	d.models = models
	d.mu.Unlock()

	slog.Info("Bridge model list refreshed", "count", len(models))
	return nil
}

func (d *Daemon) currentModels() []LocalModel {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]LocalModel, len(d.models))
	copy(out, d.models)
	return out
}

func (d *Daemon) handleHealth(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (d *Daemon) handleModels(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(struct {
		Models []LocalModel `json:"models"`
	}{Models: d.currentModels()})
}
