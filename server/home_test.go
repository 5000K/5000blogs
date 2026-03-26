package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/go-chi/chi/v5"
)

func homeRouter(indexer *stubIndexer, renderer *stubRenderer) chi.Router {
	r := chi.NewRouter()
	m := NewHomeModule(indexer, nil, renderer)
	_ = m.RegisterRoutes(r, config.NewConfigLoaderFromConfig(config.Config{SiteURL: "http://example.com"}))
	return r
}

func TestHomeModule_ServesPostListWhenNoIndexPost(t *testing.T) {
	posts := []*service.Post{newStubPost("hello", "Hello")}
	indexer := &stubIndexer{posts: posts}
	renderer := &stubRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	homeRouter(indexer, renderer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if renderer.servedListCount != 1 {
		t.Errorf("want ServePostList called once, got %d", renderer.servedListCount)
	}
}

func TestHomeModule_ServesIndexPostWhenPresent(t *testing.T) {
	raw := []byte("---\ntitle: Home\n---\n\n# Welcome\n\nThis is the home page.\n")
	indexPost, err := convertedPost("index", raw)
	if err != nil {
		t.Fatalf("convertedPost: %v", err)
	}
	indexer := &stubIndexer{posts: []*service.Post{indexPost}}
	renderer := &stubRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	homeRouter(indexer, renderer).ServeHTTP(w, req)

	if renderer.servedPost == nil || renderer.servedPost.Data().Slug != "index" {
		t.Errorf("want index post to be served, got %v", renderer.servedPost)
	}
}
