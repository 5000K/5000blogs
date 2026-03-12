---
title: Configuration
description: Complete reference of all configuration options.
date: 2025-01-03
tags: [config]
---

Configuration is loaded from a YAML file (path set by `CONFIG_PATH` env var, default `config.yml`). Every key can also be set via environment variable. Env vars take precedence.

## General

| YAML key | Env var | Default | Description |
|---|---|---|---|
| `address` | `SERVER_ADDRESS` | `:8080` | HTTP listen address |
| `blog_name` | `BLOG_NAME` | `Blog` | Displayed in header, feeds, og:image |
| `site_url` | `SITE_URL` | `http://localhost:8080` | Absolute base URL. Must include scheme. Used in feeds, sitemap, og:image |
| `log_level` | `LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `rescan_cron` | `RESCAN_CRON` | `* * * * *` | Cron expression for post rescan interval |
| `skip_unchanged_mod_time` | `SKIP_UNCHANGED_MOD_TIME` | `true` | Skip re-reading files whose modification time hasn't changed |
| `page_size` | `PAGE_SIZE` | `10` | Posts per page on list view. Must be > 0 |

## Paths

| YAML key | Env var | Default | Description |
|---|---|---|---|
| `paths.posts` | `POSTS_PATH` | `./posts/` | Directory to read posts from (when no `sources` are configured) |
| `paths.template` | `TEMPLATE_PATH` | GitHub raw URL | Path or URL to the HTML template file |
| `paths.icon` | `ICON_PATH` | GitHub raw URL | Path or URL to the site icon (PNG). Served at `/favicon.ico` and `/og-logo.png` |

Template and icon accept both local file paths and HTTP(S) URLs. By default they fetch from the official repository, so Docker images don't need to bundle them.

## Feeds

| YAML key | Env var | Default | Description |
|---|---|---|---|
| `feed_description` | `FEED_DESCRIPTION` | `""` | Description text in RSS/Atom channel |
| `feed_size` | `FEED_SIZE` | `20` | Max items in feed. Must be > 0 |
| `rss_content` | `RSS_CONTENT` | `none` | Content in feed entries: `none`, `text`, or `html` |

See [Feeds](feeds) for details.

## Navigation

```yaml
nav_links:
  - name: "Posts"
    url: "/posts"
  - name: "About"
    url: "/about"
```

Rendered in the site header. `url` can be any absolute or relative path.

## Plugins

```yaml
plugins:
  - "https://example.com/plugin.js"
```

List of JavaScript URLs injected into every page via `<script>` tags. See [Plugins](plugins).

## Sources

```yaml
sources:
  - type: filesystem
    path: "./posts"
  - type: git
    url: "https://github.com/user/posts.git"
```

When `sources` is set, `paths.posts` is ignored. See [Sources](sources).

## Features

Toggle markdown extensions:

| YAML key | Env var | Default | Description |
|---|---|---|---|
| `features.wiki_links` | `FEATURE_WIKI_LINKS` | `true` | `[[Title]]` links resolved to post slugs |
| `features.tables` | `FEATURE_TABLES` | `true` | GFM pipe tables |
| `features.strikethrough` | `FEATURE_STRIKETHROUGH` | `true` | `~~text~~` strikethrough |
| `features.autolinks` | `FEATURE_AUTOLINKS` | `false` | Auto-detect bare URLs |
| `features.task_list` | `FEATURE_TASK_LIST` | `false` | `- [x]` / `- [ ]` checkboxes |
| `features.footnotes` | `FEATURE_FOOTNOTES` | `false` | `[^1]` footnote references |

See [Markdown](markdown) for syntax details.

## OG Image

| YAML key | Env var | Default | Description |
|---|---|---|---|
| `og_image.enabled` | `OG_IMAGE_ENABLED` | `true` | Generate `og:image` PNGs |
| `og_image.bg_color` | `OG_IMAGE_BG_COLOR` | `#111111` | Background color (hex) |
| `og_image.text_color` | `OG_IMAGE_TEXT_COLOR` | `#f0f0f0` | Title text color |
| `og_image.sub_color` | `OG_IMAGE_SUB_COLOR` | `#999999` | Description text color |
| `og_image.accent_color` | `OG_IMAGE_ACCENT_COLOR` | `#7eb8f7` | Accent line color |
| `og_image.cache_size` | `OG_IMAGE_CACHE_SIZE` | `128` | LRU cache capacity. Must be > 0 |

See [OG Images](og-images).

## Validation

On startup, the config is validated. The server refuses to start if:

- `page_size` ≤ 0
- `feed_size` ≤ 0
- `rss_content` is not `none`, `text`, or `html`
- `site_url` is not an absolute URL (must include scheme)
- `og_image.cache_size` ≤ 0
- A source has an unknown type or missing required fields
- A git source has both `auth_token` and `ssh_key_path` set (mutually exclusive)
