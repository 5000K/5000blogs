package service

import (
	"fmt"
	"os"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// PostRepository manages the collection of posts.
type PostRepository interface {
	Has(path string) bool
	Get(path string) *Post
	List() []*Post
	// Page returns one page of posts sorted by date descending.
	// page is 1-based; returns an empty slice for out-of-range pages.
	Page(page, pageSize int) []*Post
	Count() int
	Add(post *Post)
	Remove(path string)
	// WritePost serializes metadata and markdown content into a .md file at path.
	// The file is created or overwritten; parent directories must already exist.
	WritePost(path string, metadata *Metadata, content string) error
}

// MemoryPostRepository is an in-memory implementation of PostRepository.
type MemoryPostRepository struct {
	posts []*Post
}

func NewMemoryPostRepository() *MemoryPostRepository {
	return &MemoryPostRepository{}
}

func (r *MemoryPostRepository) Has(path string) bool {
	for _, p := range r.posts {
		if p.path == path {
			return true
		}
	}
	return false
}

func (r *MemoryPostRepository) Get(path string) *Post {
	for _, p := range r.posts {
		if p.path == path {
			return p
		}
	}
	return nil
}

func (r *MemoryPostRepository) List() []*Post {
	return r.posts
}

func (r *MemoryPostRepository) Count() int {
	return len(r.posts)
}

func (r *MemoryPostRepository) Page(page, pageSize int) []*Post {
	// Sort by date descending into a temporary slice.
	sorted := make([]*Post, len(r.posts))
	copy(sorted, r.posts)
	sort.Slice(sorted, func(i, j int) bool {
		di, dj := time.Time{}, time.Time{}
		if sorted[i].metadata != nil {
			di = sorted[i].metadata.Date
		}
		if sorted[j].metadata != nil {
			dj = sorted[j].metadata.Date
		}
		return di.After(dj)
	})

	if pageSize <= 0 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start >= len(sorted) || start < 0 {
		return nil
	}
	end := start + pageSize
	if end > len(sorted) {
		end = len(sorted)
	}
	return sorted[start:end]
}

func (r *MemoryPostRepository) Add(post *Post) {
	r.posts = append(r.posts, post)
}

func (r *MemoryPostRepository) Remove(path string) {
	for i, p := range r.posts {
		if p.path == path {
			r.posts = append(r.posts[:i], r.posts[i+1:]...)
			return
		}
	}
}

// WritePost serializes metadata as a yaml fenced block followed by content
// and writes the result to path, creating or overwriting the file.
// Only non-zero metadata fields are included in the output.
func (r *MemoryPostRepository) WritePost(path string, metadata *Metadata, content string) error {
	fields := make(map[string]interface{})
	if metadata != nil {
		if metadata.Title != "" {
			fields["title"] = metadata.Title
		}
		if metadata.Description != "" {
			fields["description"] = metadata.Description
		}
		if !metadata.Date.IsZero() {
			fields["date"] = metadata.Date
		}
		if metadata.Author != "" {
			fields["author"] = metadata.Author
		}
		if metadata.Visible != nil {
			fields["visible"] = *metadata.Visible
		}
		if metadata.RSSVisible != nil {
			fields["rss-visible"] = *metadata.RSSVisible
		}
		for k, v := range metadata.Raw {
			if _, exists := fields[k]; !exists {
				fields[k] = v
			}
		}
	}

	yamlBytes, err := yaml.Marshal(fields)
	if err != nil {
		return fmt.Errorf("WritePost: failed to marshal metadata: %w", err)
	}

	body := fmt.Sprintf("```yaml\n%s```\n\n%s", string(yamlBytes), content)

	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return fmt.Errorf("WritePost: failed to write file %q: %w", path, err)
	}
	return nil
}
