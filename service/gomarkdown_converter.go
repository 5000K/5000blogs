package service

import (
	"path"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// GoMarkdownConverter implements Converter using gomarkdown.
// PostsBase is the URL prefix for posts (default "/posts/").
type GoMarkdownConverter struct {
	PostsBase string
}

func (c *GoMarkdownConverter) postsBase() string {
	if c.PostsBase == "" {
		return "/posts/"
	}
	return c.PostsBase
}

func (c *GoMarkdownConverter) Convert(post *Post, raw []byte) error {
	post.hash = hashBytes(raw)

	metadata, body, err := extractFrontmatter(raw)
	if err != nil {
		return err
	}
	post.metadata = metadata

	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(body)

	rewriteRelativeLinks(doc, post.slug, c.postsBase())

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	rendered := markdown.Render(doc, renderer)
	post.contents = &rendered
	plain := htmlToPlainText(rendered)
	post.plainText = &plain
	return nil
}

// rewriteRelativeLinks walks the parsed markdown AST and rewrites relative
// link destinations so they are absolute paths rooted at postsBase or /media/.
//
// For a post with slug "more+about" (i.e. the file more/about.md) and
// postsBase "/posts/", the link-directory is "/posts/more/" and the
// media-directory is "/media/more/".
//
//   - Relative links to .md files (or extensionless paths) are resolved
//     under postsBase, with the .md suffix stripped.
//   - Relative links to any other extension (.png, .jpg, .mp4, etc.) are
//     resolved under /media/, so browsers can fetch them via the media route.
//
// Absolute URLs (starting with "/" or containing "://"), anchor-only links
// ("#…"), and scheme links ("mailto:", etc.) are left unchanged.
func rewriteRelativeLinks(doc ast.Node, slug string, postsBase string) {
	// Build the URL directories for this post.
	// slug "more+about" → parts ["more","about"] → subdir "more/"
	// slug "about"      → parts ["about"]        → subdir ""
	parts := strings.Split(slug, "+")
	subdir := ""
	if len(parts) > 1 {
		subdir = strings.Join(parts[:len(parts)-1], "/") + "/"
	}
	postsDir := postsBase + subdir
	mediaDir := "/media/" + subdir

	ast.Walk(doc, &linkRewriter{postsDir: postsDir, mediaDir: mediaDir})
}

// linkRewriter implements ast.NodeVisitor to rewrite relative link/image destinations.
type linkRewriter struct {
	postsDir string
	mediaDir string
}

func (r *linkRewriter) Visit(node ast.Node, entering bool) ast.WalkStatus {
	if !entering {
		return ast.GoToNext
	}
	switch n := node.(type) {
	case *ast.Link:
		n.Destination = rewriteRelativeDest(n.Destination, r.postsDir, r.mediaDir)
	case *ast.Image:
		n.Destination = rewriteRelativeDest(n.Destination, r.postsDir, r.mediaDir)
	}
	return ast.GoToNext
}

// rewriteRelativeDest resolves a single link destination relative to postsDir
// or mediaDir depending on the file extension, stripping any trailing ".md" suffix.
//
// Links with no extension or a .md extension are treated as post links and
// resolved under postsDir. Links with any other extension (e.g. .png, .mp4)
// are treated as media and resolved under mediaDir so they map to /media/….
func rewriteRelativeDest(dest []byte, postsDir, mediaDir string) []byte {
	s := string(dest)
	// Leave absolute URLs, absolute paths, anchor-only, and scheme links.
	if strings.HasPrefix(s, "/") || strings.Contains(s, "://") ||
		strings.HasPrefix(s, "#") || strings.Contains(s, ":") {
		return dest
	}

	// Separate fragment and query string before resolving the path.
	fragment := ""
	if idx := strings.IndexByte(s, '#'); idx != -1 {
		fragment = s[idx:]
		s = s[:idx]
	}
	query := ""
	if idx := strings.IndexByte(s, '?'); idx != -1 {
		query = s[idx:]
		s = s[:idx]
	}

	if s == "" {
		// Anchor-only or empty — leave as-is.
		return dest
	}

	ext := path.Ext(s)
	if ext == "" || ext == ".md" {
		// Post link: resolve under postsDir and strip the .md extension.
		resolved := path.Join(postsDir, s)
		resolved = strings.TrimSuffix(resolved, ".md")
		return []byte(resolved + query + fragment)
	}
	// Media link: resolve under mediaDir; keep extension as-is.
	resolved := path.Join(mediaDir, s)
	return []byte(resolved + query + fragment)
}
