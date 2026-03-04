package service

import (
	"sort"
	"time"
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
