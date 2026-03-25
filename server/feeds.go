package server

import (
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

func (m *PostFeedModule) RegisterRoutes(r chi.Router, conf *config.ConfigLoader) error {
	var feedConfigs []PostFeedConfig
	err := conf.Load("feeds", &feedConfigs)
	if err != nil {
		feedConfigs = []PostFeedConfig{
			{Name: "feed", Tags: make([]string, 0), Query: ""},
		}
	}
	return nil
}
