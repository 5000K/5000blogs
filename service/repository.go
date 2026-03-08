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
	Search(query string) []PostSummary
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

	// rescanMu serializes concurrent rescan calls so that file I/O and the
	// subsequent write-lock phase are never interleaved by two goroutines.
	rescanMu sync.Mutex

	// postsMu guards posts. Readers take RLock; rescan takes Lock only for the
	// short mutation phase (slice append/replace/remove), not during file I/O.
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

// Search returns summaries of all visible posts whose title, description, or
// plain-text body contains query (case-insensitive). No pagination is applied.
func (r *MemoryPostRepository) Search(query string) []PostSummary {
	if query == "" {
		return []PostSummary{}
	}
	q := strings.ToLower(query)
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()
	var results []PostSummary
	for _, p := range r.posts {
		if !p.IsVisible() {
			continue
		}
		d := p.Data()
		plain := ""
		if pt := p.PlainText(); pt != nil {
			plain = strings.ToLower(string(pt))
		}
		if strings.Contains(strings.ToLower(d.Title), q) ||
			strings.Contains(strings.ToLower(d.Description), q) ||
			strings.Contains(plain, q) {
			results = append(results, PostSummary{
				Slug:        d.Slug,
				Title:       d.Title,
				Description: d.Description,
				Date:        d.Date,
				Author:      d.Author,
				Tags:        d.Tags,
			})
		}
	}
	if results == nil {
		return []PostSummary{}
	}
	return results
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

// pendingChange describes a single mutation to apply to r.posts.
type pendingChange struct {
	path string
	post *Post // non-nil: add or replace; nil: remove
}

func (r *MemoryPostRepository) rescan() {
	r.log.Debug("rescanning posts")
	r.rescanMu.Lock()
	defer r.rescanMu.Unlock()

	if err := r.source.Sync(); err != nil {
		r.log.Error("failed to sync source", "err", err)
	}

	paths, err := r.source.ListPosts()
	if err != nil {
		r.log.Error("failed to list posts", "err", err)
		return
	}

	// Snapshot current state under read lock — no file I/O yet.
	r.postsMu.RLock()
	snapshot := make(map[string]*Post, len(r.posts))
	for _, p := range r.posts {
		snapshot[p.path] = p
	}
	r.postsMu.RUnlock()

	// Compute all changes outside the write lock.
	found := make(map[string]bool, len(paths))
	var changes []pendingChange

	for _, path := range paths {
		found[path] = true
		if existing, ok := snapshot[path]; ok {
			if p, updated := r.checkPostChanged(path, existing); updated {
				changes = append(changes, pendingChange{path: path, post: p})
				r.log.Info("updated post", "path", path)
			}
		} else {
			if p, ok := r.preparePost(path); ok {
				changes = append(changes, pendingChange{path: path, post: p})
				r.log.Info("added post", "path", path)
			}
		}
	}

	for path := range snapshot {
		if !found[path] {
			changes = append(changes, pendingChange{path: path, post: nil})
			r.log.Info("removed post", "path", path)
		}
	}

	if len(changes) == 0 {
		r.log.Debug("rescan complete, no changes", "total", len(paths))
		return
	}

	// Apply all mutations under write lock.
	r.postsMu.Lock()
	for _, ch := range changes {
		if ch.post == nil {
			r.remove(ch.path)
		} else if _, exists := snapshot[ch.path]; exists {
			for i, p := range r.posts {
				if p.path == ch.path {
					r.posts[i] = ch.post
					break
				}
			}
		} else {
			r.posts = append(r.posts, ch.post)
		}
	}
	r.postsMu.Unlock()

	r.invalidateFeedCache()
	r.log.Debug("rescan complete", "total", len(paths))
}

// preparePost reads, stats, and converts a brand-new post. No lock needed.
func (r *MemoryPostRepository) preparePost(path string) (*Post, bool) {
	modTime, err := r.source.StatPost(path)
	if err != nil {
		r.log.Error("failed to stat post", "path", path, "err", err)
		return nil, false
	}
	buf, err := r.source.ReadPost(path)
	if err != nil {
		r.log.Error("failed to read post", "path", path, "err", err)
		return nil, false
	}
	post := &Post{path: path, modTime: modTime}
	if err := r.converter.Convert(post, buf); err != nil {
		r.log.Error("failed to convert post", "path", path, "err", err)
		return nil, false
	}
	return post, true
}

// checkPostChanged determines whether the on-disk post differs from existing.
// Returns a freshly prepared Post and true when content has changed.
// When SkipUnchangedModTime is true, an unchanged mod-time is an early exit.
// No lock needed; operates on a snapshot copy of existing.
func (r *MemoryPostRepository) checkPostChanged(path string, existing *Post) (*Post, bool) {
	var modTime time.Time
	if r.conf.SkipUnchangedModTime {
		var err error
		modTime, err = r.source.StatPost(path)
		if err != nil {
			r.log.Error("failed to stat post", "path", path, "err", err)
			return nil, false
		}
		if modTime.Equal(existing.modTime) {
			return nil, false
		}
	}
	buf, err := r.source.ReadPost(path)
	if err != nil {
		r.log.Error("failed to read post", "path", path, "err", err)
		return nil, false
	}
	if hashBytes(buf) == existing.hash {
		return nil, false
	}
	// Stat for a fresh mod-time if we haven't done so yet.
	if modTime.IsZero() {
		modTime, err = r.source.StatPost(path)
		if err != nil {
			r.log.Error("failed to stat post", "path", path, "err", err)
			return nil, false
		}
	}
	post := &Post{path: path, modTime: modTime}
	if err := r.converter.Convert(post, buf); err != nil {
		r.log.Error("failed to convert post", "path", path, "err", err)
		return nil, false
	}
	return post, true
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
