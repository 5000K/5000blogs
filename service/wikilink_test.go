package service

import (
	"5000blogs/config"
	"strings"
	"testing"
)

// wikiConvert runs a full convert with wiki-links enabled and a custom resolver.
func wikiConvert(t *testing.T, post *Post, raw []byte, resolver func(string) string) {
	t.Helper()
	c := &GoldmarkConverter{Features: config.Features{WikiLinks: true}}
	body, err := c.ExtractMetadata(post, raw)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if err := c.Convert(post, body, resolver); err != nil {
		t.Fatalf("Convert: %v", err)
	}
}

// --- basic resolution ---

func TestWikiLink_KnownTitle_RendersAnchor(t *testing.T) {
	resolver := func(title string) string {
		if title == "Another Example" {
			return "example"
		}
		return ""
	}
	post := &Post{slug: "about"}
	wikiConvert(t, post, []byte("See [[Another Example]] for details.\n"), resolver)

	html := string(*post.contents)
	if !strings.Contains(html, `href="/example"`) {
		t.Errorf("want href=/example, got:\n%s", html)
	}
	if !strings.Contains(html, `>Another Example<`) {
		t.Errorf("want link text 'Another Example', got:\n%s", html)
	}
}

func TestWikiLink_UnknownTitle_FallsBackToURLEncodedHref(t *testing.T) {
	post := &Post{slug: "about"}
	wikiConvert(t, post, []byte("See [[No Such Post]] here.\n"), func(string) string { return "" })

	html := string(*post.contents)
	if !strings.Contains(html, `href="/No%20Such%20Post"`) {
		t.Errorf("want href=/No%%20Such%%20Post, got:\n%s", html)
	}
	if !strings.Contains(html, `>No Such Post<`) {
		t.Errorf("want link text in output, got:\n%s", html)
	}
}

func TestWikiLink_NilResolver_FallsBackToURLEncodedHref(t *testing.T) {
	post := &Post{slug: "about"}
	wikiConvert(t, post, []byte("[[Hello World]]\n"), nil)

	html := string(*post.contents)
	if !strings.Contains(html, `href="/Hello%20World"`) {
		t.Errorf("want href=/Hello%%20World, got:\n%s", html)
	}
}

// --- nested slug (slug with +) ---

func TestWikiLink_NestedSlug_RewritesSlashSeparator(t *testing.T) {
	resolver := func(title string) string {
		if title == "About Me" {
			return "more/about"
		}
		return ""
	}
	post := &Post{slug: "index"}
	wikiConvert(t, post, []byte("[[About Me]]\n"), resolver)

	html := string(*post.contents)
	if !strings.Contains(html, `href="/more/about"`) {
		t.Errorf("want href=/more/about, got:\n%s", html)
	}
}

// --- multiple wiki-links ---

func TestWikiLink_Multiple_AllResolved(t *testing.T) {
	slugs := map[string]string{
		"First Post":  "first",
		"Second Post": "second",
	}
	resolver := func(title string) string { return slugs[title] }
	post := &Post{}
	wikiConvert(t, post, []byte("[[First Post]] and [[Second Post]].\n"), resolver)

	html := string(*post.contents)
	if !strings.Contains(html, `href="/first"`) {
		t.Errorf("want first post link, got:\n%s", html)
	}
	if !strings.Contains(html, `href="/second"`) {
		t.Errorf("want second post link, got:\n%s", html)
	}
}

// --- feature flag ---

func TestWikiLink_FeatureDisabled_LeftAsLiteral(t *testing.T) {
	c := &GoldmarkConverter{} // WikiLinks: false (default)
	post := &Post{}
	raw := []byte("[[Some Title]]\n")
	body, err := c.ExtractMetadata(post, raw)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if err := c.Convert(post, body, func(string) string { return "some-title" }); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	html := string(*post.contents)
	if strings.Contains(html, `<a href=`) {
		t.Errorf("wiki-links should not be parsed when feature is disabled, got:\n%s", html)
	}
}

// --- coexistence with regular markdown links ---

func TestWikiLink_CoexistsWithRegularLinks(t *testing.T) {
	resolver := func(title string) string {
		if title == "Wiki Page" {
			return "wiki-page"
		}
		return ""
	}
	post := &Post{}
	wikiConvert(t, post, []byte("[Regular](https://example.com) and [[Wiki Page]].\n"), resolver)

	html := string(*post.contents)
	if !strings.Contains(html, `href="https://example.com"`) {
		t.Errorf("regular link should be preserved, got:\n%s", html)
	}
	if !strings.Contains(html, `href="/wiki-page"`) {
		t.Errorf("wiki-link should be resolved, got:\n%s", html)
	}
}

// --- custom postsBase ---

func TestWikiLink_CustomPostsBase_UsedInHref(t *testing.T) {
	resolver := func(title string) string {
		if title == "Notes" {
			return "notes"
		}
		return ""
	}
	c := &GoldmarkConverter{
		PostsBase: "/articles/",
		Features:  config.Features{WikiLinks: true},
	}
	post := &Post{}
	body, err := c.ExtractMetadata(post, []byte("[[Notes]]\n"))
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if err := c.Convert(post, body, resolver); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	html := string(*post.contents)
	if !strings.Contains(html, `href="/articles/notes"`) {
		t.Errorf("want href=/articles/notes, got:\n%s", html)
	}
}
