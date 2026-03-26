package service

import (
	"encoding/xml"
	"log/slog"
	"testing"
	"time"

	"github.com/5000K/5000blogs/config"
)

func newTestBleveRepo(t *testing.T, conf *config.Config, source PostSource) *BlevePostIndexer {
	t.Helper()
	repo, err := NewBlevePostIndexer(*conf, slog.Default())

	if err != nil {
		t.Fatalf("NewBlevePostIndexer: %v", err)
	}

	err = repo.Initialize(source, &GoldmarkConverter{})

	if err != nil {
		t.Fatalf("Initialize: %v", err)
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

	data, err := BuildRSSFeed(repo.conf, repo.FeedPosts(nil, ""))
	if err != nil {
		t.Fatalf("BuildRSSFeed: %v", err)
	}
	if err := xml.Unmarshal(data, new(interface{})); err != nil {
		t.Errorf("BuildRSSFeed produced invalid XML: %v", err)
	}
}

func TestBleve_AtomFeed_ValidXML(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\ndate: 2024-01-01\n---\n# Hello"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	data, err := BuildAtomFeed(repo.conf, repo.FeedPosts(nil, ""))
	if err != nil {
		t.Fatalf("BuildAtomFeed: %v", err)
	}
	if err := xml.Unmarshal(data, new(interface{})); err != nil {
		t.Errorf("BuildAtomFeed produced invalid XML: %v", err)
	}
}

// --- LastModified / Sitemap ---

func TestBleve_LastModified_ReturnsLatest(t *testing.T) {
	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	repo := newTestBleveRepo(t, newTestConf(10), newStubSource(nil))
	repo.posts["posts/a.md"] = &Post{path: "posts/a.md", modTime: older, metadata: &Metadata{}}
	repo.posts["posts/b.md"] = &Post{path: "posts/b.md", modTime: newer, metadata: &Metadata{}}

	got := repo.LastModified()
	if !got.Equal(newer) {
		t.Errorf("want %v, got %v", newer, got)
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

// --- ImplementsPostIndexer ---

func TestBleve_ImplementsInterface(t *testing.T) {
	repo := newTestBleveRepo(t, newTestConf(10), newStubSource(nil))
	var _ PostIndexer = repo
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

// --- ListFiltered ---

func TestBleve_ListFiltered_VisibleOnly(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/visible.md": []byte("---\ntitle: Visible\n---\n# body"),
		"posts/hidden.md":  []byte("---\ntitle: Hidden\nvisible: false\n---\n# body"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	results := repo.ListFiltered(PostFilter{})
	if len(results) != 1 {
		t.Fatalf("want 1 visible post, got %d", len(results))
	}
	if results[0].slug != "visible" {
		t.Errorf("want slug 'visible', got %q", results[0].slug)
	}
}

func TestBleve_ListFiltered_TagFilter(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/go.md":   []byte("---\ntitle: Go\ntags: [go]\n---\n# body"),
		"posts/rust.md": []byte("---\ntitle: Rust\ntags: [rust]\n---\n# body"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	results := repo.ListFiltered(PostFilter{Tags: []string{"go"}})
	if len(results) != 1 {
		t.Fatalf("want 1 post, got %d", len(results))
	}
	if results[0].slug != "go" {
		t.Errorf("want slug 'go', got %q", results[0].slug)
	}
}

func TestBleve_ListFiltered_QueryFilter(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("---\ntitle: Alpaca Post\n---\n# body"),
		"posts/b.md": []byte("---\ntitle: Banana Post\n---\n# body"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	results := repo.ListFiltered(PostFilter{Query: "alpaca"})
	if len(results) != 1 {
		t.Fatalf("want 1 post, got %d", len(results))
	}
	if results[0].slug != "a" {
		t.Errorf("want slug 'a', got %q", results[0].slug)
	}
}

func TestBleve_ListFiltered_TagAndQueryFilter(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("---\ntitle: Alpaca Go\ntags: [go]\n---\n# body"),
		"posts/b.md": []byte("---\ntitle: Alpaca Rust\ntags: [rust]\n---\n# body"),
		"posts/c.md": []byte("---\ntitle: Other Go\ntags: [go]\n---\n# body"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	results := repo.ListFiltered(PostFilter{Tags: []string{"go"}, Query: "alpaca"})
	if len(results) != 1 {
		t.Fatalf("want 1 post, got %d", len(results))
	}
	if results[0].slug != "a" {
		t.Errorf("want slug 'a', got %q", results[0].slug)
	}
}

func TestBleve_ListFiltered_SortedByDateDescending(t *testing.T) {
	older := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newTestBleveRepo(t, newTestConf(10), newStubSource(nil))
	oldPost := NewPost("posts/old.md", &Metadata{Title: "Old", Date: older}, []byte("x"))
	newPost := NewPost("posts/new.md", &Metadata{Title: "New", Date: newer}, []byte("x"))
	repo.posts["posts/old.md"] = oldPost
	repo.posts["posts/new.md"] = newPost
	_ = repo.index.Index("posts/old.md", toPostDoc(oldPost))
	_ = repo.index.Index("posts/new.md", toPostDoc(newPost))

	results := repo.ListFiltered(PostFilter{})
	if len(results) != 2 {
		t.Fatalf("want 2 posts, got %d", len(results))
	}
	if results[0].slug != "new" {
		t.Errorf("want newest first, got %q", results[0].slug)
	}
}

// --- ListFilteredPaged ---

func TestBleve_ListFilteredPaged_Pagination(t *testing.T) {
	repo := newTestBleveRepo(t, newTestConf(1), newStubSource(nil)) // Config.PageSize=1, must NOT be used
	for _, slug := range []string{"a", "b", "c", "d"} {
		p := NewPost("posts/"+slug+".md", &Metadata{}, []byte("x"))
		repo.posts["posts/"+slug+".md"] = p
		_ = repo.index.Index("posts/"+slug+".md", toPostDoc(p))
	}

	result := repo.ListFilteredPaged(PostFilter{}, 2, 1)
	if result.TotalPosts != 4 {
		t.Errorf("want TotalPosts=4, got %d", result.TotalPosts)
	}
	if result.TotalPages != 2 {
		t.Errorf("want TotalPages=2, got %d", result.TotalPages)
	}
	if len(result.Posts) != 2 {
		t.Errorf("want 2 posts on page 1, got %d", len(result.Posts))
	}
	if !result.HasNext {
		t.Error("page 1 should have next")
	}
	if result.HasPrev {
		t.Error("page 1 should not have prev")
	}

	result2 := repo.ListFilteredPaged(PostFilter{}, 2, 2)
	if len(result2.Posts) != 2 {
		t.Errorf("want 2 posts on page 2, got %d", len(result2.Posts))
	}
	if result2.HasNext {
		t.Error("page 2 should not have next")
	}
	if !result2.HasPrev {
		t.Error("page 2 should have prev")
	}
}

func TestBleve_ListFilteredPaged_UsesProvidedPageSizeNotConfig(t *testing.T) {
	repo := newTestBleveRepo(t, newTestConf(1), newStubSource(nil)) // Config.PageSize=1
	for _, slug := range []string{"a", "b", "c"} {
		p := NewPost("posts/"+slug+".md", &Metadata{}, []byte("x"))
		repo.posts["posts/"+slug+".md"] = p
		_ = repo.index.Index("posts/"+slug+".md", toPostDoc(p))
	}

	result := repo.ListFilteredPaged(PostFilter{}, 10, 1)
	if len(result.Posts) != 3 {
		t.Errorf("want all 3 posts on one page (pageSize=10), got %d", len(result.Posts))
	}
}

func TestBleve_ListFilteredPaged_PageClamping(t *testing.T) {
	repo := newTestBleveRepo(t, newTestConf(10), newStubSource(nil))
	p := NewPost("posts/a.md", &Metadata{}, []byte("x"))
	repo.posts["posts/a.md"] = p
	_ = repo.index.Index("posts/a.md", toPostDoc(p))

	high := repo.ListFilteredPaged(PostFilter{}, 10, 99)
	if high.Page != 1 {
		t.Errorf("page 99 should clamp to 1, got %d", high.Page)
	}
	low := repo.ListFilteredPaged(PostFilter{}, 10, 0)
	if low.Page != 1 {
		t.Errorf("page 0 should clamp to 1, got %d", low.Page)
	}
}

func TestBleve_ListFilteredPaged_WithFilter(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/go.md":   []byte("---\ntitle: Go\ntags: [go]\n---\n# body"),
		"posts/rust.md": []byte("---\ntitle: Rust\ntags: [rust]\n---\n# body"),
	})
	repo := newTestBleveRepo(t, newTestConf(10), src)
	repo.rescan()

	result := repo.ListFilteredPaged(PostFilter{Tags: []string{"go"}}, 10, 1)
	if result.TotalPosts != 1 {
		t.Errorf("want 1 post, got %d", result.TotalPosts)
	}
	if len(result.Posts) != 1 || result.Posts[0].Slug != "go" {
		t.Errorf("want slug 'go', got %v", result.Posts)
	}
}
