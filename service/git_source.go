package service

import (
	"errors"
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
	auth transport.AuthMethod
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
		auth: auth,
		repo: repo,
		fs:   fs,
		log:  logger.With("component", "GitSource"),
	}, nil
}

func (g *GitSource) Sync() error {
	g.log.Debug("pulling", "url", g.url)
	wt, err := g.repo.Worktree()
	if err != nil {
		return fmt.Errorf("git sync %s: %w", g.url, err)
	}
	err = wt.Pull(&gogit.PullOptions{Auth: g.auth})
	if errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("git sync %s: %w", g.url, err)
	}
	return nil
}

func (g *GitSource) SlugForPath(p string) string {
	// git paths use forward slashes; root is g.dir
	root := g.dir
	if root == "." {
		root = ""
	}
	rel := p
	if root != "" && strings.HasPrefix(p, root+"/") {
		rel = p[len(root)+1:]
	}
	ext := path.Ext(rel)
	if ext != "" {
		rel = rel[:len(rel)-len(ext)]
	}
	parts := strings.Split(rel, "/")
	for i, seg := range parts {
		parts[i] = strings.ReplaceAll(seg, "+", "-")
	}
	return strings.Join(parts, "+")
}

func (g *GitSource) ListPosts() ([]string, error) {
	g.log.Debug("listing posts", "url", g.url, "dir", g.dir)
	var paths []string
	if err := g.walkDir(g.dir, &paths); err != nil {
		return nil, fmt.Errorf("git list posts: %w", err)
	}
	return paths, nil
}

func (g *GitSource) walkDir(dir string, paths *[]string) error {
	entries, err := g.fs.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		p := path.Join(dir, entry.Name())
		if entry.IsDir() {
			if err := g.walkDir(p, paths); err != nil {
				return err
			}
		} else if strings.HasSuffix(entry.Name(), ".md") {
			*paths = append(*paths, p)
		}
	}
	return nil
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
