package service

import (
	"bytes"
	"path"
	"strings"

	"5000blogs/config"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// GoldmarkConverter implements Converter using goldmark.
// PostsBase is the URL prefix for posts (default "/").
type GoldmarkConverter struct {
	PostsBase string
	Features  config.Features
}

func NewGoldmarkConverter(postsBase string, features config.Features) *GoldmarkConverter {
	return &GoldmarkConverter{PostsBase: postsBase, Features: features}
}

func (c *GoldmarkConverter) postsBase() string {
	if c.PostsBase == "" {
		return "/"
	}
	return c.PostsBase
}

func (c *GoldmarkConverter) ExtractMetadata(post *Post, raw []byte) ([]byte, error) {
	post.hash = hashBytes(raw)
	metadata, body, err := extractFrontmatter(raw)
	if err != nil {
		return nil, err
	}
	post.metadata = metadata
	return body, nil
}

func (c *GoldmarkConverter) Convert(post *Post, body []byte, resolveSlugByTitle func(string) string) error {
	opts := []goldmark.Option{
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(
				util.Prioritized(&goldmarkLinkRewriter{
					slug:      post.slug,
					postsBase: c.postsBase(),
					source:    body,
				}, 100),
			),
		),
	}
	if c.Features.WikiLinks {
		opts = append(opts, goldmark.WithExtensions(&WikiLinkExtension{
			postsBase:          c.postsBase(),
			resolveSlugByTitle: resolveSlugByTitle,
		}))
	}

	if c.Features.Tables {
		opts = append(opts, goldmark.WithExtensions(extension.Table))
	}

	if c.Features.Strikethrough {
		opts = append(opts, goldmark.WithExtensions(extension.Strikethrough))
	}
	
	if c.Features.Autolinks {
		opts = append(opts, goldmark.WithExtensions(extension.Linkify))
	}

	if c.Features.TaskList {
		opts = append(opts, goldmark.WithExtensions(extension.TaskList))
	}
	
	if c.Features.Footnotes {
		opts = append(opts, goldmark.WithExtensions(extension.Footnote))
	}

	md := goldmark.New(opts...)

	var buf bytes.Buffer
	if err := md.Convert(body, &buf); err != nil {
		return err
	}

	rendered := buf.Bytes()
	post.contents = &rendered
	plain := htmlToPlainText(rendered)
	post.plainText = &plain
	return nil
}

// goldmarkLinkRewriter is a goldmark AST transformer that rewrites relative
// link/image destinations, matching the logic of GoMarkdownConverter.
type goldmarkLinkRewriter struct {
	slug      string
	postsBase string
	source    []byte
}

func (t *goldmarkLinkRewriter) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	parts := strings.Split(t.slug, "/")
	subdir := ""
	if len(parts) > 1 {
		subdir = strings.Join(parts[:len(parts)-1], "/") + "/"
	}
	postsDir := t.postsBase + subdir
	mediaDir := "/media/" + subdir

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.Kind() {
		case ast.KindLink:
			link := n.(*ast.Link)
			link.Destination = goldmarkRewriteDest(link.Destination, postsDir, mediaDir)
		case ast.KindImage:
			img := n.(*ast.Image)
			img.Destination = goldmarkRewriteDest(img.Destination, postsDir, mediaDir)
		}
		return ast.WalkContinue, nil
	})
}

// goldmarkRewriteDest rewrites a relative link destination to an absolute one,
// using the same rules as rewriteRelativeDest.
func goldmarkRewriteDest(dest []byte, postsDir, mediaDir string) []byte {
	s := string(dest)

	if strings.HasPrefix(s, "/") || strings.Contains(s, "://") ||
		strings.HasPrefix(s, "#") || strings.Contains(s, ":") {
		return dest
	}

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
		return dest
	}

	ext := path.Ext(s)
	if ext == "" || ext == ".md" {
		resolved := path.Join(postsDir, s)
		resolved = strings.TrimSuffix(resolved, ".md")
		return []byte(resolved + query + fragment)
	}
	resolved := path.Join(mediaDir, s)
	return []byte(resolved + query + fragment)
}
