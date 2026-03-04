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
	LogLevel             string `env:"LOG_LEVEL" env-default:"info" yaml:"log_level"`
	PageSize             int    `env:"PAGE_SIZE" env-default:"10" yaml:"page_size"`

	SiteURL         string `env:"SITE_URL" env-default:"http://localhost:8080" yaml:"site_url"`
	FeedTitle       string `env:"FEED_TITLE" env-default:"Blog" yaml:"feed_title"`
	FeedDescription string `env:"FEED_DESCRIPTION" env-default:"" yaml:"feed_description"`
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
