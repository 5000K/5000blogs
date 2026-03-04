package service

import (
	"bytes"
	"fmt"
	"hash/fnv"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"gopkg.in/yaml.v3"
)

// Converter parses raw post bytes into metadata and rendered HTML on a Post.
type Converter interface {
	Convert(post *Post, raw []byte) error
}

// GoMarkdownConverter implements Converter using gomarkdown.
type GoMarkdownConverter struct{}

func (c *GoMarkdownConverter) Convert(post *Post, raw []byte) error {
	post.hash = hashBytes(raw)

	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(raw)

	metadata, err := extractMetadata(doc)
	if err != nil {
		return err
	}
	post.metadata = metadata

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	rendered := markdown.Render(doc, renderer)
	post.contents = &rendered
	return nil
}

func extractMetadata(doc ast.Node) (*Metadata, error) {
	var metaNode *ast.CodeBlock

	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if metaNode != nil {
			return ast.Terminate
		}
		cb, ok := node.(*ast.CodeBlock)
		if !ok || !entering {
			return ast.GoToNext
		}
		if bytes.EqualFold(cb.Info, []byte("yaml")) {
			metaNode = cb
			return ast.Terminate
		}
		return ast.GoToNext
	})

	if metaNode == nil {
		return &Metadata{}, nil
	}

	var meta Metadata
	if err := yaml.Unmarshal(metaNode.Literal, &meta); err != nil {
		return nil, fmt.Errorf("extractMetadata: failed to parse yaml: %w", err)
	}

	ast.RemoveFromTree(metaNode)

	return &meta, nil
}

func hashBytes(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}
