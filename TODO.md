## Bugs / correctness

- **Data race in `MemoryPostRepository`**: `r.posts` is read by HTTP handlers and written by `rescan()` without any mutex. Under load this is a concurrent map/slice race. Guard `r.posts` with a `sync.RWMutex`.
- **Concurrent rescan runs**: no lock prevents two overlapping `rescan()` calls if the cron fires while a previous scan is still running.
- **Zip slip in `extractZip`** (`service/setup.go`): zip entry paths are not sanitised before joining with `destDir`. A malicious zip could write files outside the target directory. Validate that the cleaned destination path is still under `destDir`.
- **API returns hidden posts**: `apiListPosts` and `apiSearchPosts` include posts with `visible: false`. Decide whether that is intentional and document it, or filter them out.
- **Out-of-bounds page**: `GetPage` with a page number beyond the last page silently returns an empty result. Return the last valid page or a clear error instead.

## Testing gaps

- `MemoryPostRepository`: no tests for `rescan`, `GetPage`, `RSSFeed` (cache invalidation), `Sitemap`.
- `Post.Data()`, `IsVisible()`, `IsRSSVisible()`, `slugFromPath`: no unit tests.
- `FileSystemSource`: no tests (list, read, stat, write).
- `service/setup.go` (`RunInitialSetup`, `ensureTemplate`, `extractZip`): no tests.
- `incoming/api.go`: no HTTP handler tests for list, get, search, stats.
- `incoming/server.go`: no integration tests for routing, 404 handling, pagination, feed, sitemap.
- `buildFeed` / RSS sorting / item trimming: no tests.

## Performance / reliability

- **Unbounded OGImage cache**: `OGImageGenerator.cache` is a plain map with no eviction. Add a size cap or LRU eviction to bound memory usage on large blogs.
- **HTTP caching headers**: post pages and the RSS feed don't set `ETag` or `Last-Modified`. Browsers and proxies re-fetch on every navigation.
- **`rescan` replaces the whole slice every minute** even when nothing changed. After the hash-check short-circuit, avoid touching the slice at all if the set of posts is unchanged.

## Features

- **Graceful shutdown**: `incoming.Serve` does not handle OS signals. Wrap `http.Server` and call `Shutdown` on `SIGINT`/`SIGTERM` so in-flight requests finish cleanly.
- **Health check endpoint**: add `/healthz` (or `/health`) returning `200 OK` for container/load-balancer probes.
- **Template hot-reload**: `view.Renderer` parses the template once at startup; any edit needs a process restart. Support reloading on `SIGHUP` or via an admin API endpoint.
- **Tag / category metadata**: `Metadata.Raw` already captures unknown YAML keys but nothing consumes them. Add first-class `tags []string` field in `Metadata`, expose it through `PostData`, and thread it into the template data contract and sitemap.
- **Atom feed**: provide `/feed.atom` (Atom 1.0) alongside the existing RSS 2.0 feed.
- **RSS full content**: current RSS items only include `description`. Support an opt-in `rss-full-content: true` metadata flag (or global config) to include the full rendered HTML in `<content:encoded>`.
- **API write endpoints**: `WritePost` exists on `PostSource` but is never exposed over HTTP. Add `POST /api/v1/posts` and `PUT /api/v1/post/{name}` so posts can be created/updated without direct file access.
- **`TemplateURL` direct download**: setup only handles zip archives. Support a plain `.html` URL as well.

## Configuration / documentation

- **Configuration reference doc**: no `docs/config.md`. Document every config key (env var name, YAML key, default, description) — especially `rescan_cron`, `skip_unchanged_mod_time`, `template-url`, `plugins`.
- **Config validation at startup**: `config.Get` doesn't validate values (e.g. `SiteURL` must be an absolute URL, `PageSize` must be > 0). Add a `Validate()` step in `main.go` after loading.
- **Document builtin post override**: it isn't documented anywhere that placing `home.md` or `404.md` in the user's posts directory silently overrides the builtin versions.
- **`Plugins` env var**: `cleanenv` may not split a space/comma-separated string into `[]string` as expected. Verify and document the expected env format.
