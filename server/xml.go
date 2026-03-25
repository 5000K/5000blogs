package server

import (
	"net/http"
	"strings"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/go-chi/chi/v5"
)

type XmlFeedModule struct {
	indexer service.PostIndexer
}

func NewFeedModule(indexer service.PostIndexer) *XmlFeedModule {
	return &XmlFeedModule{indexer: indexer}
}

func (f *XmlFeedModule) RegisterRoutes(r chi.Router, conf *config.ConfigLoader) error {
	baseConf := conf.BaseConfig()

	r.Get("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		var tags []string
		if t := r.URL.Query().Get("tags"); t != "" {
			tags = strings.Split(t, ",")
		}
		q := r.URL.Query().Get("q")
		data, err := service.BuildRSSFeed(&baseConf, f.indexer.FeedPosts(tags, q))
		if err != nil {
			http.Error(w, "failed to generate feed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		_, _ = w.Write(data)
	})

	r.Get("/feed.atom", func(w http.ResponseWriter, r *http.Request) {
		var tags []string
		if t := r.URL.Query().Get("tags"); t != "" {
			tags = strings.Split(t, ",")
		}
		q := r.URL.Query().Get("q")
		data, err := service.BuildAtomFeed(&baseConf, f.indexer.FeedPosts(tags, q))
		if err != nil {
			http.Error(w, "failed to generate atom feed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		_, _ = w.Write(data)
	})
	return nil
}
