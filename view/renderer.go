package view

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
)

type navLink struct {
	Name string
	URL  string
}

// templateData is the data passed to the HTML template for every page type.
// IsListPage selects the list view; all other fields are for the post view.
type templateData struct {
	// Shared
	Title       string
	Description string
	URL         string // canonical page URL
	OGImageURL  string // absolute URL for og:image; empty when not available
	OGLogoURL   string // absolute URL for og:logo; empty when not configured
	Plugins     []string
	BlogName    string
	NavLinks    []navLink
	Slug        string // for setting active nav link; empty on list pages

	// Post view
	DateStr string
	DateISO string // RFC 3339, for <time datetime>
	Author  string
	Tags    []string
	Content template.HTML
	NoIndex bool

	// Injected by SetFooter
	FooterContent template.HTML

	// List view
	IsListPage  bool
	SearchQuery string // non-empty when rendering search results
	FilterTags  []string
	Posts       []postListItem
	Pagination  paginationData
}

type postListItem struct {
	Slug        string
	Title       string
	Description string
	DateStr     string
	Author      string
	Tags        []string
}

type paginationData struct {
	Page       int
	TotalPages int
	TotalPosts int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
	TagParam   string // "&tags=foo,bar" when tag filter is active; empty otherwise
}

type Renderer interface {
	Serve404(post *service.Post, w http.ResponseWriter)
	ServePost(post *service.Post, w http.ResponseWriter, pageURL string, ogImageURL string)
	ServeSearchResults(query string, tags []string, results []service.PostSummary, w http.ResponseWriter, pageURL string)
	ServePostList(pr service.PageResult, w http.ResponseWriter, pageURL string)
	SetFooter(fn func() template.HTML)
}

// DefaultRenderer loads an HTML template and renders posts through it.
type DefaultRenderer struct {
	cfg        *config.Config
	log        *slog.Logger
	tmpl       *template.Template
	footerHTML func() template.HTML
}

// SetFooter registers a function that returns the rendered footer content.
// It is called on every request, so changes to the underlying post are picked up automatically.
func (r *DefaultRenderer) SetFooter(fn func() template.HTML) {
	r.footerHTML = fn
}

func (r *DefaultRenderer) footer() template.HTML {
	if r.footerHTML == nil {
		return ""
	}
	return r.footerHTML()
}

// NewRenderer creates a Renderer using the provided template bytes.
// Returns an error if the template cannot be parsed.
func NewRenderer(cfg config.Config, tmplData []byte, logger *slog.Logger) (*DefaultRenderer, error) {
	tmpl, err := template.New("template.html").Parse(string(tmplData))
	if err != nil {
		return nil, fmt.Errorf("view.Renderer: parse template: %w", err)
	}
	return &DefaultRenderer{
		cfg:  &cfg,
		log:  logger.With("component", "Renderer"),
		tmpl: tmpl,
	}, nil
}

func (r *DefaultRenderer) Initialize() error {
	return nil
}

func (r *DefaultRenderer) navLinks() []navLink {
	links := make([]navLink, len(r.cfg.NavLinks))
	for i, l := range r.cfg.NavLinks {
		links[i] = navLink{Name: l.Name, URL: l.URL}
	}
	return links
}

// ogLogoURL returns the absolute URL of the logo if an icon is configured.
func (r *DefaultRenderer) ogLogoURL() string {
	if r.cfg.Paths.Icon == "" {
		return ""
	}
	return r.cfg.SiteURL + "/og-logo.png"
}

// Serve404 renders a 404 page with HTTP 404 status. If post is provided and
// has content it is rendered; otherwise a placeholder title is used.
func (r *DefaultRenderer) Serve404(post *service.Post, w http.ResponseWriter) {
	td := templateData{
		Title:         "404 - Page Not Found",
		OGLogoURL:     r.ogLogoURL(),
		Plugins:       r.cfg.Plugins,
		BlogName:      r.cfg.BlogName,
		NavLinks:      r.navLinks(),
		FooterContent: r.footer(),
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
// ogImageURL is the absolute URL of the og:image for this post; pass empty string to omit.
func (r *DefaultRenderer) ServePost(post *service.Post, w http.ResponseWriter, pageURL string, ogImageURL string) {
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
		Title:         data.Title,
		Description:   data.Description,
		URL:           pageURL,
		OGImageURL:    ogImageURL,
		OGLogoURL:     r.ogLogoURL(),
		Author:        data.Author,
		Tags:          data.Tags,
		Content:       template.HTML(data.Content), //nolint:gosec // content is markdown-rendered HTML
		DateISO:       data.DateISO,
		NoIndex:       data.NoIndex,
		Plugins:       r.cfg.Plugins,
		BlogName:      r.cfg.BlogName,
		NavLinks:      r.navLinks(),
		FooterContent: r.footer(),
		Slug:          data.Slug,
	}
	if !data.Date.IsZero() {
		td.DateStr = data.Date.Format("January 2, 2006")
	}

	r.execute(w, td)
}

// ServeSearchResults renders a list of search results through the HTML template.
// It reuses the list-page layout; SearchQuery and FilterTags are exposed to the template.
func (r *DefaultRenderer) ServeSearchResults(query string, tags []string, results []service.PostSummary, w http.ResponseWriter, pageURL string) {
	items := make([]postListItem, 0, len(results))
	for _, p := range results {
		item := postListItem{
			Slug:        p.Slug,
			Title:       p.Title,
			Description: p.Description,
			Author:      p.Author,
			Tags:        p.Tags,
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
		Title:         "Search: " + query,
		URL:           pageURL,
		OGLogoURL:     r.ogLogoURL(),
		IsListPage:    true,
		SearchQuery:   query,
		FilterTags:    tags,
		Posts:         items,
		Plugins:       r.cfg.Plugins,
		BlogName:      r.cfg.BlogName,
		NavLinks:      r.navLinks(),
		FooterContent: r.footer(),
	}
	r.execute(w, td)
}

// ServePostList renders a paginated post list through the HTML template.
func (r *DefaultRenderer) ServePostList(pr service.PageResult, w http.ResponseWriter, pageURL string) {
	items := make([]postListItem, 0, len(pr.Posts))
	for _, p := range pr.Posts {
		item := postListItem{
			Slug:        p.Slug,
			Title:       p.Title,
			Description: p.Description,
			Author:      p.Author,
			Tags:        p.Tags,
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
		Title:         "Posts",
		URL:           pageURL,
		OGLogoURL:     r.ogLogoURL(),
		IsListPage:    true,
		FilterTags:    pr.FilterTags,
		Posts:         items,
		Plugins:       r.cfg.Plugins,
		BlogName:      r.cfg.BlogName,
		NavLinks:      r.navLinks(),
		FooterContent: r.footer(),
		Pagination: paginationData{
			Page:       pr.Page,
			TotalPages: pr.TotalPages,
			TotalPosts: pr.TotalPosts,
			HasPrev:    pr.HasPrev,
			HasNext:    pr.HasNext,
			PrevPage:   pr.PrevPage,
			NextPage:   pr.NextPage,
			TagParam:   pr.TagParam,
		},
	}

	r.execute(w, td)
}

func (r *DefaultRenderer) execute(w http.ResponseWriter, td templateData) {
	r.executeStatus(w, td, http.StatusOK)
}

func (r *DefaultRenderer) executeStatus(w http.ResponseWriter, td templateData, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := r.tmpl.Execute(w, td); err != nil {
		r.log.Error("failed to execute template", "title", td.Title, "err", err)
	}
}
