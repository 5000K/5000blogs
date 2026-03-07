package service

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PostSource interface {
	// ListPosts returns the canonical paths of all available posts.
	ListPosts() ([]string, error)
	// ReadPost returns the raw bytes of the post at the given path.
	ReadPost(path string) ([]byte, error)
	// StatPost returns the modification time of the post at the given path.
	StatPost(path string) (time.Time, error)
}

// FileSystemSource reads posts from a directory on disk.
type FileSystemSource struct {
	dir string
	log *slog.Logger
}

func NewFileSystemSource(dir string, logger *slog.Logger) *FileSystemSource {
	return &FileSystemSource{dir: dir, log: logger.With("component", "FileSystemSource")}
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
