package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/incoming"
	"github.com/5000K/5000blogs/service"
	"github.com/5000K/5000blogs/view"

	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func main() {
	cfg, err := config.Get()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		log.Fatalf("invalid log level %q: %v", cfg.LogLevel, err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	userSources, err := buildSources(cfg, logger)
	if err != nil {
		log.Fatalf("failed to build sources: %v", err)
	}
	source := service.NewLayeredSource(append(userSources, service.NewBuiltinSource())...)
	converter := service.NewGoldmarkConverter("", cfg.Features)
	repo, err := service.NewBlevePostRepository(cfg, source, converter, logger)
	if err != nil {
		log.Fatalf("failed to create repository: %v", err)
	}
	if err := repo.Start(); err != nil {
		log.Fatalf("failed to start repository: %v", err)
	}
	defer repo.Stop()

	tmplData, err := config.FetchResource(cfg.Paths.Template)
	if err != nil {
		log.Fatalf("failed to load template: %v", err)
	}

	var iconData []byte
	if cfg.Paths.Icon != "" {
		iconData, err = config.FetchResource(cfg.Paths.Icon)
		if err != nil {
			log.Fatalf("failed to load icon: %v", err)
		}
	}

	renderer, err := view.NewRenderer(cfg, tmplData, logger)
	if err != nil {
		log.Fatalf("failed to create renderer: %v", err)
	}

	var ogGen service.OGImageGenerator
	if cfg.OGImage.Enabled {
		ogGen, err = service.NewOGImageGenerator(cfg.OGImage, cfg.BlogName, iconData)
		if err != nil {
			log.Fatalf("failed to create og:image generator: %v", err)
		}
	}

	incoming.Serve(cfg, repo, renderer, ogGen, iconData)
}

func buildSources(cfg *config.Config, logger *slog.Logger) ([]service.PostSource, error) {
	if len(cfg.Sources) == 0 {
		return []service.PostSource{service.NewFileSystemSource(cfg.Paths.Posts, logger)}, nil
	}
	sources := make([]service.PostSource, 0, len(cfg.Sources))
	for _, sc := range cfg.Sources {
		switch sc.Type {
		case "filesystem":
			sources = append(sources, service.NewFileSystemSource(sc.Path, logger))
		case "git":
			dir := sc.Dir
			if dir == "" {
				dir = "."
			}
			auth, err := gitAuth(sc)
			if err != nil {
				return nil, err
			}
			gs, err := service.NewGitSource(sc.URL, dir, auth, logger)
			if err != nil {
				return nil, err
			}
			sources = append(sources, gs)
		}
	}
	return sources, nil
}

func gitAuth(sc config.SourceConfig) (transport.AuthMethod, error) {
	if sc.SSHKeyPath != "" {
		auth, err := gitssh.NewPublicKeysFromFile("git", sc.SSHKeyPath, sc.SSHKeyPassphrase)
		if err != nil {
			return nil, fmt.Errorf("git source %s: load SSH key: %w", sc.URL, err)
		}
		return auth, nil
	}
	if sc.AuthToken != "" {
		user := sc.AuthUser
		if user == "" {
			user = "git"
		}
		return &githttp.BasicAuth{Username: user, Password: sc.AuthToken}, nil
	}
	return nil, nil
}
