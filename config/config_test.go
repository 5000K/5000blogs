package config

import (
	"testing"
)

func TestValidate(t *testing.T) {
	valid := Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080"}
	valid.OGImage.CacheSize = 128

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid", valid, false},
		{"page_size zero", Config{PageSize: 0, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}}, true},
		{"page_size negative", Config{PageSize: -1, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}}, true},
		{"feed_size zero", Config{PageSize: 10, FeedSize: 0, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}}, true},
		{"feed_size negative", Config{PageSize: 10, FeedSize: -1, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}}, true},
		{"rss_content invalid", Config{PageSize: 10, FeedSize: 20, RSSContent: "full", SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}}, true},
		{"site_url relative", Config{PageSize: 1, FeedSize: 20, SiteURL: "/relative", OGImage: OGImageConfig{CacheSize: 128}}, true},
		{"site_url empty", Config{PageSize: 1, FeedSize: 20, SiteURL: "", OGImage: OGImageConfig{CacheSize: 128}}, true},
		{"site_url no scheme", Config{PageSize: 1, FeedSize: 20, SiteURL: "example.com", OGImage: OGImageConfig{CacheSize: 128}}, true},
		{"site_url https", Config{PageSize: 5, FeedSize: 20, SiteURL: "https://example.com", OGImage: OGImageConfig{CacheSize: 128}}, false},
		{"cache_size zero", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 0}}, true},
		{"cache_size negative", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: -1}}, true},
		{"pages valid", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}, Pages: []PageRoute{{Path: "/about", Slug: "about"}}}, false},
		{"pages path no slash", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}, Pages: []PageRoute{{Path: "about", Slug: "about"}}}, true},
		{"pages slug empty", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}, Pages: []PageRoute{{Path: "/about", Slug: ""}}}, true}, {"source filesystem valid", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}, Sources: []SourceConfig{{Type: "filesystem", Path: "./posts"}}}, false},
		{"source filesystem no path", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}, Sources: []SourceConfig{{Type: "filesystem"}}}, true},
		{"source git valid", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}, Sources: []SourceConfig{{Type: "git", URL: "https://github.com/x/y"}}}, false},
		{"source git no url", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}, Sources: []SourceConfig{{Type: "git"}}}, true},
		{"source unknown type", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}, Sources: []SourceConfig{{Type: "s3"}}}, true},
		{"source git token auth", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}, Sources: []SourceConfig{{Type: "git", URL: "https://github.com/x/y", AuthToken: "tok"}}}, false},
		{"source git ssh auth", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}, Sources: []SourceConfig{{Type: "git", URL: "git@github.com:x/y", SSHKeyPath: "/id_rsa"}}}, false},
		{"source git conflicting auth", Config{PageSize: 10, FeedSize: 20, SiteURL: "http://localhost:8080", OGImage: OGImageConfig{CacheSize: 128}, Sources: []SourceConfig{{Type: "git", URL: "https://github.com/x/y", AuthToken: "tok", SSHKeyPath: "/id_rsa"}}}, true}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
