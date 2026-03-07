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

// --- WritePost ---

func TestFileSystemSource_WritePost_CreatesFile(t *testing.T) {
	fs, dir := newFSSource(t)
	path := filepath.Join(dir, "new-post.md")
	meta := &Metadata{
		Title:       "New Post",
		Description: "A description",
		Author:      "Alice",
	}

	if err := fs.WritePost(path, meta, "Body content here."); err != nil {
		t.Fatalf("WritePost: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after WritePost: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "New Post") {
		t.Errorf("title missing in written file: %q", body)
	}
	if !strings.Contains(body, "Body content here.") {
		t.Errorf("content missing in written file: %q", body)
	}
	if !strings.HasPrefix(body, "---\n") {
		snip := body
		if len(snip) > 20 {
			snip = snip[:20]
		}
		t.Errorf("expected front matter delimiter, got: %q", snip)
	}
}

func TestFileSystemSource_WritePost_RoundTrip(t *testing.T) {
	fs, dir := newFSSource(t)
	path := filepath.Join(dir, "roundtrip.md")
	date := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	meta := &Metadata{
		Title:  "Round Trip",
		Date:   date,
		Author: "Bob",
	}
	body := "Hello, world!"

	if err := fs.WritePost(path, meta, body); err != nil {
		t.Fatalf("WritePost: %v", err)
	}

	raw, err := fs.ReadPost(path)
	if err != nil {
		t.Fatalf("ReadPost: %v", err)
	}

	parsed, parsedBody, err := extractFrontmatter(raw)
	if err != nil {
		t.Fatalf("extractFrontmatter: %v", err)
	}
	if parsed.Title != "Round Trip" {
		t.Errorf("Title round-trip failed: got %q", parsed.Title)
	}
	if parsed.Author != "Bob" {
		t.Errorf("Author round-trip failed: got %q", parsed.Author)
	}
	if !strings.Contains(string(parsedBody), body) {
		t.Errorf("body round-trip failed: got %q", string(parsedBody))
	}
}

func TestFileSystemSource_WritePost_NilMetadata(t *testing.T) {
	fs, dir := newFSSource(t)
	path := filepath.Join(dir, "no-meta.md")

	if err := fs.WritePost(path, nil, "Just content."); err != nil {
		t.Fatalf("WritePost with nil metadata: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "Just content.") {
		t.Errorf("content missing: %q", string(data))
	}
}
