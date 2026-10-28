package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cybozu-go/log"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/internal/metrics"
)

const (
	httpReadTimeout     = 30 * time.Second
	httpShutdownTimeout = 3 * time.Minute
	healthCheckTimeout  = 2 * time.Second
)

// restHandler handles the REST API of cke.
type restHandler struct {
	etcd *clientv3.Client
}

type version struct {
	Version string `json:"version"`
}

type health struct {
	Health string `json:"health"`
}

// runHTTPServer serves the metrics and the REST API on addr until ctx is
// canceled, then shuts down the server gracefully.
func runHTTPServer(ctx context.Context, addr string, etcd *clientv3.Client) error {
	mux := http.NewServeMux()
	collector := metrics.NewCollector(&cke.Storage{Client: etcd})
	mux.Handle("/metrics", metrics.GetHandler(collector))
	mux.Handle("/", restHandler{etcd: etcd})

	srv := &http.Server{
		Addr:        addr,
		Handler:     mux,
		ReadTimeout: httpReadTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func (h restHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		renderError(w, http.StatusMethodNotAllowed)
		return
	}

	switch r.URL.Path {
	case "/version":
		renderJSON(w, http.StatusOK, version{Version: cke.Version})
	case "/health":
		h.handleHealth(w, r)
	default:
		renderError(w, http.StatusNotFound)
	}
}

func (h restHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctxWithTimeout, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
	defer cancel()

	_, err := h.etcd.Get(ctxWithTimeout, "health")
	if err == nil || err == rpctypes.ErrPermissionDenied {
		renderJSON(w, http.StatusOK, health{Health: "healthy"})
	} else {
		renderJSON(w, http.StatusInternalServerError, health{Health: "unhealthy"})
	}
}

func renderJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Error("failed to output JSON", map[string]any{
			log.FnError: err.Error(),
		})
	}
}

func renderError(w http.ResponseWriter, status int) {
	renderJSON(w, status, map[string]any{
		"status": status,
		"error":  http.StatusText(status),
	})
}
