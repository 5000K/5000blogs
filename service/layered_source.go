package service

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed builtin
var builtinFS embed.FS

// BuiltinSource is a read-only PostSource backed by files embedded at compile time.
// Paths are virtual: "builtin/<name>.md".
type BuiltinSource struct {
	paths []string
	media []string
}

func NewBuiltinSource() *BuiltinSource {
	return &BuiltinSource{
		paths: []string{"builtin/index.md", "builtin/404.md", "builtin/footer.md"},
		media: []string{"builtin/robots.txt"},
	}
}

func (b *BuiltinSource) Sync() error { return nil }

// SlugForPath for builtin paths strips the "builtin/" prefix: "builtin/home.md" → "home".
func (b *BuiltinSource) SlugForPath(p string) string {
	base := p
	if after, ok := strings.CutPrefix(p, "builtin/"); ok {
		base = after
	}
	if ext := filepath.Ext(base); ext != "" {
		base = base[:len(base)-len(ext)]
	}
	return base
}

func (b *BuiltinSource) ListPosts() ([]string, error) {
	return b.paths, nil
}

func (b *BuiltinSource) ReadPost(path string) ([]byte, error) {
	data, err := builtinFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("builtin source: %w", err)
	}
	return data, nil
}

func (b *BuiltinSource) StatPost(path string) (time.Time, error) {
	if !b.has(path) {
		return time.Time{}, fmt.Errorf("builtin source: unknown path %q", path)
	}
	return time.Time{}, nil
}

func (b *BuiltinSource) ReadMedia(path string) ([]byte, time.Time, error) {
	for _, m := range b.media {
		if m == path {
			data, err := builtinFS.ReadFile(m)
			if err != nil {
				return nil, time.Time{}, fmt.Errorf("builtin source: %w", err)
			}
			return data, time.Time{}, nil
		}
	}
	return nil, time.Time{}, os.ErrNotExist
}

func (b *BuiltinSource) ResolveAssetByFilename(filename string) string {
	for _, m := range b.media {
		if strings.Contains(filename, m) {
			return m
		}
	}
	return ""
}

func (b *BuiltinSource) has(path string) bool {
	for _, p := range b.paths {
		if p == path {
			return true
		}
	}
	return false
}

// LayeredSource combines multiple PostSources into one. Earlier sources take
// priority: if two sources expose a post with the same slug, only the first
// one's path is returned from ListPosts. ReadPost/StatPost/SlugForPath are
// routed to the source that owns the path.
type LayeredSource struct {
	sources []PostSource
}

func NewLayeredSource(sources ...PostSource) *LayeredSource {
	return &LayeredSource{sources: sources}
}

func (l *LayeredSource) Sync() error {
	var errs []string
	for _, s := range l.sources {
		if err := s.Sync(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("sync errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (l *LayeredSource) ListPosts() ([]string, error) {
	seen := make(map[string]bool)
	var all []string
	for _, s := range l.sources {
		paths, err := s.ListPosts()
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			slug := s.SlugForPath(p)
			if seen[slug] {
				continue
			}
			seen[slug] = true
			all = append(all, p)
		}
	}
	return all, nil
}

func (l *LayeredSource) SlugForPath(p string) string {
	for _, s := range l.sources {
		paths, err := s.ListPosts()
		if err != nil {
			continue
		}
		for _, sp := range paths {
			if sp == p {
				return s.SlugForPath(p)
			}
		}
	}
	// fall back: treat p as a plain filename
	base := filepath.Base(p)
	if ext := filepath.Ext(base); ext != "" {
		base = base[:len(base)-len(ext)]
	}
	return base
}

func (l *LayeredSource) ReadPost(path string) ([]byte, error) {
	for _, s := range l.sources {
		data, err := s.ReadPost(path)
		if err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("layered source: no source has path %q", path)
}

func (l *LayeredSource) StatPost(path string) (time.Time, error) {
	for _, s := range l.sources {
		t, err := s.StatPost(path)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("layered source: no source has path %q", path)
}

// ReadMedia tries each source in order and returns the first successful result.
func (l *LayeredSource) ReadMedia(relPath string) ([]byte, time.Time, error) {
	for _, s := range l.sources {
		data, t, err := s.ReadMedia(relPath)
		if err == nil {
			return data, t, nil
		}
	}
	return nil, time.Time{}, fmt.Errorf("layered source: no source has media %q", relPath)
}

// ResolveAssetByFilename tries each source in order and returns the first non-empty result.
func (l *LayeredSource) ResolveAssetByFilename(filename string) string {
	for _, s := range l.sources {
		if rel := s.ResolveAssetByFilename(filename); rel != "" {
			return rel
		}
	}
	return ""
}
