package service

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/5000K/5000blogs/config"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
)

func configToGitAuth(sc config.SourceConfig) (transport.AuthMethod, error) {
	if sc.SSHKeyPath != "" {
		auth, err := gitssh.NewPublicKeysFromFile("git", sc.SSHKeyPath, sc.SSHKeyPassphrase)
		if err != nil {
			return nil, fmt.Errorf("git source %s: load SSH key: %w", sc.URL, err)
		}
		return auth, nil
	}
	if sc.AuthToken != "" {
		user := sc.AuthUser
		if user == "" {
			user = "git"
		}
		return &githttp.BasicAuth{Username: user, Password: sc.AuthToken}, nil
	}
	return nil, nil
}

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

func NewGitSource(logger *slog.Logger) (*GitSource, error) {
	fs := memfs.New()

	return &GitSource{
		fs:  fs,
		log: logger.With("component", "GitSource"),
	}, nil
}

func (g *GitSource) Initialize(conf config.SourceConfig) error {
	g.url = conf.URL
	g.dir = conf.Dir
	auth, err := configToGitAuth(conf)
	if err != nil {
		return fmt.Errorf("git source auth: %w", err)
	}
	g.auth = auth

	g.log.Debug("cloning repository", "url", g.url)
	repo, err := gogit.Clone(memory.NewStorage(), g.fs, &gogit.CloneOptions{
		URL:  g.url,
		Auth: g.auth,
	})
	if err != nil {
		return fmt.Errorf("git clone %s: %w", g.url, err)
	}
	g.repo = repo
	return nil
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
	return rel
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

// ReadMedia returns the raw bytes and modification time of a media file at
// relPath relative to the source's directory within the repository.
func (g *GitSource) ReadMedia(relPath string) ([]byte, time.Time, error) {
	// Prevent path traversal by resolving inside a virtual root.
	cleaned := path.Clean("/" + relPath)
	cleaned = strings.TrimPrefix(cleaned, "/")
	p := cleaned
	if g.dir != "." && g.dir != "" {
		p = path.Join(g.dir, cleaned)
	}
	f, err := g.fs.Open(p)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, time.Time{}, err
	}
	info, err := g.fs.Stat(p)
	if err != nil {
		return data, time.Time{}, nil
	}
	return data, info.ModTime(), nil
}

// ResolveAssetByFilename searches breadth-first from the source directory for a
// file matching filename (basename only) and returns its path relative to the source root.
// Returns "" when not found.
func (g *GitSource) ResolveAssetByFilename(filename string) string {
	root := g.dir
	if root == "" {
		root = "."
	}
	queue := []string{root}
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]
		entries, err := g.fs.ReadDir(dir)
		if err != nil {
			continue
		}
		var subdirs []string
		for _, e := range entries {
			p := path.Join(dir, e.Name())
			if e.IsDir() {
				subdirs = append(subdirs, p)
				continue
			}
			if e.Name() == filename {
				rel := p
				if root != "." && root != "" {
					rel = strings.TrimPrefix(p, root+"/")
				}
				return rel
			}
		}
		queue = append(queue, subdirs...)
	}
	return ""
}
