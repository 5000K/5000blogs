package server

import (
	"net/http"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/go-chi/chi/v5"
)

type PlainModule struct {
	indexer service.PostIndexer
}

func NewPlainModule(indexer service.PostIndexer) *PlainModule {
	return &PlainModule{indexer: indexer}
}

func (m *PlainModule) RegisterRoutes(r chi.Router, conf *config.ConfigLoader) error {

	r.Get("/plain/*", func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "*")
		post := m.indexer.GetBySlug(slug)
		if post == nil {
			m.serve404(w, r)
			return
		}
		plain := post.PlainText()
		if plain == nil {
			http.Error(w, "plain text not available", http.StatusNotFound)
			return
		}
		if checkLastModified(w, r, post.ModTime()) {
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(plain)
	})

	return nil
}

func (m *PlainModule) serve404(w http.ResponseWriter, r *http.Request) {
	post := m.indexer.GetBySlug("404")
	if post == nil {
		http.NotFound(w, r)
		return
	}
	if checkLastModified(w, r, post.ModTime()) {
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(post.PlainText())
}
