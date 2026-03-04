package service

import (
	"5000blogs/config"
	"bytes"
	"fmt"
	"hash/fnv"
	"net/http"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/robfig/cron/v3"
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
	hash     uint64
	metadata *Metadata
	contents *[]byte
}

type Service struct {
	conf   *config.Config
	source PostSource

	posts     []*Post
	scheduler *cron.Cron
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

func NewService(conf *config.Config) *Service {
	return &Service{
		conf:   conf,
		source: NewFileSystemSource(conf.Paths.Posts),
	}
}

// Start performs an initial rescan and then schedules periodic rescans
// according to the cron expression in the config. Call Stop to release resources.
func (s *Service) Start() error {
	s.rescan()

	s.scheduler = cron.New()
	_, err := s.scheduler.AddFunc(s.conf.RescanCron, s.rescan)
	if err != nil {
		return fmt.Errorf("service.Start: invalid rescan cron expression %q: %w", s.conf.RescanCron, err)
	}
	s.scheduler.Start()
	return nil
}

// Stop gracefully shuts down the rescan scheduler.
func (s *Service) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}

func (s *Service) hasPost(path string) bool {
	for _, post := range s.posts {
		if post.path == path {
			return true
		}
	}
	return false
}

func (s *Service) addPost(path string) {
	post := &Post{path: path}
	if err := s.renderPost(post); err != nil {
		// todo: log error
	}
	s.posts = append(s.posts, post)
}

func (s *Service) removePost(path string) {
	for i, post := range s.posts {
		if post.path == path {
			s.posts = append(s.posts[:i], s.posts[i+1:]...)
			return
		}
	}
}

func (s *Service) updatePost(path string) {
	for _, post := range s.posts {
		if post.path == path {
			buf, err := s.source.ReadPost(path)
			if err != nil {
				// todo: log error
				return
			}
			if hashBytes(buf) == post.hash {
				return // file unchanged, no re-render needed
			}
			if err := parseAndRender(post, buf); err != nil {
				// todo: log error
			}
			return
		}
	}
}

func (s *Service) rescan() {
	paths, err := s.source.ListPosts()
	if err != nil {
		// todo: log error
		return
	}

	// Track which paths exist on disk during this scan.
	found := make(map[string]bool)
	for _, path := range paths {
		found[path] = true
		if s.hasPost(path) {
			s.updatePost(path)
		} else {
			s.addPost(path)
		}
	}

	// Remove posts that are no longer present on disk.
	var toRemove []string
	for _, post := range s.posts {
		if !found[post.path] {
			toRemove = append(toRemove, post.path)
		}
	}
	for _, path := range toRemove {
		s.removePost(path)
	}
}

func (s *Service) GetPost(path string) *Post {
	for _, post := range s.posts {
		if post.path == path {
			return post
		}
	}

	return nil
}

func hashBytes(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}

// parseAndRender parses raw markdown bytes, extracts metadata, renders HTML,
// and stores the hash, metadata and rendered contents on the post.
func parseAndRender(post *Post, buf []byte) error {
	post.hash = hashBytes(buf)

	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(buf)

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

// renderPost reads the file at post.path via the source and calls parseAndRender.
func (s *Service) renderPost(post *Post) error {
	buf, err := s.source.ReadPost(post.path)
	if err != nil {
		return err
	}
	return parseAndRender(post, buf)
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
