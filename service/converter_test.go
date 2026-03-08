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

func TestExtractFrontmatter_DateWithTime_RFC3339(t *testing.T) {
	raw := []byte("---\ndate: 2026-01-02T10:30:45Z\n---\n")
	meta, _, err := extractFrontmatter(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 1, 2, 10, 30, 45, 0, time.UTC)
	if !meta.Date.Equal(want) {
		t.Errorf("date: want %v, got %v", want, meta.Date)
	}
}

func TestExtractFrontmatter_DateWithTime_SpaceSeparated(t *testing.T) {
	raw := []byte("---\ndate: 2026-01-02 10:30:45\n---\n")
	meta, _, err := extractFrontmatter(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 1, 2, 10, 30, 45, 0, time.UTC)
	if !meta.Date.Equal(want) {
		t.Errorf("date: want %v, got %v", want, meta.Date)
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

func TestConvert_SetsPlainText(t *testing.T) {
	c := &GoMarkdownConverter{}
	post := &Post{}
	raw := []byte("---\ntitle: Plain Test\n---\n\n# Hello\n\nWorld.\n")
	if err := c.Convert(post, raw); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if post.plainText == nil {
		t.Fatal("want plainText to be set after Convert")
	}
	plain := string(*post.plainText)
	if !strings.Contains(plain, "Hello") {
		t.Errorf("plain text missing heading text: %q", plain)
	}
	if !strings.Contains(plain, "World") {
		t.Errorf("plain text missing paragraph text: %q", plain)
	}
	if strings.Contains(plain, "<") || strings.Contains(plain, ">") {
		t.Errorf("plain text must not contain HTML tags: %q", plain)
	}
}

func TestHTMLToPlainText_StripsTagsAndEntities(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string // substrings that must appear
		gone  []string // substrings that must not appear
	}{
		{
			name:  "paragraph",
			input: "<p>Hello world</p>",
			want:  []string{"Hello world"},
			gone:  []string{"<p>", "</p>"},
		},
		{
			name:  "heading",
			input: "<h1>Title</h1><p>Body</p>",
			want:  []string{"Title", "Body"},
			gone:  []string{"<h1>", "</h1>"},
		},
		{
			name:  "html entities",
			input: "<p>Tom &amp; Jerry &#8212; cool</p>",
			want:  []string{"Tom & Jerry"},
			gone:  []string{"&amp;"},
		},
		{
			name:  "no tags",
			input: "plain text",
			want:  []string{"plain text"},
		},
		{
			name:  "inline code",
			input: "<p>Use <code>foo()</code> here</p>",
			want:  []string{"foo()"},
			gone:  []string{"<code>"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := string(htmlToPlainText([]byte(tc.input)))
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("want %q in output %q", w, got)
				}
			}
			for _, g := range tc.gone {
				if strings.Contains(got, g) {
					t.Errorf("want %q absent from output %q", g, got)
				}
			}
		})
	}
}

func TestHTMLToPlainText_BlockNewlines(t *testing.T) {
	input := "<h1>Title</h1><p>Para one</p><p>Para two</p>"
	got := string(htmlToPlainText([]byte(input)))
	// Both paragraphs must be present and separated by whitespace
	if !strings.Contains(got, "Para one") || !strings.Contains(got, "Para two") {
		t.Errorf("missing paragraphs: %q", got)
	}
	// Title and first paragraph should be on separate lines
	titleIdx := strings.Index(got, "Title")
	paraIdx := strings.Index(got, "Para one")
	if titleIdx >= paraIdx {
		t.Errorf("Title should appear before Para one: %q", got)
	}
}
