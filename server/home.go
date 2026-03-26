package server

import (
	"net/http"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/5000K/5000blogs/view"
	"github.com/go-chi/chi/v5"
)

type HomeModule struct {
	indexer  service.PostIndexer
	ogGen    service.OGImageGenerator
	renderer view.Renderer
}

func NewHomeModule(indexer service.PostIndexer, ogGen service.OGImageGenerator, renderer view.Renderer) *HomeModule {
	return &HomeModule{indexer: indexer, ogGen: ogGen, renderer: renderer}
}

func (m *HomeModule) RegisterRoutes(r chi.Router, conf *config.ConfigLoader) error {
	var cfg SiteUrlConfig
	err := conf.Load("", &cfg)
	if err != nil {
		return err
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if home := m.indexer.GetBySlug("index"); home != nil {
			if data := home.Data(); len(data.Content) > 0 {
				if checkLastModified(w, r, home.ModTime()) {
					return
				}
				m.renderer.ServePost(home, w, cfg.SiteURL+"/", "")
				return
			}
		}
		if checkLastModified(w, r, m.indexer.LastModified()) {
			return
		}
		m.renderer.ServePostList(m.indexer.GetPage(1, nil), w, cfg.SiteURL+"/posts")
	})

	return nil
}
