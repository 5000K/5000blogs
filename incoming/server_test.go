package incoming

import (
	"5000blogs/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
