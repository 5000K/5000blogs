---
title: API
description: REST API reference for programmatic access.
date: 2025-01-10
tags: [api]
---

All API endpoints are mounted at `/api/v1`. Responses are JSON.

## Endpoints

### List posts

Returns an array of post slugs for all visible posts. Results match what is shown on the default post list (same visibility and ordering rules).

```
GET /api/v1/posts
GET /api/v1/posts?tags=tag1,tag2
GET /api/v1/posts?q=Query%20Text
GET /api/v1/posts?q=Query%20Text&tags=tag1,tag2
```

- `tags` - comma-separated list; returns posts that have at least one matching tag
- `q` - case-insensitive full-text search on title, description, and body

Example response:

```json
["hello-world", "getting-started", "my-second-post"]
```

### List posts (paginated)

Same filtering as `/api/v1/posts`, but paginated. Page size matches the configured `page_size`.

```
GET /api/v1/posts/page/1
GET /api/v1/posts/page/1?tags=tag1,tag2
GET /api/v1/posts/page/1?q=Query%20Text
GET /api/v1/posts/page/1?q=Query%20Text&tags=tag1,tag2
```

Example response:

```json
{
  "posts": [
    {
      "slug": "hello-world",
      "title": "Hello World",
      "description": "My first post",
      "date": "2025-06-15T00:00:00Z",
      "author": "Jane",
      "tags": ["go", "tutorial"]
    }
  ],
  "page": 1,
  "page_size": 10,
  "total_posts": 42,
  "total_pages": 5,
  "has_prev": false,
  "has_next": true
}
```

### Get post metadata

Returns metadata of a single post by slug. Returns `404` when the slug is not found.

```
GET /api/v1/post/{slug}
```

Example response:

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

### List tags

Returns a sorted array of all tags used across visible posts.

```
GET /api/v1/tags
```

Example response:

```json
["go", "rust", "tutorial", "web"]
```

### Basic stats

Returns basic statistics about the server.

```
GET /api/v1/stats
```

Example response:

```json
{
  "last_change": "2025-06-15T12:34:56Z",
  "visible_post_count": 42
}
```

- `last_change` - timestamp of the most recently modified visible post
- `visible_post_count` - number of posts with `visible: true` (default)

## Other relevant HTTP endpoints

These are not part of the API but useful for integration:

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
