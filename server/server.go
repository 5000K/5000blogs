package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/5000K/5000blogs/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type ServerModule interface {
	RegisterRoutes(r chi.Router, conf *config.ConfigLoader) error
}

type ServerConfig struct {
	ServerAddress string `env:"SERVER_ADDRESS" env-default:":8080" yaml:"address"`
	Timeout       int    `env:"SERVER_TIMEOUT" env-default:"60" yaml:"timeout"`
}

func Listen(conf *config.ConfigLoader, modules []ServerModule, logger *slog.Logger) error {
	log := logger.With("component", "server")

	var serverConf ServerConfig
	conf.Load("", &serverConf)

	r := chi.NewRouter()
	r.Use(RequestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(time.Duration(serverConf.Timeout) * time.Second))

	for _, module := range modules {
		module.RegisterRoutes(r, conf)
	}

	log.Info("Running server", "address", serverConf.ServerAddress)

	return http.ListenAndServe(serverConf.ServerAddress, r)
}
