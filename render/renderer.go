package render

import (
	"fmt"
	"html"
	"strings"
)

type Renderer struct {
	wikiResolver WikiLinkResolver
	builder      strings.Builder
}

type WikiLinkResolver interface {
	Resolve(title string) (url string, ok bool)
}

// WikiLinkResolverFunc is an adapter to allow the use of
// ordinary functions as WikiLink resolvers.
type WikiLinkResolverFunc func(title string) (url string, ok bool)

// Resolve calls f(title)
func (f WikiLinkResolverFunc) Resolve(title string) (string, bool) {
	return f(title)
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func (r *Renderer) SetWikiLinkResolver(resolver WikiLinkResolver) {
	r.wikiResolver = resolver
}

func (r *Renderer) Render(node *Node) (string, error) {
	r.builder.Reset()
	if err := r.renderNode(node); err != nil {
		return "", err
	}
	return r.builder.String(), nil
}

func (r *Renderer) renderNode(node *Node) error {
	switch node.Type {
	case NodeDocument:
		for _, child := range node.Children {
			if err := r.renderNode(child); err != nil {
				return err
			}
		}
	case NodeFrontmatter:
		// not rendered as html right now, but maybe later? e.g. include styling or scripts?
		r.builder.WriteString("<!-- Frontmatter -->\n")
	case NodeParagraph:
		r.builder.WriteString("<p>")
		for _, child := range node.Children {
			if err := r.renderNode(child); err != nil {
				return err
			}
		}
		r.builder.WriteString("</p>\n")
	case NodeHeader:
		tag := fmt.Sprintf("h%d", node.Level)
		r.builder.WriteString(fmt.Sprintf("<%s>", tag))
		for _, child := range node.Children {
			if err := r.renderNode(child); err != nil {
				return err
			}
		}
		r.builder.WriteString(fmt.Sprintf("</%s>\n", tag))
	case NodeBold:
		r.builder.WriteString("<strong>")
		for _, child := range node.Children {
			if err := r.renderNode(child); err != nil {
				return err
			}
		}
		r.builder.WriteString("</strong>")
	case NodeItalic:
		r.builder.WriteString("<em>")
		for _, child := range node.Children {
			if err := r.renderNode(child); err != nil {
				return err
			}
		}
		r.builder.WriteString("</em>")
	case NodeLink:
		r.builder.WriteString(fmt.Sprintf(`<a href="%s">`, html.EscapeString(node.Destination)))
		for _, child := range node.Children {
			if err := r.renderNode(child); err != nil {
				return err
			}
		}
		r.builder.WriteString("</a>")
	case NodeWikiLink:
		url := "#"
		if r.wikiResolver != nil {
			if u, ok := r.wikiResolver.Resolve(node.Literal); ok {
				url = u
			}
		}
		r.builder.WriteString(fmt.Sprintf(`<a href="%s" class="wikilink">%s</a>`,
			html.EscapeString(url), html.EscapeString(node.Literal)))
	case NodeCode:
		r.builder.WriteString("<code>")
		r.builder.WriteString(html.EscapeString(node.Literal))
		r.builder.WriteString("</code>")
	case NodeCodeBlock:
		lang := node.Attributes["lang"]
		if lang != "" {
			r.builder.WriteString(fmt.Sprintf(`<pre><code class="language-%s">`, html.EscapeString(lang)))
		} else {
			r.builder.WriteString("<pre><code>")
		}
		r.builder.WriteString(html.EscapeString(node.Literal))
		r.builder.WriteString("</code></pre>\n")
	case NodeText:
		r.builder.WriteString(html.EscapeString(node.Literal))
	case NodeSoftBreak:
		r.builder.WriteString("\n")
	default:
		return fmt.Errorf("unknown node type: %s", node.Type)
	}
	return nil
}
