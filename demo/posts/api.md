---
title: API
description: REST API reference for programmatic access.
date: 2025-01-10
tags: [api]
---

All API endpoints are mounted at `/api/v1`. Responses are JSON.

## Endpoints

### List posts

```
GET /api/v1/posts
GET /api/v1/posts?tags=go,docker
```

Returns an array of post slugs. Optional `tags` parameter filters by tag (OR logic).

**Response:**

```json
["hello-world", "guides/setup", "about"]
```

### Get post metadata

```
GET /api/v1/post/{slug}
```

Returns full metadata for a single post. Nested slugs use path syntax: `/api/v1/post/guides/setup`.

**Response:**

```json
{
  "slug": "hello-world",
  "title": "Hello World",
  "description": "My first post",
  "date": "2025-06-15T00:00:00Z",
  "author": "Jane",
  "tags": ["go", "tutorial"],
  "visible": true,
  "rss_visible": true,
  "noindex": false
}
```

### Search posts (substring)

```
GET /api/v1/posts/search?q=setup
```

Case-insensitive substring search on title and description. Returns only visible posts.

**Response:**

```json
[
  {
    "slug": "guides/setup",
    "title": "Setup Guide",
    "description": "How to set up the blog"
  }
]
```

### Full-text search

```
GET /api/v1/search?q=docker+compose
```

Full-text search across title, description, and post body content. Returns matching slugs.

**Response:**

```json
["setup-docker", "configuration"]
```

### List tags

```
GET /api/v1/posts/tags
```

Returns a sorted list of all tags across visible posts.

**Response:**

```json
["config", "docker", "go", "setup", "tutorial"]
```

### Stats

```
GET /api/v1/stats
```

Aggregate blog statistics.

**Response:**

```json
{
  "total_posts": 42,
  "latest_post_date": "2025-06-15T00:00:00Z"
}
```

`latest_post_date` is `null` when no posts have dates.

## Other HTTP endpoints

These are not part of the API but are useful for integration:

| Endpoint | Description |
|---|---|
| `/health` | Returns `200 ok` (plain text) |
| `/feed.xml` | RSS 2.0 feed |
| `/feed.atom` | Atom 1.0 feed |
| `/sitemap.xml` | XML sitemap of all visible posts |
| `/robots.txt` | Auto-generated, includes sitemap URL - can be overwritten by adding a robots.txt to your own post source |
| `/plain/{slug}` | Plain text version of a post |
| `/media/{path}` | Media files from post sources |
| `/{slug}/og-image.png` | Generated OG image for a post |
