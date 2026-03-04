package service

import (
	"sort"
	"time"
)

type PostRepository interface {
	Has(path string) bool
	Get(path string) *Post
	List() []*Post
	Page(page, pageSize int) []*Post
	Count() int
	Add(post *Post)
	Remove(path string)
}

// todo: scans should probably be part of the repository instead of the service - since e.g. a db doesn't actually need a scan.
// not that a db is planned, but... not a reason to not do it properly.

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
