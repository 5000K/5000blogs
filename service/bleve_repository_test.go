package service

import (
	"5000blogs/config"
	"encoding/xml"
	"log/slog"
	"testing"
	"time"
)

func newTestBleveRepo(t *testing.T, conf *config.Config, source PostSource) *BlevePostRepository {
	t.Helper()
	repo, err := NewBlevePostRepository(conf, source, &GoMarkdownConverter{}, slog.Default())
	if err != nil {
		t.Fatalf("NewBlevePostRepository: %v", err)
	}
	return repo
}

// --- rescan ---

func TestBleve_Rescan_AddsNewPosts(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("---\ntitle: A\n---\n\n# A"),
		"posts/b.md": []byte("---\ntitle: B\n---\n\n# B"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	if repo.Count() != 2 {
		t.Errorf("want 2 posts, got %d", repo.Count())
	}
}

func TestBleve_Rescan_RemovesDeletedPosts(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("# A"),
		"posts/b.md": []byte("# B"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
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

func TestBleve_Rescan_UpdatesChangedPost(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("---\ntitle: Old\n---\n\n# Old"),
	})
	cfg := newTestConf(10)
	cfg.SkipUnchangedModTime = false
	repo := newTestBleveRepo(t, cfg, src)
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

func TestBleve_Rescan_InvalidatesFeedCacheOnChange(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("# A"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	_, _ = repo.RSSFeed()

	repo.feedMu.RLock()
	before := repo.feedCache
	repo.feedMu.RUnlock()
	if before == nil {
		t.Fatal("feed cache should be populated after RSSFeed()")
	}

	src.posts["posts/b.md"] = []byte("# B")
	repo.rescan()

	repo.feedMu.RLock()
	after := repo.feedCache
	repo.feedMu.RUnlock()
	if after != nil {
		t.Error("feed cache should be nil after rescan with changes")
	}
}

// --- Get / GetBySlug ---

func TestBleve_Get_ReturnsPostByPath(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\n---\n\n# Hello"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	p := repo.Get("posts/hello.md")
	if p == nil {
		t.Fatal("expected post, got nil")
	}
	if p.metadata.Title != "Hello" {
		t.Errorf("want title 'Hello', got %q", p.metadata.Title)
	}
}

func TestBleve_GetBySlug_ReturnsPost(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/my-post.md": []byte("---\ntitle: My Post\n---\n# body"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	p := repo.GetBySlug("my-post")
	if p == nil {
		t.Fatal("expected post by slug, got nil")
	}
	if p.metadata.Title != "My Post" {
		t.Errorf("want title 'My Post', got %q", p.metadata.Title)
	}
}

func TestBleve_GetBySlug_MissingReturnsNil(t *testing.T) {
	repo := newTestBleveRepo(t, newTestConf(10), newStubSource(nil))
	repo.rescan()
	if repo.GetBySlug("nope") != nil {
		t.Error("want nil for missing slug")
	}
}

// --- GetPage ---

func TestBleve_GetPage_SortedByDateDescending(t *testing.T) {
	older := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := newTestBleveRepo(t, newTestConf(10), newStubSource(nil))
	repo.posts["posts/old.md"] = NewPost("posts/old.md", &Metadata{Title: "Old", Date: older}, []byte("x"))
	repo.posts["posts/new.md"] = NewPost("posts/new.md", &Metadata{Title: "New", Date: newer}, []byte("x"))
	if err := repo.index.Index("posts/old.md", toPostDoc(repo.posts["posts/old.md"])); err != nil {
		t.Fatal(err)
	}
	if err := repo.index.Index("posts/new.md", toPostDoc(repo.posts["posts/new.md"])); err != nil {
		t.Fatal(err)
	}

	page := repo.GetPage(1, nil)
	if len(page.Posts) != 2 {
		t.Fatalf("want 2 posts, got %d", len(page.Posts))
	}
	if page.Posts[0].Slug != "new" {
		t.Errorf("want newest first, got %q", page.Posts[0].Slug)
	}
}

func TestBleve_GetPage_HiddenPostsExcluded(t *testing.T) {
	visible := NewPost("posts/visible.md", &Metadata{Title: "Visible"}, []byte("x"))
	hidden := NewPost("posts/hidden.md", &Metadata{Title: "Hidden", Visible: boolPtr(false)}, []byte("x"))
	repo := newTestBleveRepo(t, newTestConf(10), newStubSource(nil))
	repo.posts["posts/visible.md"] = visible
	repo.posts["posts/hidden.md"] = hidden
	_ = repo.index.Index("posts/visible.md", toPostDoc(visible))
	_ = repo.index.Index("posts/hidden.md", toPostDoc(hidden))

	page := repo.GetPage(1, nil)
	if page.TotalPosts != 1 {
		t.Errorf("want 1 visible post, got %d", page.TotalPosts)
	}
	if len(page.Posts) != 1 || page.Posts[0].Slug != "visible" {
		t.Errorf("want slug 'visible', got %v", page.Posts)
	}
}

func TestBleve_GetPage_Pagination(t *testing.T) {
	repo := newTestBleveRepo(t, newTestConf(2), newStubSource(nil))
	for _, s := range []string{"a", "b", "c", "d", "e"} {
		p := NewPost("posts/"+s+".md", &Metadata{}, []byte("x"))
		repo.posts["posts/"+s+".md"] = p
		_ = repo.index.Index("posts/"+s+".md", toPostDoc(p))
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

func TestBleve_GetPage_TagFilter(t *testing.T) {
	goPost := NewPost("posts/go.md", &Metadata{Title: "Go", Tags: []string{"go"}}, []byte("x"))
	rubyPost := NewPost("posts/ruby.md", &Metadata{Title: "Ruby", Tags: []string{"ruby"}}, []byte("x"))
	repo := newTestBleveRepo(t, newTestConf(10), newStubSource(nil))
	repo.posts["posts/go.md"] = goPost
	repo.posts["posts/ruby.md"] = rubyPost
	_ = repo.index.Index("posts/go.md", toPostDoc(goPost))
	_ = repo.index.Index("posts/ruby.md", toPostDoc(rubyPost))

	page := repo.GetPage(1, []string{"go"})
	if page.TotalPosts != 1 {
		t.Errorf("want 1 post with tag 'go', got %d", page.TotalPosts)
	}
	if page.Posts[0].Slug != "go" {
		t.Errorf("want slug 'go', got %q", page.Posts[0].Slug)
	}
}

func TestBleve_GetPage_EmptyRepo(t *testing.T) {
	repo := newTestBleveRepo(t, newTestConf(10), newStubSource(nil))
	page := repo.GetPage(1, nil)
	if page.TotalPosts != 0 {
		t.Errorf("want 0 posts, got %d", page.TotalPosts)
	}
	if page.TotalPages != 1 {
		t.Errorf("want TotalPages=1 for empty repo, got %d", page.TotalPages)
	}
}

// --- AllTags ---

func TestBleve_AllTags_ReturnsSortedUnique(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("---\ntags: [go, web]\n---\n# a"),
		"posts/b.md": []byte("---\ntags: [go, rust]\n---\n# b"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	tags := repo.AllTags()
	if len(tags) != 3 {
		t.Fatalf("want 3 unique tags, got %d: %v", len(tags), tags)
	}
	for i, want := range []string{"go", "rust", "web"} {
		if tags[i] != want {
			t.Errorf("tags[%d]: want %q, got %q", i, want, tags[i])
		}
	}
}

// --- RSSFeed / AtomFeed ---

func TestBleve_RSSFeed_ValidXML(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\ndate: 2024-01-01\n---\n# Hello"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	data, err := repo.RSSFeed()
	if err != nil {
		t.Fatalf("RSSFeed: %v", err)
	}
	if err := xml.Unmarshal(data, new(interface{})); err != nil {
		t.Errorf("RSSFeed produced invalid XML: %v", err)
	}
}

func TestBleve_AtomFeed_ValidXML(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\ndate: 2024-01-01\n---\n# Hello"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	data, err := repo.AtomFeed()
	if err != nil {
		t.Fatalf("AtomFeed: %v", err)
	}
	if err := xml.Unmarshal(data, new(interface{})); err != nil {
		t.Errorf("AtomFeed produced invalid XML: %v", err)
	}
}

// --- LastModified / Sitemap ---

func TestBleve_LastModified_ReturnsLatest(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("# A"),
		"posts/b.md": []byte("# B"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	lm := repo.LastModified()
	if lm.IsZero() {
		t.Error("LastModified should not be zero after loading posts")
	}
}

func TestBleve_Sitemap_OnlyVisiblePosts(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/visible.md": []byte("# Visible"),
		"posts/hidden.md":  []byte("---\nvisible: false\n---\n# Hidden"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	entries := repo.Sitemap()
	if len(entries) != 1 {
		t.Fatalf("want 1 sitemap entry, got %d", len(entries))
	}
	if entries[0].Slug != "visible" {
		t.Errorf("want slug 'visible', got %q", entries[0].Slug)
	}
}

// --- ImplementsPostRepository ---

func TestBleve_ImplementsInterface(t *testing.T) {
	repo := newTestBleveRepo(t, newTestConf(10), newStubSource(nil))
	var _ PostRepository = repo
}

// --- Search ---

func TestBleve_Search_MatchesTitle(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/go.md":   []byte("---\ntitle: Learning Go\n---\nsome content"),
		"posts/rust.md": []byte("---\ntitle: Learning Rust\n---\nother content"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	results := repo.Search("rust")
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Slug != "rust" {
		t.Errorf("want slug 'rust', got %q", results[0].Slug)
	}
}

func TestBleve_Search_EmptyQueryReturnsEmpty(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("---\ntitle: A\n---\n# body"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	results := repo.Search("")
	if len(results) != 0 {
		t.Errorf("empty query should return no results, got %d", len(results))
	}
}

func TestBleve_Search_ExcludesHiddenPosts(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/visible.md": []byte("---\ntitle: Visible Post\n---\n# body"),
		"posts/hidden.md":  []byte("---\ntitle: Hidden Post\nvisible: false\n---\n# body"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	results := repo.Search("post")
	if len(results) != 1 {
		t.Fatalf("want 1 result (only visible), got %d", len(results))
	}
	if results[0].Slug != "visible" {
		t.Errorf("want slug 'visible', got %q", results[0].Slug)
	}
}

func TestBleve_Search_NoMatchReturnsEmptySlice(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("---\ntitle: A\n---\n# body"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	results := repo.Search("zzznomatch")
	if results == nil {
		t.Error("Search should return empty slice, not nil")
	}
	if len(results) != 0 {
		t.Errorf("want 0 results, got %d", len(results))
	}
}
