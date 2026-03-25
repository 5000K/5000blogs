package server

import (
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

func Listen(conf config.ConfigLoader, modules []ServerModule) error {
	var serverConf ServerConfig
	conf.Load("", &serverConf)

	r := chi.NewRouter()
	r.Use(middleware.Logger) // todo: migrate to goware/httplog
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(time.Duration(serverConf.Timeout) * time.Second))

	for _, module := range modules {
		module.RegisterRoutes(r, &conf)
	}

	return http.ListenAndServe(serverConf.ServerAddress, r)
}
