package incoming

import (
	"5000blogs/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// plainRouter is a minimal router containing only the /plain/{slug} handler,
// used to test that route in isolation without a full config/renderer setup.
func plainRouter(repo service.PostRepository) chi.Router {
	r := chi.NewRouter()
	r.Get("/plain/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
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
	c := &service.GoMarkdownConverter{}
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
	atomRouter := func() http.Handler {
		r := chi.NewRouter()
		r.Get("/feed.atom", func(w http.ResponseWriter, r *http.Request) {
			data, err := repo.AtomFeed()
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
