package service

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PostSource interface {
	// Sync updates the source's backing data (e.g. git pull). Called before each rescan.
	Sync() error
	// ListPosts returns the canonical paths of all available posts.
	ListPosts() ([]string, error)
	// ReadPost returns the raw bytes of the post at the given path.
	ReadPost(path string) ([]byte, error)
	// StatPost returns the modification time of the post at the given path.
	StatPost(path string) (time.Time, error)
	// SlugForPath derives the URL slug for a path returned by ListPosts.
	SlugForPath(path string) string
}

// slugFromSegments builds a slug from path segments relative to a root:
// segments are joined with "+" and any literal "+" in a segment is replaced
// with "-" to keep slugs predictable.
func slugFromSegments(root, fullPath string) string {
	rel := fullPath
	if root != "." && root != "" {
		var err error
		rel, err = filepath.Rel(root, fullPath)
		if err != nil {
			rel = fullPath
		}
	}
	// Strip extension from the last segment.
	ext := filepath.Ext(rel)
	if ext != "" {
		rel = rel[:len(rel)-len(ext)]
	}
	// Split on OS path separator and filter empty parts.
	parts := strings.FieldsFunc(rel, func(r rune) bool {
		return r == filepath.Separator || r == '/'
	})
	for i, p := range parts {
		parts[i] = strings.ReplaceAll(p, "+", "-")
	}
	return strings.Join(parts, "+")
}

// FileSystemSource reads posts from a directory on disk.
type FileSystemSource struct {
	dir string
	log *slog.Logger
}

func NewFileSystemSource(dir string, logger *slog.Logger) *FileSystemSource {
	return &FileSystemSource{dir: dir, log: logger.With("component", "FileSystemSource")}
}

func (fs *FileSystemSource) Sync() error { return nil }

func (fs *FileSystemSource) SlugForPath(path string) string {
	return slugFromSegments(fs.dir, path)
}

func (fs *FileSystemSource) ListPosts() ([]string, error) {
	fs.log.Debug("listing posts", "dir", fs.dir)
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		paths = append(paths, filepath.Join(fs.dir, entry.Name()))
	}
	return paths, nil
}

func (fs *FileSystemSource) ReadPost(path string) ([]byte, error) {
	fs.log.Debug("reading post", "path", path)
	return os.ReadFile(path)
}

func (fs *FileSystemSource) StatPost(path string) (time.Time, error) {
	fs.log.Debug("stat post", "path", path)
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}
