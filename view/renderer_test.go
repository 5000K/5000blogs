package view

import (
	"5000blogs/config"
	"5000blogs/service"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const minimalTemplate = `<!DOCTYPE html><html><head><title>{{.Title}}</title></head><body>{{.Content}}<footer>{{.FooterContent}}</footer></body></html>`

func newTestRenderer(t *testing.T) *Renderer {
	t.Helper()
	dir := t.TempDir()
	staticDir := filepath.Join(dir, "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("mkdir static: %v", err)
	}
	tmplPath := filepath.Join(staticDir, "template.html")
	if err := os.WriteFile(tmplPath, []byte(minimalTemplate), 0644); err != nil {
		t.Fatalf("write template: %v", err)
	}
	cfg := &config.Config{}
	cfg.Paths.Static = staticDir
	r, err := NewRenderer(cfg, slog.Default())
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	return r
}

func TestServe404_NilPost(t *testing.T) {
	r := newTestRenderer(t)
	w := httptest.NewRecorder()
	r.Serve404(nil, w)
	if w.Code != http.StatusNotFound {
		t.Errorf("status: want 404 got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "404") {
		t.Errorf("want 404 in body, got: %s", w.Body.String())
	}
}

func TestServe404_CustomPost(t *testing.T) {
	r := newTestRenderer(t)
	w := httptest.NewRecorder()
	post := service.NewPost("404.md", &service.Metadata{Title: "Not Here"}, []byte("<p>nope</p>"))
	r.Serve404(post, w)
	if w.Code != http.StatusNotFound {
		t.Errorf("status: want 404 got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Not Here") {
		t.Errorf("want title Not Here in body, got: %s", body)
	}
	if !strings.Contains(body, "nope") {
		t.Errorf("want custom content in body, got: %s", body)
	}
}

func TestServePost_Nil(t *testing.T) {
	r := newTestRenderer(t)
	w := httptest.NewRecorder()
	r.ServePost(nil, w, "http://example.com/posts/x", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status: want 404 got %d", w.Code)
	}
}

func TestServePost_Valid(t *testing.T) {
	r := newTestRenderer(t)
	w := httptest.NewRecorder()
	post := service.NewPost("hello.md", &service.Metadata{Title: "Hello"}, []byte("<p>world</p>"))
	r.ServePost(post, w, "http://example.com/posts/hello", "")
	if w.Code != http.StatusOK {
		t.Errorf("status: want 200 got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "world") {
		t.Errorf("want world in body, got: %s", w.Body.String())
	}
}

func TestSetFooter_InjectedIntoPost(t *testing.T) {
	r := newTestRenderer(t)
	r.SetFooter(func() template.HTML { return template.HTML("<p>footer-content</p>") })
	w := httptest.NewRecorder()
	post := service.NewPost("hello.md", &service.Metadata{Title: "Hello"}, []byte("<p>body</p>"))
	r.ServePost(post, w, "http://example.com/posts/hello", "")
	if !strings.Contains(w.Body.String(), "footer-content") {
		t.Errorf("want footer-content in body, got: %s", w.Body.String())
	}
}

func TestSetFooter_InjectedIntoList(t *testing.T) {
	r := newTestRenderer(t)
	r.SetFooter(func() template.HTML { return template.HTML("<p>footer-content</p>") })
	w := httptest.NewRecorder()
	r.ServePostList(service.PageResult{}, w, "http://example.com/posts")
	if !strings.Contains(w.Body.String(), "footer-content") {
		t.Errorf("want footer-content in list body, got: %s", w.Body.String())
	}
}

func TestSetFooter_NotSetReturnsEmpty(t *testing.T) {
	r := newTestRenderer(t)
	w := httptest.NewRecorder()
	post := service.NewPost("hello.md", &service.Metadata{Title: "Hello"}, []byte("<p>body</p>"))
	r.ServePost(post, w, "http://example.com/posts/hello", "")
	body := w.Body.String()
	if !strings.Contains(body, "<footer></footer>") {
		t.Errorf("want empty footer, got: %s", body)
	}
}
