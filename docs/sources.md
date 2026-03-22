---
title: Sources
description: Configure filesystem and git post sources.
date: 2025-01-04
tags: [config, sources]
---

Sources define where posts are loaded from. Two types are supported: `filesystem` and `git`.

## Default behavior

When no `sources` array is configured, only the builtin pages are served. If you want to have any content, you need to define a source.

## Multiple sources

Multiple sources can be combined. Earlier sources take priority when slugs collide:

```yaml
sources:
  - type: filesystem
    path: "./local-posts"
  - type: git
    url: "https://github.com/user/posts.git"
```

## Filesystem source

Reads `.md` files recursively from a local directory.

| Key | Required | Description |
|---|---|---|
| `type` | yes | `filesystem` |
| `path` | yes | Directory path |

```yaml
sources:
  - type: filesystem
    path: "./posts"
```

Subdirectories create nested slugs: `posts/guides/setup.md` → slug `guides/setup`, URL `/guides/setup`.

## Git source

Clones a git repository into memory and reads `.md` files from it. Syncs on every rescan cycle.

| Key | Required | Description |
|---|---|---|
| `type` | yes | `git` |
| `url` | yes | Repository URL (HTTPS or SSH) |
| `dir` | no | Subdirectory within the repo. Default: `.` (root) |
| `auth_user` | no | HTTP basic auth username. Default: `git` |
| `auth_token` | no | HTTP basic auth password or token |
| `ssh_key_path` | no | Path to SSH private key file |
| `ssh_key_passphrase` | no | Passphrase for SSH private key |

`auth_token` and `ssh_key_path` are mutually exclusive. Startup fails if both are set.

### Public repository (HTTPS)

```yaml
sources:
  - type: git
    url: "https://github.com/user/posts.git"
```

### Private repository (token)

```yaml
sources:
  - type: git
    url: "https://github.com/user/private-posts.git"
    auth_user: "git"
    auth_token: "ghp_xxxxxxxxxxxx"
```

### Private repository (SSH)

```yaml
sources:
  - type: git
    url: "git@github.com:user/private-posts.git"
    ssh_key_path: "/path/to/id_ed25519"
    ssh_key_passphrase: "optional-passphrase"
```

### Subdirectory

Read only from a specific folder within the repo:

```yaml
sources:
  - type: git
    url: "https://github.com/user/monorepo.git"
    dir: "blog/posts"
```

## Builtin source

A built-in source is always loaded as the lowest-priority layer. It provides default content for:

- `index` - welcome page (replaced by your own `index.md`)
- `404` - not-found page (replaced by your own `404.md`)
- `footer` - page footer (replaced by your own `footer.md`)

See [Writing Posts: Special Posts](writing-posts#special-posts).

## Source layering

Sources are layered in order: user-defined sources first, then the builtin source. When two sources provide a post with the same slug, the first source wins.

Media files (images, etc.) are also resolved through the source chain - first match wins.

## Rescan

Sources are synced and posts are rescanned on a cron schedule (`rescan_cron`, default: every minute). Git sources run `git pull` on each sync. Filesystem sources re-list the directory.

The `skip_unchanged_mod_time` option (default: `true`) avoids re-reading files whose modification time hasn't changed. Even when a file is re-read, it is only re-rendered if the content hash (FNV-64a) differs.
