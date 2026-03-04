package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ServerAddress string `env:"SERVER_ADDRESS" env-default:":8080" yaml:"address"`

	Paths struct {
		Config string `env:"CONFIG_PATH" env-default:"config.yml"`
		Posts  string `env:"POSTS_PATH" env-default:"./posts/" yaml:"posts"`
		Static string `env:"STATIC_PATH" env-default:"./static/" yaml:"static"`
	} `yaml:"paths"`

	RescanCron           string `env:"RESCAN_CRON" env-default:"0 * * * *" yaml:"rescan_cron"`
	SkipUnchangedModTime bool   `env:"SKIP_UNCHANGED_MOD_TIME" env-default:"true" yaml:"skip_unchanged_mod_time"`
}

func Get() (*Config, error) {
	var cfg Config

	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		return nil, err
	}

	err = cleanenv.ReadConfig(cfg.Paths.Config, &cfg)

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
