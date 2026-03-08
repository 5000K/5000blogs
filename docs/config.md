# Configuration Reference

Config is loaded from a YAML file whose path is set by the `CONFIG_PATH` environment variable (default: `config.yml`). Environment variables override YAML values.

## Keys

| YAML key | Env var | Default | Description |
|---|---|---|---|
| `address` | `SERVER_ADDRESS` | `:8080` | TCP address the HTTP server listens on. |
| `paths.posts` | `POSTS_PATH` | `./posts/` | Directory containing Markdown post files. |
| `paths.static` | `STATIC_PATH` | `./static/` | Directory containing `template.html` and other static assets. |
| `rescan_cron` | `RESCAN_CRON` | `* * * * *` | Cron expression controlling how often the posts directory is rescanned. Default is every minute. |
| `skip_unchanged_mod_time` | `SKIP_UNCHANGED_MOD_TIME` | `true` | Skip re-parsing a post file if its modification time has not changed since the last scan. |
| `log_level` | `LOG_LEVEL` | `info` | Log verbosity. Accepted values: `debug`, `info`, `warn`, `error`. |
| `page_size` | `PAGE_SIZE` | `10` | Number of posts per page on the `/posts` list. Must be > 0. |
| `site_url` | `SITE_URL` | `http://localhost:8080` | Absolute base URL of the site. Used in feed links and canonical URLs. Must be an absolute URL. |
| `feed_description` | `FEED_DESCRIPTION` | _(empty)_ | Description of the RSS/Atom feed. |
| `rss_full_content` | `RSS_FULL_CONTENT` | `false` | Include full rendered HTML in feed entries instead of description only. |
| `blog_name` | `BLOG_NAME` | `Blog` | Site name shown in the header, used as the RSS/Atom feed title, and used as a fallback page title. |
| `icon` | `ICON` | _(empty)_ | Path to a PNG file served as `/favicon.ico` and `/og-logo.png`. Also used as the logo in generated `og:image` PNGs. |
| `nav_links` | — | _(none)_ | List of header navigation links. Each entry has `name` (display text) and `url`. YAML array only; no env var equivalent. |
| `pages` | — | _(none)_ | Custom page routes. Each entry has `path` (URL path, must start with `/`) and `slug` (post slug to serve at that path). The `/` path defaults to slug `home` if not listed here. Paths already owned by the static router (e.g. `/posts`, `/feed.xml`) are silently ignored. YAML array only; no env var equivalent. |
| `plugins` | — | _(none)_ | List of JavaScript URLs injected as `<script>` tags via `.Plugins` in the template to quickly extend the client. YAML array only; no env var equivalent. |
| `sources` | — | _(none)_ | List of post sources. If omitted, a single filesystem source using `paths.posts` is used. See [Sources](#sources) below. YAML array only; no env var equivalent. |

### og_image

| YAML key | Env var | Default | Description |
|---|---|---|---|
| `og_image.enabled` | `OG_IMAGE_ENABLED` | `true` | Enable dynamic `og:image` generation. |
| `og_image.bg_color` | `OG_IMAGE_BG_COLOR` | `#111111` | Background colour (hex). |
| `og_image.text_color` | `OG_IMAGE_TEXT_COLOR` | `#f0f0f0` | Title text colour (hex). |
| `og_image.sub_color` | `OG_IMAGE_SUB_COLOR` | `#999999` | Subtitle/secondary text colour (hex). |
| `og_image.accent_color` | `OG_IMAGE_ACCENT_COLOR` | `#7eb8f7` | Accent colour used for decorative elements (hex). |
| `og_image.cache_size` | `OG_IMAGE_CACHE_SIZE` | `128` | Maximum number of generated `og:image` PNGs to keep in the in-memory LRU cache. |

## Sources

`sources` is an optional list of post sources. When omitted, a single `filesystem` source pointing at `paths.posts` is used automatically.

Each entry requires a `type` field:

### `filesystem`

| Field | Required | Description |
|---|---|---|
| `path` | yes | Directory path containing `.md` files. |

```yaml
sources:
  - type: filesystem
    path: ./posts
```

### `git`

Clones the repository in-memory on startup. No local checkout is written to disk.

| Field | Required | Description |
|---|---|---|
| `url` | yes | Repository URL (HTTPS or SSH). |
| `dir` | no | Subdirectory within the repo containing posts. Defaults to the repo root (`.`). |
| `auth_user` | no | Username for HTTP basic auth. Defaults to `git` when `auth_token` is set. |
| `auth_token` | no | Password or personal access token for HTTP basic auth. Mutually exclusive with `ssh_key_path`. |
| `ssh_key_path` | no | Path to a PEM-encoded SSH private key file. Mutually exclusive with `auth_token`. |
| `ssh_key_passphrase` | no | Passphrase for the SSH private key. Leave empty for unencrypted keys. |

```yaml
# Public repo — no auth needed
sources:
  - type: git
    url: https://github.com/yourname/blog-posts
    dir: posts
```

```yaml
# Private repo via HTTPS token (e.g. GitHub personal access token)
sources:
  - type: git
    url: https://github.com/yourname/private-blog
    auth_token: ghp_yourpersonalaccesstoken
```

```yaml
# Private repo via SSH key
sources:
  - type: git
    url: git@github.com:yourname/private-blog.git
    ssh_key_path: /home/user/.ssh/id_ed25519
    ssh_key_passphrase: optional-passphrase  # omit if key is unencrypted
```

Multiple sources can be combined. Earlier sources take priority: when two sources expose the same slug, only the first one's post is used.

## Well-known posts

Three posts are built in and served automatically:

| Slug | Purpose |
|---|---|
| `home` | Rendered at `/` (the root page). Override the slug via `pages: [{path: /, slug: my-home}]`. |
| `footer` | Rendered as the page footer on every page. Override by placing `footer.md` in your `paths.posts` directory. |
| `404` | Rendered for any unknown URL. |

To override either, place a file with the matching name in your `paths.posts` directory (e.g. `home.md` or `404.md`). Your file takes precedence over the built-in version.
