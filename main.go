package main

import (
	"5000blogs/config"
	"5000blogs/incoming"
	"5000blogs/service"
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

	svc := service.NewService(cfg, logger)
	if err := svc.Start(); err != nil {
		log.Fatalf("failed to start service: %v", err)
	}
	defer svc.Stop()

	incoming.Serve(cfg, svc)
}
