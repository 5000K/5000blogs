package service

import (
	"github.com/5000K/5000blogs/config"
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
	Content string      `xml:",innerxml"`
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

// BuildAtomFeed constructs an Atom 1.0 feed document from the given posts.
// posts need not be pre-sorted; sorting and truncation are applied internally.
func BuildAtomFeed(conf *config.Config, posts []*Post) ([]byte, error) {
	filtered := posts
	size := conf.FeedSize
	if size <= 0 {
		size = 20
	}

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
		link := fmt.Sprintf("%s/%s", conf.SiteURL, d.Slug)
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
		switch conf.RSSContent {
		case "text":
			if plain := p.PlainText(); plain != nil {
				entry.Content = `<content type="text"><![CDATA[` + escapeCDATA(string(plain)) + "]]></content>"
			}
		case "html":
			if html := p.Data().Content; html != nil {
				entry.Content = `<content type="html"><![CDATA[` + escapeCDATA(string(html)) + "]]></content>"
			}
		}
		entries = append(entries, entry)
	}

	selfURL := fmt.Sprintf("%s/feed.atom", conf.SiteURL)
	feed := atomFeedDoc{
		Xmlns: "http://www.w3.org/2005/Atom",
		Title: conf.BlogName,
		Links: []atomLink{
			{Href: conf.SiteURL, Rel: "alternate", Type: "text/html"},
			{Href: selfURL, Rel: "self", Type: "application/atom+xml"},
		},
		Updated: feedUpdated.UTC().Format(time.RFC3339),
		ID:      conf.SiteURL + "/",
		Entries: entries,
	}

	out, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("repository.AtomFeed: marshal: %w", err)
	}
	return append([]byte(xml.Header), out...), nil
}
