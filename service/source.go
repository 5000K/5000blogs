package service

import (
	"os"
	"path/filepath"
	"strings"
)

// PostSource is the interface for discovering and reading post files.
type PostSource interface {
	// ListPosts returns the canonical paths of all available posts.
	ListPosts() ([]string, error)
	// ReadPost returns the raw bytes of the post at the given path.
	ReadPost(path string) ([]byte, error)
}

// FileSystemSource reads posts from a directory on disk.
type FileSystemSource struct {
	dir string
}

// NewFileSystemSource creates a FileSystemSource that reads .md files from dir.
func NewFileSystemSource(dir string) *FileSystemSource {
	return &FileSystemSource{dir: dir}
}

// ListPosts returns the paths of all .md files in the source directory.
func (fs *FileSystemSource) ListPosts() ([]string, error) {
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

// ReadPost returns the raw bytes of the file at path.
func (fs *FileSystemSource) ReadPost(path string) ([]byte, error) {
	return os.ReadFile(path)
}
