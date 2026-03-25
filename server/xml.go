package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/go-chi/chi/v5"
)

type XmlFeedModule struct {
	indexer service.PostIndexer
}

func NewFeedModule(indexer service.PostIndexer) *XmlFeedModule {
	return &XmlFeedModule{indexer: indexer}
}

func (f *XmlFeedModule) RegisterRoutes(r chi.Router, conf *config.ConfigLoader) error {
	baseConf := conf.BaseConfig()

	r.Get("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		entries := f.indexer.Sitemap()
		type url struct {
			Loc     string `xml:"loc"`
			LastMod string `xml:"lastmod,omitempty"`
		}
		type urlset struct {
			XMLName xml.Name `xml:"urlset"`
			Xmlns   string   `xml:"xmlns,attr"`
			URLs    []url    `xml:"url"`
		}
		urls := make([]url, 0, len(entries)+1)
		urls = append(urls, url{Loc: baseConf.SiteURL + "/posts"})
		for _, e := range entries {
			u := url{Loc: baseConf.SiteURL + "/" + e.Slug}
			if !e.LastMod.IsZero() {
				u.LastMod = e.LastMod.UTC().Format(time.RFC3339)
			}
			urls = append(urls, u)
		}
		set := urlset{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write([]byte(xml.Header))
		enc := xml.NewEncoder(w)
		enc.Indent("", "  ")
		_ = enc.Encode(set)
	})

	r.Get("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		var tags []string
		if t := r.URL.Query().Get("tags"); t != "" {
			tags = strings.Split(t, ",")
		}
		q := r.URL.Query().Get("q")
		data, err := service.BuildRSSFeed(&baseConf, f.indexer.FeedPosts(tags, q))
		if err != nil {
			http.Error(w, "failed to generate feed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		_, _ = w.Write(data)
	})

	r.Get("/feed.atom", func(w http.ResponseWriter, r *http.Request) {
		var tags []string
		if t := r.URL.Query().Get("tags"); t != "" {
			tags = strings.Split(t, ",")
		}
		q := r.URL.Query().Get("q")
		data, err := service.BuildAtomFeed(&baseConf, f.indexer.FeedPosts(tags, q))
		if err != nil {
			http.Error(w, "failed to generate atom feed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		_, _ = w.Write(data)
	})
	return nil
}
