package incoming

import (
	"5000blogs/config"
	"5000blogs/service"
	"5000blogs/view"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Serve(cfg *config.Config, repo service.PostRepository, renderer *view.Renderer) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, cfg.Paths.Static))
	FileServer(r, "/static", filesDir)

	notFoundPath := filepath.Join(cfg.Paths.Posts, "404.md")
	serve404 := func(w http.ResponseWriter, r *http.Request) {
		renderer.Serve404(repo.Get(notFoundPath), w)
	}

	r.NotFound(serve404)

	r.Get("/posts/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		path := filepath.Join(cfg.Paths.Posts, slug+".md")
		post := repo.Get(path)
		if post == nil {
			serve404(w, r)
			return
		}
		renderer.ServePost(post, w, cfg.SiteURL+r.URL.RequestURI())
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		homePath := filepath.Join(cfg.Paths.Posts, "home.md")
		if home := repo.Get(homePath); home != nil {
			if data := home.Data(); len(data.Content) > 0 {
				renderer.ServePost(home, w, cfg.SiteURL+"/")
				return
			}
		}
		renderer.ServePostList(repo.GetPage(1), w, cfg.SiteURL+"/posts")
	})

	r.Get("/posts", func(w http.ResponseWriter, r *http.Request) {
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				page = n
			}
		}
		renderer.ServePostList(repo.GetPage(page), w, cfg.SiteURL+r.URL.RequestURI())
	})

	r.Get("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		data, err := repo.RSSFeed()
		if err != nil {
			http.Error(w, "failed to generate feed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		_, _ = w.Write(data)
	})

	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", cfg.SiteURL)
	})

	r.Get("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		entries := repo.Sitemap()
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
		urls = append(urls, url{Loc: cfg.SiteURL + "/posts"})
		for _, e := range entries {
			u := url{Loc: cfg.SiteURL + "/posts/" + e.Slug}
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

	_ = http.ListenAndServe(cfg.ServerAddress, r)
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
