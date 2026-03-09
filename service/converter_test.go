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

// --- rewriteRelativeDest ---

func TestRewriteRelativeDest_AbsoluteURLUnchanged(t *testing.T) {
	cases := []string{
		"https://example.com/page",
		"http://example.com/",
		"/absolute/path",
		"#anchor",
		"mailto:user@example.com",
	}
	for _, c := range cases {
		got := string(rewriteRelativeDest([]byte(c), "/posts/more/", "/media/more/"))
		if got != c {
			t.Errorf("rewriteRelativeDest(%q): want unchanged, got %q", c, got)
		}
	}
}

func TestRewriteRelativeDest_SimpleRelative(t *testing.T) {
	cases := []struct {
		dest     string
		postsDir string
		want     string
	}{
		{"./example.md", "/posts/more/", "/posts/more/example"},
		{"example.md", "/posts/more/", "/posts/more/example"},
		{"../other.md", "/posts/more/", "/posts/other"},
		{"./example.md", "/posts/", "/posts/example"},
		{"sub/page.md", "/posts/", "/posts/sub/page"},
		// no .md extension — leave path as-is but still absolutise
		{"./notes", "/posts/more/", "/posts/more/notes"},
	}
	for _, tc := range cases {
		got := string(rewriteRelativeDest([]byte(tc.dest), tc.postsDir, "/media/more/"))
		if got != tc.want {
			t.Errorf("rewriteRelativeDest(%q, %q): want %q, got %q", tc.dest, tc.postsDir, tc.want, got)
		}
	}
}

func TestRewriteRelativeDest_WithFragment(t *testing.T) {
	got := string(rewriteRelativeDest([]byte("./example.md#section"), "/posts/more/", "/media/more/"))
	if got != "/posts/more/example#section" {
		t.Errorf("want /posts/more/example#section, got %q", got)
	}
}

func TestRewriteRelativeDest_WithQuery(t *testing.T) {
	got := string(rewriteRelativeDest([]byte("./example.md?foo=bar"), "/posts/more/", "/media/more/"))
	if got != "/posts/more/example?foo=bar" {
		t.Errorf("want /posts/more/example?foo=bar, got %q", got)
	}
}

func TestRewriteRelativeDest_MediaFiles(t *testing.T) {
	cases := []struct {
		desc     string
		dest     string
		postsDir string
		mediaDir string
		want     string
	}{
		{"png in subdir", "./funny.png", "/posts/more/", "/media/more/", "/media/more/funny.png"},
		{"jpg at root", "./photo.jpg", "/posts/", "/media/", "/media/photo.jpg"},
		{"parent dir traversal", "../banner.gif", "/posts/more/", "/media/more/", "/media/banner.gif"},
		{"nested media", "assets/image.svg", "/posts/more/", "/media/more/", "/media/more/assets/image.svg"},
		{"media with fragment", "./img.png#L1", "/posts/", "/media/", "/media/img.png#L1"},
		{"video", "./demo.mp4", "/posts/more/", "/media/more/", "/media/more/demo.mp4"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := string(rewriteRelativeDest([]byte(tc.dest), tc.postsDir, tc.mediaDir))
			if got != tc.want {
				t.Errorf("want %q, got %q", tc.want, got)
			}
		})
	}
}

// --- GoMarkdownConverter link rewriting integration ---

func TestConvert_RelativeLinksRewritten(t *testing.T) {
	c := &GoMarkdownConverter{}
	post := &Post{slug: "more+about"}
	raw := []byte("# Page\n\n[Example](./example.md)\n")
	if err := c.Convert(post, raw); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	html := string(*post.contents)
	if !strings.Contains(html, `href="/posts/more/example"`) {
		t.Errorf("want href=/posts/more/example in HTML, got:\n%s", html)
	}
	if strings.Contains(html, "example.md") {
		t.Errorf("bare .md href should have been rewritten, got:\n%s", html)
	}
}

func TestConvert_RelativeLinksRewritten_TopLevelPost(t *testing.T) {
	c := &GoMarkdownConverter{}
	post := &Post{slug: "about"}
	raw := []byte("[Go](./other.md)\n")
	if err := c.Convert(post, raw); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	html := string(*post.contents)
	if !strings.Contains(html, `href="/posts/other"`) {
		t.Errorf("want href=/posts/other, got:\n%s", html)
	}
}

func TestConvert_AbsoluteLinksUnchanged(t *testing.T) {
	c := &GoMarkdownConverter{}
	post := &Post{slug: "more+about"}
	raw := []byte("[External](https://example.com) [Anchor](#section) [Root](/posts/hello)\n")
	if err := c.Convert(post, raw); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	html := string(*post.contents)
	if !strings.Contains(html, `href="https://example.com"`) {
		t.Errorf("external link changed: %s", html)
	}
	if !strings.Contains(html, `href="#section"`) {
		t.Errorf("anchor link changed: %s", html)
	}
	if !strings.Contains(html, `href="/posts/hello"`) {
		t.Errorf("absolute path link changed: %s", html)
	}
}

func TestConvert_RelativeImageRewrittenToMedia(t *testing.T) {
	c := &GoMarkdownConverter{}
	post := &Post{slug: "more+about"}
	raw := []byte("# Page\n\n![Alt](./funny.png)\n")
	if err := c.Convert(post, raw); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	html := string(*post.contents)
	if !strings.Contains(html, `src="/media/more/funny.png"`) {
		t.Errorf("want src=/media/more/funny.png in HTML, got:\n%s", html)
	}
}

func TestConvert_RelativeImageTopLevelPost(t *testing.T) {
	c := &GoMarkdownConverter{}
	post := &Post{slug: "about"}
	raw := []byte("![Banner](./banner.jpg)\n")
	if err := c.Convert(post, raw); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	html := string(*post.contents)
	if !strings.Contains(html, `src="/media/banner.jpg"`) {
		t.Errorf("want src=/media/banner.jpg in HTML, got:\n%s", html)
	}
}
