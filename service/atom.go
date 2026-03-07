package service

import (
	"encoding/xml"
	"fmt"
	"sort"
	"time"
)

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type atomPerson struct {
	Name string `xml:"name"`
}

type atomEntry struct {
	XMLName xml.Name    `xml:"entry"`
	Title   string      `xml:"title"`
	Link    atomLink    `xml:"link"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Summary string      `xml:"summary,omitempty"`
	Author  *atomPerson `xml:"author,omitempty"`
}

type atomFeedDoc struct {
	XMLName xml.Name    `xml:"feed"`
	Xmlns   string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	Links   []atomLink  `xml:"link"`
	Updated string      `xml:"updated"`
	ID      string      `xml:"id"`
	Entries []atomEntry `xml:""`
}

// AtomFeed returns an Atom 1.0 feed document, cached (invalidated on rescan changes).
func (r *MemoryPostRepository) AtomFeed() ([]byte, error) {
	r.atomFeedMu.RLock()
	cached := r.atomFeedCache
	r.atomFeedMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	data, err := r.buildAtomFeed()
	if err != nil {
		return nil, err
	}
	r.atomFeedMu.Lock()
	r.atomFeedCache = data
	r.atomFeedMu.Unlock()
	return data, nil
}

// buildAtomFeed constructs the Atom 1.0 document without consulting the cache.
func (r *MemoryPostRepository) buildAtomFeed() ([]byte, error) {
	size := r.conf.PageSize
	if size <= 0 {
		size = 10
	}

	r.postsMu.RLock()
	filtered := make([]*Post, 0, len(r.posts))
	for _, p := range r.posts {
		if p.IsRSSVisible() {
			filtered = append(filtered, p)
		}
	}
	r.postsMu.RUnlock()

	sort.Slice(filtered, func(i, j int) bool {
		di, dj := time.Time{}, time.Time{}
		if filtered[i].metadata != nil {
			di = filtered[i].metadata.Date
		}
		if filtered[j].metadata != nil {
			dj = filtered[j].metadata.Date
		}
		return di.After(dj)
	})
	if len(filtered) > size {
		filtered = filtered[:size]
	}

	feedUpdated := time.Now()
	entries := make([]atomEntry, 0, len(filtered))
	for _, p := range filtered {
		d := p.Data()
		link := fmt.Sprintf("%s/posts/%s", r.conf.SiteURL, d.Slug)
		entry := atomEntry{
			Title:   d.Title,
			Link:    atomLink{Href: link, Rel: "alternate", Type: "text/html"},
			ID:      link,
			Summary: d.Description,
		}
		if !d.Date.IsZero() {
			entry.Updated = d.Date.UTC().Format(time.RFC3339)
		} else {
			entry.Updated = feedUpdated.UTC().Format(time.RFC3339)
		}
		if d.Author != "" {
			entry.Author = &atomPerson{Name: d.Author}
		}
		entries = append(entries, entry)
	}

	selfURL := fmt.Sprintf("%s/feed.atom", r.conf.SiteURL)
	feed := atomFeedDoc{
		Xmlns: "http://www.w3.org/2005/Atom",
		Title: r.conf.FeedTitle,
		Links: []atomLink{
			{Href: r.conf.SiteURL, Rel: "alternate", Type: "text/html"},
			{Href: selfURL, Rel: "self", Type: "application/atom+xml"},
		},
		Updated: feedUpdated.UTC().Format(time.RFC3339),
		ID:      r.conf.SiteURL + "/",
		Entries: entries,
	}

	out, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("repository.AtomFeed: marshal: %w", err)
	}
	return append([]byte(xml.Header), out...), nil
}
