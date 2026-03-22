package service

import (
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/5000K/5000blogs/config"
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
	// ReadMedia returns the raw bytes and modification time of a non-markdown
	// media file located at relPath relative to the source root.
	// Returns os.ErrNotExist if the file is not present in this source.
	ReadMedia(relPath string) ([]byte, time.Time, error)
	// ResolveAssetByFilename searches for a file matching filename (basename only)
	// breadth-first from the source root and returns its path relative to the root.
	// Returns "" when not found.
	ResolveAssetByFilename(filename string) string
}

// slugFromSegments builds a slug from path segments relative to a root:
// segments are joined with "/" to form a path-based slug.
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
	return strings.Join(parts, "/")
}

// FileSystemSource reads posts from a directory on disk.
type FileSystemSource struct {
	dir string
	log *slog.Logger
}

func NewFileSystemSource(logger *slog.Logger) *FileSystemSource {
	return &FileSystemSource{log: logger.With("component", "FileSystemSource")}
}

func (fs *FileSystemSource) Initialize(conf config.SourceConfig) error {
	fs.dir = conf.Dir
	return nil
}

func (fs *FileSystemSource) Sync() error { return nil }

func (fs *FileSystemSource) SlugForPath(path string) string {
	return slugFromSegments(fs.dir, path)
}

func (fs *FileSystemSource) ListPosts() ([]string, error) {
	fs.log.Debug("listing posts", "dir", fs.dir)
	var paths []string
	err := filepath.WalkDir(fs.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
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

// ReadMedia returns the raw bytes and modification time of a media file located
// at relPath (relative to the source directory). Path traversal is prevented by
// resolving the path inside a virtual root before joining with the real dir.
func (fs *FileSystemSource) ReadMedia(relPath string) ([]byte, time.Time, error) {
	// Resolve inside a virtual root so "../../etc/passwd" becomes "etc/passwd".
	cleaned := path.Clean("/" + relPath)
	cleaned = strings.TrimPrefix(cleaned, "/")
	fullPath := filepath.Join(fs.dir, filepath.FromSlash(cleaned))
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, time.Time{}, err
	}
	if info.IsDir() {
		return nil, time.Time{}, os.ErrNotExist
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, time.Time{}, err
	}
	return data, info.ModTime(), nil
}

// ResolveAssetByFilename searches breadth-first from the source directory for a
// file whose basename matches filename, and returns its path relative to fs.dir.
// Returns "" when no match is found.
func (fs *FileSystemSource) ResolveAssetByFilename(filename string) string {
	queue := []string{fs.dir}
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		var subdirs []string
		for _, e := range entries {
			if e.IsDir() {
				subdirs = append(subdirs, filepath.Join(dir, e.Name()))
				continue
			}
			if e.Name() == filename {
				rel, err := filepath.Rel(fs.dir, filepath.Join(dir, e.Name()))
				if err != nil {
					return filename
				}
				return filepath.ToSlash(rel)
			}
		}
		queue = append(queue, subdirs...)
	}
	return ""
}
