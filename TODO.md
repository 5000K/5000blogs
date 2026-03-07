## Testing gaps

- `incoming/server.go`: no integration tests for routing, 404 handling, pagination, feed, sitemap.

## Performance / reliability

- **HTTP caching headers**: post pages and feeds now set `Last-Modified` / return `304 Not Modified`. `ETag` support not yet added.
