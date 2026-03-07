package service

import (
	"bytes"
	"fmt"
	"hash/fnv"

	"github.com/gomarkdown/markdown"
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

	metadata, body, err := extractFrontmatter(raw)
	if err != nil {
		return err
	}
	post.metadata = metadata

	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(body)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	rendered := markdown.Render(doc, renderer)
	post.contents = &rendered
	return nil
}

// extractFrontmatter parses --- delimited YAML front matter from raw markdown.
// Returns metadata, the remaining markdown body, and any error.
func extractFrontmatter(raw []byte) (*Metadata, []byte, error) {
	const open = "---\n"
	const close = "\n---\n"

	if !bytes.HasPrefix(raw, []byte(open)) {
		return &Metadata{}, raw, nil
	}

	rest := raw[len(open):]

	var yamlBytes, body []byte
	if bytes.HasPrefix(rest, []byte("---\n")) {
		// empty front matter block
		body = rest[len("---\n"):]
	} else {
		idx := bytes.Index(rest, []byte(close))
		if idx == -1 {
			return &Metadata{}, raw, nil
		}
		yamlBytes = rest[:idx]
		body = rest[idx+len(close):]
	}

	var meta Metadata
	if err := yaml.Unmarshal(yamlBytes, &meta); err != nil {
		return nil, nil, fmt.Errorf("extractFrontmatter: failed to parse yaml: %w", err)
	}

	return &meta, body, nil
}

func hashBytes(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}
