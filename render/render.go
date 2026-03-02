package render

import (
	"fmt"
)

// Document represents a parsed blog post
type Document struct {
	Frontmatter *Frontmatter
	Root        *Node
	RawContent  string
}

// Engine is the main entry point
type Engine struct {
	renderer *Renderer
}

func NewEngine() *Engine {
	return &Engine{
		renderer: NewRenderer(),
	}
}

func (e *Engine) SetWikiLinkResolver(resolver WikiLinkResolver) {
	e.renderer.SetWikiLinkResolver(resolver)
}

// Parse parses markdown input into a Document
func (e *Engine) Parse(input string) (*Document, error) {
	// Extract frontmatter
	fm, content, err := ExtractFrontmatter(input)
	if err != nil {
		return nil, err
	}

	lexer := NewLexer(content)
	parser := NewParser(lexer)
	root, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &Document{
		Frontmatter: fm,
		Root:        root,
		RawContent:  content,
	}, nil
}

// Render converts a Document to HTML
func (e *Engine) Render(doc *Document) (string, error) {
	return e.renderer.Render(doc.Root)
}

// ParseAndRender is a convenience method
func (e *Engine) ParseAndRender(input string) (string, *Frontmatter, error) {
	doc, err := e.Parse(input)
	if err != nil {
		return "", nil, err
	}

	html, err := e.Render(doc)
	if err != nil {
		return "", nil, err
	}

	return html, doc.Frontmatter, nil
}
