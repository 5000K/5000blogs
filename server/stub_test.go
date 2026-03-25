package server

import (
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/5000K/5000blogs/service"
)

// stubIndexer is a minimal in-memory PostIndexer for handler tests.
type stubIndexer struct {
	posts        []*service.Post
	lastModified time.Time
	sitemap      []service.SitemapEntry
	feedPosts    []*service.Post // override for FeedPosts; falls back to posts
}

func (s *stubIndexer) Get(path string) *service.Post {
	for _, p := range s.posts {
		if p.Data().Slug == path {
			return p
		}
	}
	return nil
}

func (s *stubIndexer) GetBySlug(slug string) *service.Post {
	for _, p := range s.posts {
		if p.Data().Slug == slug {
			return p
		}
	}
	return nil
}

func (s *stubIndexer) List() []*service.Post { return s.posts }
func (s *stubIndexer) Count() int            { return len(s.posts) }

func (s *stubIndexer) ListFiltered(filter service.PostFilter) []*service.Post {
	var result []*service.Post
	for _, p := range s.posts {
		if matchesFilter(p, filter) {
			result = append(result, p)
		}
	}
	return result
}

func (s *stubIndexer) ListFilteredPaged(filter service.PostFilter, pageSize int, page int) *service.PageResult {
	all := s.ListFiltered(filter)
	total := len(all)
	start := (page - 1) * pageSize
	if start >= total {
		return &service.PageResult{}
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return &service.PageResult{Posts: toSummaries(all[start:end])}
}

func (s *stubIndexer) GetPage(page int, tags []string) service.PageResult {
	return service.PageResult{Posts: toSummaries(s.posts)}
}

func toSummaries(posts []*service.Post) []service.PostSummary {
	out := make([]service.PostSummary, 0, len(posts))
	for _, p := range posts {
		d := p.Data()
		out = append(out, service.PostSummary{
			Slug:  d.Slug,
			Title: d.Title,
		})
	}
	return out
}

func (s *stubIndexer) AllTags() []string                          { return nil }
func (s *stubIndexer) FeedPosts([]string, string) []*service.Post {
	if s.feedPosts != nil {
		return s.feedPosts
	}
	return s.posts
}
func (s *stubIndexer) LastModified() time.Time                    { return s.lastModified }
func (s *stubIndexer) Sitemap() []service.SitemapEntry            { return s.sitemap }
func (s *stubIndexer) Start() error                               { return nil }
func (s *stubIndexer) Stop()                                      {}
func (s *stubIndexer) ReadMedia(_ string) ([]byte, time.Time, error) {
	return nil, time.Time{}, errors.New("not found")
}

func (s *stubIndexer) Search(query string) []service.PostSummary {
	if query == "" {
		return []service.PostSummary{}
	}
	q := strings.ToLower(query)
	var out []service.PostSummary
	for _, p := range s.posts {
		d := p.Data()
		if strings.Contains(strings.ToLower(d.Title), q) ||
			strings.Contains(strings.ToLower(d.Description), q) {
			out = append(out, service.PostSummary{Slug: d.Slug, Title: d.Title, Description: d.Description})
		}
	}
	return out
}

func matchesFilter(p *service.Post, f service.PostFilter) bool {
	if len(f.Tags) > 0 {
		d := p.Data()
		for _, ft := range f.Tags {
			found := false
			for _, pt := range d.Tags {
				if pt == ft {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	if f.Query != "" {
		d := p.Data()
		q := strings.ToLower(f.Query)
		if !strings.Contains(strings.ToLower(d.Title), q) &&
			!strings.Contains(strings.ToLower(d.Description), q) {
			return false
		}
	}
	return true
}

// newStubPost creates a post using NewPost (no plain text).
func newStubPost(slug, title string) *service.Post {
	return service.NewPost(slug+".md", &service.Metadata{Title: title}, []byte("<p>content</p>"))
}

// convertedPost creates a post with plain text populated via the converter.
func convertedPost(slug string, raw []byte) (*service.Post, error) {
	post := service.NewPostWithSlug(slug+".md", slug, nil, nil)
	c := &service.GoldmarkConverter{}
	body, err := c.ExtractMetadata(post, raw)
	if err != nil {
		return nil, err
	}
	if err := c.Convert(post, body, nil); err != nil {
		return nil, err
	}
	return post, nil
}

// stubRenderer is a minimal Renderer that records the most recent call.
type stubRenderer struct {
	servedPost      *service.Post
	served404Count  int
	servedListCount int
}

func (r *stubRenderer) ServePost(post *service.Post, w http.ResponseWriter, _ string, _ string) {
	r.servedPost = post
	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte("<html>post:" + post.Data().Slug + "</html>"))
}

func (r *stubRenderer) Serve404(post *service.Post, w http.ResponseWriter) {
	r.served404Count++
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("404"))
}

func (r *stubRenderer) ServePostList(pr service.PageResult, w http.ResponseWriter, _ string) {
	r.servedListCount++
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("list"))
}

func (r *stubRenderer) ServeSearchResults(_ string, _ []string, _ []service.PostSummary, w http.ResponseWriter, _ string) {
	w.WriteHeader(http.StatusOK)
}

func (r *stubRenderer) SetFooter(_ func() template.HTML) {}
