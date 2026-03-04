package incoming

import (
	"5000blogs/config"
	"5000blogs/service"
	"5000blogs/view"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Serve(cfg *config.Config, svc *service.Service, renderer *view.Renderer) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, cfg.Paths.Static))
	FileServer(r, "/static", filesDir)

	r.Get("/posts/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		path := filepath.Join(cfg.Paths.Posts, slug+".md")
		renderer.ServePost(svc.GetPost(path), w)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		homePath := filepath.Join(cfg.Paths.Posts, "home.md")
		if home := svc.GetPost(homePath); home != nil {
			if data := home.Data(); len(data.Content) > 0 {
				renderer.ServePost(home, w)
				return
			}
		}
		renderer.ServePostList(svc.GetPage(1), w)
	})

	r.Get("/posts", func(w http.ResponseWriter, r *http.Request) {
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				page = n
			}
		}
		renderer.ServePostList(svc.GetPage(page), w)
	})

	r.Get("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		data, err := svc.RSSFeed()
		if err != nil {
			http.Error(w, "failed to generate feed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		_, _ = w.Write(data)
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
