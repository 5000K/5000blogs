package incoming

import (
	"5000blogs/service"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func apiRouter(repo service.PostRepository) chi.Router {
	r := chi.NewRouter()
	r.Get("/posts", apiListPosts(repo))
	r.Get("/posts/search", apiSearchPosts(repo))
	r.Get("/post/{name}", apiGetPost(repo))
	r.Get("/stats", apiStats(repo))
	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// GET /api/v1/posts — all post slugs, no sorting.
func apiListPosts(repo service.PostRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		posts := repo.List()
		names := make([]string, 0, len(posts))
		for _, p := range posts {
			names = append(names, p.Data().Slug)
		}
		writeJSON(w, http.StatusOK, names)
	}
}

// GET /api/v1/post/{name} — full metadata for a single post.
func apiGetPost(repo service.PostRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		post := repo.GetBySlug(name)
		if post == nil {
			http.NotFound(w, r)
			return
		}
		d := post.Data()
		writeJSON(w, http.StatusOK, toPostMeta(d))
	}
}

// GET /api/v1/posts/search?q={query} — title/description substring match.
func apiSearchPosts(repo service.PostRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.ToLower(r.URL.Query().Get("q"))
		posts := repo.List()
		type result struct {
			Slug        string `json:"slug"`
			Title       string `json:"title"`
			Description string `json:"description,omitempty"`
		}
		results := make([]result, 0)
		for _, p := range posts {
			d := p.Data()
			if strings.Contains(strings.ToLower(d.Title), q) ||
				strings.Contains(strings.ToLower(d.Description), q) {
				results = append(results, result{
					Slug:        d.Slug,
					Title:       d.Title,
					Description: d.Description,
				})
			}
		}
		writeJSON(w, http.StatusOK, results)
	}
}

// GET /api/v1/stats — aggregate blog stats.
func apiStats(repo service.PostRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		posts := repo.List()
		var count int
		var latest time.Time
		for _, p := range posts {
			if !p.IsVisible() {
				continue
			}
			count++
			d := p.Data()
			if d.Date.After(latest) {
				latest = d.Date
			}
		}
		type stats struct {
			TotalPosts     int        `json:"total_posts"`
			LatestPostDate *time.Time `json:"latest_post_date"`
		}
		s := stats{TotalPosts: count}
		if !latest.IsZero() {
			s.LatestPostDate = &latest
		}
		writeJSON(w, http.StatusOK, s)
	}
}

// postMeta is the JSON shape for a single post's metadata.
type postMeta struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Date        time.Time `json:"date,omitempty"`
	Author      string    `json:"author,omitempty"`
	Visible     bool      `json:"visible"`
	RSSVisible  bool      `json:"rss_visible"`
	NoIndex     bool      `json:"noindex,omitempty"`
}

func toPostMeta(d service.PostData) postMeta {
	return postMeta{
		Slug:        d.Slug,
		Title:       d.Title,
		Description: d.Description,
		Date:        d.Date,
		Author:      d.Author,
		Visible:     d.Visible,
		RSSVisible:  d.RSSVisible,
		NoIndex:     d.NoIndex,
	}
}
