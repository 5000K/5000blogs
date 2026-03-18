package service

import (
	"bytes"
	"fmt"
	"hash/fnv"
	stdhtml "html"
	"strings"

	"gopkg.in/yaml.v3"
)

// AssetResolver provides resolution callbacks used during wiki-link conversion.
// A nil AssetResolver is valid; all methods behave as if nothing is found.
type AssetResolver interface {
	// ResolveSlugByTitle maps a post title to its URL slug.
	// Returns "" when not found.
	ResolveSlugByTitle(title string) string
	// ResolveAssetByFilename returns the URL for a media asset found by filename.
	// Returns "" when not found.
	ResolveAssetByFilename(filename string) string
}

// Converter parses raw post bytes into metadata and rendered HTML on a Post.
// The two methods are intended to be called in sequence:
//
//  1. ExtractMetadata sets post.metadata and post.hash, and returns the markdown body stripped of front matter.
//  2. Convert renders that body to HTML and sets post.contents and post.plainText.
type Converter interface {
	ExtractMetadata(post *Post, raw []byte) (body []byte, err error)
	Convert(post *Post, body []byte, resolver AssetResolver) error
}

// blockElements are HTML tags that represent block boundaries and map to newlines in plain text.
var blockElements = []string{"p", "br", "h1", "h2", "h3", "h4", "h5", "h6", "li", "blockquote", "pre", "div", "tr", "hr"}

// htmlToPlainText strips HTML tags from src, inserting newlines at block boundaries,
// then unescapes HTML entities and normalises whitespace.
func htmlToPlainText(src []byte) []byte {
	var buf bytes.Buffer
	i := 0
	for i < len(src) {
		if src[i] != '<' {
			buf.WriteByte(src[i])
			i++
			continue
		}
		end := bytes.IndexByte(src[i:], '>')
		if end == -1 {
			buf.Write(src[i:])
			break
		}
		inner := src[i+1 : i+end] // everything between < and >
		if len(inner) > 0 && inner[0] == '/' {
			inner = inner[1:] // strip leading /
		}
		tagName := strings.ToLower(string(inner))
		if sp := strings.IndexByte(tagName, ' '); sp != -1 {
			tagName = tagName[:sp]
		}
		for _, bt := range blockElements {
			if tagName == bt {
				buf.WriteByte('\n')
				break
			}
		}
		i += end + 1
	}
	text := stdhtml.UnescapeString(buf.String())
	// Collapse runs of 3+ newlines to a single blank line.
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	return []byte(strings.TrimSpace(text))
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
