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
	Tags        []string  `yaml:"tags"`
	MetaTags    []string  `yaml:"meta-tags"`
	Visible     *bool     `yaml:"visible"`
	RSSVisible  *bool     `yaml:"rss-visible"`
	NoIndex     *bool     `yaml:"noindex"`

	Raw map[string]interface{} `yaml:",inline"`
}

type Post struct {
	Slug    string
	hash    uint64
	modTime time.Time

	Metadata  *Metadata
	Contents  *[]byte
	plainText *[]byte
}

// PostData holds the rendered data for a post, safe to pass to a view layer.
type PostData struct {
	Slug        string
	Title       string
	Description string
	Date        time.Time
	DateISO     string // RFC 3339, empty when no date
	Author      string
	Tags        []string
	Content     []byte // rendered HTML
	Visible     bool
	RSSVisible  bool
	NoIndex     bool
}

// Data returns a PostData view of the post.
func (p *Post) Data() PostData {
	slug := p.Slug
	if slug == "" {
		slug = slugFromPath(p.Slug)
	}
	d := PostData{
		Slug:       slug,
		Visible:    p.IsVisible(),
		RSSVisible: p.IsRSSVisible(),
	}
	if p.Metadata != nil {
		d.Title = p.Metadata.Title
		d.Description = p.Metadata.Description
		d.Date = p.Metadata.Date
		d.Author = p.Metadata.Author
		d.Tags = p.Metadata.Tags
		if p.Metadata.NoIndex != nil {
			d.NoIndex = *p.Metadata.NoIndex
		}
	}
	// Fall back to file modification time when no date is set in metadata.
	if d.Date.IsZero() {
		d.Date = p.modTime
	}
	if !d.Date.IsZero() {
		d.DateISO = d.Date.Format(time.RFC3339)
	}
	if p.Contents != nil {
		d.Content = *p.Contents
	}
	return d
}

// ModTime returns the file modification time of the post, used for HTTP caching.
func (p *Post) ModTime() time.Time {
	return p.modTime
}

// PlainText returns the post body as plain text (HTML and markdown formatting stripped).
func (p *Post) PlainText() []byte {
	if p.plainText == nil {
		return nil
	}
	return *p.plainText
}

func (p *Post) IsVisible() bool {
	if p.Metadata == nil || p.Metadata.Visible == nil {
		return true
	}
	return *p.Metadata.Visible
}

func (p *Post) IsRSSVisible() bool {
	if !p.IsVisible() {
		return false
	}
	if p.Metadata == nil || p.Metadata.RSSVisible == nil {
		return true
	}
	return *p.Metadata.RSSVisible
}

// PostSummary is a lightweight view of a post for list pages.
type PostSummary struct {
	Slug        string
	Title       string
	Description string
	Date        time.Time
	Author      string
	Tags        []string
	MetaTags    []string // not rendered; used for internal tag filtering only
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
	FilterTags []string // active tag filter; nil when unfiltered
	TagParam   string   // e.g. "&tags=foo,bar" for use in pagination links; empty when no filter
}

// slugFromPath derives URL slug from file path (basename without extension).
// Used only as a fallback; sources should derive slugs via SlugForPath.
func slugFromPath(path string) string {
	base := filepath.Base(path)
	if ext := filepath.Ext(base); ext != "" {
		return base[:len(base)-len(ext)]
	}
	return base
}
