package service

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// stubSource is a minimal in-memory PostSource for testing.
type stubSource struct {
	posts map[string][]byte
}

func newStubSource(posts map[string][]byte) *stubSource {
	return &stubSource{posts: posts}
}

func (s *stubSource) Sync() error { return nil }

// SlugForPath uses the plain basename (no directory awareness needed for tests).
func (s *stubSource) SlugForPath(p string) string { return slugFromPath(p) }

func (s *stubSource) ListPosts() ([]string, error) {
	paths := make([]string, 0, len(s.posts))
	for p := range s.posts {
		paths = append(paths, p)
	}
	return paths, nil
}

func (s *stubSource) ReadPost(path string) ([]byte, error) {
	data, ok := s.posts[path]
	if !ok {
		return nil, &notFoundError{path}
	}
	return data, nil
}

func (s *stubSource) StatPost(path string) (time.Time, error) {
	if _, ok := s.posts[path]; !ok {
		return time.Time{}, &notFoundError{path}
	}
	return time.Time{}, nil
}

type notFoundError struct{ path string }

func (e *notFoundError) Error() string { return "not found: " + e.path }

// --- BuiltinSource tests ---

func TestBuiltinSource_ListPosts(t *testing.T) {
	b := NewBuiltinSource()
	paths, err := b.ListPosts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 builtin posts, got %d", len(paths))
	}
	slugs := map[string]bool{}
	for _, p := range paths {
		slugs[slugFromPath(p)] = true
	}
	for _, want := range []string{"home", "404", "footer"} {
		if !slugs[want] {
			t.Errorf("expected builtin slug %q", want)
		}
	}
}

func TestBuiltinSource_ReadPost(t *testing.T) {
	b := NewBuiltinSource()
	paths, _ := b.ListPosts()
	for _, p := range paths {
		data, err := b.ReadPost(p)
		if err != nil {
			t.Errorf("ReadPost(%q) unexpected error: %v", p, err)
		}
		if len(data) == 0 {
			t.Errorf("ReadPost(%q) returned empty content", p)
		}
	}
}

func TestBuiltinSource_ReadPost_Unknown(t *testing.T) {
	b := NewBuiltinSource()
	_, err := b.ReadPost("builtin/unknown.md")
	if err == nil {
		t.Error("expected error for unknown builtin path")
	}
}

func TestBuiltinSource_StatPost_Unknown(t *testing.T) {
	b := NewBuiltinSource()
	_, err := b.StatPost("builtin/unknown.md")
	if err == nil {
		t.Error("expected error for unknown builtin path")
	}
}

func TestBuiltinSource_HomeContent(t *testing.T) {
	b := NewBuiltinSource()
	data, err := b.ReadPost("builtin/home.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(data), "title:") {
		t.Error("home.md should contain front matter with title")
	}
}

// --- LayeredSource tests ---

func TestLayeredSource_ListPosts_Priority(t *testing.T) {
	// First source has home.md; second also has home.md — first wins.
	s1 := newStubSource(map[string][]byte{
		"/user/posts/home.md": []byte("---\ntitle: User Home\n---\n"),
	})
	s2 := newStubSource(map[string][]byte{
		"builtin/home.md": []byte("---\ntitle: Builtin Home\n---\n"),
		"builtin/404.md":  []byte("---\ntitle: 404\n---\n"),
	})
	l := NewLayeredSource(s1, s2)

	paths, err := l.ListPosts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slugs := map[string]bool{}
	for _, p := range paths {
		slugs[slugFromPath(p)] = true
	}
	if !slugs["home"] {
		t.Error("expected slug 'home' in result")
	}
	if !slugs["404"] {
		t.Error("expected slug '404' from fallback source")
	}
	if len(paths) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(paths), paths)
	}
	// The user's home.md should be used, not the builtin one.
	if paths[0] != "/user/posts/home.md" {
		t.Errorf("expected user's home path first, got %q", paths[0])
	}
}

func TestLayeredSource_ListPosts_FallbackOnly(t *testing.T) {
	// Primary source is empty; all posts come from fallback.
	s1 := newStubSource(map[string][]byte{})
	s2 := newStubSource(map[string][]byte{
		"builtin/home.md": []byte("---\ntitle: Home\n---\n"),
	})
	l := NewLayeredSource(s1, s2)

	paths, err := l.ListPosts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 1 || slugFromPath(paths[0]) != "home" {
		t.Errorf("expected fallback home post, got %v", paths)
	}
}

func TestLayeredSource_ReadPost_Routing(t *testing.T) {
	s1 := newStubSource(map[string][]byte{
		"/posts/mine.md": []byte("user content"),
	})
	s2 := newStubSource(map[string][]byte{
		"builtin/home.md": []byte("builtin content"),
	})
	l := NewLayeredSource(s1, s2)

	data, err := l.ReadPost("/posts/mine.md")
	if err != nil || string(data) != "user content" {
		t.Errorf("expected user content, got %q err=%v", data, err)
	}

	data, err = l.ReadPost("builtin/home.md")
	if err != nil || string(data) != "builtin content" {
		t.Errorf("expected builtin content, got %q err=%v", data, err)
	}
}

func TestLayeredSource_ReadPost_UnknownPath(t *testing.T) {
	l := NewLayeredSource(newStubSource(map[string][]byte{}))
	_, err := l.ReadPost("/nonexistent.md")
	if err == nil {
		t.Error("expected error for unknown path")
	}
}

type syncTrackingSource struct {
	*stubSource
	synced  int
	syncErr error
}

func (s *syncTrackingSource) Sync() error {
	s.synced++
	return s.syncErr
}

func TestLayeredSource_Sync_PropagatesAll(t *testing.T) {
	s1 := &syncTrackingSource{stubSource: newStubSource(nil)}
	s2 := &syncTrackingSource{stubSource: newStubSource(nil)}
	l := NewLayeredSource(s1, s2)

	if err := l.Sync(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s1.synced != 1 || s2.synced != 1 {
		t.Errorf("expected both sources synced once, got s1=%d s2=%d", s1.synced, s2.synced)
	}
}

func TestLayeredSource_Sync_CollectsErrors(t *testing.T) {
	s1 := &syncTrackingSource{stubSource: newStubSource(nil), syncErr: errors.New("oops")}
	s2 := &syncTrackingSource{stubSource: newStubSource(nil)}
	l := NewLayeredSource(s1, s2)

	err := l.Sync()
	if err == nil {
		t.Fatal("expected error")
	}
	if s2.synced != 1 {
		t.Errorf("expected s2 synced even after s1 error, got %d", s2.synced)
	}
}
