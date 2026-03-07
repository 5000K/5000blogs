# REST API

All endpoints are mounted under `/api/v1` and return `application/json`.

## GET /api/v1/posts

Returns an unordered array of all post slugs (visible and hidden).

```json
["introduction", "about", "customizing-your-blog"]
```

## GET /api/v1/post/{name}

Returns the full metadata for one post. `name` is the slug (filename without `.md`).

```json
{
  "slug": "introduction",
  "title": "Introduction",
  "description": "Welcome to the blog.",
  "date": "2024-01-01T00:00:00Z",
  "author": "Alice",
  "visible": true,
  "rss_visible": true,
  "noindex": false
}
```

Returns `404` when the slug does not exist. `description`, `author`, and `noindex` are omitted from the response when empty/false.

## GET /api/v1/posts/search?q={query}

Returns **visible** posts whose `title` or `description` contain `query` (case-insensitive). Returns an empty array when nothing matches. Posts with `visible: false` are excluded.

```json
[
  { "slug": "introduction", "title": "Introduction", "description": "Welcome to the blog." }
]
```

`description` is omitted when empty.

## GET /api/v1/stats

Returns aggregate stats for visible posts only.

```json
{
  "total_posts": 3,
  "latest_post_date": "2024-06-15T00:00:00Z"
}
```

`latest_post_date` is omitted when no posts exist.
