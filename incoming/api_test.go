package incoming

import (
	"5000blogs/service"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// stubRepo is a minimal in-memory PostRepository for handler tests.
type stubRepo struct {
	posts []*service.Post
}

func (r *stubRepo) Get(path string) *service.Post {
	for _, p := range r.posts {
		if p.Data().Slug == path {
			return p
		}
	}
	return nil
}

func (r *stubRepo) GetBySlug(slug string) *service.Post {
	for _, p := range r.posts {
		if p.Data().Slug == slug {
			return p
		}
	}
	return nil
}

func (r *stubRepo) List() []*service.Post                      { return r.posts }
func (r *stubRepo) Count() int                                 { return len(r.posts) }
func (r *stubRepo) GetPage(int, []string) service.PageResult   { return service.PageResult{} }
func (r *stubRepo) AllTags() []string                          { return nil }
func (r *stubRepo) FeedPosts([]string, string) []*service.Post { return r.posts }
func (r *stubRepo) LastModified() time.Time                    { return time.Time{} }
func (r *stubRepo) Sitemap() []service.SitemapEntry            { return nil }
func (r *stubRepo) Start() error                               { return nil }
func (r *stubRepo) Stop()                                      {}
func (r *stubRepo) ReadMedia(_ string) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (r *stubRepo) Search(query string) []service.PostSummary {
	if query == "" {
		return []service.PostSummary{}
	}
	q := strings.ToLower(query)
	var out []service.PostSummary
	for _, p := range r.posts {
		if !p.IsVisible() {
			continue
		}
		d := p.Data()
		if strings.Contains(strings.ToLower(d.Title), q) ||
			strings.Contains(strings.ToLower(d.Description), q) {
			out = append(out, service.PostSummary{Slug: d.Slug, Title: d.Title, Description: d.Description})
		}
	}
	return out
}

func newPost(slug, title, description string) *service.Post {
	return service.NewPost(slug+".md", &service.Metadata{
		Title:       title,
		Description: description,
	}, []byte("<p>content</p>"))
}

func doRequest(t *testing.T, repo service.PostRepository, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	apiRouter(repo).ServeHTTP(w, req)
	return w
}

// --- GET /posts ---

func TestAPIListPosts_Empty(t *testing.T) {
	repo := &stubRepo{}
	w := doRequest(t, repo, http.MethodGet, "/posts")

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var slugs []string
	if err := json.Unmarshal(w.Body.Bytes(), &slugs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(slugs) != 0 {
		t.Errorf("want empty list, got %v", slugs)
	}
}

func TestAPIListPosts_ReturnsSlugs(t *testing.T) {
	repo := &stubRepo{posts: []*service.Post{
		newPost("hello", "Hello", ""),
		newPost("world", "World", ""),
	}}
	w := doRequest(t, repo, http.MethodGet, "/posts")

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var slugs []string
	if err := json.Unmarshal(w.Body.Bytes(), &slugs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(slugs) != 2 {
		t.Fatalf("want 2 slugs, got %v", slugs)
	}
	found := make(map[string]bool)
	for _, s := range slugs {
		found[s] = true
	}
	if !found["hello"] || !found["world"] {
		t.Errorf("unexpected slugs: %v", slugs)
	}
}

// --- GET /post/{name} ---

func TestAPIGetPost_Found(t *testing.T) {
	repo := &stubRepo{posts: []*service.Post{
		newPost("intro", "Introduction", "Welcome"),
	}}
	w := doRequest(t, repo, http.MethodGet, "/post/intro")

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode: %v – body: %s", err, w.Body.String())
	}
	if m["slug"] != "intro" {
		t.Errorf("slug: got %v", m["slug"])
	}
	if m["title"] != "Introduction" {
		t.Errorf("title: got %v", m["title"])
	}
	if m["description"] != "Welcome" {
		t.Errorf("description: got %v", m["description"])
	}
}

func TestAPIGetPost_NotFound(t *testing.T) {
	repo := &stubRepo{}
	w := doRequest(t, repo, http.MethodGet, "/post/nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestAPIGetPost_VisibilityFields(t *testing.T) {
	f := false
	post := service.NewPost("hidden.md", &service.Metadata{
		Title:   "Hidden",
		Visible: &f,
	}, []byte("<p>x</p>"))
	repo := &stubRepo{posts: []*service.Post{post}}

	w := doRequest(t, repo, http.MethodGet, "/post/hidden")
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var m map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	if m["visible"] != false {
		t.Errorf("want visible=false, got %v", m["visible"])
	}
}

// --- GET /posts/search ---

func TestAPISearch_MatchesTitle(t *testing.T) {
	repo := &stubRepo{posts: []*service.Post{
		newPost("go-tips", "Go Tips", "Useful tips"),
		newPost("rust-guide", "Rust Guide", "Learn Rust"),
	}}
	w := doRequest(t, repo, http.MethodGet, "/posts/search?q=go")

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d: %v", len(results), results)
	}
	if results[0]["slug"] != "go-tips" {
		t.Errorf("unexpected result: %v", results[0])
	}
}

func TestAPISearch_MatchesDescription(t *testing.T) {
	repo := &stubRepo{posts: []*service.Post{
		newPost("post-a", "First Post", "Discusses concurrency patterns"),
		newPost("post-b", "Second Post", "About web frameworks"),
	}}
	w := doRequest(t, repo, http.MethodGet, "/posts/search?q=concurrency")

	var results []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &results)
	if len(results) != 1 || results[0]["slug"] != "post-a" {
		t.Errorf("expected 1 match on description, got %v", results)
	}
}

func TestAPISearch_CaseInsensitive(t *testing.T) {
	repo := &stubRepo{posts: []*service.Post{
		newPost("hello", "Hello World", ""),
	}}
	w := doRequest(t, repo, http.MethodGet, "/posts/search?q=HELLO")

	var results []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &results)
	if len(results) != 1 {
		t.Errorf("want case-insensitive match, got %v", results)
	}
}

func TestAPISearch_NoMatch(t *testing.T) {
	repo := &stubRepo{posts: []*service.Post{
		newPost("hello", "Hello", ""),
	}}
	w := doRequest(t, repo, http.MethodGet, "/posts/search?q=zzz")

	var results []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &results)
	if len(results) != 0 {
		t.Errorf("want empty results, got %v", results)
	}
}

func TestAPISearch_ExcludesHiddenPosts(t *testing.T) {
	f := false
	repo := &stubRepo{posts: []*service.Post{
		newPost("visible-post", "Visible Post", ""),
		service.NewPost("hidden-post.md", &service.Metadata{Title: "Hidden Post", Visible: &f}, []byte("<p>x</p>")),
	}}
	w := doRequest(t, repo, http.MethodGet, "/posts/search?q=post")

	var results []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &results)
	if len(results) != 1 {
		t.Fatalf("want 1 result (hidden post excluded), got %d: %v", len(results), results)
	}
	if results[0]["slug"] != "visible-post" {
		t.Errorf("unexpected result: %v", results[0])
	}
}

func TestAPISearch_EmptyQuery(t *testing.T) {
	repo := &stubRepo{posts: []*service.Post{
		newPost("a", "Alpha", ""),
		newPost("b", "Beta", ""),
	}}
	w := doRequest(t, repo, http.MethodGet, "/posts/search?q=")

	var results []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &results)
	// Empty query matches everything (substring of any string).
	if len(results) != 2 {
		t.Errorf("empty query should match all posts, got %d", len(results))
	}
}

// --- GET /stats ---

func TestAPIStats_NoVisiblePosts(t *testing.T) {
	f := false
	repo := &stubRepo{posts: []*service.Post{
		service.NewPost("hidden.md", &service.Metadata{Visible: &f}, []byte("x")),
	}}
	w := doRequest(t, repo, http.MethodGet, "/stats")

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var s map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &s)
	if s["total_posts"].(float64) != 0 {
		t.Errorf("want total_posts=0 for all-hidden repo, got %v", s["total_posts"])
	}
	// latest_post_date is null (not omitted) when no visible posts have a date.
	if v, ok := s["latest_post_date"]; ok && v != nil {
		t.Errorf("latest_post_date should be null or absent, got %v", v)
	}
}

func TestAPIStats_CountsVisiblePosts(t *testing.T) {
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	f := false
	repo := &stubRepo{posts: []*service.Post{
		service.NewPost("a.md", &service.Metadata{Title: "A", Date: date}, []byte("x")),
		service.NewPost("b.md", &service.Metadata{Title: "B"}, []byte("x")),
		service.NewPost("c.md", &service.Metadata{Title: "C", Visible: &f}, []byte("x")),
	}}
	w := doRequest(t, repo, http.MethodGet, "/stats")

	var s map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &s)
	if s["total_posts"].(float64) != 2 {
		t.Errorf("want total_posts=2, got %v", s["total_posts"])
	}
	if _, hasDate := s["latest_post_date"]; !hasDate {
		t.Error("latest_post_date should be present")
	}
}

func TestAPIStats_ContentType(t *testing.T) {
	repo := &stubRepo{}
	w := doRequest(t, repo, http.MethodGet, "/stats")

	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: got %q", ct)
	}
}

// --- GET /posts/tags ---

func newPostWithTags(slug string, tags []string) *service.Post {
	return service.NewPost(slug+".md", &service.Metadata{
		Title: slug,
		Tags:  tags,
	}, []byte("<p>content</p>"))
}

func TestAPIListTags_Empty(t *testing.T) {
	repo := &stubRepo{}
	w := doRequest(t, repo, http.MethodGet, "/posts/tags")

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestAPIGetPost_IncludesTags(t *testing.T) {
	repo := &stubRepo{posts: []*service.Post{
		newPostWithTags("hello", []string{"go", "web"}),
	}}
	w := doRequest(t, repo, http.MethodGet, "/post/hello")

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var result map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	tags, ok := result["tags"]
	if !ok {
		t.Fatal("want 'tags' field in response")
	}
	tagList, ok := tags.([]interface{})
	if !ok || len(tagList) != 2 {
		t.Errorf("want 2 tags, got %v", tags)
	}
}

func TestAPIListPosts_TagFilter(t *testing.T) {
	repo := &stubRepo{posts: []*service.Post{
		newPostWithTags("alpha", []string{"go"}),
		newPostWithTags("beta", []string{"rust"}),
		newPostWithTags("gamma", []string{"go", "web"}),
	}}
	w := doRequest(t, repo, http.MethodGet, "/posts?tags=go")

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var slugs []string
	_ = json.Unmarshal(w.Body.Bytes(), &slugs)
	if len(slugs) != 2 {
		t.Errorf("want 2 posts tagged 'go', got %d: %v", len(slugs), slugs)
	}
	for _, s := range slugs {
		if s == "beta" {
			t.Error("'beta' (rust only) should be excluded from 'go' filter")
		}
	}
}

// --- GET /search ---

func TestAPISearch_ReturnsMatchingSlugs(t *testing.T) {
	repo := &stubRepo{posts: []*service.Post{
		newPost("hello", "Hello World", ""),
		newPost("goodbye", "Goodbye World", ""),
		newPost("other", "Unrelated", ""),
	}}
	w := doRequest(t, repo, http.MethodGet, "/search?q=world")

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var slugs []string
	_ = json.Unmarshal(w.Body.Bytes(), &slugs)
	if len(slugs) != 2 {
		t.Errorf("want 2 results for 'world', got %d: %v", len(slugs), slugs)
	}
	for _, s := range slugs {
		if s == "other" {
			t.Error("'other' should not appear in results")
		}
	}
}

func TestAPISearch_EmptyQueryReturnsEmptySlice(t *testing.T) {
	repo := &stubRepo{posts: []*service.Post{
		newPost("a", "Alpha", ""),
	}}
	w := doRequest(t, repo, http.MethodGet, "/search?q=")

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var slugs []string
	_ = json.Unmarshal(w.Body.Bytes(), &slugs)
	if len(slugs) != 0 {
		t.Errorf("empty query should return no results, got %v", slugs)
	}
}

func TestAPISearch_ContentType(t *testing.T) {
	repo := &stubRepo{}
	w := doRequest(t, repo, http.MethodGet, "/search?q=anything")

	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: got %q", ct)
	}
}
