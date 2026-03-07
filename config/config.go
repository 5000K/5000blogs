package config

import (
	"fmt"
	"net/url"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ServerAddress string `env:"SERVER_ADDRESS" env-default:":8080" yaml:"address"`

	Paths struct {
		Config string `env:"CONFIG_PATH" env-default:"config.yml"`
		Posts  string `env:"POSTS_PATH" env-default:"./posts/" yaml:"posts"`
		Static string `env:"STATIC_PATH" env-default:"./static/" yaml:"static"`
	} `yaml:"paths"`

	RescanCron           string `env:"RESCAN_CRON" env-default:"* * * * *" yaml:"rescan_cron"`
	SkipUnchangedModTime bool   `env:"SKIP_UNCHANGED_MOD_TIME" env-default:"true" yaml:"skip_unchanged_mod_time"`
	LogLevel             string `env:"LOG_LEVEL" env-default:"info" yaml:"log_level"`
	PageSize             int    `env:"PAGE_SIZE" env-default:"10" yaml:"page_size"`

	SiteURL         string `env:"SITE_URL" env-default:"http://localhost:8080" yaml:"site_url"`
	FeedTitle       string `env:"FEED_TITLE" env-default:"Blog" yaml:"feed_title"`
	FeedDescription string `env:"FEED_DESCRIPTION" env-default:"" yaml:"feed_description"`
	RSSFullContent  bool   `env:"RSS_FULL_CONTENT" env-default:"false" yaml:"rss_full_content"`

	Plugins []string `yaml:"plugins"`

	OGImage OGImageConfig `yaml:"og_image"`
}

type OGImageConfig struct {
	Enabled     bool   `env:"OG_IMAGE_ENABLED" env-default:"true" yaml:"enabled"`
	BlogName    string `env:"OG_IMAGE_BLOG_NAME" env-default:"" yaml:"blog_name"`
	BlogIcon    string `env:"OG_IMAGE_BLOG_ICON" env-default:"" yaml:"blog_icon"` // path to PNG file
	BgColor     string `env:"OG_IMAGE_BG_COLOR" env-default:"#111111" yaml:"bg_color"`
	TextColor   string `env:"OG_IMAGE_TEXT_COLOR" env-default:"#f0f0f0" yaml:"text_color"`
	SubColor    string `env:"OG_IMAGE_SUB_COLOR" env-default:"#999999" yaml:"sub_color"`
	AccentColor string `env:"OG_IMAGE_ACCENT_COLOR" env-default:"#7eb8f7" yaml:"accent_color"`
	CacheSize   int    `env:"OG_IMAGE_CACHE_SIZE" env-default:"128" yaml:"cache_size"`
}

func (c *Config) Validate() error {
	if c.PageSize <= 0 {
		return fmt.Errorf("page_size must be > 0, got %d", c.PageSize)
	}
	u, err := url.Parse(c.SiteURL)
	if err != nil || !u.IsAbs() {
		return fmt.Errorf("site_url must be an absolute URL, got %q", c.SiteURL)
	}
	if c.OGImage.CacheSize <= 0 {
		return fmt.Errorf("og_image.cache_size must be > 0, got %d", c.OGImage.CacheSize)
	}
	return nil
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
