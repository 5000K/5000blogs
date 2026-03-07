package view

import (
	"5000blogs/config"
	"5000blogs/service"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

// templateData is the data passed to the HTML template for every page type.
// IsListPage selects the list view; all other fields are for the post view.
type templateData struct {
	// Shared
	Title       string
	Description string
	URL         string // canonical page URL
	Plugins     []string

	// Post view
	DateStr string
	DateISO string // RFC 3339, for <time datetime>
	Author  string
	Content template.HTML
	NoIndex bool

	// List view
	IsListPage bool
	Posts      []postListItem
	Pagination paginationData
}

type postListItem struct {
	Slug        string
	Title       string
	Description string
	DateStr     string
	Author      string
}

type paginationData struct {
	Page       int
	TotalPages int
	TotalPosts int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
}

// Renderer loads an HTML template from disk and renders posts through it.
type Renderer struct {
	cfg      *config.Config
	log      *slog.Logger
	tmplPath string
	tmpl     *template.Template
}

// NewRenderer creates a Renderer, loading the template from the static directory.
// Returns an error if the template file cannot be parsed.
func NewRenderer(cfg *config.Config, logger *slog.Logger) (*Renderer, error) {
	r := &Renderer{
		cfg:      cfg,
		log:      logger.With("component", "Renderer"),
		tmplPath: filepath.Join(cfg.Paths.Static, "template.html"),
	}
	if err := r.reload(); err != nil {
		return nil, err
	}
	return r, nil
}

// reload re-parses the template file from disk.
func (r *Renderer) reload() error {
	raw, err := os.ReadFile(r.tmplPath)
	if err != nil {
		return fmt.Errorf("view.Renderer: read template %q: %w", r.tmplPath, err)
	}
	tmpl, err := template.New(filepath.Base(r.tmplPath)).Parse(string(raw))
	if err != nil {
		return fmt.Errorf("view.Renderer: parse template %q: %w", r.tmplPath, err)
	}
	r.tmpl = tmpl
	return nil
}

// Serve404 renders a 404 page with HTTP 404 status. If post is provided and
// has content it is rendered; otherwise a placeholder title is used.
func (r *Renderer) Serve404(post *service.Post, w http.ResponseWriter) {
	td := templateData{
		Title:   "404 - Page Not Found",
		Plugins: r.cfg.Plugins,
	}
	if post != nil {
		data := post.Data()
		if len(data.Content) > 0 {
			td.Title = data.Title
			td.Description = data.Description
			td.Author = data.Author
			td.Content = template.HTML(data.Content) //nolint:gosec
		}
	}
	r.executeStatus(w, td, http.StatusNotFound)
}

// ServePost renders the given post through the HTML template and writes the
// response. Responds with 404 when the post is nil or has no rendered content.
func (r *Renderer) ServePost(post *service.Post, w http.ResponseWriter, pageURL string) {
	if post == nil {
		r.log.Debug("post not found")
		r.Serve404(nil, w)
		return
	}

	data := post.Data()
	if len(data.Content) == 0 {
		r.log.Debug("post has no rendered content", "title", data.Title)
		r.Serve404(nil, w)
		return
	}

	td := templateData{
		Title:       data.Title,
		Description: data.Description,
		URL:         pageURL,
		Author:      data.Author,
		Content:     template.HTML(data.Content), //nolint:gosec // content is markdown-rendered HTML
		DateISO:     data.DateISO,
		NoIndex:     data.NoIndex,
		Plugins:     r.cfg.Plugins,
	}
	if !data.Date.IsZero() {
		td.DateStr = data.Date.Format("January 2, 2006")
	}

	r.execute(w, td)
}

// ServePostList renders a paginated post list through the HTML template.
func (r *Renderer) ServePostList(pr service.PageResult, w http.ResponseWriter, pageURL string) {
	items := make([]postListItem, 0, len(pr.Posts))
	for _, p := range pr.Posts {
		item := postListItem{
			Slug:        p.Slug,
			Title:       p.Title,
			Description: p.Description,
			Author:      p.Author,
		}
		if !p.Date.IsZero() {
			item.DateStr = p.Date.Format("January 2, 2006")
		}
		if item.Title == "" {
			item.Title = p.Slug
		}
		items = append(items, item)
	}

	td := templateData{
		Title:      "Posts",
		URL:        pageURL,
		IsListPage: true,
		Posts:      items,
		Plugins:    r.cfg.Plugins,
		Pagination: paginationData{
			Page:       pr.Page,
			TotalPages: pr.TotalPages,
			TotalPosts: pr.TotalPosts,
			HasPrev:    pr.HasPrev,
			HasNext:    pr.HasNext,
			PrevPage:   pr.PrevPage,
			NextPage:   pr.NextPage,
		},
	}

	r.execute(w, td)
}

func (r *Renderer) execute(w http.ResponseWriter, td templateData) {
	r.executeStatus(w, td, http.StatusOK)
}

func (r *Renderer) executeStatus(w http.ResponseWriter, td templateData, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := r.tmpl.Execute(w, td); err != nil {
		r.log.Error("failed to execute template", "title", td.Title, "err", err)
	}
}
