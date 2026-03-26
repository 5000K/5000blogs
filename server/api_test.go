package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/go-chi/chi/v5"
)

func apiRouter(indexer *stubIndexer) chi.Router {
	r := chi.NewRouter()
	m := NewApiModule(indexer)
	_ = m.RegisterRoutes(r, config.NewConfigLoaderFromConfig(config.Config{PageSize: 10}))
	return r
}

func TestApiListPosts_ReturnsAllSlugs(t *testing.T) {
	indexer := &stubIndexer{posts: []*service.Post{
		newStubPost("alpha", "Alpha"),
		newStubPost("beta", "Beta"),
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var slugs []string
	if err := json.Unmarshal(w.Body.Bytes(), &slugs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(slugs) != 2 {
		t.Errorf("want 2 slugs, got %d", len(slugs))
	}
}

func TestApiListPosts_FiltersByTag(t *testing.T) {
	indexer := &stubIndexer{posts: []*service.Post{
		service.NewPost("go-post.md", &service.Metadata{Title: "Go Post", Tags: []string{"go"}}, nil),
		service.NewPost("rust-post.md", &service.Metadata{Title: "Rust Post", Tags: []string{"rust"}}, nil),
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?tags=go", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	var slugs []string
	json.Unmarshal(w.Body.Bytes(), &slugs) //nolint:errcheck
	if len(slugs) != 1 || slugs[0] != "go-post" {
		t.Errorf("want [go-post], got %v", slugs)
	}
}

func TestApiListPosts_FiltersByQuery(t *testing.T) {
	indexer := &stubIndexer{posts: []*service.Post{
		service.NewPost("hello.md", &service.Metadata{Title: "Hello World"}, nil),
		service.NewPost("other.md", &service.Metadata{Title: "Something Else"}, nil),
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?q=hello", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	var slugs []string
	json.Unmarshal(w.Body.Bytes(), &slugs) //nolint:errcheck
	if len(slugs) != 1 || slugs[0] != "hello" {
		t.Errorf("want [hello], got %v", slugs)
	}
}

func TestApiListPosts_EmptyReturnsEmptyArray(t *testing.T) {
	indexer := &stubIndexer{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	body := w.Body.String()
	// must be a JSON array, not null
	if body[0] != '[' {
		t.Errorf("want JSON array, got %q", body)
	}
}

func TestApiListPostsPaged_ReturnsPaginatedResult(t *testing.T) {
	posts := make([]*service.Post, 15)
	for i := range posts {
		posts[i] = service.NewPost("post.md", &service.Metadata{Title: "Post"}, nil)
	}
	indexer := &stubIndexer{posts: posts}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/page/1", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var result pagedPostsJSON
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if result.TotalPosts != 15 {
		t.Errorf("want TotalPosts=15, got %d", result.TotalPosts)
	}
	if result.Page != 1 {
		t.Errorf("want Page=1, got %d", result.Page)
	}
}

func TestApiListPostsPaged_InvalidPageDefaultsToOne(t *testing.T) {
	indexer := &stubIndexer{posts: []*service.Post{newStubPost("a", "A")}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/page/abc", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestApiGetPost_ReturnsMetadata(t *testing.T) {
	date := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	p := service.NewPost("hello-world.md", &service.Metadata{
		Title:       "Hello World",
		Description: "My first post",
		Date:        date,
		Author:      "Jane",
		Tags:        []string{"go", "tutorial"},
	}, nil)
	indexer := &stubIndexer{posts: []*service.Post{p}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/post/hello-world", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var meta postMetaJSON
	if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if meta.Slug != "hello-world" {
		t.Errorf("want slug=hello-world, got %q", meta.Slug)
	}
	if meta.Title != "Hello World" {
		t.Errorf("want title=Hello World, got %q", meta.Title)
	}
	if meta.Author != "Jane" {
		t.Errorf("want author=Jane, got %q", meta.Author)
	}
	if len(meta.Tags) != 2 {
		t.Errorf("want 2 tags, got %d", len(meta.Tags))
	}
	if meta.Visible != true {
		t.Errorf("want visible=true")
	}
}

func TestApiGetPost_NotFound(t *testing.T) {
	indexer := &stubIndexer{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/post/missing", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestApiGetPost_TagsNotNull(t *testing.T) {
	p := service.NewPost("no-tags.md", &service.Metadata{Title: "No Tags"}, nil)
	indexer := &stubIndexer{posts: []*service.Post{p}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/post/no-tags", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	var meta postMetaJSON
	json.Unmarshal(w.Body.Bytes(), &meta) //nolint:errcheck
	if meta.Tags == nil {
		t.Error("want non-nil tags slice, got nil")
	}
}

func TestApiListTags_ReturnsTags(t *testing.T) {
	posts := []*service.Post{
		service.NewPost("a.md", &service.Metadata{Tags: []string{"go", "web"}}, nil),
		service.NewPost("b.md", &service.Metadata{Tags: []string{"go"}}, nil),
	}
	indexer := &stubIndexer{posts: posts}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	// tags from stubIndexer.AllTags returns nil, normalized to []
	var tags []string
	if err := json.Unmarshal(w.Body.Bytes(), &tags); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
}

func TestApiListTags_NeverNull(t *testing.T) {
	indexer := &stubIndexer{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	body := w.Body.String()
	if body[0] != '[' {
		t.Errorf("want JSON array, got %q", body)
	}
}

func TestApiGetStats_ReturnsStats(t *testing.T) {
	now := time.Now()
	indexer := &stubIndexer{
		posts:        []*service.Post{newStubPost("a", "A"), newStubPost("b", "B")},
		lastModified: now,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	apiRouter(indexer).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var stats statsJSON
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if stats.VisiblePostCount != 2 {
		t.Errorf("want VisiblePostCount=2, got %d", stats.VisiblePostCount)
	}
}

func TestApiContentTypeIsJSON(t *testing.T) {
	indexer := &stubIndexer{}
	endpoints := []string{
		"/api/v1/posts",
		"/api/v1/posts/page/1",
		"/api/v1/tags",
		"/api/v1/stats",
	}
	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		w := httptest.NewRecorder()
		apiRouter(indexer).ServeHTTP(w, req)
		ct := w.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("%s: want Content-Type application/json, got %q", ep, ct)
		}
	}
}
