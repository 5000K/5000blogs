package service

import (
	"embed"
	"fmt"
	"time"
)

//go:embed builtin
var builtinFS embed.FS

// BuiltinSource is a read-only PostSource backed by files embedded at compile time.
// Paths are virtual: "builtin/<name>.md".
type BuiltinSource struct {
	paths []string
}

func NewBuiltinSource() *BuiltinSource {
	return &BuiltinSource{
		paths: []string{"builtin/home.md", "builtin/404.md"},
	}
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
// one's path is returned from ListPosts. ReadPost/StatPost are routed to the
// source that owns the path.
type LayeredSource struct {
	sources []PostSource
}

func NewLayeredSource(sources ...PostSource) *LayeredSource {
	return &LayeredSource{sources: sources}
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
			slug := slugFromPath(p)
			if seen[slug] {
				continue
			}
			seen[slug] = true
			all = append(all, p)
		}
	}
	return all, nil
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
