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
	FeedPosts(tags []string, query string) []*Post
	LastModified() time.Time
	Sitemap() []SitemapEntry
	ReadMedia(relPath string) ([]byte, time.Time, error)
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
		if p.slug == slug {
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

// FeedPosts returns all RSS-visible posts, optionally filtered by tags (OR logic)
// and/or a full-text search query (case-insensitive match on title, description, body).
func (r *MemoryPostRepository) FeedPosts(tags []string, query string) []*Post {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()
	q := strings.ToLower(query)
	var filtered []*Post
	for _, p := range r.posts {
		if !p.IsRSSVisible() {
			continue
		}
		if len(tags) > 0 && !hasAnyTag(p, tags) {
			continue
		}
		if q != "" {
			d := p.Data()
			plain := ""
			if pt := p.PlainText(); pt != nil {
				plain = strings.ToLower(string(pt))
			}
			if !strings.Contains(strings.ToLower(d.Title), q) &&
				!strings.Contains(strings.ToLower(d.Description), q) &&
				!strings.Contains(plain, q) {
				continue
			}
		}
		filtered = append(filtered, p)
	}
	return filtered
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

	// Snapshot current state under read lock - no file I/O yet.
	r.postsMu.RLock()
	snapshot := make(map[string]*Post, len(r.posts))
	for _, p := range r.posts {
		snapshot[p.path] = p
	}
	r.postsMu.RUnlock()

	type pendingRender struct {
		path  string
		body  []byte
		post  *Post
		isNew bool
	}

	found := make(map[string]bool, len(paths))
	var toRender []pendingRender
	var removals []string

	// Phase 1: extract metadata for all new/changed posts.
	for _, path := range paths {
		found[path] = true
		if existing, ok := snapshot[path]; ok {
			if post, body, ok := r.extractMetadataIfChanged(path, existing); ok {
				toRender = append(toRender, pendingRender{path: path, body: body, post: post, isNew: false})
			}
		} else {
			if post, body, ok := r.extractMetadataForNew(path); ok {
				toRender = append(toRender, pendingRender{path: path, body: body, post: post, isNew: true})
			}
		}
	}
	for path := range snapshot {
		if !found[path] {
			removals = append(removals, path)
		}
	}

	// Phase 2: build a title→slug index across all current posts, then render HTML.
	titleIndex := make(map[string]string)
	for _, p := range snapshot {
		if p.metadata != nil && p.metadata.Title != "" {
			titleIndex[p.metadata.Title] = p.slug
		}
	}
	for _, pr := range toRender {
		if pr.post.metadata != nil && pr.post.metadata.Title != "" {
			titleIndex[pr.post.metadata.Title] = pr.post.slug
		}
	}
	for _, path := range removals {
		if p, ok := snapshot[path]; ok && p.metadata != nil {
			delete(titleIndex, p.metadata.Title)
		}
	}
	resolveSlugByTitle := func(title string) string { return titleIndex[title] }
	baseResolver := &repoAssetResolver{
		slugByTitle: resolveSlugByTitle,
		source:      r.source,
		converter:   r.converter,
		getBySlug: func(slug string) *Post {
			r.postsMu.RLock()
			defer r.postsMu.RUnlock()
			return r.getBySlug(slug)
		},
		log: r.log,
	}

	var changes []pendingChange
	for _, pr := range toRender {
		resolver := &repoAssetResolver{
			slugByTitle: baseResolver.slugByTitle,
			source:      baseResolver.source,
			converter:   baseResolver.converter,
			getBySlug:   baseResolver.getBySlug,
			inProgress:  []string{pr.post.slug},
			log:         baseResolver.log,
		}
		if err := r.converter.Convert(pr.post, pr.body, resolver); err != nil {
			r.log.Error("failed to convert post", "path", pr.path, "err", err)
			continue
		}
		if pr.isNew {
			r.log.Info("added post", "path", pr.path)
		} else {
			r.log.Info("updated post", "path", pr.path)
		}
		changes = append(changes, pendingChange{path: pr.path, post: pr.post})
	}
	for _, path := range removals {
		r.log.Info("removed post", "path", path)
		changes = append(changes, pendingChange{path: path, post: nil})
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
	r.log.Debug("rescan complete", "total", len(paths))
}

// extractMetadataForNew reads and extracts metadata for a brand-new post.
// Returns the post (with metadata set), the markdown body, and whether it succeeded.
func (r *MemoryPostRepository) extractMetadataForNew(path string) (*Post, []byte, bool) {
	modTime, err := r.source.StatPost(path)
	if err != nil {
		r.log.Error("failed to stat post", "path", path, "err", err)
		return nil, nil, false
	}
	buf, err := r.source.ReadPost(path)
	if err != nil {
		r.log.Error("failed to read post", "path", path, "err", err)
		return nil, nil, false
	}
	post := &Post{path: path, slug: r.source.SlugForPath(path), modTime: modTime}
	body, err := r.converter.ExtractMetadata(post, buf)
	if err != nil {
		r.log.Error("failed to extract metadata", "path", path, "err", err)
		return nil, nil, false
	}
	return post, body, true
}

// extractMetadataIfChanged reads and extracts metadata when the on-disk post
// differs from existing. Returns nil when the post is unchanged.
func (r *MemoryPostRepository) extractMetadataIfChanged(path string, existing *Post) (*Post, []byte, bool) {
	var modTime time.Time
	if r.conf.SkipUnchangedModTime {
		var err error
		modTime, err = r.source.StatPost(path)
		if err != nil {
			r.log.Error("failed to stat post", "path", path, "err", err)
			return nil, nil, false
		}
		if modTime.Equal(existing.modTime) {
			return nil, nil, false
		}
	}
	buf, err := r.source.ReadPost(path)
	if err != nil {
		r.log.Error("failed to read post", "path", path, "err", err)
		return nil, nil, false
	}
	if hashBytes(buf) == existing.hash {
		return nil, nil, false
	}
	if modTime.IsZero() {
		modTime, err = r.source.StatPost(path)
		if err != nil {
			r.log.Error("failed to stat post", "path", path, "err", err)
			return nil, nil, false
		}
	}
	post := &Post{path: path, slug: r.source.SlugForPath(path), modTime: modTime}
	body, err := r.converter.ExtractMetadata(post, buf)
	if err != nil {
		r.log.Error("failed to extract metadata", "path", path, "err", err)
		return nil, nil, false
	}
	return post, body, true
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

// repoAssetResolver implements AssetResolver using a title→slug map and a PostSource.
type repoAssetResolver struct {
	slugByTitle func(string) string
	source      PostSource
	converter   Converter
	getBySlug   func(slug string) *Post
	inProgress  []string // slugs currently being embedded; used for recursion detection
	log         *slog.Logger
}

func (r *repoAssetResolver) ResolveSlugByTitle(title string) string {
	return r.slugByTitle(title)
}

func (r *repoAssetResolver) ResolveAssetByFilename(filename string) string {
	rel := r.source.ResolveAssetByFilename(filename)
	if rel == "" {
		return ""
	}
	return "/media/" + rel
}

func (r *repoAssetResolver) ResolveEmbedBySlug(slug string) []byte {
	for _, s := range r.inProgress {
		if s == slug {
			r.log.Error("embed recursion detected", "slug", slug)
			return []byte(fmt.Sprintf("<!-- post %q would be here, but it couldn't be loaded (recursion) -->", slug))
		}
	}

	post := r.getBySlug(slug)

	if post == nil {
		return nil
	}

	// If the post already has rendered HTML, return it directly.
	if post.contents != nil {
		return *post.contents
	}

	// Post exists but has no rendered contents yet - render it now.
	buf, err := r.source.ReadPost(post.path)
	if err != nil {
		r.log.Error("embed: failed to read post", "slug", slug, "err", err)
		return nil
	}
	tmp := &Post{path: post.path, slug: post.slug}
	body, err := r.converter.ExtractMetadata(tmp, buf)
	if err != nil {
		r.log.Error("embed: failed to extract metadata", "slug", slug, "err", err)
		return nil
	}
	sub := &repoAssetResolver{
		slugByTitle: r.slugByTitle,
		source:      r.source,
		converter:   r.converter,
		getBySlug:   r.getBySlug,
		inProgress:  append(append([]string(nil), r.inProgress...), slug),
		log:         r.log,
	}
	if err := r.converter.Convert(tmp, body, sub); err != nil {
		r.log.Error("embed: failed to convert post", "slug", slug, "err", err)
		return nil
	}
	return *tmp.contents
}
