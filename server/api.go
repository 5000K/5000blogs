package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/go-chi/chi/v5"
)

type ApiModule struct {
	indexer service.PostIndexer
}

func NewApiModule(indexer service.PostIndexer) *ApiModule {
	return &ApiModule{indexer: indexer}
}

type apiPageConfig struct {
	PageSize int `env:"PAGE_SIZE" env-default:"10" yaml:"page_size"`
}

type postMetaJSON struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Author      string    `json:"author"`
	Tags        []string  `json:"tags"`
	Visible     bool      `json:"visible"`
	RSSVisible  bool      `json:"rss_visible"`
	NoIndex     bool      `json:"noindex"`
}

type postSummaryJSON struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Author      string    `json:"author"`
	Tags        []string  `json:"tags"`
}

type pagedPostsJSON struct {
	Posts      []postSummaryJSON `json:"posts"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPosts int               `json:"total_posts"`
	TotalPages int               `json:"total_pages"`
	HasPrev    bool              `json:"has_prev"`
	HasNext    bool              `json:"has_next"`
}

type statsJSON struct {
	LastChange       time.Time `json:"last_change"`
	VisiblePostCount int       `json:"visible_post_count"`
}

func (m *ApiModule) RegisterRoutes(r chi.Router, conf *config.ConfigLoader) error {
	var cfg apiPageConfig
	if err := conf.Load("", &cfg); err != nil {
		return err
	}

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/posts", m.listPosts)
		r.Get("/posts/page/{page}", m.listPostsPagedHandler(cfg.PageSize))
		r.Get("/post/{slug}", m.getPost)
		r.Get("/tags", m.listTags)
		r.Get("/stats", m.getStats)
	})

	return nil
}

func (m *ApiModule) listPosts(w http.ResponseWriter, r *http.Request) {
	filter := service.PostFilter{
		Tags:  parseTags(r),
		Query: r.URL.Query().Get("q"),
	}
	posts := m.indexer.ListFiltered(filter)
	slugs := make([]string, 0, len(posts))
	for _, p := range posts {
		slugs = append(slugs, p.Data().Slug)
	}
	writeJSON(w, slugs)
}

func (m *ApiModule) listPostsPagedHandler(pageSize int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(chi.URLParam(r, "page"))
		if page < 1 {
			page = 1
		}
		filter := service.PostFilter{
			Tags:  parseTags(r),
			Query: r.URL.Query().Get("q"),
		}
		result := m.indexer.ListFilteredPaged(filter, pageSize, page)
		summaries := make([]postSummaryJSON, 0, len(result.Posts))
		for _, s := range result.Posts {
			summaries = append(summaries, postSummaryJSON{
				Slug:        s.Slug,
				Title:       s.Title,
				Description: s.Description,
				Date:        s.Date,
				Author:      s.Author,
				Tags:        normalizeTags(s.Tags),
			})
		}
		writeJSON(w, pagedPostsJSON{
			Posts:      summaries,
			Page:       result.Page,
			PageSize:   result.PageSize,
			TotalPosts: result.TotalPosts,
			TotalPages: result.TotalPages,
			HasPrev:    result.HasPrev,
			HasNext:    result.HasNext,
		})
	}
}

func (m *ApiModule) getPost(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	post := m.indexer.GetBySlug(slug)
	if post == nil {
		http.NotFound(w, r)
		return
	}
	d := post.Data()
	writeJSON(w, postMetaJSON{
		Slug:        d.Slug,
		Title:       d.Title,
		Description: d.Description,
		Date:        d.Date,
		Author:      d.Author,
		Tags:        normalizeTags(d.Tags),
		Visible:     d.Visible,
		RSSVisible:  d.RSSVisible,
		NoIndex:     d.NoIndex,
	})
}

func (m *ApiModule) listTags(w http.ResponseWriter, r *http.Request) {
	tags := m.indexer.AllTags()
	writeJSON(w, normalizeTags(tags))
}

func (m *ApiModule) getStats(w http.ResponseWriter, r *http.Request) {
	visible := m.indexer.ListFiltered(service.PostFilter{})
	writeJSON(w, statsJSON{
		LastChange:       m.indexer.LastModified(),
		VisiblePostCount: len(visible),
	})
}

func parseTags(r *http.Request) []string {
	raw := r.URL.Query().Get("tags")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func normalizeTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
