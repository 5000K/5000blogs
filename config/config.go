package config

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ServerAddress string `env:"SERVER_ADDRESS" env-default:":8080" yaml:"address"`

	Paths struct {
		Config   string `env:"CONFIG_PATH" env-default:"config.yml"`
		Posts    string `env:"POSTS_PATH" env-default:"./posts/" yaml:"posts"`
		Template string `env:"TEMPLATE_PATH" env-default:"./template/template.html" yaml:"template"`
		Icon     string `env:"ICON_PATH" env-default:"" yaml:"icon"`
	} `yaml:"paths"`

	RescanCron           string `env:"RESCAN_CRON" env-default:"* * * * *" yaml:"rescan_cron"`
	SkipUnchangedModTime bool   `env:"SKIP_UNCHANGED_MOD_TIME" env-default:"true" yaml:"skip_unchanged_mod_time"`
	LogLevel             string `env:"LOG_LEVEL" env-default:"info" yaml:"log_level"`
	PageSize             int    `env:"PAGE_SIZE" env-default:"10" yaml:"page_size"`

	SiteURL         string `env:"SITE_URL" env-default:"http://localhost:8080" yaml:"site_url"`
	FeedDescription string `env:"FEED_DESCRIPTION" env-default:"" yaml:"feed_description"`
	FeedSize        int    `env:"FEED_SIZE" env-default:"20" yaml:"feed_size"`
	RSSContent      string `env:"RSS_CONTENT" env-default:"none" yaml:"rss_content"`

	BlogName string         `env:"BLOG_NAME" env-default:"Blog" yaml:"blog_name"`
	NavLinks []NavLink      `yaml:"nav_links"`
	Pages    []PageRoute    `yaml:"pages"`
	Plugins  []string       `yaml:"plugins"`
	Sources  []SourceConfig `yaml:"sources"`

	Features Features `yaml:"features"`

	OGImage OGImageConfig `yaml:"og_image"`
}

type Features struct{
	WikiLinks bool `env:"FEATURE_WIKI_LINKS" env-default:"false" yaml:"wiki_links"`
}

// FetchResource reads a file from disk or downloads it over HTTP/HTTPS.
func FetchResource(urlOrPath string) ([]byte, error) {
	if strings.HasPrefix(urlOrPath, "http://") || strings.HasPrefix(urlOrPath, "https://") {
		resp, err := http.Get(urlOrPath) //nolint:noctx
		if err != nil {
			return nil, fmt.Errorf("fetch %q: %w", urlOrPath, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch %q: HTTP %d", urlOrPath, resp.StatusCode)
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("fetch %q: read body: %w", urlOrPath, err)
		}
		return data, nil
	}
	data, err := os.ReadFile(urlOrPath)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", urlOrPath, err)
	}
	return data, nil
}

// NavLink is a navigation entry rendered in the site header.
type NavLink struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// PageRoute maps a URL path to a post slug for serving static page content.
type PageRoute struct {
	Path string `yaml:"path"`
	Slug string `yaml:"slug"`
}

// SourceConfig defines a post source. Type is required: "filesystem" or "git".
type SourceConfig struct {
	Type string `yaml:"type"`
	// filesystem
	Path string `yaml:"path"`
	// git
	URL string `yaml:"url"`
	Dir string `yaml:"dir"` // subdirectory within the repo (default: ".")
	// git auth (mutually exclusive: use either token or SSH key, not both)
	AuthUser         string `yaml:"auth_user"`          // HTTP basic auth username
	AuthToken        string `yaml:"auth_token"`         // HTTP basic auth password or token
	SSHKeyPath       string `yaml:"ssh_key_path"`       // path to SSH private key file
	SSHKeyPassphrase string `yaml:"ssh_key_passphrase"` // passphrase for the SSH private key
}

type OGImageConfig struct {
	Enabled     bool   `env:"OG_IMAGE_ENABLED" env-default:"true" yaml:"enabled"`
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
	if c.FeedSize <= 0 {
		return fmt.Errorf("feed_size must be > 0, got %d", c.FeedSize)
	}
	switch c.RSSContent {
	case "", "none", "text", "html":
		// valid
	default:
		return fmt.Errorf("rss_content must be \"none\", \"text\", or \"html\", got %q", c.RSSContent)
	}
	u, err := url.Parse(c.SiteURL)
	if err != nil || !u.IsAbs() {
		return fmt.Errorf("site_url must be an absolute URL, got %q", c.SiteURL)
	}
	if c.OGImage.CacheSize <= 0 {
		return fmt.Errorf("og_image.cache_size must be > 0, got %d", c.OGImage.CacheSize)
	}
	for _, p := range c.Pages {
		if !strings.HasPrefix(p.Path, "/") {
			return fmt.Errorf("pages: path %q must start with /", p.Path)
		}
		if p.Slug == "" {
			return fmt.Errorf("pages: slug for path %q must not be empty", p.Path)
		}
	}
	for i, s := range c.Sources {
		switch s.Type {
		case "filesystem":
			if s.Path == "" {
				return fmt.Errorf("sources[%d]: filesystem source requires path", i)
			}
		case "git":
			if s.URL == "" {
				return fmt.Errorf("sources[%d]: git source requires url", i)
			}
			if s.AuthToken != "" && s.SSHKeyPath != "" {
				return fmt.Errorf("sources[%d]: auth_token and ssh_key_path are mutually exclusive", i)
			}
		default:
			return fmt.Errorf("sources[%d]: unknown type %q (must be \"filesystem\" or \"git\")", i, s.Type)
		}
	}
	return nil
}

func Get() (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	data, err := FetchResource(cfg.Paths.Config)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if err := cleanenv.ParseYAML(bytes.NewReader(data), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}
