package incoming

import (
	"5000blogs/config"
	"5000blogs/service"
	"5000blogs/view"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// checkLastModified sets the Last-Modified header and returns true (with 304)
// if the client's If-Modified-Since indicates the resource is still fresh.
func checkLastModified(w http.ResponseWriter, r *http.Request, t time.Time) bool {
	if t.IsZero() {
		return false
	}
	t = t.UTC().Truncate(time.Second)
	w.Header().Set("Last-Modified", t.Format(http.TimeFormat))
	if ims := r.Header.Get("If-Modified-Since"); ims != "" {
		if parsed, err := http.ParseTime(ims); err == nil && !t.After(parsed.UTC()) {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}
	return false
}

func Serve(cfg *config.Config, repo service.PostRepository, renderer *view.Renderer, ogGen *service.OGImageGenerator, iconData []byte) {
	srv := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: buildRouter(cfg, repo, renderer, ogGen, iconData),
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
	}
}

// reservedPaths lists URL paths explicitly registered by the static router.
// Configured page routes matching these paths or their prefixes are skipped.
var reservedPaths = map[string]bool{
	"/":            true,
	"/posts":       true,
	"/feed.xml":    true,
	"/feed.atom":   true,
	"/health":      true,
	"/favicon.ico": true,
	"/og-logo.png": true,
	"/robots.txt":  true,
	"/sitemap.xml": true,
}

var reservedPrefixes = []string{"/api/", "/posts/", "/plain/", "/media/"}

func isReservedPath(path string) bool {
	if reservedPaths[path] {
		return true
	}
	for _, prefix := range reservedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// pathToSlug converts a URL sub-path to a post slug.
// "hello" → "hello", "more/things/hello-world" → "more+things+hello-world"
func pathToSlug(urlPath string) string {
	parts := strings.FieldsFunc(urlPath, func(r rune) bool { return r == '/' })
	return strings.Join(parts, "+")
}

func buildRouter(cfg *config.Config, repo service.PostRepository, renderer *view.Renderer, ogGen *service.OGImageGenerator, iconData []byte) chi.Router {
	renderer.SetFooter(func() template.HTML {
		if post := repo.GetBySlug("footer"); post != nil {
			if data := post.Data(); len(data.Content) > 0 {
				return template.HTML(data.Content) //nolint:gosec
			}
		}
		return ""
	})

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Mount("/api/v1", apiRouter(repo))

	// Serve media files (images, videos, etc.) from the post sources.
	r.Get("/media/*", func(w http.ResponseWriter, r *http.Request) {
		relPath := chi.URLParam(r, "*")
		// Prevent serving raw markdown through the media endpoint.
		if strings.HasSuffix(relPath, ".md") {
			http.NotFound(w, r)
			return
		}
		// Sanitise the path: resolve inside a virtual root to prevent traversal.
		relPath = strings.TrimPrefix(path.Clean("/"+relPath), "/")
		if relPath == "" {
			http.NotFound(w, r)
			return
		}
		data, modTime, err := repo.ReadMedia(relPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		// http.ServeContent handles Content-Type detection, Range requests,
		// If-Modified-Since / Last-Modified, and ETag caching automatically.
		http.ServeContent(w, r, relPath, modTime, bytes.NewReader(data))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	serveIcon := func(w http.ResponseWriter, r *http.Request) {
		if len(iconData) == 0 {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, "icon.png", time.Time{}, bytes.NewReader(iconData))
	}
	r.Get("/favicon.ico", serveIcon)
	r.Get("/og-logo.png", serveIcon)

	serve404 := func(w http.ResponseWriter, r *http.Request) {
		renderer.Serve404(repo.GetBySlug("404"), w)
	}

	r.NotFound(serve404)

	r.Get("/posts/*", func(w http.ResponseWriter, r *http.Request) {
		rest := chi.URLParam(r, "*")

		if strings.Contains(rest, "+") {
			http.Redirect(w, r, "/posts/"+strings.ReplaceAll(rest, "+", "/"), http.StatusMovedPermanently)
			return
		}

		if strings.HasSuffix(rest, "/og-image.png") {
			if ogGen == nil {
				http.NotFound(w, r)
				return
			}
			slug := pathToSlug(strings.TrimSuffix(rest, "/og-image.png"))
			post := repo.GetBySlug(slug)
			if post == nil {
				http.NotFound(w, r)
				return
			}
			data, err := ogGen.Generate(post)
			if err != nil {
				http.Error(w, "failed to generate og:image", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "public, max-age=86400")
			_, _ = w.Write(data)
			return
		}

		slug := pathToSlug(rest)
		post := repo.GetBySlug(slug)
		if post == nil {
			serve404(w, r)
			return
		}
		if checkLastModified(w, r, post.ModTime()) {
			return
		}
		var ogImageURL string
		if ogGen != nil {
			ogImageURL = cfg.SiteURL + "/posts/" + rest + "/og-image.png"
		}
		renderer.ServePost(post, w, cfg.SiteURL+r.URL.RequestURI(), ogImageURL)
	})

	r.Get("/plain/*", func(w http.ResponseWriter, r *http.Request) {
		slug := pathToSlug(chi.URLParam(r, "*"))
		post := repo.GetBySlug(slug)
		if post == nil {
			serve404(w, r)
			return
		}
		plain := post.PlainText()
		if plain == nil {
			http.Error(w, "plain text not available", http.StatusNotFound)
			return
		}
		if checkLastModified(w, r, post.ModTime()) {
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(plain)
	})

	homeSlug := "home"
	for _, p := range cfg.Pages {
		if p.Path == "/" {
			homeSlug = p.Slug
			break
		}
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if home := repo.GetBySlug(homeSlug); home != nil {
			if data := home.Data(); len(data.Content) > 0 {
				if checkLastModified(w, r, home.ModTime()) {
					return
				}
				renderer.ServePost(home, w, cfg.SiteURL+"/", "")
				return
			}
		}
		if checkLastModified(w, r, repo.LastModified()) {
			return
		}
		renderer.ServePostList(repo.GetPage(1, nil), w, cfg.SiteURL+"/posts")
	})

	r.Get("/posts", func(w http.ResponseWriter, r *http.Request) {
		var tags []string
		if t := r.URL.Query().Get("tags"); t != "" {
			tags = strings.Split(t, ",")
		}
		if q := r.URL.Query().Get("q"); q != "" {
			results := repo.Search(q)
			if len(tags) > 0 {
				filtered := results[:0]
				for _, p := range results {
					for _, pt := range p.Tags {
						for _, ft := range tags {
							if pt == ft {
								filtered = append(filtered, p)
								goto nextResult
							}
						}
					}
				nextResult:
				}
				results = filtered
			}
			renderer.ServeSearchResults(q, tags, results, w, cfg.SiteURL+r.URL.RequestURI())
			return
		}
		if checkLastModified(w, r, repo.LastModified()) {
			return
		}
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				page = n
			}
		}
		renderer.ServePostList(repo.GetPage(page, tags), w, cfg.SiteURL+r.URL.RequestURI())
	})

	r.Get("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		var tags []string
		if t := r.URL.Query().Get("tags"); t != "" {
			tags = strings.Split(t, ",")
		}
		q := r.URL.Query().Get("q")
		data, err := service.BuildRSSFeed(cfg, repo.FeedPosts(tags, q))
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
		data, err := service.BuildAtomFeed(cfg, repo.FeedPosts(tags, q))
		if err != nil {
			http.Error(w, "failed to generate atom feed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		_, _ = w.Write(data)
	})

	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", cfg.SiteURL)
	})

	for _, pg := range cfg.Pages {
		if isReservedPath(pg.Path) {
			continue
		}
		slug := pg.Slug
		path := pg.Path
		r.Get(path, func(w http.ResponseWriter, r *http.Request) {
			post := repo.GetBySlug(slug)
			if post == nil {
				serve404(w, r)
				return
			}
			if data := post.Data(); len(data.Content) == 0 {
				serve404(w, r)
				return
			}
			if checkLastModified(w, r, post.ModTime()) {
				return
			}
			renderer.ServePost(post, w, cfg.SiteURL+path, "")
		})
	}

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

	return r
}
