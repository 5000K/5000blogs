package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ConfigPath string `env:"CONFIG_PATH" env-default:"config.yml"`

	ServerAddress string `env:"SERVER_ADDRESS" env-default:":8080"`
}

func Get() (*Config, error) {
	var cfg Config

	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		return nil, err
	}

	err = cleanenv.ReadConfig(cfg.ConfigPath, &cfg)

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
