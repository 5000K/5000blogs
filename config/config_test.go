package config

import (
	"testing"
)

func TestValidate(t *testing.T) {
	valid := Config{PageSize: 10, SiteURL: "http://localhost:8080"}

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid", valid, false},
		{"page_size zero", Config{PageSize: 0, SiteURL: "http://localhost:8080"}, true},
		{"page_size negative", Config{PageSize: -1, SiteURL: "http://localhost:8080"}, true},
		{"site_url relative", Config{PageSize: 1, SiteURL: "/relative"}, true},
		{"site_url empty", Config{PageSize: 1, SiteURL: ""}, true},
		{"site_url no scheme", Config{PageSize: 1, SiteURL: "example.com"}, true},
		{"site_url https", Config{PageSize: 5, SiteURL: "https://example.com"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
