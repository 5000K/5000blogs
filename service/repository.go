package service

// PostRepository manages the collection of posts.
type PostRepository interface {
	Has(path string) bool
	Get(path string) *Post
	List() []*Post
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
