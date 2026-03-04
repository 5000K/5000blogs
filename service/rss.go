package service

import (
	"encoding/xml"
	"fmt"
	"sort"
	"time"
)

// rssTime wraps time.Time and marshals to the RFC 1123Z format required by
// RSS 2.0 (e.g. "Mon, 02 Jan 2006 15:04:05 -0700").
type rssTime struct{ time.Time }

func (t rssTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if t.IsZero() {
		return nil
	}
	return e.EncodeElement(t.Format(time.RFC1123Z), start)
}

type rssItem struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	PubDate     rssTime  `xml:"pubDate"`
	GUID        string   `xml:"guid"`
}

type rssChannel struct {
	XMLName       xml.Name  `xml:"channel"`
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	LastBuildDate rssTime   `xml:"lastBuildDate"`
	Items         []rssItem `xml:""`
}

type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel rssChannel
}

// RSSFeed returns a complete RSS 2.0 feed document for the most recent
// RSS-visible posts (up to one full page). The result is cached and only
// regenerated when a post is added, updated, or removed.
func (s *Service) RSSFeed() ([]byte, error) {
	// Fast path: return cached feed under a read lock.
	s.feedMu.RLock()
	cached := s.feedCache
	s.feedMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	// Slow path: build the feed, then store under a write lock.
	data, err := s.buildFeed()
	if err != nil {
		return nil, err
	}
	s.feedMu.Lock()
	s.feedCache = data
	s.feedMu.Unlock()
	return data, nil
}

// buildFeed constructs the RSS 2.0 document without consulting the cache.
func (s *Service) buildFeed() ([]byte, error) {
	size := s.conf.PageSize
	if size <= 0 {
		size = 10
	}

	// Collect RSS-visible posts and sort by date descending.
	all := s.repo.List()
	filtered := make([]*Post, 0, len(all))
	for _, p := range all {
		if p.IsRSSVisible() {
			filtered = append(filtered, p)
		}
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

	items := make([]rssItem, 0, len(filtered))
	for _, p := range filtered {
		d := p.Data()
		link := fmt.Sprintf("%s/posts/%s", s.conf.SiteURL, d.Slug)
		item := rssItem{
			Title:       d.Title,
			Link:        link,
			Description: d.Description,
			GUID:        link,
		}
		if !d.Date.IsZero() {
			item.PubDate = rssTime{d.Date}
		}
		items = append(items, item)
	}

	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:         s.conf.FeedTitle,
			Link:          s.conf.SiteURL,
			Description:   s.conf.FeedDescription,
			LastBuildDate: rssTime{time.Now()},
			Items:         items,
		},
	}

	out, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("service.RSSFeed: marshal: %w", err)
	}
	return append([]byte(xml.Header), out...), nil
}
