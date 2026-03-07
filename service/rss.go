package service

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"
)

// rssTime -> RSS 2.0 needs RFC 1123Z (e.g. "Mon, 02 Jan 2006 15:04:05 -0700").
type rssTime struct{ time.Time }

func (t rssTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if t.IsZero() {
		return nil
	}
	return e.EncodeElement(t.Format(time.RFC1123Z), start)
}

type rssItem struct {
	XMLName        xml.Name `xml:"item"`
	Title          string   `xml:"title"`
	Link           string   `xml:"link"`
	Description    string   `xml:"description"`
	PubDate        rssTime  `xml:"pubDate"`
	GUID           string   `xml:"guid"`
	ContentEncoded string   `xml:",innerxml"`
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
	XMLName   xml.Name `xml:"rss"`
	Version   string   `xml:"version,attr"`
	ContentNS string   `xml:"xmlns:content,attr,omitempty"`
	Channel   rssChannel
}

// RSSFeed returns a RSS 2.0 feed document, cached (invalidated on rescan changes).
func (r *MemoryPostRepository) RSSFeed() ([]byte, error) {
	// Fast path: return cached feed under a read lock.
	r.feedMu.RLock()
	cached := r.feedCache
	r.feedMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	// Slow path: build the feed, then store under a write lock.
	data, err := r.buildFeed()
	if err != nil {
		return nil, err
	}
	r.feedMu.Lock()
	r.feedCache = data
	r.feedMu.Unlock()
	return data, nil
}

// buildFeed constructs the RSS 2.0 document without consulting the cache.
func (r *MemoryPostRepository) buildFeed() ([]byte, error) {
	size := r.conf.PageSize
	if size <= 0 {
		size = 10
	}

	r.postsMu.RLock()
	// Collect RSS-visible posts, sort by date descending.
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

	items := make([]rssItem, 0, len(filtered))
	for _, p := range filtered {
		d := p.Data()
		link := fmt.Sprintf("%s/posts/%s", r.conf.SiteURL, d.Slug)
		item := rssItem{
			Title:       d.Title,
			Link:        link,
			Description: d.Description,
			GUID:        link,
		}
		if r.conf.RSSFullContent {
			if plain := p.PlainText(); plain != nil {
				item.ContentEncoded = "<content:encoded><![CDATA[" + escapeCDATA(string(plain)) + "]]></content:encoded>"
			}
		}
		if !d.Date.IsZero() {
			item.PubDate = rssTime{d.Date}
		}
		items = append(items, item)
	}

	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:         r.conf.BlogName,
			Link:          r.conf.SiteURL,
			Description:   r.conf.FeedDescription,
			LastBuildDate: rssTime{time.Now()},
			Items:         items,
		},
	}
	if r.conf.RSSFullContent {
		feed.ContentNS = "http://purl.org/rss/1.0/modules/content/"
	}

	out, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("repository.RSSFeed: marshal: %w", err)
	}
	return append([]byte(xml.Header), out...), nil
}

// escapeCDATA escapes the CDATA end sequence "]]>" so the content can be
// safely embedded inside a <![CDATA[...]]> section.
func escapeCDATA(s string) string {
	return strings.ReplaceAll(s, "]]>", "]]]]><![CDATA[>")
}
