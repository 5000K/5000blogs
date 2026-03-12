package service

import (
	"strings"
	"testing"
)

// fullConvert runs ExtractMetadata then Convert - the two steps repositories use.
func fullConvert(c *GoldmarkConverter, post *Post, raw []byte) error {
	body, err := c.ExtractMetadata(post, raw)
	if err != nil {
		return err
	}
	return c.Convert(post, body, nil)
}

// --- ExtractMetadata ---

func TestGoldmarkExtractMetadata_ParsesFrontmatter(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{}
	raw := []byte("---\ntitle: Test Post\nauthor: Bob\n---\n\n# Hello\n\nWorld.\n")
	body, err := c.ExtractMetadata(post, raw)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if post.metadata.Title != "Test Post" {
		t.Errorf("title: want Test Post, got %q", post.metadata.Title)
	}
	if post.metadata.Author != "Bob" {
		t.Errorf("author: want Bob, got %q", post.metadata.Author)
	}
	if post.hash == 0 {
		t.Error("want non-zero hash after ExtractMetadata")
	}
	if strings.Contains(string(body), "title:") {
		t.Errorf("front matter should be stripped from body: %s", body)
	}
	if !strings.Contains(string(body), "Hello") {
		t.Errorf("body should contain markdown content: %s", body)
	}
}

func TestGoldmarkExtractMetadata_NoFrontmatter(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{}
	raw := []byte("# No Front Matter\n\nJust content.\n")
	body, err := c.ExtractMetadata(post, raw)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if post.metadata.Title != "" {
		t.Errorf("want empty title, got %q", post.metadata.Title)
	}
	if string(body) != string(raw) {
		t.Errorf("body should equal raw when no frontmatter")
	}
}

func TestGoldmarkExtractMetadata_SetsHash(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{}
	raw := []byte("# Hello\n")
	if _, err := c.ExtractMetadata(post, raw); err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if post.hash == 0 {
		t.Error("want non-zero hash")
	}
}

// --- GoldmarkConverter basic rendering ---

func TestGoldmarkConvert_RendersHTML(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{}
	raw := []byte("---\ntitle: Test Post\nauthor: Bob\n---\n\n# Hello\n\nWorld.\n")
	if err := fullConvert(c, post, raw); err != nil {
		t.Fatalf("fullConvert: %v", err)
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
	if !strings.Contains(contents, "Hello") {
		t.Errorf("rendered HTML missing heading: %s", contents)
	}
	if !strings.Contains(contents, "World") {
		t.Errorf("rendered HTML missing paragraph: %s", contents)
	}
	if strings.Contains(contents, "title:") {
		t.Errorf("front matter leaked into HTML: %s", contents)
	}
}

func TestGoldmarkConvert_NoFrontmatter(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{}
	raw := []byte("# No Front Matter\n\nJust content.\n")
	if err := fullConvert(c, post, raw); err != nil {
		t.Fatalf("fullConvert: %v", err)
	}
	if post.metadata.Title != "" {
		t.Errorf("want empty title, got %q", post.metadata.Title)
	}
	if post.contents == nil || string(*post.contents) == "" {
		t.Error("want rendered HTML")
	}
}

func TestGoldmarkConvert_SetsPlainText(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{}
	raw := []byte("---\ntitle: Plain Test\n---\n\n# Hello\n\nWorld.\n")
	if err := fullConvert(c, post, raw); err != nil {
		t.Fatalf("fullConvert: %v", err)
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

func TestGoldmarkConvert_AutoHeadingID(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{}
	raw := []byte("# My Heading\n")
	if err := fullConvert(c, post, raw); err != nil {
		t.Fatalf("fullConvert: %v", err)
	}
	contents := string(*post.contents)
	if !strings.Contains(contents, `id="`) {
		t.Errorf("want auto heading ID in HTML, got:\n%s", contents)
	}
}

func TestGoldmarkConvert_SetsHash(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{}
	raw := []byte("# Hello\n")
	if err := fullConvert(c, post, raw); err != nil {
		t.Fatalf("fullConvert: %v", err)
	}
	if post.hash == 0 {
		t.Error("want non-zero hash")
	}
}

// --- GoldmarkConverter link rewriting ---

func TestGoldmarkConvert_RelativeLinksRewritten(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{slug: "more/about"}
	raw := []byte("# Page\n\n[Example](./example.md)\n")
	if err := fullConvert(c, post, raw); err != nil {
		t.Fatalf("fullConvert: %v", err)
	}
	html := string(*post.contents)
	if !strings.Contains(html, `href="/more/example"`) {
		t.Errorf("want href=/more/example in HTML, got:\n%s", html)
	}
	if strings.Contains(html, "example.md") {
		t.Errorf("bare .md href should have been rewritten, got:\n%s", html)
	}
}

func TestGoldmarkConvert_RelativeLinksRewritten_TopLevelPost(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{slug: "about"}
	raw := []byte("[Go](./other.md)\n")
	if err := fullConvert(c, post, raw); err != nil {
		t.Fatalf("fullConvert: %v", err)
	}
	html := string(*post.contents)
	if !strings.Contains(html, `href="/other"`) {
		t.Errorf("want href=/other, got:\n%s", html)
	}
}

func TestGoldmarkConvert_AbsoluteLinksUnchanged(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{slug: "more/about"}
	raw := []byte("[External](https://example.com) [Anchor](#section) [Root](/hello)\n")
	if err := fullConvert(c, post, raw); err != nil {
		t.Fatalf("fullConvert: %v", err)
	}
	html := string(*post.contents)
	if !strings.Contains(html, `href="https://example.com"`) {
		t.Errorf("external link changed: %s", html)
	}
	if !strings.Contains(html, `href="#section"`) {
		t.Errorf("anchor link changed: %s", html)
	}
	if !strings.Contains(html, `href="/hello"`) {
		t.Errorf("absolute path link changed: %s", html)
	}
}

func TestGoldmarkConvert_RelativeImageRewrittenToMedia(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{slug: "more/about"}
	raw := []byte("# Page\n\n![Alt](./funny.png)\n")
	if err := fullConvert(c, post, raw); err != nil {
		t.Fatalf("fullConvert: %v", err)
	}
	html := string(*post.contents)
	if !strings.Contains(html, `src="/media/more/funny.png"`) {
		t.Errorf("want src=/media/more/funny.png in HTML, got:\n%s", html)
	}
}

func TestGoldmarkConvert_RelativeImageTopLevelPost(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{slug: "about"}
	raw := []byte("![Banner](./banner.jpg)\n")
	if err := fullConvert(c, post, raw); err != nil {
		t.Fatalf("fullConvert: %v", err)
	}
	html := string(*post.contents)
	if !strings.Contains(html, `src="/media/banner.jpg"`) {
		t.Errorf("want src=/media/banner.jpg in HTML, got:\n%s", html)
	}
}

// --- goldmarkRewriteDest unit tests ---

func TestGoldmarkRewriteDest_AbsoluteURLUnchanged(t *testing.T) {
	cases := []string{
		"https://example.com/page",
		"http://example.com/",
		"/absolute/path",
		"#anchor",
		"mailto:user@example.com",
	}
	for _, c := range cases {
		got := string(goldmarkRewriteDest([]byte(c), "/more/", "/media/more/"))
		if got != c {
			t.Errorf("goldmarkRewriteDest(%q): want unchanged, got %q", c, got)
		}
	}
}

func TestGoldmarkRewriteDest_SimpleRelative(t *testing.T) {
	cases := []struct {
		dest     string
		postsDir string
		want     string
	}{
		{"./example.md", "/more/", "/more/example"},
		{"example.md", "/more/", "/more/example"},
		{"../other.md", "/more/", "/other"},
		{"./example.md", "/", "/example"},
		{"sub/page.md", "/", "/sub/page"},
		{"./notes", "/more/", "/more/notes"},
	}
	for _, tc := range cases {
		got := string(goldmarkRewriteDest([]byte(tc.dest), tc.postsDir, "/media/more/"))
		if got != tc.want {
			t.Errorf("goldmarkRewriteDest(%q, %q): want %q, got %q", tc.dest, tc.postsDir, tc.want, got)
		}
	}
}

func TestGoldmarkRewriteDest_WithFragment(t *testing.T) {
	got := string(goldmarkRewriteDest([]byte("./example.md#section"), "/more/", "/media/more/"))
	if got != "/more/example#section" {
		t.Errorf("want /more/example#section, got %q", got)
	}
}

func TestGoldmarkRewriteDest_WithQuery(t *testing.T) {
	got := string(goldmarkRewriteDest([]byte("./example.md?foo=bar"), "/more/", "/media/more/"))
	if got != "/more/example?foo=bar" {
		t.Errorf("want /more/example?foo=bar, got %q", got)
	}
}

func TestGoldmarkRewriteDest_MediaFiles(t *testing.T) {
	cases := []struct {
		desc     string
		dest     string
		postsDir string
		mediaDir string
		want     string
	}{
		{"png in subdir", "./funny.png", "/more/", "/media/more/", "/media/more/funny.png"},
		{"jpg at root", "./photo.jpg", "/", "/media/", "/media/photo.jpg"},
		{"parent dir traversal", "../banner.gif", "/more/", "/media/more/", "/media/banner.gif"},
		{"nested media", "assets/image.svg", "/more/", "/media/more/", "/media/more/assets/image.svg"},
		{"media with fragment", "./img.png#L1", "/", "/media/", "/media/img.png#L1"},
		{"video", "./demo.mp4", "/more/", "/media/more/", "/media/more/demo.mp4"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := string(goldmarkRewriteDest([]byte(tc.dest), tc.postsDir, tc.mediaDir))
			if got != tc.want {
				t.Errorf("want %q, got %q", tc.want, got)
			}
		})
	}
}
