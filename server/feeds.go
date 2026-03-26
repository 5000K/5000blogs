package server

import (
	"net/http"
	"strconv"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/5000K/5000blogs/view"
	"github.com/go-chi/chi/v5"
)

type PostFeedModule struct {
	indexer  service.PostIndexer
	renderer view.Renderer
}

type PostFeedConfig struct {
	Name  string   `yaml:"name"`
	Tags  []string `yaml:"tags"`
	Query string   `yaml:"query"`
}

func NewPostFeedModule(indexer service.PostIndexer, renderer view.Renderer) *PostFeedModule {
	return &PostFeedModule{indexer: indexer, renderer: renderer}
}

func filterFromRequest(r *http.Request, feedConf PostFeedConfig) service.PostFilter {
	// tags
	tags := feedConf.Tags
	if t := r.URL.Query().Get("tags"); t != "" {
		tags = append(tags, t)
	}
	// query
	q := feedConf.Query
	if q == "" {
		q = r.URL.Query().Get("q")
	}

	filter := service.PostFilter{Tags: tags, Query: q}
	filter.Tags = append(filter.Tags, feedConf.Tags...)

	return filter
}

func (m *PostFeedModule) RegisterRoutes(r chi.Router, conf *config.ConfigLoader) error {
	var feedConfigs []PostFeedConfig
	err := conf.Load("feeds", &feedConfigs)
	if err != nil {
		feedConfigs = []PostFeedConfig{
			{Name: "posts", Tags: make([]string, 0), Query: ""},
		}
	}

	cfg := conf.BaseConfig()

	for _, feedConf := range feedConfigs {
		feedConf := feedConf
		r.Get("/"+feedConf.Name, func(w http.ResponseWriter, r *http.Request) {
			// page
			page := r.URL.Query().Get("page")
			if page == "" {
				page = "1"
			}

			pageInt := 1
			if p, err := strconv.Atoi(page); err == nil && p > 0 {
				pageInt = p
			}

			filter := filterFromRequest(r, feedConf)

			pageResult := m.indexer.ListFilteredPaged(filter, cfg.FeedSize, pageInt)

			if pageResult == nil {
				http.NotFound(w, r)
				return
			}

			m.renderer.ServePostList(*pageResult, w, r.URL.RequestURI())

		})
	}

	return nil
}
