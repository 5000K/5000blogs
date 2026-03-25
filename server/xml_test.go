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

func xmlRouter(indexer *stubIndexer, siteURL string) chi.Router {
	r := chi.NewRouter()
	m := NewFeedModule(indexer)
	cfg := config.Config{SiteURL: siteURL}
	_ = m.RegisterRoutes(r, config.NewConfigLoaderFromConfig(cfg))
	return r
}

func TestXmlFeedModule_RSSFeed(t *testing.T) {
	indexer := &stubIndexer{}
	req := httptest.NewRequest(http.MethodGet, "/feed.xml", nil)
	w := httptest.NewRecorder()
	xmlRouter(indexer, "http://example.com").ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/rss+xml") {
		t.Errorf("want application/rss+xml, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), "<rss") {
		t.Errorf("body should contain <rss element, got %q", w.Body.String())
	}
}

func TestXmlFeedModule_AtomFeed(t *testing.T) {
	indexer := &stubIndexer{}
	req := httptest.NewRequest(http.MethodGet, "/feed.atom", nil)
	w := httptest.NewRecorder()
	xmlRouter(indexer, "http://example.com").ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/atom+xml") {
		t.Errorf("want application/atom+xml, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), "<feed") {
		t.Errorf("body should contain <feed element, got %q", w.Body.String())
	}
}

func TestXmlFeedModule_Sitemap(t *testing.T) {
	indexer := &stubIndexer{}
	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	w := httptest.NewRecorder()
	xmlRouter(indexer, "http://example.com").ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/xml") {
		t.Errorf("want application/xml, got %q", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<urlset") {
		t.Errorf("body should contain <urlset, got %q", body)
	}
	// root /posts entry is always included
	if !strings.Contains(body, "/posts") {
		t.Errorf("sitemap should include /posts entry, got %q", body)
	}
}

func TestXmlFeedModule_SitemapIncludesPostSlugs(t *testing.T) {
	indexer := &stubIndexer{
		sitemap: []service.SitemapEntry{
			{Slug: "hello-world"},
			{Slug: "about"},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	w := httptest.NewRecorder()
	xmlRouter(indexer, "http://example.com").ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "hello-world") {
		t.Errorf("want 'hello-world' slug in sitemap, got %q", body)
	}
	if !strings.Contains(body, "about") {
		t.Errorf("want 'about' slug in sitemap, got %q", body)
	}
}

func TestXmlFeedModule_RSSTagFilter(t *testing.T) {
	indexer := &stubIndexer{}
	req := httptest.NewRequest(http.MethodGet, "/feed.xml?tags=go", nil)
	w := httptest.NewRecorder()
	xmlRouter(indexer, "http://example.com").ServeHTTP(w, req)

	// A tag filter must not break the endpoint.
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}
