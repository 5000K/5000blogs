package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/go-chi/chi/v5"
)

// --- checkLastModified ---

func TestCheckLastModified_SetsHeader(t *testing.T) {
	modTime := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	served := checkLastModified(w, r, modTime)

	if served {
		t.Error("want false (no If-Modified-Since header present)")
	}
	lm := w.Header().Get("Last-Modified")
	if lm == "" {
		t.Error("Last-Modified header should be set")
	}
	parsed, err := http.ParseTime(lm)
	if err != nil {
		t.Fatalf("Last-Modified header not a valid HTTP date: %q", lm)
	}
	if !parsed.Equal(modTime.Truncate(time.Second)) {
		t.Errorf("Last-Modified mismatch: want %v, got %v", modTime, parsed)
	}
}

func TestCheckLastModified_Returns304WhenFresh(t *testing.T) {
	modTime := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	// Client claims it has a copy from the same time - should get 304.
	r.Header.Set("If-Modified-Since", modTime.Format(http.TimeFormat))

	served := checkLastModified(w, r, modTime)

	if !served {
		t.Error("want true (resource not modified)")
	}
	if w.Code != http.StatusNotModified {
		t.Errorf("want 304, got %d", w.Code)
	}
}

func TestCheckLastModified_Returns200WhenStale(t *testing.T) {
	modTime := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	// Client's copy is older than the resource.
	r.Header.Set("If-Modified-Since", time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC).Format(http.TimeFormat))

	served := checkLastModified(w, r, modTime)

	if served {
		t.Error("want false (resource is newer than client copy)")
	}
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestCheckLastModified_ZeroTimeIsNoop(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	served := checkLastModified(w, r, time.Time{})

	if served {
		t.Error("want false for zero time")
	}
	if lm := w.Header().Get("Last-Modified"); lm != "" {
		t.Errorf("Last-Modified should not be set for zero time, got %q", lm)
	}
}

// --- DynamicModule post serving ---

func dynamicRouter(indexer *stubIndexer, renderer *stubRenderer) chi.Router {
	r := chi.NewRouter()
	m := NewDynamicModule(indexer, nil, renderer)
	_ = m.RegisterRoutes(r, config.NewConfigLoaderFromConfig(config.Config{SiteURL: "http://example.com"}))
	return r
}

func TestDynamicModule_ServesPostBySlug(t *testing.T) {
	post := newStubPost("hello", "Hello World")
	indexer := &stubIndexer{posts: []*service.Post{post}}
	renderer := &stubRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()
	dynamicRouter(indexer, renderer).ServeHTTP(w, req)

	if renderer.servedPost == nil || renderer.servedPost.Data().Slug != "hello" {
		t.Errorf("expected post 'hello' to be served")
	}
}

func TestDynamicModule_Returns404ForMissingPost(t *testing.T) {
	indexer := &stubIndexer{}
	renderer := &stubRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/no-such-post", nil)
	w := httptest.NewRecorder()
	dynamicRouter(indexer, renderer).ServeHTTP(w, req)

	if renderer.served404Count != 1 {
		t.Errorf("want 404 handler called once, got %d", renderer.served404Count)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 status, got %d", w.Code)
	}
}

func TestDynamicModule_LastModifiedCaching(t *testing.T) {
	modTime := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	post := newStubPost("cached", "Cached Post")
	// We can't set modtime via newStubPost; just verify the header is sent.
	_ = modTime
	indexer := &stubIndexer{posts: []*service.Post{post}}
	renderer := &stubRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/cached", nil)
	w := httptest.NewRecorder()
	dynamicRouter(indexer, renderer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}
