package service

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type PostSource interface {
	// ListPosts returns the canonical paths of all available posts.
	ListPosts() ([]string, error)
	// ReadPost returns the raw bytes of the post at the given path.
	ReadPost(path string) ([]byte, error)
	// StatPost returns the modification time of the post at the given path.
	StatPost(path string) (time.Time, error)
	// WritePost serialises metadata and content to the given path.
	WritePost(path string, metadata *Metadata, content string) error
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

func (fs *FileSystemSource) WritePost(path string, metadata *Metadata, content string) error {
	fields := make(map[string]interface{})
	if metadata != nil {
		if metadata.Title != "" {
			fields["title"] = metadata.Title
		}
		if metadata.Description != "" {
			fields["description"] = metadata.Description
		}
		if !metadata.Date.IsZero() {
			fields["date"] = metadata.Date
		}
		if metadata.Author != "" {
			fields["author"] = metadata.Author
		}
		if metadata.Visible != nil {
			fields["visible"] = *metadata.Visible
		}
		if metadata.RSSVisible != nil {
			fields["rss-visible"] = *metadata.RSSVisible
		}
		for k, v := range metadata.Raw {
			if _, exists := fields[k]; !exists {
				fields[k] = v
			}
		}
	}

	yamlBytes, err := yaml.Marshal(fields)
	if err != nil {
		return fmt.Errorf("WritePost: failed to marshal metadata: %w", err)
	}

	body := fmt.Sprintf("```yaml\n%s```\n\n%s", string(yamlBytes), content)

	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return fmt.Errorf("WritePost: failed to write file %q: %w", path, err)
	}
	return nil
}
