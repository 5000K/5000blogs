package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/modules"
	"github.com/5000K/5000blogs/run"
)

func main() {
	loader, err := config.NewConfigLoader()

	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(loader.BaseConfig().LogLevel)); err != nil {
		log.Fatalf("invalid log level %q: %v", loader.BaseConfig().LogLevel, err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	ctx := modules.RuntimeContext{
		Converters:        modules.Converters(loader),
		PostRepositories:  modules.PostRepositories(loader, logger),
		PostSources:       modules.PostSources(loader, logger),
		OGImageGenerators: modules.OGImageGenerators(loader, logger),
		Renderers:         modules.Renderers(loader, logger),
		Assets:            modules.NewAssetFetchRegistry(logger),
		Loader:            loader,
		Log:               logger,
	}

	err = run.Run(ctx)

	if err != nil {
		logger.Error("application error", "error", err)
		os.Exit(1)
	}
}
