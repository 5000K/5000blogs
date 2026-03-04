package service

import (
	"5000blogs/config"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type PostRepository interface {
	Get(path string) *Post
	List() []*Post
	Count() int
	GetPage(page int) PageResult
	RSSFeed() ([]byte, error)
	Start() error
	Stop()
}

// MemoryPostRepository is an in-memory implementation of PostRepository.
type MemoryPostRepository struct {
	conf      *config.Config
	posts     []*Post
	source    PostSource
	converter Converter
	log       *slog.Logger

	scheduler *cron.Cron

	feedMu    sync.RWMutex
	feedCache []byte
}

func NewMemoryPostRepository(conf *config.Config, source PostSource, converter Converter, logger *slog.Logger) *MemoryPostRepository {
	return &MemoryPostRepository{
		conf:      conf,
		source:    source,
		converter: converter,
		log:       logger.With("component", "MemoryPostRepository"),
	}
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

func (r *MemoryPostRepository) Start() error {
	r.log.Info("starting repository")
	r.rescan()

	r.scheduler = cron.New()
	_, err := r.scheduler.AddFunc(r.conf.RescanCron, r.rescan)
	if err != nil {
		return fmt.Errorf("repository.Start: invalid rescan cron expression %q: %w", r.conf.RescanCron, err)
	}
	r.scheduler.Start()
	return nil
}

func (r *MemoryPostRepository) Stop() {
	r.log.Info("stopping repository")
	if r.scheduler != nil {
		r.scheduler.Stop()
	}
}

func (r *MemoryPostRepository) GetPage(page int) PageResult {
	size := r.conf.PageSize
	if size <= 0 {
		size = 10
	}

	filtered := make([]*Post, 0, len(r.posts))
	for _, p := range r.posts {
		if p.IsVisible() {
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

	total := len(filtered)
	totalPages := (total + size - 1) / size
	if totalPages == 0 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}

	start := (page - 1) * size
	var pagePosts []*Post
	if start < total {
		end := start + size
		if end > total {
			end = total
		}
		pagePosts = filtered[start:end]
	}

	summaries := make([]PostSummary, 0, len(pagePosts))
	for _, p := range pagePosts {
		d := p.Data()
		summaries = append(summaries, PostSummary{
			Slug:        d.Slug,
			Title:       d.Title,
			Description: d.Description,
			Date:        d.Date,
			Author:      d.Author,
		})
	}

	return PageResult{
		Posts:      summaries,
		Page:       page,
		PageSize:   size,
		TotalPosts: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
	}
}

func (r *MemoryPostRepository) invalidateFeedCache() {
	r.feedMu.Lock()
	r.feedCache = nil
	r.feedMu.Unlock()
}

func (r *MemoryPostRepository) rescan() {
	r.log.Debug("rescanning posts")
	paths, err := r.source.ListPosts()
	if err != nil {
		r.log.Error("failed to list posts", "err", err)
		return
	}

	changed := false
	found := make(map[string]bool, len(paths))
	for _, path := range paths {
		found[path] = true
		if r.has(path) {
			if r.updatePost(path) {
				changed = true
			}
		} else {
			if r.addPost(path) {
				changed = true
			}
		}
	}

	var toRemove []string
	for _, post := range r.posts {
		if !found[post.path] {
			toRemove = append(toRemove, post.path)
		}
	}
	for _, path := range toRemove {
		r.log.Info("removed post", "path", path)
		r.remove(path)
		changed = true
	}

	if changed {
		r.invalidateFeedCache()
	}
	r.log.Debug("rescan complete", "total", len(paths))
}

func (r *MemoryPostRepository) has(path string) bool {
	return r.Get(path) != nil
}

func (r *MemoryPostRepository) addPost(path string) bool {
	post := &Post{path: path}

	modTime, err := r.source.StatPost(path)
	if err != nil {
		r.log.Error("failed to stat post", "path", path, "err", err)
		return false
	}
	buf, err := r.source.ReadPost(path)
	if err != nil {
		r.log.Error("failed to read post", "path", path, "err", err)
		return false
	}
	post.modTime = modTime
	if err := r.converter.Convert(post, buf); err != nil {
		r.log.Error("failed to convert post", "path", path, "err", err)
		return false
	}

	r.log.Info("added post", "path", path)
	r.posts = append(r.posts, post)
	return true
}

func (r *MemoryPostRepository) updatePost(path string) bool {
	post := r.Get(path)
	if post == nil {
		return false
	}

	if r.conf.SkipUnchangedModTime {
		modTime, err := r.source.StatPost(path)
		if err != nil {
			r.log.Error("failed to stat post", "path", path, "err", err)
			return false
		}
		if modTime.Equal(post.modTime) {
			return false
		}
		post.modTime = modTime
	}

	buf, err := r.source.ReadPost(path)
	if err != nil {
		r.log.Error("failed to read post", "path", path, "err", err)
		return false
	}
	if hashBytes(buf) == post.hash {
		return false
	}
	if err := r.converter.Convert(post, buf); err != nil {
		r.log.Error("failed to convert post", "path", path, "err", err)
		return false
	}
	return true
}

func (r *MemoryPostRepository) remove(path string) {
	for i, p := range r.posts {
		if p.path == path {
			r.posts = append(r.posts[:i], r.posts[i+1:]...)
			return
		}
	}
}
