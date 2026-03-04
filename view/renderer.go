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

	// Post view
	DateStr string
	Author  string
	Content template.HTML

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

// ServePost renders the given post through the HTML template and writes the
// response. Responds with 404 when the post is nil or has no rendered content.
func (r *Renderer) ServePost(post *service.Post, w http.ResponseWriter) {
	if post == nil {
		r.log.Debug("post not found")
		http.NotFound(w, nil)
		return
	}

	data := post.Data()
	if len(data.Content) == 0 {
		r.log.Debug("post has no rendered content", "title", data.Title)
		http.NotFound(w, nil)
		return
	}

	td := templateData{
		Title:       data.Title,
		Description: data.Description,
		Author:      data.Author,
		Content:     template.HTML(data.Content), //nolint:gosec // content is markdown-rendered HTML
	}
	if !data.Date.IsZero() {
		td.DateStr = data.Date.Format("January 2, 2006")
	}

	r.execute(w, td)
}

// ServePostList renders a paginated post list through the HTML template.
func (r *Renderer) ServePostList(pr service.PageResult, w http.ResponseWriter) {
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
		IsListPage: true,
		Posts:      items,
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := r.tmpl.Execute(w, td); err != nil {
		r.log.Error("failed to execute template", "title", td.Title, "err", err)
	}
}
