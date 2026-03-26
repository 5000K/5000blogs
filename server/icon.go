package server

import (
	"bytes"
	"net/http"
	"time"

	"github.com/5000K/5000blogs/config"
	"github.com/go-chi/chi/v5"
)

type IconModule struct {
	iconData []byte
}

func NewIconModule(iconData []byte) *IconModule {
	return &IconModule{iconData: iconData}
}

func (m *IconModule) RegisterRoutes(r chi.Router, conf *config.ConfigLoader) error {
	serveIcon := func(w http.ResponseWriter, r *http.Request) {
		if len(m.iconData) == 0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeContent(w, r, "icon.png", time.Time{}, bytes.NewReader(m.iconData))
	}

	r.Get("/favicon.ico", serveIcon)
	r.Get("/og-logo.png", serveIcon)
	return nil
}
