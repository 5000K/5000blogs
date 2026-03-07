package service

import (
	"5000blogs/config"
	"encoding/xml"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func newTestConf(pageSize int) *config.Config {
	cfg := &config.Config{}
	cfg.RescanCron = "* * * * *"
	cfg.PageSize = pageSize
	cfg.SiteURL = "http://example.com"
	return cfg
}

func newTestRepo(conf *config.Config, source PostSource) *MemoryPostRepository {
	return NewMemoryPostRepository(conf, source, &GoMarkdownConverter{}, slog.Default())
}

// --- rescan ---

func TestRescan_AddsNewPosts(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("---\ntitle: A\n---\n\n# A"),
		"posts/b.md": []byte("---\ntitle: B\n---\n\n# B"),
	})
	repo := newTestRepo(newTestConf(10), src)
	repo.rescan()

	if repo.Count() != 2 {
		t.Errorf("want 2 posts, got %d", repo.Count())
	}
}

func TestRescan_RemovesDeletedPosts(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("# A"),
		"posts/b.md": []byte("# B"),
	})
	repo := newTestRepo(newTestConf(10), src)
	repo.rescan()

	delete(src.posts, "posts/b.md")
	repo.rescan()

	if repo.Count() != 1 {
		t.Errorf("want 1 post after removal, got %d", repo.Count())
	}
	if repo.GetBySlug("b") != nil {
		t.Error("post 'b' should have been removed")
	}
}

func TestRescan_UpdatesChangedPost(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("---\ntitle: Old\n---\n\n# Old"),
	})
	cfg := newTestConf(10)
	cfg.SkipUnchangedModTime = false
	repo := newTestRepo(cfg, src)
	repo.rescan()

	src.posts["posts/a.md"] = []byte("---\ntitle: New\n---\n\n# New")
	repo.rescan()

	p := repo.GetBySlug("a")
	if p == nil {
		t.Fatal("post 'a' not found")
	}
	if p.metadata.Title != "New" {
		t.Errorf("want title 'New', got %q", p.metadata.Title)
	}
}

func TestRescan_InvalidatesFeedCacheOnChange(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("# A"),
	})
	repo := newTestRepo(newTestConf(10), src)
	repo.rescan()

	// Prime the feed cache.
	_, _ = repo.RSSFeed()

	repo.feedMu.RLock()
	before := repo.feedCache
	repo.feedMu.RUnlock()
	if before == nil {
		t.Fatal("feed cache should be populated after RSSFeed()")
	}

	// Adding a new post invalidates the cache.
	src.posts["posts/b.md"] = []byte("# B")
	repo.rescan()

	repo.feedMu.RLock()
	after := repo.feedCache
	repo.feedMu.RUnlock()
	if after != nil {
		t.Error("feed cache should be nil after rescan with changes")
	}
}

func TestRescan_NoChangeDoesNotInvalidateCache(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("# A"),
	})
	cfg := newTestConf(10)
	cfg.SkipUnchangedModTime = true
	repo := newTestRepo(cfg, src)
	repo.rescan()

	_, _ = repo.RSSFeed()

	repo.feedMu.RLock()
	before := repo.feedCache
	repo.feedMu.RUnlock()

	// Second rescan with identical source — should not invalidate.
	repo.rescan()

	repo.feedMu.RLock()
	after := repo.feedCache
	repo.feedMu.RUnlock()

	if after == nil && before != nil {
		t.Error("feed cache should remain populated when nothing changed")
	}
}

// --- GetPage ---

func TestGetPage_SortedByDateDescending(t *testing.T) {
	older := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/old.md", &Metadata{Title: "Old", Date: older}, []byte("x")),
		NewPost("posts/new.md", &Metadata{Title: "New", Date: newer}, []byte("x")),
	}
	page := repo.GetPage(1, nil)

	if len(page.Posts) != 2 {
		t.Fatalf("want 2 posts, got %d", len(page.Posts))
	}
	if page.Posts[0].Slug != "new" {
		t.Errorf("want newest first, got %q", page.Posts[0].Slug)
	}
}

func TestGetPage_HiddenPostsExcluded(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/visible.md", &Metadata{Title: "Visible"}, []byte("x")),
		NewPost("posts/hidden.md", &Metadata{Title: "Hidden", Visible: boolPtr(false)}, []byte("x")),
	}
	page := repo.GetPage(1, nil)

	if page.TotalPosts != 1 {
		t.Errorf("want 1 visible post, got %d", page.TotalPosts)
	}
	if page.Posts[0].Slug != "visible" {
		t.Errorf("want slug 'visible', got %q", page.Posts[0].Slug)
	}
}

func TestGetPage_Pagination(t *testing.T) {
	repo := newTestRepo(newTestConf(2), newStubSource(nil))
	slugs := []string{"a", "b", "c", "d", "e"}
	for _, s := range slugs {
		repo.posts = append(repo.posts, NewPost("posts/"+s+".md", &Metadata{}, []byte("x")))
	}

	p1 := repo.GetPage(1, nil)
	if p1.TotalPages != 3 {
		t.Errorf("want 3 total pages, got %d", p1.TotalPages)
	}
	if p1.HasPrev {
		t.Error("page 1 should not have prev")
	}
	if !p1.HasNext {
		t.Error("page 1 should have next")
	}
	if len(p1.Posts) != 2 {
		t.Errorf("want 2 posts on page 1, got %d", len(p1.Posts))
	}

	p3 := repo.GetPage(3, nil)
	if p3.HasNext {
		t.Error("last page should not have next")
	}
	if !p3.HasPrev {
		t.Error("last page should have prev")
	}
	if len(p3.Posts) != 1 {
		t.Errorf("want 1 post on last page, got %d", len(p3.Posts))
	}
}

func TestGetPage_EmptyRepo(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	page := repo.GetPage(1, nil)

	if page.TotalPosts != 0 {
		t.Errorf("want 0 total posts, got %d", page.TotalPosts)
	}
	if page.TotalPages != 1 {
		t.Errorf("want TotalPages=1 for empty repo, got %d", page.TotalPages)
	}
}

func TestGetPage_OutOfBoundsClampsToLastPage(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/a.md", &Metadata{}, []byte("x")),
	}
	page := repo.GetPage(99, nil)
	if page.Page != 1 {
		t.Errorf("want page clamped to 1, got %d", page.Page)
	}
	if len(page.Posts) != 1 {
		t.Errorf("want 1 post on clamped last page, got %d", len(page.Posts))
	}
}

// --- RSSFeed ---

func TestRSSFeed_ValidXML(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/hello.md", &Metadata{Title: "Hello", Date: time.Now()}, []byte("<p>hi</p>")),
	}

	data, err := repo.RSSFeed()
	if err != nil {
		t.Fatalf("RSSFeed: %v", err)
	}

	if err := xml.Unmarshal(data, new(interface{})); err != nil {
		t.Errorf("RSSFeed produced invalid XML: %v", err)
	}
}

func TestRSSFeed_CachedOnSecondCall(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/a.md", &Metadata{Title: "A"}, []byte("x")),
	}

	first, _ := repo.RSSFeed()
	second, _ := repo.RSSFeed()

	// Pointer equality on the first byte confirms cache hit.
	if &first[0] != &second[0] {
		t.Error("expected cached result on second call")
	}
}

func TestRSSFeed_ExcludesRSSHiddenPosts(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/pub.md", &Metadata{Title: "Public"}, []byte("<p>pub</p>")),
		NewPost("posts/rss-hidden.md", &Metadata{Title: "RSSHidden", RSSVisible: boolPtr(false)}, []byte("<p>x</p>")),
	}

	data, err := repo.RSSFeed()
	if err != nil {
		t.Fatalf("RSSFeed: %v", err)
	}
	if strings.Contains(string(data), "RSSHidden") {
		t.Error("RSS feed should not include rss-hidden posts")
	}
}

func TestRSSFeed_LimitedToPageSize(t *testing.T) {
	repo := newTestRepo(newTestConf(2), newStubSource(nil))
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		slug := string(rune('a' + i))
		repo.posts = append(repo.posts, NewPost(
			"posts/"+slug+".md",
			&Metadata{Title: "Post " + slug, Date: base.Add(time.Duration(i) * 24 * time.Hour)},
			[]byte("<p>x</p>"),
		))
	}

	data, err := repo.RSSFeed()
	if err != nil {
		t.Fatalf("RSSFeed: %v", err)
	}
	if count := strings.Count(string(data), "<item>"); count != 2 {
		t.Errorf("want 2 RSS items (page size), got %d", count)
	}
}

// --- Sitemap ---

func TestSitemap_OnlyVisiblePosts(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/visible.md", &Metadata{Title: "Visible"}, []byte("x")),
		NewPost("posts/hidden.md", &Metadata{Title: "Hidden", Visible: boolPtr(false)}, []byte("x")),
	}

	entries := repo.Sitemap()
	if len(entries) != 1 {
		t.Fatalf("want 1 sitemap entry, got %d", len(entries))
	}
	if entries[0].Slug != "visible" {
		t.Errorf("want slug 'visible', got %q", entries[0].Slug)
	}
}

func TestSitemap_LastModFromDate(t *testing.T) {
	date := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/dated.md", &Metadata{Date: date}, []byte("x")),
	}

	entries := repo.Sitemap()
	if len(entries) != 1 {
		t.Fatal("want 1 entry")
	}
	if !entries[0].LastMod.Equal(date) {
		t.Errorf("LastMod: got %v, want %v", entries[0].LastMod, date)
	}
}

func TestSitemap_LastModFallsBackToModTime(t *testing.T) {
	mod := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	p := &Post{path: "posts/no-date.md", modTime: mod, metadata: &Metadata{}}
	repo.posts = []*Post{p}

	entries := repo.Sitemap()
	if len(entries) != 1 {
		t.Fatal("want 1 entry")
	}
	if !entries[0].LastMod.Equal(mod) {
		t.Errorf("LastMod fallback: got %v, want %v", entries[0].LastMod, mod)
	}
}

func TestLastModified_ReturnsZeroWithNoPosts(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	if !repo.LastModified().IsZero() {
		t.Error("want zero time when no posts")
	}
}

func TestLastModified_ReturnsMaxModTime(t *testing.T) {
	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		{path: "posts/a.md", modTime: older, metadata: &Metadata{}},
		{path: "posts/b.md", modTime: newer, metadata: &Metadata{}},
	}

	got := repo.LastModified()
	if !got.Equal(newer) {
		t.Errorf("want %v, got %v", newer, got)
	}
}

func TestLastModified_IgnoresInvisiblePosts(t *testing.T) {
	visible := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hidden := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	hiddenBool := false
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		{path: "posts/a.md", modTime: visible, metadata: &Metadata{}},
		{path: "posts/b.md", modTime: hidden, metadata: &Metadata{Visible: &hiddenBool}},
	}

	got := repo.LastModified()
	if !got.Equal(visible) {
		t.Errorf("want %v (ignoring hidden post), got %v", visible, got)
	}
}

func TestModTime(t *testing.T) {
	mod := time.Date(2026, 5, 10, 8, 30, 0, 0, time.UTC)
	p := &Post{path: "posts/x.md", modTime: mod}
	if !p.ModTime().Equal(mod) {
		t.Errorf("ModTime: want %v, got %v", mod, p.ModTime())
	}
}

// --- Tags ---

func TestGetPage_TagFilter_MatchingPosts(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/a.md", &Metadata{Title: "A", Tags: []string{"go", "web"}}, []byte("x")),
		NewPost("posts/b.md", &Metadata{Title: "B", Tags: []string{"go"}}, []byte("x")),
		NewPost("posts/c.md", &Metadata{Title: "C", Tags: []string{"rust"}}, []byte("x")),
	}

	page := repo.GetPage(1, []string{"go"})

	if page.TotalPosts != 2 {
		t.Errorf("want 2 posts with tag 'go', got %d", page.TotalPosts)
	}
	for _, p := range page.Posts {
		if p.Slug == "c" {
			t.Error("post 'c' (rust only) should not appear in 'go' filter")
		}
	}
}

func TestGetPage_TagFilter_ORLogic(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/a.md", &Metadata{Title: "A", Tags: []string{"go"}}, []byte("x")),
		NewPost("posts/b.md", &Metadata{Title: "B", Tags: []string{"rust"}}, []byte("x")),
		NewPost("posts/c.md", &Metadata{Title: "C", Tags: []string{"java"}}, []byte("x")),
	}

	page := repo.GetPage(1, []string{"go", "rust"})

	if page.TotalPosts != 2 {
		t.Errorf("want 2 posts (OR filter), got %d", page.TotalPosts)
	}
}

func TestGetPage_TagFilter_CaseInsensitive(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/a.md", &Metadata{Title: "A", Tags: []string{"Go"}}, []byte("x")),
	}

	page := repo.GetPage(1, []string{"go"})
	if page.TotalPosts != 1 {
		t.Errorf("tag filter should be case-insensitive, got %d posts", page.TotalPosts)
	}
}

func TestGetPage_TagFilter_TagParam(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/a.md", &Metadata{Title: "A", Tags: []string{"go"}}, []byte("x")),
	}

	page := repo.GetPage(1, []string{"go", "web"})
	if page.TagParam != "&tags=go,web" {
		t.Errorf("TagParam: want \"&tags=go,web\", got %q", page.TagParam)
	}
	if len(page.FilterTags) != 2 {
		t.Errorf("FilterTags: want 2, got %d", len(page.FilterTags))
	}
}

func TestGetPage_NoTagFilter_EmptyTagParam(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/a.md", &Metadata{Title: "A"}, []byte("x")),
	}

	page := repo.GetPage(1, nil)
	if page.TagParam != "" {
		t.Errorf("TagParam should be empty without filter, got %q", page.TagParam)
	}
}

func TestAllTags_ReturnsSortedUnique(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/a.md", &Metadata{Tags: []string{"go", "web"}}, []byte("x")),
		NewPost("posts/b.md", &Metadata{Tags: []string{"go", "rust"}}, []byte("x")),
		NewPost("posts/c.md", &Metadata{Tags: []string{"java"}}, []byte("x")),
	}

	tags := repo.AllTags()
	if len(tags) != 4 {
		t.Fatalf("want 4 unique tags, got %d: %v", len(tags), tags)
	}
	// Should be sorted alphabetically
	expected := []string{"go", "java", "rust", "web"}
	for i, want := range expected {
		if tags[i] != want {
			t.Errorf("tags[%d]: want %q, got %q", i, want, tags[i])
		}
	}
}

func TestAllTags_IgnoresHiddenPosts(t *testing.T) {
	hiddenBool := false
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/a.md", &Metadata{Tags: []string{"visible-tag"}}, []byte("x")),
		NewPost("posts/b.md", &Metadata{Tags: []string{"hidden-tag"}, Visible: &hiddenBool}, []byte("x")),
	}

	tags := repo.AllTags()
	if len(tags) != 1 || tags[0] != "visible-tag" {
		t.Errorf("AllTags should only return tags from visible posts, got %v", tags)
	}
}

func TestAllTags_Empty(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	tags := repo.AllTags()
	if tags == nil {
		t.Error("AllTags should return empty slice, not nil")
	}
	if len(tags) != 0 {
		t.Errorf("want 0 tags, got %d", len(tags))
	}
}

func TestPostSummary_IncludesTags(t *testing.T) {
	repo := newTestRepo(newTestConf(10), newStubSource(nil))
	repo.posts = []*Post{
		NewPost("posts/a.md", &Metadata{Title: "A", Tags: []string{"go", "test"}}, []byte("x")),
	}

	page := repo.GetPage(1, nil)
	if len(page.Posts) != 1 {
		t.Fatal("want 1 post")
	}
	if len(page.Posts[0].Tags) != 2 {
		t.Errorf("want 2 tags in PostSummary, got %v", page.Posts[0].Tags)
	}
}
