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

// wikilinkInlineParser implements parser.InlineParser for [[Title]] links.
// It triggers on '[' and only claims the input when it sees '[['.
// Priority is set above goldmark's standard link parser (200) so wiki-links are
// tested first; a plain '[' returns nil and lets the standard parser continue.
type wikilinkInlineParser struct {
	postsBase          string
	resolveSlugByTitle func(string) string
}

func (p *wikilinkInlineParser) Trigger() []byte { return []byte{'['} }

func (p *wikilinkInlineParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if len(line) < 5 || line[0] != '[' || line[1] != '[' {
		return nil
	}
	rest := string(line[2:])
	end := strings.Index(rest, "]]")
	if end <= 0 {
		return nil
	}
	title := rest[:end]
	block.Advance(2 + end + 2) // consume [[title]]
	return &WikiLinkNode{Title: title, Href: p.resolveHref(title)}
}

func (p *wikilinkInlineParser) resolveHref(title string) string {
	if p.resolveSlugByTitle != nil {
		if slug := p.resolveSlugByTitle(title); slug != "" {
			return p.postsBase + slug
		}
	}
	return "/" + url.PathEscape(title)
}

// wikilinkNodeRenderer renders WikiLinkNode as an HTML anchor.
type wikilinkNodeRenderer struct{}

func (r *wikilinkNodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(WikiLinkNodeKind, r.render)
}

func (r *wikilinkNodeRenderer) render(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
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

// WikiLinkExtension is a goldmark.Extender that adds [[Title]] wiki-link syntax.
type WikiLinkExtension struct {
	postsBase          string
	resolveSlugByTitle func(string) string
}

func (e *WikiLinkExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&wikilinkInlineParser{
			postsBase:          e.postsBase,
			resolveSlugByTitle: e.resolveSlugByTitle,
		}, 199),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&wikilinkNodeRenderer{}, 199),
	))
}
