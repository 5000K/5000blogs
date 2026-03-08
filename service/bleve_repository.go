package service

import (
	"5000blogs/config"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve"
	_ "github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/robfig/cron/v3"
)

// postDoc is the bleve-indexed representation of a post.
// Tags are stored lowercase for case-insensitive matching.
// Visible and RSSVisible are stored as "true"/"false" keyword strings.
type postDoc struct {
	Path        string    `json:"path"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Author      string    `json:"author"`
	Tags        []string  `json:"tags"`
	Content     string    `json:"content"`
	Visible     string    `json:"visible"`
	RSSVisible  string    `json:"rss_visible"`
	ModTime     time.Time `json:"mod_time"`
}

func boolKw(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func toPostDoc(p *Post) postDoc {
	d := p.Data()
	tags := make([]string, len(d.Tags))
	for i, t := range d.Tags {
		tags[i] = strings.ToLower(t)
	}
	content := ""
	if plain := p.PlainText(); plain != nil {
		content = string(plain)
	}
	return postDoc{
		Path:        p.path,
		Slug:        d.Slug,
		Title:       d.Title,
		Description: d.Description,
		Date:        d.Date,
		Author:      d.Author,
		Tags:        tags,
		Content:     content,
		Visible:     boolKw(d.Visible),
		RSSVisible:  boolKw(d.RSSVisible),
		ModTime:     p.modTime,
	}
}

func newBleveIndex() (bleve.Index, error) {
	text := bleve.NewTextFieldMapping()

	keyword := bleve.NewTextFieldMapping()
	keyword.Analyzer = "keyword"

	dt := bleve.NewDateTimeFieldMapping()

	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("path", keyword)
	docMapping.AddFieldMappingsAt("slug", keyword)
	docMapping.AddFieldMappingsAt("title", text)
	docMapping.AddFieldMappingsAt("description", text)
	docMapping.AddFieldMappingsAt("content", text)
	docMapping.AddFieldMappingsAt("author", keyword)
	docMapping.AddFieldMappingsAt("tags", keyword)
	docMapping.AddFieldMappingsAt("date", dt)
	docMapping.AddFieldMappingsAt("mod_time", dt)
	docMapping.AddFieldMappingsAt("visible", keyword)
	docMapping.AddFieldMappingsAt("rss_visible", keyword)

	m := bleve.NewIndexMapping()
	m.DefaultMapping = docMapping

	return bleve.NewUsing("", m, bleve.Config.DefaultIndexType, bleve.Config.DefaultMemKVStore, nil)
}

// BlevePostRepository is a PostRepository backed by an in-memory bleve index.
// It enables full-text search across all post content and metadata, at the cost
// of additional memory compared to the slice-based MemoryPostRepository.
// GetPage and GetBySlug leverage the index; other lookups use a complementary map.
type BlevePostRepository struct {
	conf      *config.Config
	source    PostSource
	converter Converter
	log       *slog.Logger

	scheduler *cron.Cron

	// rescanMu serializes concurrent rescan calls.
	rescanMu sync.Mutex

	// postsMu guards both posts and index so map and index stay consistent.
	// Rescan holds Lock during mutation; readers hold RLock.
	postsMu sync.RWMutex
	posts   map[string]*Post // path -> Post for direct lookups
	index   bleve.Index      // in-memory bleve index

	feedMu    sync.RWMutex
	feedCache []byte

	atomFeedMu    sync.RWMutex
	atomFeedCache []byte
}

func NewBlevePostRepository(conf *config.Config, source PostSource, converter Converter, logger *slog.Logger) (*BlevePostRepository, error) {
	idx, err := newBleveIndex()
	if err != nil {
		return nil, fmt.Errorf("BlevePostRepository: create index: %w", err)
	}
	return &BlevePostRepository{
		conf:      conf,
		source:    source,
		converter: converter,
		log:       logger.With("component", "BlevePostRepository"),
		posts:     make(map[string]*Post),
		index:     idx,
	}, nil
}

func (r *BlevePostRepository) Start() error {
	r.log.Info("starting repository")
	r.rescan()

	r.scheduler = cron.New()
	_, err := r.scheduler.AddFunc(r.conf.RescanCron, r.rescan)
	if err != nil {
		return fmt.Errorf("BlevePostRepository.Start: invalid rescan cron expression %q: %w", r.conf.RescanCron, err)
	}
	r.scheduler.Start()
	return nil
}

func (r *BlevePostRepository) Stop() {
	r.log.Info("stopping repository")
	if r.scheduler != nil {
		r.scheduler.Stop()
	}
}

func (r *BlevePostRepository) Get(path string) *Post {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()
	return r.posts[path]
}

func (r *BlevePostRepository) GetBySlug(slug string) *Post {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()

	q := bleve.NewTermQuery(slug)
	q.SetField("slug")
	req := bleve.NewSearchRequestOptions(q, 1, 0, false)
	result, err := r.index.Search(req)
	if err != nil || result.Total == 0 {
		return nil
	}
	return r.posts[result.Hits[0].ID]
}

func (r *BlevePostRepository) List() []*Post {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()
	out := make([]*Post, 0, len(r.posts))
	for _, p := range r.posts {
		out = append(out, p)
	}
	return out
}

func (r *BlevePostRepository) Count() int {
	r.postsMu.RLock()
	defer r.postsMu.RUnlock()
	return len(r.posts)
}

func (r *BlevePostRepository) GetPage(page int, tags []string) PageResult {
	size := r.conf.PageSize
	if size <= 0 {
		size = 10
	}

	countReq, pageReq := r.buildPageRequests(tags, size)

	r.postsMu.RLock()
	defer r.postsMu.RUnlock()

	countResult, err := r.index.Search(countReq)
	if err != nil {
		r.log.Error("bleve count query failed", "err", err)
		return PageResult{Page: 1, PageSize: size, TotalPages: 1}
	}
	total := int(countResult.Total)
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

	pageReq.From = (page - 1) * size
	pageResult, err := r.index.Search(pageReq)
	if err != nil {
		r.log.Error("bleve page query failed", "err", err)
		return PageResult{Page: page, PageSize: size, TotalPosts: total, TotalPages: totalPages}
	}

	summaries := make([]PostSummary, 0, len(pageResult.Hits))
	for _, hit := range pageResult.Hits {
		p, ok := r.posts[hit.ID]
		if !ok {
			continue
		}
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

// buildPageRequests constructs the count and page search requests for GetPage.
// Using a helper avoids the need to name the bleve.Query interface directly.
func (r *BlevePostRepository) buildPageRequests(tags []string, size int) (*bleve.SearchRequest, *bleve.SearchRequest) {
	visQ := bleve.NewTermQuery("true")
	visQ.SetField("visible")

	if len(tags) == 0 {
		countReq := bleve.NewSearchRequest(visQ)
		countReq.Size = 0
		pageReq := bleve.NewSearchRequestOptions(visQ, size, 0, false)
		pageReq.SortBy([]string{"-date"})
		return countReq, pageReq
	}

	tagQ := bleve.NewBooleanQuery()
	for _, tag := range tags {
		tq := bleve.NewTermQuery(strings.ToLower(tag))
		tq.SetField("tags")
		tagQ.AddShould(tq)
	}
	conjQ := bleve.NewConjunctionQuery(visQ, tagQ)

	countReq := bleve.NewSearchRequest(conjQ)
	countReq.Size = 0
	pageReq := bleve.NewSearchRequestOptions(conjQ, size, 0, false)
	pageReq.SortBy([]string{"-date"})
	return countReq, pageReq
}

// Search returns summaries of visible posts matching query via full-text search.
// Returns an empty slice when query is empty.
func (r *BlevePostRepository) Search(query string) []PostSummary {
	if query == "" {
		return []PostSummary{}
	}

	visQ := bleve.NewTermQuery("true")
	visQ.SetField("visible")
	matchQ := bleve.NewMatchQuery(query)
	conjQ := bleve.NewConjunctionQuery(visQ, matchQ)

	req := bleve.NewSearchRequestOptions(conjQ, 200, 0, false)

	r.postsMu.RLock()
	defer r.postsMu.RUnlock()

	result, err := r.index.Search(req)
	if err != nil {
		r.log.Error("bleve search failed", "err", err)
		return []PostSummary{}
	}

	summaries := make([]PostSummary, 0, len(result.Hits))
	for _, hit := range result.Hits {
		p, ok := r.posts[hit.ID]
		if !ok {
			continue
		}
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
	return summaries
}

// AllTags returns a sorted, deduplicated list of all tags across visible posts.
func (r *BlevePostRepository) AllTags() []string {
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

// LastModified returns the most recent mod-time across visible posts.
func (r *BlevePostRepository) LastModified() time.Time {
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

// Sitemap returns one entry per visible post for use in sitemap.xml.
func (r *BlevePostRepository) Sitemap() []SitemapEntry {
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

func (r *BlevePostRepository) invalidateFeedCache() {
	r.feedMu.Lock()
	r.feedCache = nil
	r.feedMu.Unlock()
	r.atomFeedMu.Lock()
	r.atomFeedCache = nil
	r.atomFeedMu.Unlock()
}

func (r *BlevePostRepository) RSSFeed() ([]byte, error) {
	r.feedMu.RLock()
	cached := r.feedCache
	r.feedMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	r.postsMu.RLock()
	posts := make([]*Post, 0, len(r.posts))
	for _, p := range r.posts {
		if p.IsRSSVisible() {
			posts = append(posts, p)
		}
	}
	r.postsMu.RUnlock()

	data, err := buildRSSXML(r.conf, posts)
	if err != nil {
		return nil, err
	}
	r.feedMu.Lock()
	r.feedCache = data
	r.feedMu.Unlock()
	return data, nil
}

func (r *BlevePostRepository) AtomFeed() ([]byte, error) {
	r.atomFeedMu.RLock()
	cached := r.atomFeedCache
	r.atomFeedMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	r.postsMu.RLock()
	posts := make([]*Post, 0, len(r.posts))
	for _, p := range r.posts {
		if p.IsRSSVisible() {
			posts = append(posts, p)
		}
	}
	r.postsMu.RUnlock()

	data, err := buildAtomXML(r.conf, posts)
	if err != nil {
		return nil, err
	}
	r.atomFeedMu.Lock()
	r.atomFeedCache = data
	r.atomFeedMu.Unlock()
	return data, nil
}

func (r *BlevePostRepository) rescan() {
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

	r.postsMu.RLock()
	snapshot := make(map[string]*Post, len(r.posts))
	for path, p := range r.posts {
		snapshot[path] = p
	}
	r.postsMu.RUnlock()

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
			if p, ok2 := r.preparePost(path); ok2 {
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

	r.postsMu.Lock()
	for _, ch := range changes {
		if ch.post == nil {
			delete(r.posts, ch.path)
			if err := r.index.Delete(ch.path); err != nil {
				r.log.Error("bleve delete failed", "path", ch.path, "err", err)
			}
		} else {
			r.posts[ch.path] = ch.post
			if err := r.index.Index(ch.path, toPostDoc(ch.post)); err != nil {
				r.log.Error("bleve index failed", "path", ch.path, "err", err)
			}
		}
	}
	r.postsMu.Unlock()

	r.invalidateFeedCache()
	r.log.Debug("rescan complete", "total", len(paths))
}

func (r *BlevePostRepository) preparePost(path string) (*Post, bool) {
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
	post := &Post{path: path, slug: r.source.SlugForPath(path), modTime: modTime}
	if err := r.converter.Convert(post, buf); err != nil {
		r.log.Error("failed to convert post", "path", path, "err", err)
		return nil, false
	}
	return post, true
}

func (r *BlevePostRepository) checkPostChanged(path string, existing *Post) (*Post, bool) {
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
	if modTime.IsZero() {
		modTime, err = r.source.StatPost(path)
		if err != nil {
			r.log.Error("failed to stat post", "path", path, "err", err)
			return nil, false
		}
	}
	post := &Post{path: path, slug: r.source.SlugForPath(path), modTime: modTime}
	if err := r.converter.Convert(post, buf); err != nil {
		r.log.Error("failed to convert post", "path", path, "err", err)
		return nil, false
	}
	return post, true
}
