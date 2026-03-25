package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/go-chi/chi/v5"
)

func feedsRouter(indexer *stubIndexer, renderer *stubRenderer) chi.Router {
	r := chi.NewRouter()
	m := NewPostFeedModule(indexer, renderer)
	_ = m.RegisterRoutes(r, config.NewConfigLoaderFromConfig(config.Config{}))
	return r
}

func TestPostFeedModule_ListPostsDefault(t *testing.T) {
	posts := []*service.Post{
		newStubPost("first", "First Post"),
		newStubPost("second", "Second Post"),
	}
	indexer := &stubIndexer{posts: posts}
	renderer := &stubRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()
	feedsRouter(indexer, renderer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if renderer.servedListCount != 1 {
		t.Errorf("want ServePostList called once, got %d", renderer.servedListCount)
	}
}

func TestPostFeedModule_PaginationParameter(t *testing.T) {
	indexer := &stubIndexer{}
	renderer := &stubRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/posts?page=2", nil)
	w := httptest.NewRecorder()
	feedsRouter(indexer, renderer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestPostFeedModule_TagFilterParameter(t *testing.T) {
	indexer := &stubIndexer{}
	renderer := &stubRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/posts?tags=go", nil)
	w := httptest.NewRecorder()
	feedsRouter(indexer, renderer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}
