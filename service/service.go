package service

import (
	"5000blogs/config"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"gopkg.in/yaml.v3"
)

type Metadata struct {
	Title       string    `yaml:"title"`
	Description string    `yaml:"description"`
	Date        time.Time `yaml:"date"`

	Raw map[string]interface{} `yaml:",inline"`
}

type Post struct {
	path     string
	contents *string
}

type Service struct {
	conf *config.Config

	posts []*Post
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
		if bytes.EqualFold(cb.Info, []byte("meta")) {
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

func render(md []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

func NewService(conf *config.Config) *Service {
	return &Service{
		conf: conf,
	}
}

func (s *Service) GetPosts() []*Post {
	return s.posts
}

func (s *Service) GetPost(path string) *Post {
	for _, post := range s.posts {
		if post.path == path {
			return post
		}
	}

	return nil
}

func renderPost(post *Post) *string {
	if post.contents == nil {
		// read file and render markdown to HTML, then set post.contents

		// read file

		fi, err := os.Open("input.txt")
		if err != nil {
			// todo: log error
			return nil
		}

		defer func(fi *os.File) {
			err := fi.Close()
			if err != nil {
				// todo: log error
			}
		}(fi)

		buf, err := io.ReadAll(fi)
		if err != nil {
			// todo: log error
			return nil
		}

		// render markdown to HTML
		// For now, just return the raw content as a placeholder
		content := string(buf)
		return &content
	}

	return post.contents
}

func (s *Service) ServePost(path string, w http.ResponseWriter) {
	post := s.GetPost(path)

	if post == nil {
		http.NotFound(w, nil)
		return
	}

	if post.contents == nil {
		http.NotFound(w, nil)
		return
	}

	_, _ = w.Write([]byte(*post.contents))
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}
