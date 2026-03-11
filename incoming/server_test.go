package incoming

import (
	"5000blogs/config"
	"5000blogs/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// plainRouter is a minimal router containing only the /plain/* handler,
// used to test that route in isolation without a full config/renderer setup.
func plainRouter(repo service.PostRepository) chi.Router {
	r := chi.NewRouter()
	r.Get("/plain/*", func(w http.ResponseWriter, r *http.Request) {
		slug := pathToSlug(chi.URLParam(r, "*"))
		post := repo.GetBySlug(slug)
		if post == nil {
			http.NotFound(w, r)
			return
		}
		plain := post.PlainText()
		if plain == nil {
			http.Error(w, "plain text not available", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(plain)
	})
	return r
}

func doPlainRequest(t *testing.T, repo service.PostRepository, slug string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/plain/"+slug, nil)
	w := httptest.NewRecorder()
	plainRouter(repo).ServeHTTP(w, req)
	return w
}

// convertedPost creates a *service.Post with plain text populated via the converter.
func convertedPost(t *testing.T, slug string, raw []byte) *service.Post {
	t.Helper()
	post := service.NewPost(slug+".md", nil, nil)
	c := &service.GoldmarkConverter{}
	if err := c.Convert(post, raw); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	return post
}

func TestPlainEndpoint_ReturnsPlainText(t *testing.T) {
	raw := []byte("---\ntitle: Hello\n---\n\n# Hello\n\nThis is a paragraph.\n")
	post := convertedPost(t, "hello", raw)
	repo := &stubRepo{posts: []*service.Post{post}}

	w := doPlainRequest(t, repo, "hello")

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

func TestPlainEndpoint_NotFound(t *testing.T) {
	repo := &stubRepo{}
	w := doPlainRequest(t, repo, "missing")
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestPlainEndpoint_NestedSlug(t *testing.T) {
	raw := []byte("---\ntitle: Nested\n---\n\n# Nested\n\nContent here.\n")
	// Simulate a post with slug "more+things+hello" (as stored by the repo)
	post := convertedPost(t, "more+things+hello", raw)
	repo := &stubRepo{posts: []*service.Post{post}}

	// URL: /plain/more/things/hello should resolve to slug "more+things+hello"
	w := doPlainRequest(t, repo, "more/things/hello")
	if w.Code != http.StatusOK {
		t.Errorf("want 200 for nested slug, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Nested") {
		t.Errorf("want 'Nested' in body, got %q", w.Body.String())
	}
}

func TestPlainEndpoint_PostWithoutPlainText(t *testing.T) {
	// NewPost without going through the converter has no plainText set.
	post := service.NewPost("no-plain.md", &service.Metadata{Title: "No Plain"}, []byte("<p>content</p>"))
	repo := &stubRepo{posts: []*service.Post{post}}

	w := doPlainRequest(t, repo, "no-plain")
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 when plain text unavailable, got %d", w.Code)
	}
}

// --- /feed.atom ---

func doAtomRequest(t *testing.T, repo service.PostRepository) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/feed.atom", nil)
	w := httptest.NewRecorder()
	// Use a minimal router that wires only the atom feed handler.
	cfg := &config.Config{}
	atomRouter := func() http.Handler {
		r := chi.NewRouter()
		r.Get("/feed.atom", func(w http.ResponseWriter, r *http.Request) {
			data, err := service.BuildAtomFeed(cfg, repo.FeedPosts(nil, ""))
			if err != nil {
				http.Error(w, "failed to generate atom feed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
			_, _ = w.Write(data)
		})
		return r
	}()
	atomRouter.ServeHTTP(w, req)
	return w
}

func TestAtomFeedEndpoint_ContentTypeAndBody(t *testing.T) {
	repo := &stubRepo{}
	w := doAtomRequest(t, repo)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/atom+xml") {
		t.Errorf("want application/atom+xml content type, got %q", ct)
	}
}

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
	// Client claims it has a copy from the same time — should get 304.
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

// --- isReservedPath ---

func TestIsReservedPath_KnownStaticPaths(t *testing.T) {
	reserved := []string{
		"/", "/posts", "/feed.xml", "/feed.atom",
		"/health", "/favicon.ico", "/og-logo.png",
		"/robots.txt", "/sitemap.xml",
	}
	for _, p := range reserved {
		if !isReservedPath(p) {
			t.Errorf("want %q to be reserved", p)
		}
	}
}

func TestIsReservedPath_ReservedPrefixes(t *testing.T) {
	reserved := []string{
		"/api/v1/posts",
		"/posts/my-slug",
		"/plain/my-slug",
	}
	for _, p := range reserved {
		if !isReservedPath(p) {
			t.Errorf("want %q to be reserved", p)
		}
	}
}

func TestIsReservedPath_CustomPathsAreNotReserved(t *testing.T) {
	free := []string{"/about", "/contact", "/info", "/services/web"}
	for _, p := range free {
		if isReservedPath(p) {
			t.Errorf("want %q to be free (not reserved), but it was reserved", p)
		}
	}
}

// --- dynamic page routes ---

// pageRouter builds a minimal router that registers one dynamic page route,
// mirroring the logic in buildRouter for cfg.Pages entries.
func pageRouter(path, slug string, repo service.PostRepository) chi.Router {
	r := chi.NewRouter()
	serve404 := func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}
	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		post := repo.GetBySlug(slug)
		if post == nil {
			serve404(w, r)
			return
		}
		if data := post.Data(); len(data.Content) == 0 {
			serve404(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(post.Data().Content)
	})
	return r
}

func TestDynamicPageRoute_ServesPost(t *testing.T) {
	post := service.NewPost("about.md", &service.Metadata{Title: "About"}, []byte("<p>about us</p>"))
	repo := &stubRepo{posts: []*service.Post{post}}

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	w := httptest.NewRecorder()
	pageRouter("/about", "about", repo).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestDynamicPageRoute_NotFoundWhenSlugMissing(t *testing.T) {
	repo := &stubRepo{}

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	w := httptest.NewRecorder()
	pageRouter("/about", "about", repo).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestDynamicPageRoute_NotFoundWhenPostHasNoContent(t *testing.T) {
	post := service.NewPost("about.md", &service.Metadata{Title: "About"}, nil)
	repo := &stubRepo{posts: []*service.Post{post}}

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	w := httptest.NewRecorder()
	pageRouter("/about", "about", repo).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

// --- /health ---

func TestHealthEndpoint(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

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

func TestPathToSlug(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"hello-world", "hello-world"},
		{"more/hello", "more+hello"},
		{"more/things/hello-world", "more+things+hello-world"},
		{"a/b/c/d", "a+b+c+d"},
		{"/leading/slash", "leading+slash"},
	}
	for _, tc := range cases {
		if got := pathToSlug(tc.input); got != tc.want {
			t.Errorf("pathToSlug(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
