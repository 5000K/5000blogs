package incoming

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/5000K/5000blogs/service"

	"github.com/go-chi/chi/v5"
)

func apiRouter(repo service.PostIndexer) chi.Router {
	r := chi.NewRouter()
	r.Get("/posts", apiListPosts(repo))
	r.Get("/posts/search", apiSearchPosts(repo))
	r.Get("/posts/tags", apiListTags(repo))
	r.Get("/post/*", apiGetPost(repo))
	r.Get("/search", apiSearch(repo))
	r.Get("/stats", apiStats(repo))
	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// GET /api/v1/posts - all post slugs, optionally filtered by ?tags=tag1,tag2 (OR).
func apiListPosts(repo service.PostIndexer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var filterTags []string
		if t := r.URL.Query().Get("tags"); t != "" {
			filterTags = strings.Split(t, ",")
		}
		posts := repo.List()
		names := make([]string, 0, len(posts))
		for _, p := range posts {
			if len(filterTags) > 0 {
				d := p.Data()
				if !hasTagMatch(d.Tags, filterTags) {
					continue
				}
			}
			names = append(names, p.Data().Slug)
		}
		writeJSON(w, http.StatusOK, names)
	}
}

// GET /api/v1/posts/tags - sorted list of all tags across visible posts.
func apiListTags(repo service.PostIndexer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, repo.AllTags())
	}
}

// hasTagMatch reports whether postTags contains any of the filter tags (case-insensitive).
func hasTagMatch(postTags, filter []string) bool {
	for _, want := range filter {
		for _, have := range postTags {
			if strings.EqualFold(have, want) {
				return true
			}
		}
	}
	return false
}

// GET /api/v1/post/{name} - full metadata for a single post.
func apiGetPost(repo service.PostIndexer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "*")
		post := repo.GetBySlug(slug)
		if post == nil {
			http.NotFound(w, r)
			return
		}
		d := post.Data()
		writeJSON(w, http.StatusOK, toPostMeta(d))
	}
}

// GET /api/v1/search?q={query} - full-text search; returns matching slugs.
func apiSearch(repo service.PostIndexer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		results := repo.Search(q)
		slugs := make([]string, 0, len(results))
		for _, s := range results {
			slugs = append(slugs, s.Slug)
		}
		writeJSON(w, http.StatusOK, slugs)
	}
}

// GET /api/v1/posts/search?q={query} - title/description substring match on visible posts.
func apiSearchPosts(repo service.PostIndexer) http.HandlerFunc {
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
			if !d.Visible {
				continue
			}
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

// GET /api/v1/stats - aggregate blog stats.
func apiStats(repo service.PostIndexer) http.HandlerFunc {
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
	Tags        []string  `json:"tags,omitempty"`
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
		Tags:        d.Tags,
		Visible:     d.Visible,
		RSSVisible:  d.RSSVisible,
		NoIndex:     d.NoIndex,
	}
}
