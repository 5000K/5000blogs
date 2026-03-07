package service

import (
	"strings"
	"testing"
	"time"
)

func TestExtractFrontmatter_Valid(t *testing.T) {
	raw := []byte("---\ntitle: Hello\nauthor: Alice\ndate: 2026-01-02\n---\n\n# Content\n")
	meta, body, err := extractFrontmatter(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Title != "Hello" {
		t.Errorf("title: want Hello, got %q", meta.Title)
	}
	if meta.Author != "Alice" {
		t.Errorf("author: want Alice, got %q", meta.Author)
	}
	want := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if !meta.Date.Equal(want) {
		t.Errorf("date: want %v, got %v", want, meta.Date)
	}
	if string(body) != "\n# Content\n" {
		t.Errorf("body: got %q", string(body))
	}
}

func TestExtractFrontmatter_NoFrontmatter(t *testing.T) {
	raw := []byte("# Just content\n\nNo front matter here.\n")
	meta, body, err := extractFrontmatter(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Title != "" {
		t.Errorf("want empty title, got %q", meta.Title)
	}
	if string(body) != string(raw) {
		t.Errorf("body should equal raw when no front matter found")
	}
}

func TestExtractFrontmatter_EmptyMetadata(t *testing.T) {
	raw := []byte("---\n---\n\nContent.\n")
	meta, body, err := extractFrontmatter(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Title != "" {
		t.Errorf("want empty title, got %q", meta.Title)
	}
	if string(body) != "\nContent.\n" {
		t.Errorf("body: got %q", string(body))
	}
}

func TestExtractFrontmatter_UnclosedDelimiter(t *testing.T) {
	raw := []byte("---\ntitle: Oops\n\n# No closing delimiter\n")
	meta, body, err := extractFrontmatter(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Falls back to treating the whole file as content
	if string(body) != string(raw) {
		t.Errorf("body should equal raw for unclosed front matter")
	}
	if meta.Title != "" {
		t.Errorf("want empty title for unclosed front matter, got %q", meta.Title)
	}
}

func TestExtractFrontmatter_InvalidYAML(t *testing.T) {
	raw := []byte("---\n: invalid: yaml: [\n---\n\n# Content\n")
	_, _, err := extractFrontmatter(raw)
	if err == nil {
		t.Error("want error for invalid YAML, got nil")
	}
}

func TestExtractFrontmatter_VisibleFalse(t *testing.T) {
	raw := []byte("---\ntitle: Hidden\nvisible: false\n---\n")
	meta, _, err := extractFrontmatter(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Visible == nil || *meta.Visible != false {
		t.Errorf("want visible=false, got %v", meta.Visible)
	}
}

func TestConvert_RendersHTML(t *testing.T) {
	c := &GoMarkdownConverter{}
	post := &Post{}
	raw := []byte("---\ntitle: Test Post\nauthor: Bob\n---\n\n# Hello\n\nWorld.\n")
	if err := c.Convert(post, raw); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if post.metadata.Title != "Test Post" {
		t.Errorf("title: want Test Post, got %q", post.metadata.Title)
	}
	if post.metadata.Author != "Bob" {
		t.Errorf("author: want Bob, got %q", post.metadata.Author)
	}
	contents := string(*post.contents)
	if contents == "" {
		t.Error("want non-empty HTML output")
	}
	// The rendered HTML should contain the heading and paragraph text
	if !strings.Contains(contents, "Hello") {
		t.Errorf("rendered HTML missing heading: %s", contents)
	}
	if !strings.Contains(contents, "World") {
		t.Errorf("rendered HTML missing paragraph: %s", contents)
	}
	// Front matter must not appear in HTML output
	if strings.Contains(contents, "title:") {
		t.Errorf("front matter leaked into HTML: %s", contents)
	}
}

func TestConvert_NoFrontmatter(t *testing.T) {
	c := &GoMarkdownConverter{}
	post := &Post{}
	raw := []byte("# No Front Matter\n\nJust content.\n")
	if err := c.Convert(post, raw); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if post.metadata.Title != "" {
		t.Errorf("want empty title, got %q", post.metadata.Title)
	}
	if post.contents == nil || string(*post.contents) == "" {
		t.Error("want rendered HTML")
	}
}
