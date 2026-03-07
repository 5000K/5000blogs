package service

import (
	"path/filepath"
	"time"
)

type Metadata struct {
	Title       string    `yaml:"title"`
	Description string    `yaml:"description"`
	Date        time.Time `yaml:"date"`
	Author      string    `yaml:"author"`
	Visible     *bool     `yaml:"visible"`
	RSSVisible  *bool     `yaml:"rss-visible"`
	NoIndex     *bool     `yaml:"noindex"`

	Raw map[string]interface{} `yaml:",inline"`
}

type Post struct {
	path    string
	hash    uint64
	modTime time.Time

	metadata *Metadata
	contents *[]byte
}

// PostData holds the rendered data for a post, safe to pass to a view layer.
type PostData struct {
	Slug        string
	Title       string
	Description string
	Date        time.Time
	DateISO     string // RFC 3339, empty when no date
	Author      string
	Content     []byte // rendered HTML
	Visible     bool
	RSSVisible  bool
	NoIndex     bool
}

// Data returns a PostData view of the post.
func (p *Post) Data() PostData {
	d := PostData{
		Slug:       slugFromPath(p.path),
		Visible:    p.IsVisible(),
		RSSVisible: p.IsRSSVisible(),
	}
	if p.metadata != nil {
		d.Title = p.metadata.Title
		d.Description = p.metadata.Description
		d.Date = p.metadata.Date
		d.Author = p.metadata.Author
		if p.metadata.NoIndex != nil {
			d.NoIndex = *p.metadata.NoIndex
		}
	}
	// Fall back to file modification time when no date is set in metadata.
	if d.Date.IsZero() {
		d.Date = p.modTime
	}
	if !d.Date.IsZero() {
		d.DateISO = d.Date.Format(time.RFC3339)
	}
	if p.contents != nil {
		d.Content = *p.contents
	}
	return d
}

func (p *Post) IsVisible() bool {
	if p.metadata == nil || p.metadata.Visible == nil {
		return true
	}
	return *p.metadata.Visible
}

func (p *Post) IsRSSVisible() bool {
	if !p.IsVisible() {
		return false
	}
	if p.metadata == nil || p.metadata.RSSVisible == nil {
		return true
	}
	return *p.metadata.RSSVisible
}

// PostSummary is a lightweight view of a post for list pages.
type PostSummary struct {
	Slug        string
	Title       string
	Description string
	Date        time.Time
	Author      string
}

// PageResult is the output of GetPage.
type PageResult struct {
	Posts      []PostSummary
	Page       int
	PageSize   int
	TotalPosts int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
}

// slugFromPath derives URL slug from file path
func slugFromPath(path string) string {
	base := filepath.Base(path)
	if ext := filepath.Ext(base); ext != "" {
		return base[:len(base)-len(ext)]
	}
	return base
}
