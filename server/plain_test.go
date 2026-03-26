package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/go-chi/chi/v5"
)

func plainRouter(indexer *stubIndexer) chi.Router {
	r := chi.NewRouter()
	m := NewPlainModule(indexer)
	_ = m.RegisterRoutes(r, config.NewConfigLoaderFromConfig(config.Config{}))
	return r
}

func doPlainRequest(t *testing.T, indexer *stubIndexer, slug string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/plain/"+slug, nil)
	w := httptest.NewRecorder()
	plainRouter(indexer).ServeHTTP(w, req)
	return w
}

func TestPlainModule_ReturnsPlainText(t *testing.T) {
	raw := []byte("---\ntitle: Hello\n---\n\n# Hello\n\nThis is a paragraph.\n")
	post, err := convertedPost("hello", raw)
	if err != nil {
		t.Fatalf("convertedPost: %v", err)
	}
	indexer := &stubIndexer{posts: []*service.Post{post}}

	w := doPlainRequest(t, indexer, "hello")

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("want text/plain content type, got %q", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Hello") {
		t.Errorf("want 'Hello' in body, got %q", body)
	}
	if !strings.Contains(body, "This is a paragraph") {
		t.Errorf("want paragraph text in body, got %q", body)
	}
	if strings.Contains(body, "<") || strings.Contains(body, ">") {
		t.Errorf("body must not contain HTML tags: %q", body)
	}
}

func TestPlainModule_NotFound(t *testing.T) {
	indexer := &stubIndexer{}
	w := doPlainRequest(t, indexer, "missing")
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestPlainModule_NestedSlug(t *testing.T) {
	raw := []byte("---\ntitle: Nested\n---\n\n# Nested\n\nContent here.\n")
	post, err := convertedPost("more/things/hello", raw)
	if err != nil {
		t.Fatalf("convertedPost: %v", err)
	}
	indexer := &stubIndexer{posts: []*service.Post{post}}

	w := doPlainRequest(t, indexer, "more/things/hello")
	if w.Code != http.StatusOK {
		t.Errorf("want 200 for nested slug, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Nested") {
		t.Errorf("want 'Nested' in body, got %q", w.Body.String())
	}
}

func TestPlainModule_PostWithoutPlainText(t *testing.T) {
	post := newStubPost("no-plain", "No Plain")
	indexer := &stubIndexer{posts: []*service.Post{post}}

	w := doPlainRequest(t, indexer, "no-plain")
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 when plain text unavailable, got %d", w.Code)
	}
}
