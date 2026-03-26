package server

import (
	"bytes"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/5000K/5000blogs/view"
	"github.com/go-chi/chi/v5"
)

type DynamicModule struct {
	indexer  service.PostIndexer
	ogGen    service.OGImageGenerator
	renderer view.Renderer
}

func NewDynamicModule(indexer service.PostIndexer, ogGen service.OGImageGenerator, renderer view.Renderer) *DynamicModule {
	return &DynamicModule{indexer: indexer, ogGen: ogGen, renderer: renderer}
}

type SiteUrlConfig struct {
	SiteURL string `env:"SITE_URL" env-default:"http://localhost:8080" yaml:"site_url"`
}

func (m *DynamicModule) RegisterRoutes(r chi.Router, conf *config.ConfigLoader) error {
	var cfg SiteUrlConfig
	err := conf.Load("", &cfg)
	if err != nil {
		return err
	}

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		rest := chi.URLParam(r, "*")

		// Serve og:image for a post: /{slug}/og-image.png
		if strings.HasSuffix(rest, "/og-image.png") {
			if m.ogGen == nil {
				http.NotFound(w, r)
				return
			}
			slug := strings.TrimSuffix(rest, "/og-image.png")
			post := m.indexer.GetBySlug(slug)
			if post == nil {
				http.NotFound(w, r)
				return
			}
			data, err := m.ogGen.Generate(post)
			if err != nil {
				http.Error(w, "failed to generate og:image", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "public, max-age=86400")
			_, _ = w.Write(data)
			return
		}

		slug := rest
		post := m.indexer.GetBySlug(slug)
		if post == nil { // try to serve as media file.
			if !m.serveMedia(w, r) {
				m.serve404(w, r)
			}
			return
		}
		if checkLastModified(w, r, post.ModTime()) {
			return
		}
		var ogImageURL string
		if m.ogGen != nil {
			ogImageURL = cfg.SiteURL + "/" + slug + "/og-image.png"
		}
		m.renderer.ServePost(post, w, cfg.SiteURL+r.URL.RequestURI(), ogImageURL)
	})

	r.NotFound(m.serve404)

	return nil
}

func (m *DynamicModule) serveMedia(w http.ResponseWriter, r *http.Request) bool {
	relPath := chi.URLParam(r, "*")
	// Prevent serving raw markdown through the media endpoint.
	if strings.HasSuffix(relPath, ".md") {
		return false
	}
	// Sanitise the path: resolve inside a virtual root to prevent traversal.
	relPath = strings.TrimPrefix(path.Clean("/"+relPath), "/")
	if relPath == "" {
		return false
	}
	data, modTime, err := m.indexer.ReadMedia(relPath)
	if err != nil {
		return false
	}
	// http.ServeContent handles Content-Type detection, Range requests,
	// If-Modified-Since / Last-Modified, and ETag caching automatically.
	http.ServeContent(w, r, relPath, modTime, bytes.NewReader(data))

	return true
}

func (m *DynamicModule) serve404(w http.ResponseWriter, r *http.Request) {
	m.renderer.Serve404(m.indexer.GetBySlug("404"), w)
}

func checkLastModified(w http.ResponseWriter, r *http.Request, t time.Time) bool {
	if t.IsZero() {
		return false
	}
	t = t.UTC().Truncate(time.Second)
	w.Header().Set("Last-Modified", t.Format(http.TimeFormat))
	if ims := r.Header.Get("If-Modified-Since"); ims != "" {
		if parsed, err := http.ParseTime(ims); err == nil && !t.After(parsed.UTC()) {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}
	return false
}
