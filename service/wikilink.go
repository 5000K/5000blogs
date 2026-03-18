package service

import (
	stdhtml "html"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// PostEmbedNodeKind is the goldmark AST node kind for embedded posts.
var PostEmbedNodeKind = ast.NewNodeKind("PostEmbed")

// PostEmbedNode holds the pre-rendered HTML of an embedded post.
type PostEmbedNode struct {
	ast.BaseBlock
	HTML []byte
}

func (n *PostEmbedNode) Kind() ast.NodeKind { return PostEmbedNodeKind }

func (n *PostEmbedNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// WikiLinkNodeKind is the goldmark AST node kind for [[Title]] wiki-links.
var WikiLinkNodeKind = ast.NewNodeKind("WikiLink")

// WikiLinkNode represents a [[Title]] wiki-link in the AST.
type WikiLinkNode struct {
	ast.BaseInline
	Title string
	Href  string // pre-resolved absolute href
}

func (n *WikiLinkNode) Kind() ast.NodeKind { return WikiLinkNodeKind }

func (n *WikiLinkNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Title": n.Title, "Href": n.Href}, nil)
}

// WikiImageNodeKind is the goldmark AST node kind for ![[filename]] wiki-images.
var WikiImageNodeKind = ast.NewNodeKind("WikiImage")

// WikiImageNode represents a ![[filename]] wiki-image in the AST.
type WikiImageNode struct {
	ast.BaseInline
	Filename string
	Src      string // pre-resolved absolute URL
}

func (n *WikiImageNode) Kind() ast.NodeKind { return WikiImageNodeKind }

func (n *WikiImageNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Filename": n.Filename, "Src": n.Src}, nil)
}

// wikilinkInlineParser implements parser.InlineParser for [[Title]] links and
// ![[filename]] images. It triggers on '[' and '!' and only claims input when
// the exact patterns are detected, letting the standard parser handle anything else.
type wikilinkInlineParser struct {
	postsBase string
	resolver  AssetResolver
}

func (p *wikilinkInlineParser) Trigger() []byte { return []byte{'!', '['} }

func (p *wikilinkInlineParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()

	// ![[filename]] → post embed (if title resolves to a post) or wiki image
	if len(line) >= 6 && line[0] == '!' && line[1] == '[' && line[2] == '[' {
		rest := string(line[3:])
		end := strings.Index(rest, "]]")
		if end <= 0 {
			return nil
		}
		filename := rest[:end]
		block.Advance(3 + end + 2) // consume ![[filename]]
		if p.resolver != nil {
			if slug := p.resolver.ResolveSlugByTitle(filename); slug != "" {
				if html := p.resolver.ResolveEmbedBySlug(slug); html != nil {
					return &PostEmbedNode{HTML: html}
				}
			}
		}
		return &WikiImageNode{Filename: filename, Src: p.resolveImageSrc(filename)}
	}

	// [[Title]] → wiki link
	if len(line) >= 5 && line[0] == '[' && line[1] == '[' {
		rest := string(line[2:])
		end := strings.Index(rest, "]]")
		if end <= 0 {
			return nil
		}
		title := rest[:end]
		block.Advance(2 + end + 2) // consume [[title]]
		return &WikiLinkNode{Title: title, Href: p.resolveHref(title)}
	}

	return nil
}

func (p *wikilinkInlineParser) resolveHref(title string) string {
	if p.resolver != nil {
		if slug := p.resolver.ResolveSlugByTitle(title); slug != "" {
			return p.postsBase + slug
		}
	}
	return "/" + url.PathEscape(title)
}

func (p *wikilinkInlineParser) resolveImageSrc(filename string) string {
	if p.resolver != nil {
		if u := p.resolver.ResolveAssetByFilename(filename); u != "" {
			return u
		}
	}
	return "/media/" + url.PathEscape(filename)
}

// wikilinkNodeRenderer renders WikiLinkNode, WikiImageNode, and PostEmbedNode.
type wikilinkNodeRenderer struct{}

func (r *wikilinkNodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(WikiLinkNodeKind, r.renderLink)
	reg.Register(WikiImageNodeKind, r.renderImage)
	reg.Register(PostEmbedNodeKind, r.renderEmbed)
}

func (r *wikilinkNodeRenderer) renderEmbed(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	_, _ = w.Write(n.(*PostEmbedNode).HTML)
	return ast.WalkSkipChildren, nil
}

func (r *wikilinkNodeRenderer) renderLink(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	node := n.(*WikiLinkNode)
	_, _ = w.WriteString(`<a href="`)
	_, _ = w.WriteString(stdhtml.EscapeString(node.Href))
	_, _ = w.WriteString(`">`)
	_, _ = w.WriteString(stdhtml.EscapeString(node.Title))
	_, _ = w.WriteString(`</a>`)
	return ast.WalkContinue, nil
}

func (r *wikilinkNodeRenderer) renderImage(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	node := n.(*WikiImageNode)
	_, _ = w.WriteString(`<img src="`)
	_, _ = w.WriteString(stdhtml.EscapeString(node.Src))
	_, _ = w.WriteString(`" alt="`)
	_, _ = w.WriteString(stdhtml.EscapeString(node.Filename))
	_, _ = w.WriteString(`">`)
	return ast.WalkContinue, nil
}

// WikiLinkExtension is a goldmark.Extender that adds [[Title]] and ![[file]] syntax.
type WikiLinkExtension struct {
	postsBase string
	resolver  AssetResolver
}

func (e *WikiLinkExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&wikilinkInlineParser{
			postsBase: e.postsBase,
			resolver:  e.resolver,
		}, 199),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&wikilinkNodeRenderer{}, 199),
	))
}
