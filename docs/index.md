---
title: 5000blogs Documentation
description: Complete documentation for 5000blogs - a file-backed markdown blog engine.
visible: false
rss-visible: false
---

## What it is

5000blogs is a single-binary server for serving markdown files. The project focus is on doing one thing well: serving static markdowns. It is opiniated towards blogs - with features like feed generation, author and date metadata, or social media previews. But can be used to serve all kinds of "a bunch of markdown files" - like this docs page.

## What it does

- Serves `.md` files as HTML pages with zero build step
- Serves all other files next to your markdown files as-is
- Periodically rescans for new/changed/deleted posts (configurable cron)
- RSS 2.0 and Atom 1.0 feeds with tag/search filtering
- Full-text search via built-in indexing
- Auto-generated social media previews, `sitemap.xml`, `robots.txt`
- Git repositories as post sources (with SSH/token auth) - push changes, and 5000blogs updates automatically
- Wikilinks, tables, footnotes, and other CommonMark extensions
- Pluggable HTML templates and JS plugins
- REST API for programmatic access

## Quick start

```yaml
# config.yml
blog_name: Blog
site_url: http://localhost:8080
nav_links:
  - name: 'Posts'
    url: '/posts'
```

```sh
docker run \
  -p 8080:8080 \
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
| [Themes](themes) | Theming of templates |
| [OG Images](og-images) | Auto-generated Open Graph images |
| [API](api) | REST API reference |
| [Plugins](plugins) | JavaScript plugins |
