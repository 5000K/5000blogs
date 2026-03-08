package service

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newFSSource(t *testing.T) (*FileSystemSource, string) {
	t.Helper()
	dir := t.TempDir()
	return NewFileSystemSource(dir, slog.Default()), dir
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	return path
}

// --- ListPosts ---

func TestFileSystemSource_ListPosts_Empty(t *testing.T) {
	fs, _ := newFSSource(t)
	paths, err := fs.ListPosts()
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("want 0 posts, got %d", len(paths))
	}
}

func TestFileSystemSource_ListPosts_OnlyMarkdown(t *testing.T) {
	fs, dir := newFSSource(t)
	writeFile(t, dir, "a.md", "# A")
	writeFile(t, dir, "b.md", "# B")
	writeFile(t, dir, "readme.txt", "not a post")
	writeFile(t, dir, "image.png", "")

	paths, err := fs.ListPosts()
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("want 2 .md files, got %d: %v", len(paths), paths)
	}
	for _, p := range paths {
		if !strings.HasSuffix(p, ".md") {
			t.Errorf("non-.md path returned: %q", p)
		}
	}
}

func TestFileSystemSource_ListPosts_SkipsSubdirectories(t *testing.T) {
	fs, dir := newFSSource(t)
	writeFile(t, dir, "post.md", "# Post")
	if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "subdir"), "nested.md", "# Nested")

	paths, err := fs.ListPosts()
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(paths) != 1 {
		t.Errorf("want 1 post (no subdirs), got %d: %v", len(paths), paths)
	}
}

func TestFileSystemSource_ListPosts_MissingDir(t *testing.T) {
	fs := NewFileSystemSource("/nonexistent/dir", slog.Default())
	_, err := fs.ListPosts()
	if err == nil {
		t.Error("want error for missing directory")
	}
}

// --- ReadPost ---

func TestFileSystemSource_ReadPost(t *testing.T) {
	fs, dir := newFSSource(t)
	path := writeFile(t, dir, "hello.md", "---\ntitle: Hello\n---\n\n# Hello")

	data, err := fs.ReadPost(path)
	if err != nil {
		t.Fatalf("ReadPost: %v", err)
	}
	if !strings.Contains(string(data), "Hello") {
		t.Errorf("unexpected content: %q", string(data))
	}
}

func TestFileSystemSource_ReadPost_NotFound(t *testing.T) {
	fs, dir := newFSSource(t)
	_, err := fs.ReadPost(filepath.Join(dir, "missing.md"))
	if err == nil {
		t.Error("want error for missing file")
	}
}

// --- StatPost ---

func TestFileSystemSource_StatPost(t *testing.T) {
	fs, dir := newFSSource(t)
	before := time.Now().Truncate(time.Second)
	path := writeFile(t, dir, "timed.md", "# Timed")

	modTime, err := fs.StatPost(path)
	if err != nil {
		t.Fatalf("StatPost: %v", err)
	}
	if modTime.Before(before) {
		t.Errorf("modTime %v is before write time %v", modTime, before)
	}
}

func TestFileSystemSource_StatPost_NotFound(t *testing.T) {
	fs, dir := newFSSource(t)
	_, err := fs.StatPost(filepath.Join(dir, "missing.md"))
	if err == nil {
		t.Error("want error for missing file")
	}
}

func TestFileSystemSource_Sync_IsNoop(t *testing.T) {
	fs, _ := newFSSource(t)
	if err := fs.Sync(); err != nil {
		t.Errorf("Sync() returned unexpected error: %v", err)
	}
}
