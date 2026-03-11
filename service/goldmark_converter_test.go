package service

import (
	"strings"
	"testing"
)

// --- GoldmarkConverter basic rendering ---

func TestGoldmarkConvert_RendersHTML(t *testing.T) {
	c := &GoldmarkConverter{}
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

func TestGoldmarkConvert_SetsPlainText(t *testing.T) {
	c := &GoldmarkConverter{}
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

func TestGoldmarkConvert_AutoHeadingID(t *testing.T) {
	c := &GoldmarkConverter{}
	post := &Post{}
	raw := []byte("# My Heading\n")
	if err := c.Convert(post, raw); err != nil {
		t.Fatalf("Convert: %v", err)
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
	if err := c.Convert(post, raw); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if post.hash == 0 {
		t.Error("want non-zero hash")
	}
}

// --- GoldmarkConverter link rewriting ---

func TestGoldmarkConvert_RelativeLinksRewritten(t *testing.T) {
	c := &GoldmarkConverter{}
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

func TestGoldmarkConvert_RelativeLinksRewritten_TopLevelPost(t *testing.T) {
	c := &GoldmarkConverter{}
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

func TestGoldmarkConvert_AbsoluteLinksUnchanged(t *testing.T) {
	c := &GoldmarkConverter{}
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

func TestGoldmarkConvert_RelativeImageRewrittenToMedia(t *testing.T) {
	c := &GoldmarkConverter{}
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

func TestGoldmarkConvert_RelativeImageTopLevelPost(t *testing.T) {
	c := &GoldmarkConverter{}
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
		got := string(goldmarkRewriteDest([]byte(c), "/posts/more/", "/media/more/"))
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
		{"./example.md", "/posts/more/", "/posts/more/example"},
		{"example.md", "/posts/more/", "/posts/more/example"},
		{"../other.md", "/posts/more/", "/posts/other"},
		{"./example.md", "/posts/", "/posts/example"},
		{"sub/page.md", "/posts/", "/posts/sub/page"},
		{"./notes", "/posts/more/", "/posts/more/notes"},
	}
	for _, tc := range cases {
		got := string(goldmarkRewriteDest([]byte(tc.dest), tc.postsDir, "/media/more/"))
		if got != tc.want {
			t.Errorf("goldmarkRewriteDest(%q, %q): want %q, got %q", tc.dest, tc.postsDir, tc.want, got)
		}
	}
}

func TestGoldmarkRewriteDest_WithFragment(t *testing.T) {
	got := string(goldmarkRewriteDest([]byte("./example.md#section"), "/posts/more/", "/media/more/"))
	if got != "/posts/more/example#section" {
		t.Errorf("want /posts/more/example#section, got %q", got)
	}
}

func TestGoldmarkRewriteDest_WithQuery(t *testing.T) {
	got := string(goldmarkRewriteDest([]byte("./example.md?foo=bar"), "/posts/more/", "/media/more/"))
	if got != "/posts/more/example?foo=bar" {
		t.Errorf("want /posts/more/example?foo=bar, got %q", got)
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
		{"png in subdir", "./funny.png", "/posts/more/", "/media/more/", "/media/more/funny.png"},
		{"jpg at root", "./photo.jpg", "/posts/", "/media/", "/media/photo.jpg"},
		{"parent dir traversal", "../banner.gif", "/posts/more/", "/media/more/", "/media/banner.gif"},
		{"nested media", "assets/image.svg", "/posts/more/", "/media/more/", "/media/more/assets/image.svg"},
		{"media with fragment", "./img.png#L1", "/posts/", "/media/", "/media/img.png#L1"},
		{"video", "./demo.mp4", "/posts/more/", "/media/more/", "/media/more/demo.mp4"},
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
