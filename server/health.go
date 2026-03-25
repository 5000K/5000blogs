package server

import (
	"net/http"

	"github.com/5000K/5000blogs/config"
	"github.com/go-chi/chi/v5"
)

type HealthModule struct{}

func NewHealthModule() *HealthModule {
	return &HealthModule{}
}

func (m *HealthModule) RegisterRoutes(r chi.Router, conf *config.ConfigLoader) error {
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return nil
}
