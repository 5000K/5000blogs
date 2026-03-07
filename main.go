package main

import (
	"5000blogs/config"
	"5000blogs/incoming"
	"5000blogs/service"
	"5000blogs/view"
	"log"
	"log/slog"
	"os"
)

func main() {
	cfg, err := config.Get()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		log.Fatalf("invalid log level %q: %v", cfg.LogLevel, err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	fsSource := service.NewFileSystemSource(cfg.Paths.Posts, logger)
	source := service.NewLayeredSource(fsSource, service.NewBuiltinSource())
	converter := &service.GoMarkdownConverter{}
	repo := service.NewMemoryPostRepository(cfg, source, converter, logger)
	if err := repo.Start(); err != nil {
		log.Fatalf("failed to start repository: %v", err)
	}
	defer repo.Stop()

	renderer, err := view.NewRenderer(cfg, logger)
	if err != nil {
		log.Fatalf("failed to create renderer: %v", err)
	}

	var ogGen *service.OGImageGenerator
	if cfg.OGImage.Enabled {
		ogGen, err = service.NewOGImageGenerator(cfg.OGImage)
		if err != nil {
			log.Fatalf("failed to create og:image generator: %v", err)
		}
	}

	incoming.Serve(cfg, repo, renderer, ogGen)
}
