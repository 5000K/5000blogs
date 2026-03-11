package service

import (
	"strings"
	"testing"
)

// --- RSS content modes ---

func TestRSSFeed_ContentText_IncludesPlainText(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\ndescription: A summary\n---\n\n# Hello\n\nSome content here.\n"),
	})
	cfg := newTestConf(10)
	cfg.RSSContent = "text"
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := BuildRSSFeed(cfg, repo.FeedPosts(nil, ""))
	if err != nil {
		t.Fatalf("BuildRSSFeed: %v", err)
	}
	xml := string(data)

	if !strings.Contains(xml, "xmlns:content=") {
		t.Error("RSS feed should declare content namespace when rss_content=text")
	}
	if !strings.Contains(xml, "<content:encoded>") {
		t.Error("RSS feed should include content:encoded element")
	}
	if !strings.Contains(xml, "Some content here") {
		t.Error("content:encoded should contain post plain text")
	}
	if !strings.Contains(xml, "<![CDATA[") {
		t.Error("content:encoded should wrap content in CDATA")
	}
}

func TestRSSFeed_ContentHTML_IncludesRenderedHTML(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\n---\n\n# Hello\n\nSome **bold** content.\n"),
	})
	cfg := newTestConf(10)
	cfg.RSSContent = "html"
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := BuildRSSFeed(cfg, repo.FeedPosts(nil, ""))
	if err != nil {
		t.Fatalf("BuildRSSFeed: %v", err)
	}
	xml := string(data)

	if !strings.Contains(xml, "xmlns:content=") {
		t.Error("RSS feed should declare content namespace when rss_content=html")
	}
	if !strings.Contains(xml, "<content:encoded>") {
		t.Error("RSS feed should include content:encoded element")
	}
	if !strings.Contains(xml, "<strong>") {
		t.Error("content:encoded should contain rendered HTML tags")
	}
	if !strings.Contains(xml, "<![CDATA[") {
		t.Error("content:encoded should wrap content in CDATA")
	}
}

func TestRSSFeed_ContentNone_OmitsContentEncoded(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\n---\n\n# Hello\n\nBody text.\n"),
	})
	cfg := newTestConf(10)
	cfg.RSSContent = "none"
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := BuildRSSFeed(cfg, repo.FeedPosts(nil, ""))
	if err != nil {
		t.Fatalf("BuildRSSFeed: %v", err)
	}
	xml := string(data)

	if strings.Contains(xml, "content:encoded") {
		t.Error("RSS feed should not include content:encoded when rss_content=none")
	}
	if strings.Contains(xml, "xmlns:content") {
		t.Error("RSS feed should not declare content namespace when rss_content=none")
	}
}

func TestEscapeCDATA(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"plain text", "plain text"},
		{"has ]]> end", "has ]]]]><![CDATA[> end"},
		{"multiple ]]> here ]]> done", "multiple ]]]]><![CDATA[> here ]]]]><![CDATA[> done"},
	}
	for _, tc := range tests {
		got := escapeCDATA(tc.input)
		if got != tc.want {
			t.Errorf("escapeCDATA(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- Atom feed ---

func TestAtomFeed_ContentText(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\n---\n\n# Hello\n\nSome content here.\n"),
	})
	cfg := newTestConf(10)
	cfg.RSSContent = "text"
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := BuildAtomFeed(cfg, repo.FeedPosts(nil, ""))
	if err != nil {
		t.Fatalf("BuildAtomFeed: %v", err)
	}
	xml := string(data)

	if !strings.Contains(xml, `<content type="text">`) {
		t.Error("Atom feed should include content element with type=text")
	}
	if !strings.Contains(xml, "Some content here") {
		t.Error("Atom content should contain plain text")
	}
	if !strings.Contains(xml, "<![CDATA[") {
		t.Error("Atom content should wrap text in CDATA")
	}
}

func TestAtomFeed_ContentHTML(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\n---\n\n# Hello\n\nSome **bold** content.\n"),
	})
	cfg := newTestConf(10)
	cfg.RSSContent = "html"
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := BuildAtomFeed(cfg, repo.FeedPosts(nil, ""))
	if err != nil {
		t.Fatalf("BuildAtomFeed: %v", err)
	}
	xml := string(data)

	if !strings.Contains(xml, `<content type="html">`) {
		t.Error("Atom feed should include content element with type=html")
	}
	if !strings.Contains(xml, "<strong>") {
		t.Error("Atom content should contain rendered HTML")
	}
	if !strings.Contains(xml, "<![CDATA[") {
		t.Error("Atom content should wrap HTML in CDATA")
	}
}

func TestAtomFeed_ContentNone_OmitsContent(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\n---\n\n# Hello\n\nBody text.\n"),
	})
	cfg := newTestConf(10)
	cfg.RSSContent = "none"
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := BuildAtomFeed(cfg, repo.FeedPosts(nil, ""))
	if err != nil {
		t.Fatalf("BuildAtomFeed: %v", err)
	}
	xml := string(data)

	if strings.Contains(xml, "<content") {
		t.Error("Atom feed should not include content element when rss_content=none")
	}
}

func TestAtomFeed_Structure(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/alpha.md": []byte("---\ntitle: Alpha\ndescription: First post summary\nauthor: Alice\ndate: 2026-01-01\n---\n\n# Alpha\n"),
		"posts/beta.md":  []byte("---\ntitle: Beta\ndate: 2026-02-01\n---\n\n# Beta\n"),
	})
	cfg := newTestConf(10)
	cfg.SiteURL = "http://example.com"
	cfg.BlogName = "My Blog"
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := BuildAtomFeed(cfg, repo.FeedPosts(nil, ""))
	if err != nil {
		t.Fatalf("BuildAtomFeed: %v", err)
	}
	xml := string(data)

	if !strings.Contains(xml, `xmlns="http://www.w3.org/2005/Atom"`) {
		t.Error("Atom feed should declare Atom namespace")
	}
	if !strings.Contains(xml, "<title>My Blog</title>") {
		t.Errorf("Atom feed should include feed title; got:\n%s", xml)
	}
	if !strings.Contains(xml, "<title>Alpha</title>") || !strings.Contains(xml, "<title>Beta</title>") {
		t.Error("Atom feed should include entry titles")
	}
	if !strings.Contains(xml, "http://example.com/alpha") {
		t.Error("Atom feed entries should include post URL")
	}
	if !strings.Contains(xml, "<summary>First post summary</summary>") {
		t.Error("Atom feed should include entry summary")
	}
	if !strings.Contains(xml, "<name>Alice</name>") {
		t.Error("Atom feed should include author when present")
	}
	if !strings.Contains(xml, "2026-01-01T00:00:00Z") {
		t.Error("Atom feed should include RFC 3339 updated date")
	}
	if strings.Contains(xml, "content:encoded") {
		t.Error("Atom feed should not include RSS-specific content:encoded")
	}
	// rel=self link
	if !strings.Contains(xml, `rel="self"`) {
		t.Error("Atom feed should include self link")
	}
	if !strings.Contains(xml, "feed.atom") {
		t.Error("Atom feed self link should point to /feed.atom")
	}
}

func TestAtomFeed_FeedSizeLimitsEntries(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("---\ntitle: A\ndate: 2026-01-01\n---\n# A"),
		"posts/b.md": []byte("---\ntitle: B\ndate: 2026-01-02\n---\n# B"),
		"posts/c.md": []byte("---\ntitle: C\ndate: 2026-01-03\n---\n# C"),
	})
	cfg := newTestConf(2)
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := BuildAtomFeed(cfg, repo.FeedPosts(nil, ""))
	if err != nil {
		t.Fatalf("BuildAtomFeed: %v", err)
	}
	xml := string(data)

	count := strings.Count(xml, "<entry>")
	if count != 2 {
		t.Errorf("want 2 entries (feed size), got %d", count)
	}
}

// --- XML declaration ---

func TestBuildAtomFeed_StartsWithXMLDeclaration(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\n---\n# Hello\n"),
	})
	cfg := newTestConf(10)
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := BuildAtomFeed(cfg, repo.FeedPosts(nil, ""))
	if err != nil {
		t.Fatalf("BuildAtomFeed: %v", err)
	}
	if !strings.HasPrefix(string(data), "<?xml") {
		t.Errorf("Atom feed should start with XML declaration, got: %q", string(data)[:50])
	}
}

// --- FeedPosts filtering ---

func TestFeedPosts_TagFilter(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/go.md":   []byte("---\ntitle: Go Post\ntags: [go]\n---\n# Go"),
		"posts/rust.md": []byte("---\ntitle: Rust Post\ntags: [rust]\n---\n# Rust"),
	})
	repo := newTestRepo(newTestConf(10), src)
	repo.rescan()

	posts := repo.FeedPosts([]string{"go"}, "")
	if len(posts) != 1 {
		t.Fatalf("want 1 post with tag 'go', got %d", len(posts))
	}
	if posts[0].Data().Slug != "go" {
		t.Errorf("want slug 'go', got %q", posts[0].Data().Slug)
	}
}

func TestFeedPosts_QueryFilter(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello World\n---\n# Hello"),
		"posts/bye.md":   []byte("---\ntitle: Goodbye\n---\n# Bye"),
	})
	repo := newTestRepo(newTestConf(10), src)
	repo.rescan()

	posts := repo.FeedPosts(nil, "hello")
	if len(posts) != 1 {
		t.Fatalf("want 1 post matching 'hello', got %d", len(posts))
	}
	if posts[0].Data().Slug != "hello" {
		t.Errorf("want slug 'hello', got %q", posts[0].Data().Slug)
	}
}

func TestBuildRSSFeed_TagFiltered(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/go.md":   []byte("---\ntitle: Go Post\ntags: [go]\ndate: 2026-01-01\n---\n# Go"),
		"posts/rust.md": []byte("---\ntitle: Rust Post\ntags: [rust]\ndate: 2026-01-02\n---\n# Rust"),
	})
	cfg := newTestConf(10)
	cfg.SiteURL = "http://example.com"
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := BuildRSSFeed(cfg, repo.FeedPosts([]string{"go"}, ""))
	if err != nil {
		t.Fatalf("BuildRSSFeed: %v", err)
	}
	xml := string(data)

	if !strings.Contains(xml, "Go Post") {
		t.Error("filtered RSS feed should include 'Go Post'")
	}
	if strings.Contains(xml, "Rust Post") {
		t.Error("filtered RSS feed should not include 'Rust Post'")
	}
}

func TestBuildAtomFeed_QueryFiltered(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello World\ndate: 2026-01-01\n---\n# Hello"),
		"posts/bye.md":   []byte("---\ntitle: Goodbye\ndate: 2026-01-02\n---\n# Bye"),
	})
	cfg := newTestConf(10)
	cfg.SiteURL = "http://example.com"
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := BuildAtomFeed(cfg, repo.FeedPosts(nil, "hello"))
	if err != nil {
		t.Fatalf("BuildAtomFeed: %v", err)
	}
	xml := string(data)

	if !strings.Contains(xml, "Hello World") {
		t.Error("filtered Atom feed should include 'Hello World'")
	}
	if strings.Contains(xml, "Goodbye") {
		t.Error("filtered Atom feed should not include 'Goodbye'")
	}
}
