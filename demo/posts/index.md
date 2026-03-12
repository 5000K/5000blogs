---
title: 5000blogs Documentation
description: Complete documentation for 5000blogs - a file-backed markdown blog engine.
visible: false
rss-visible: false
---

# 5000blogs

A single-binary blog engine. Write markdown files, get a blog.

## What it does

- Serves `.md` files as HTML pages with zero build step
- Periodically rescans for new/changed/deleted posts (configurable cron)
- RSS 2.0 and Atom 1.0 feeds with tag/search filtering
- Full-text search via built-in index
- Auto-generated `og:image` cards, `sitemap.xml`, `robots.txt`
- Git repositories as post sources (with SSH/token auth)
- Wikilinks, tables, footnotes, and other CommonMark extensions
- Pluggable HTML templates and JS plugins
- REST API for programmatic access

## Quick start

```yaml
# config.yml
blog_name: 'My Blog'
site_url: 'https://example.com'
paths:
  posts: './posts'
```

```sh
docker run -p 8080:8080 \
  -v ./config.yml:/config.yml:ro \
  -v ./posts:/posts:ro \
  ghcr.io/5000k/5000blogs:latest
```

Put `.md` files in `./posts/` - they appear on your blog instantly.

## Documentation

| Topic | Description |
|---|---|
| [Setup: Docker](setup-docker) | Run with Docker and Docker Compose |
| [Setup: Binary](setup-binary) | Build or download and run locally |
| [Configuration](configuration) | All config keys, env vars, defaults |
| [Sources](sources) | Filesystem and Git post sources |
| [Writing Posts](writing-posts) | Front matter, slugs, special posts |
| [Markdown](markdown) | CommonMark and available extensions |
| [Feeds](feeds) | RSS 2.0 and Atom 1.0 configuration |
| [Templates](templates) | HTML template engine and data contract |
| [OG Images](og-images) | Auto-generated Open Graph images |
| [API](api) | REST API reference |
| [Plugins](plugins) | JavaScript plugins |
