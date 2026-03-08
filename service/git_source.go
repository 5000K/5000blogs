package service

import (
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
)

// GitSource reads posts from a git repository cloned in-memory.
// dir is the subdirectory within the repo containing posts (use "." for the root).
type GitSource struct {
	url  string
	dir  string
	repo *gogit.Repository
	fs   billy.Filesystem
	log  *slog.Logger
}

func NewGitSource(url, dir string, auth transport.AuthMethod, logger *slog.Logger) (*GitSource, error) {
	fs := memfs.New()
	repo, err := gogit.Clone(memory.NewStorage(), fs, &gogit.CloneOptions{
		URL:  url,
		Auth: auth,
	})
	if err != nil {
		return nil, fmt.Errorf("git clone %s: %w", url, err)
	}
	return &GitSource{
		url:  url,
		dir:  dir,
		repo: repo,
		fs:   fs,
		log:  logger.With("component", "GitSource"),
	}, nil
}

func (g *GitSource) ListPosts() ([]string, error) {
	g.log.Debug("listing posts", "url", g.url, "dir", g.dir)
	entries, err := g.fs.ReadDir(g.dir)
	if err != nil {
		return nil, fmt.Errorf("git list posts: %w", err)
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		paths = append(paths, path.Join(g.dir, entry.Name()))
	}
	return paths, nil
}

func (g *GitSource) ReadPost(p string) ([]byte, error) {
	g.log.Debug("reading post", "path", p)
	f, err := g.fs.Open(p)
	if err != nil {
		return nil, fmt.Errorf("git read post %s: %w", p, err)
	}
	defer f.Close()
	return io.ReadAll(f)
}

// StatPost returns the commit time of the most recent commit that touched the file.
func (g *GitSource) StatPost(p string) (time.Time, error) {
	g.log.Debug("stat post", "path", p)
	ref, err := g.repo.Head()
	if err != nil {
		return time.Time{}, fmt.Errorf("git stat post %s: %w", p, err)
	}
	iter, err := g.repo.Log(&gogit.LogOptions{
		From: ref.Hash(),
		PathFilter: func(s string) bool {
			return s == p
		},
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("git stat post %s: %w", p, err)
	}
	defer iter.Close()
	commit, err := iter.Next()
	if err != nil {
		return time.Time{}, fmt.Errorf("git stat post %s: no commits found: %w", p, err)
	}
	return commit.Author.When, nil
}
