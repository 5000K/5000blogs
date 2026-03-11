package service

import (
	"5000blogs/config"
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

// BuildRSSFeed constructs a RSS 2.0 feed document from the given posts.
// posts need not be pre-sorted; sorting and truncation are applied internally.
func BuildRSSFeed(conf *config.Config, posts []*Post) ([]byte, error) {
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

	items := make([]rssItem, 0, len(filtered))
	for _, p := range filtered {
		d := p.Data()
		link := fmt.Sprintf("%s/%s", conf.SiteURL, d.Slug)
		item := rssItem{
			Title:       d.Title,
			Link:        link,
			Description: d.Description,
			GUID:        link,
		}
		switch conf.RSSContent {
		case "text":
			if plain := p.PlainText(); plain != nil {
				item.ContentEncoded = "<content:encoded><![CDATA[" + escapeCDATA(string(plain)) + "]]></content:encoded>"
			}
		case "html":
			if html := p.Data().Content; html != nil {
				item.ContentEncoded = "<content:encoded><![CDATA[" + escapeCDATA(string(html)) + "]]></content:encoded>"
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
			Title:         conf.BlogName,
			Link:          conf.SiteURL,
			Description:   conf.FeedDescription,
			LastBuildDate: rssTime{time.Now()},
			Items:         items,
		},
	}
	if conf.RSSContent == "text" || conf.RSSContent == "html" {
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
