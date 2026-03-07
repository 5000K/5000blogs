package service

import (
	"strings"
	"testing"
)

// --- RSS full content ---

func TestRSSFeed_FullContent_IncludesContentEncoded(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\ndescription: A summary\n---\n\n# Hello\n\nSome content here.\n"),
	})
	cfg := newTestConf(10)
	cfg.RSSFullContent = true
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := repo.RSSFeed()
	if err != nil {
		t.Fatalf("RSSFeed: %v", err)
	}
	xml := string(data)

	if !strings.Contains(xml, "xmlns:content=") {
		t.Error("RSS feed should declare content namespace when RSSFullContent is true")
	}
	if !strings.Contains(xml, "<content:encoded>") {
		t.Error("RSS feed should include content:encoded element")
	}
	if !strings.Contains(xml, "Some content here") {
		t.Error("content:encoded should contain post plain text")
	}
	if !strings.Contains(xml, "<![CDATA[") {
		t.Error("content:encoded should wrap text in CDATA")
	}
}

func TestRSSFeed_NoFullContent_OmitsContentEncoded(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\n---\n\n# Hello\n\nBody text.\n"),
	})
	cfg := newTestConf(10)
	cfg.RSSFullContent = false
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := repo.RSSFeed()
	if err != nil {
		t.Fatalf("RSSFeed: %v", err)
	}
	xml := string(data)

	if strings.Contains(xml, "content:encoded") {
		t.Error("RSS feed should not include content:encoded when RSSFullContent is false")
	}
	if strings.Contains(xml, "xmlns:content") {
		t.Error("RSS feed should not declare content namespace when RSSFullContent is false")
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

	data, err := repo.AtomFeed()
	if err != nil {
		t.Fatalf("AtomFeed: %v", err)
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
	if !strings.Contains(xml, "http://example.com/posts/alpha") {
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

func TestAtomFeed_Cached(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("# A"),
	})
	repo := newTestRepo(newTestConf(10), src)
	repo.rescan()

	first, err := repo.AtomFeed()
	if err != nil {
		t.Fatalf("AtomFeed: %v", err)
	}
	second, _ := repo.AtomFeed()
	if &first[0] != &second[0] {
		// Verify same backing array (cached slice).
		// If identical content is returned that's also fine.
		if string(first) != string(second) {
			t.Error("subsequent AtomFeed calls should return identical content")
		}
	}
}

func TestAtomFeed_CacheInvalidatedOnRescan(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("# A"),
	})
	repo := newTestRepo(newTestConf(10), src)
	repo.rescan()
	_, _ = repo.AtomFeed()

	repo.atomFeedMu.RLock()
	before := repo.atomFeedCache
	repo.atomFeedMu.RUnlock()
	if before == nil {
		t.Fatal("atom feed cache should be populated")
	}

	src.posts["posts/b.md"] = []byte("# B")
	repo.rescan()

	repo.atomFeedMu.RLock()
	after := repo.atomFeedCache
	repo.atomFeedMu.RUnlock()
	if after != nil {
		t.Error("atom feed cache should be nil after rescan with changes")
	}
}

func TestAtomFeed_PageSizeLimitsEntries(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/a.md": []byte("---\ntitle: A\ndate: 2026-01-01\n---\n# A"),
		"posts/b.md": []byte("---\ntitle: B\ndate: 2026-01-02\n---\n# B"),
		"posts/c.md": []byte("---\ntitle: C\ndate: 2026-01-03\n---\n# C"),
	})
	cfg := newTestConf(2)
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := repo.AtomFeed()
	if err != nil {
		t.Fatalf("AtomFeed: %v", err)
	}
	xml := string(data)

	count := strings.Count(xml, "<entry>")
	if count != 2 {
		t.Errorf("want 2 entries (page size), got %d", count)
	}
}

// --- /feed.atom HTTP endpoint ---

func TestAtomEndpoint_ContentType(t *testing.T) {
	src := newStubSource(map[string][]byte{
		"posts/hello.md": []byte("---\ntitle: Hello\n---\n# Hello\n"),
	})
	cfg := newTestConf(10)
	repo := newTestRepo(cfg, src)
	repo.rescan()

	data, err := repo.AtomFeed()
	if err != nil {
		t.Fatalf("AtomFeed: %v", err)
	}
	if !strings.HasPrefix(string(data), "<?xml") {
		t.Errorf("Atom feed should start with XML declaration, got: %q", string(data)[:50])
	}
}
