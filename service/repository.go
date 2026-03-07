package service

import (
	"5000blogs/config"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type PostRepository interface {
	Get(path string) *Post
	GetBySlug(slug string) *Post
	List() []*Post
	Count() int
	GetPage(page int, tags []string) PageResult
	AllTags() []string
	RSSFeed() ([]byte, error)
	AtomFeed() ([]byte, error)
	LastModified() time.Time
	Sitemap() []SitemapEntry
	Start() error
	Stop()
}

// SitemapEntry holds the data for a single sitemap URL.
type SitemapEntry struct {
	Slug    string
	LastMod time.Time
}

// MemoryPostRepository is an in-memory implementation of PostRepository.
type MemoryPostRepository struct {
	conf      *config.Config
	source    PostSource
	converter Converter
	log       *slog.Logger

	scheduler *cron.Cron

	// postsMu guards posts. All public reads take RLock; rescan takes Lock for
	// its entire duration, which also serializes it against concurrent rescan
	// calls because a second rescan will block on Lock until the first finishes.
	postsMu sync.RWMutex
	posts   []*Post

	feedMu    sync.RWMutex
	feedCache []byte

	atomFeedMu    sync.RWMutex
	atomFeedCache []byte
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
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()
	return r.get(path)
}

// get is the unlocked version of Get; callers must hold at least postsMu.RLock.
func (r *MemoryPostRepository) get(path string) *Post {
	for _, p := range r.posts {
		if p.path == path {
			return p
		}
	}
	return nil
}

func (r *MemoryPostRepository) GetBySlug(slug string) *Post {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()
	return r.getBySlug(slug)
}

// getBySlug is the unlocked version of GetBySlug.
func (r *MemoryPostRepository) getBySlug(slug string) *Post {
	for _, p := range r.posts {
		if slugFromPath(p.path) == slug {
			return p
		}
	}
	return nil
}

func (r *MemoryPostRepository) List() []*Post {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()
	cp := make([]*Post, len(r.posts))
	copy(cp, r.posts)
	return cp
}

func (r *MemoryPostRepository) Count() int {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()
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

func (r *MemoryPostRepository) GetPage(page int, tags []string) PageResult {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()

	size := r.conf.PageSize
	if size <= 0 {
		size = 10
	}

	filtered := make([]*Post, 0, len(r.posts))
	for _, p := range r.posts {
		if !p.IsVisible() {
			continue
		}
		if len(tags) > 0 && !hasAnyTag(p, tags) {
			continue
		}
		filtered = append(filtered, p)
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
	if page > totalPages {
		page = totalPages
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
			Tags:        d.Tags,
		})
	}

	tagParam := ""
	if len(tags) > 0 {
		tagParam = "&tags=" + strings.Join(tags, ",")
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
		FilterTags: tags,
		TagParam:   tagParam,
	}
}

// hasAnyTag reports whether p has at least one of the given tags.
func hasAnyTag(p *Post, tags []string) bool {
	if p.metadata == nil {
		return false
	}
	for _, want := range tags {
		for _, have := range p.metadata.Tags {
			if strings.EqualFold(have, want) {
				return true
			}
		}
	}
	return false
}

// AllTags returns a sorted, deduplicated list of all tags across visible posts.
func (r *MemoryPostRepository) AllTags() []string {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()
	seen := make(map[string]struct{})
	for _, p := range r.posts {
		if !p.IsVisible() || p.metadata == nil {
			continue
		}
		for _, t := range p.metadata.Tags {
			seen[t] = struct{}{}
		}
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// LastModified returns the most recent file-modtime across all visible posts.
// Returns zero time when there are no posts.
func (r *MemoryPostRepository) LastModified() time.Time {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()
	var latest time.Time
	for _, p := range r.posts {
		if p.IsVisible() && p.modTime.After(latest) {
			latest = p.modTime
		}
	}
	return latest
}

func (r *MemoryPostRepository) invalidateFeedCache() {
	r.feedMu.Lock()
	r.feedCache = nil
	r.feedMu.Unlock()
	r.atomFeedMu.Lock()
	r.atomFeedCache = nil
	r.atomFeedMu.Unlock()
}

func (r *MemoryPostRepository) rescan() {
	r.log.Debug("rescanning posts")
	paths, err := r.source.ListPosts()
	if err != nil {
		r.log.Error("failed to list posts", "err", err)
		return
	}

	// Hold the write lock for the entire mutation phase. This serializes
	// concurrent rescan calls and protects r.posts from concurrent readers.
	r.postsMu.Lock()
	defer r.postsMu.Unlock()

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
	return r.get(path) != nil
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
	post := r.get(path)
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

// Sitemap returns one entry per visible post for use in sitemap.xml.
// LastMod is the post's date when set, otherwise its file modification time.
func (r *MemoryPostRepository) Sitemap() []SitemapEntry {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()

	entries := make([]SitemapEntry, 0, len(r.posts))
	for _, p := range r.posts {
		if !p.IsVisible() {
			continue
		}
		d := p.Data()
		lastMod := d.Date
		if lastMod.IsZero() {
			lastMod = p.modTime
		}
		entries = append(entries, SitemapEntry{Slug: d.Slug, LastMod: lastMod})
	}
	return entries
}
