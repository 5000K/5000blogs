package main

import (
	"5000blogs/config"
	"5000blogs/incoming"
	"5000blogs/service"
	"log"
)

func main() {
	cfg, err := config.Get()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	svc := service.NewService(cfg)
	if err := svc.Start(); err != nil {
		log.Fatalf("failed to start service: %v", err)
	}
	defer svc.Stop()

	incoming.Serve(cfg, svc)
}
