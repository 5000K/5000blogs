## Testing gaps

- `incoming/server.go`: no integration tests for routing, 404 handling, pagination, feed, sitemap.

## Performance / reliability

- **Unbounded OGImage cache**: `OGImageGenerator.cache` is a plain map with no eviction. Add a size cap or LRU eviction to bound memory usage on large blogs.
- **HTTP caching headers**: post pages and feeds now set `Last-Modified` / return `304 Not Modified`. `ETag` support not yet added.
- **`rescan` replaces the whole slice every minute** even when nothing changed. After the hash-check short-circuit, avoid touching the slice at all if the set of posts is unchanged.
