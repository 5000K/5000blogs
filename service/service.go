package service

import (
	"5000blogs/config"
	"bytes"
	"fmt"
	"hash/fnv"
	"log/slog"
	"path/filepath"
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
	path    string
	hash    uint64
	modTime time.Time

	metadata *Metadata
	contents *[]byte
}

// PostData holds the rendered data for a post, safe to pass to a view layer.
type PostData struct {
	Slug        string
	Title       string
	Description string
	Date        time.Time
	Content     []byte // rendered HTML
}

// Data returns a PostData view of the post.
func (p *Post) Data() PostData {
	d := PostData{
		Slug: slugFromPath(p.path),
	}
	if p.metadata != nil {
		d.Title = p.metadata.Title
		d.Description = p.metadata.Description
		d.Date = p.metadata.Date
	}
	if p.contents != nil {
		d.Content = *p.contents
	}
	return d
}

// PostSummary is a lightweight view of a post for list pages.
type PostSummary struct {
	Slug        string
	Title       string
	Description string
	Date        time.Time
}

// PageResult is the output of GetPage.
type PageResult struct {
	Posts      []PostSummary
	Page       int
	PageSize   int
	TotalPosts int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
}

// GetPage returns a page of post summaries sorted by date descending.
func (s *Service) GetPage(page int) PageResult {
	size := s.conf.PageSize
	if size <= 0 {
		size = 10
	}
	total := s.repo.Count()
	totalPages := (total + size - 1) / size
	if totalPages == 0 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	posts := s.repo.Page(page, size)
	summaries := make([]PostSummary, 0, len(posts))
	for _, p := range posts {
		d := p.Data()
		summaries = append(summaries, PostSummary{
			Slug:        d.Slug,
			Title:       d.Title,
			Description: d.Description,
			Date:        d.Date,
		})
	}
	res := PageResult{
		Posts:      summaries,
		Page:       page,
		PageSize:   size,
		TotalPosts: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
	}
	return res
}

// slugFromPath derives a URL slug from a file path (basename without extension).
func slugFromPath(path string) string {
	base := filepath.Base(path)
	if ext := filepath.Ext(base); ext != "" {
		return base[:len(base)-len(ext)]
	}
	return base
}

type Service struct {
	conf   *config.Config
	source PostSource
	repo   PostRepository
	log    *slog.Logger

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

func NewService(conf *config.Config, logger *slog.Logger) *Service {
	log := logger.With("component", "Service")
	return &Service{
		conf:   conf,
		source: NewFileSystemSource(conf.Paths.Posts, logger),
		repo:   NewMemoryPostRepository(),
		log:    log,
	}
}

// Start performs an initial rescan and then schedules periodic rescans
// according to the cron expression in the config. Call Stop to release resources.
func (s *Service) Start() error {
	s.log.Info("starting service")
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
	s.log.Info("stopping service")
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}

func (s *Service) addPost(path string) {
	post := &Post{path: path}
	if err := s.renderPost(post); err != nil {
		s.log.Error("failed to render post", "path", path, "err", err)
		return
	}
	s.log.Info("added post", "path", path)
	s.repo.Add(post)
}

func (s *Service) updatePost(path string) {
	post := s.repo.Get(path)
	if post == nil {
		return
	}

	if s.conf.SkipUnchangedModTime {
		modTime, err := s.source.StatPost(path)
		if err != nil {
			s.log.Error("failed to stat post", "path", path, "err", err)
			return
		}
		if modTime.Equal(post.modTime) {
			return // mod time unchanged, skip read
		}
		post.modTime = modTime // update regardless of hash
	}

	buf, err := s.source.ReadPost(path)
	if err != nil {
		s.log.Error("failed to read post", "path", path, "err", err)
		return
	}
	if hashBytes(buf) == post.hash {
		return // content unchanged
	}
	if err := parseAndRender(post, buf); err != nil {
		s.log.Error("failed to parse/render post", "path", path, "err", err)
	}
}

func (s *Service) rescan() {
	s.log.Debug("rescanning posts")
	paths, err := s.source.ListPosts()
	if err != nil {
		s.log.Error("failed to list posts", "err", err)
		return
	}

	// Track which paths exist on disk during this scan.
	found := make(map[string]bool)
	for _, path := range paths {
		found[path] = true
		if s.repo.Has(path) {
			s.updatePost(path)
		} else {
			s.addPost(path)
		}
	}

	// Remove posts that are no longer present on disk.
	var toRemove []string
	for _, post := range s.repo.List() {
		if !found[post.path] {
			toRemove = append(toRemove, post.path)
		}
	}
	for _, path := range toRemove {
		s.log.Info("removed post", "path", path)
		s.repo.Remove(path)
	}
	s.log.Debug("rescan complete", "total", len(paths))
}

func (s *Service) GetPost(path string) *Post {
	return s.repo.Get(path)
}

func hashBytes(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}

// parseAndRender parses markdown bytes, extracts metadata, renders HTML,
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
// It also records the file's modification time on the post.
func (s *Service) renderPost(post *Post) error {
	modTime, err := s.source.StatPost(post.path)
	if err != nil {
		return err
	}
	buf, err := s.source.ReadPost(post.path)
	if err != nil {
		return err
	}
	post.modTime = modTime
	return parseAndRender(post, buf)
}
