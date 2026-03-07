## Testing gaps

- `incoming/server.go`: no integration tests for routing, 404 handling, pagination, feed, sitemap.

## Performance / reliability

- **Unbounded OGImage cache**: `OGImageGenerator.cache` is a plain map with no eviction. Add a size cap or LRU eviction to bound memory usage on large blogs.
- **HTTP caching headers**: post pages and feeds now set `Last-Modified` / return `304 Not Modified`. `ETag` support not yet added.
- **`rescan` replaces the whole slice every minute** even when nothing changed. After the hash-check short-circuit, avoid touching the slice at all if the set of posts is unchanged.

## Features

## Configuration / documentation

- **Configuration reference doc**: no `docs/config.md`. Document every config key (env var name, YAML key, default, description) — especially `rescan_cron`, `skip_unchanged_mod_time`, `template-url`, `plugins`.
- **Config validation at startup**: `config.Get` doesn't validate values (e.g. `SiteURL` must be an absolute URL, `PageSize` must be > 0). Add a `Validate()` step in `main.go` after loading.
- **Document builtin post override**: it isn't documented anywhere that placing `home.md` or `404.md` in the user's posts directory silently overrides the builtin versions.
- **`Plugins` env var**: `cleanenv` may not split a space/comma-separated string into `[]string` as expected. Verify and document the expected env format.
