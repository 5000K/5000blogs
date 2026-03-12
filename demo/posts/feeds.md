---
title: Feeds
description: RSS 2.0 and Atom 1.0 feed configuration and filtering.
date: 2025-01-05
tags: [config, feeds]
---

5000blogs generates both RSS 2.0 and Atom 1.0 feeds automatically.

## Endpoints

| URL | Format | Content-Type |
|---|---|---|
| `/feed.xml` | RSS 2.0 | `application/rss+xml` |
| `/feed.atom` | Atom 1.0 | `application/atom+xml` |

## Configuration

| Key | Default | Description |
|---|---|---|
| `feed_description` | `""` | Channel/feed description text |
| `feed_size` | `20` | Maximum number of entries |
| `rss_content` | `none` | Content included in entries |
| `blog_name` | `Blog` | Feed title |
| `site_url` | `http://localhost:8080` | Feed link / base URL |

## Content modes

The `rss_content` key controls what goes into each feed entry:

| Value | Behavior |
|---|---|
| `none` | Only title, link, description. No body content |
| `text` | Plain-text version of the post body in `<content:encoded>` (RSS) / `<content type="text">` (Atom) |
| `html` | Rendered HTML of the post body in `<content:encoded>` (RSS) / `<content type="html">` (Atom) |

## Filtering

Both feed endpoints accept query parameters for filtering:

```
/feed.xml?tags=go,docker
/feed.atom?tags=tutorial&q=setup
```

| Parameter | Description |
|---|---|
| `tags` | Comma-separated tag filter (OR logic) |
| `q` | Search query filter |

## Post visibility in feeds

Posts are included in feeds only if both conditions are met:

1. `visible` is `true` (default)
2. `rss-visible` is `true` (default)

Set `rss-visible: false` in front matter to exclude a post from feeds while keeping it visible on the site. Set `visible: false` to hide from both the site listing and feeds.

```yaml
---
title: Secret Post
visible: true
rss-visible: false
---
```

## Feed sorting

Entries are sorted by date (newest first) and truncated to `feed_size`. Posts without a date use their file modification time.

## Default footer

The built-in footer includes links to both `/feed.xml` and `/feed.atom`. Override by creating your own `footer.md`.
