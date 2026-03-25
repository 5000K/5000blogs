package server

import (
	"github.com/5000K/5000blogs/service"
	"github.com/5000K/5000blogs/view"
)

type PostFeedModule struct {
	indexer  service.PostIndexer
	renderer view.Renderer
}


