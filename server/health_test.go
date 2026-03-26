package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/5000K/5000blogs/config"
	"github.com/go-chi/chi/v5"
)

func healthRouter() chi.Router {
	r := chi.NewRouter()
	m := NewHealthModule()
	_ = m.RegisterRoutes(r, config.NewConfigLoaderFromConfig(config.Config{}))
	return r
}

func TestHealthModule_Health(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	healthRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if body := w.Body.String(); body != "ok" {
		t.Errorf("want body 'ok', got %q", body)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("want text/plain content type, got %q", ct)
	}
}

func TestHealthModule_Healthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	healthRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if body := w.Body.String(); body != "OK" {
		t.Errorf("want body 'OK', got %q", body)
	}
}
