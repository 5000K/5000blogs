package service

import (
	"path/filepath"
	"strings"
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
	Slug       string
	sourcePath string // original file-system path used to read from PostSource
	hash       uint64
	modTime    time.Time

	Metadata   *Metadata
	Contents   *[]byte
	plainText  *[]byte
	dateFormat string
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

	// Dates holds additional date fields parsed from frontmatter whose key ends
	// in "date" (excluding the primary "date"). Each entry exposes a human-readable
	// Str form and an RFC 3339 ISO form, keyed by the original frontmatter key.
	Dates map[string]DateStrings
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
		if d.Dates == nil {
			d.Dates = map[string]DateStrings{}
		}
		for name, val := range p.extraDates() {
			d.Dates[name] = DateStrings{
				Str: val.Format(p.resolvedDateFormat()),
				ISO: val.Format(time.RFC3339),
			}
		}
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
// DateStrings holds the two rendered string forms of a date field.
type DateStrings struct {
	Str string // human-readable form using the configured date format
	ISO string // RFC 3339 form, for <time datetime>
}

// SetDateFormat sets the layout used to render human-readable date strings
// from this post's additional date fields. When unset, Data() falls back to
// "January 2, 2006".
func (p *Post) SetDateFormat(layout string) {
	p.dateFormat = layout
}

// resolvedDateFormat returns the configured layout or the default when unset.
func (p *Post) resolvedDateFormat() string {
	if p.dateFormat == "" {
		return "January 2, 2006"
	}
	return p.dateFormat
}

// extraDates returns additional date values parsed from frontmatter keys whose
// lowercased name ends in "date", excluding the primary "date" key. The
// original frontmatter key is preserved as the map key.
func (p *Post) extraDates() map[string]time.Time {
	out := map[string]time.Time{}
	if p.Metadata == nil || p.Metadata.Raw == nil {
		return out
	}
	for key, val := range p.Metadata.Raw {
		lk := strings.ToLower(key)
		if !strings.HasSuffix(lk, "date") || lk == "date" {
			continue
		}
		switch v := val.(type) {
		case time.Time:
			if !v.IsZero() {
				out[key] = v
			}
		case string:
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				out[key] = t
			}
		}
	}
	return out
}

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
