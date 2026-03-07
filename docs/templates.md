# Templates

## Engine

Templates use Go's [`html/template`](https://pkg.go.dev/html/template) package. Refer to that documentation for all valid syntax (actions, pipelines, conditionals, range, etc.). All output is HTML-escaped by default; the sole exception is `.Content`, which is injected as `template.HTML` (pre-sanitized).

## File location

`{config.paths.static}/template.html` — parsed once at process startup. Template changes require a process restart.

## Data contract

A single value of the following shape is passed to the template on every request:

```
.Title        string        — page or post title
.Description  string        — meta description; empty string if unset
.URL          string        — canonical absolute URL of the current page
.NoIndex      bool          — true when the post has `noindex: true` in its YAML metadata

.IsListPage   bool          — true on /posts list pages, false on single-post pages

.Plugins      []string      — list of JavaScript URLs from config `plugins`; empty slice if unset
```

### Single-post view (`IsListPage = false`)

```
.DateStr      string        — formatted date ("January 2, 2006"); empty string if unset
.DateISO      string        — machine-readable date in RFC 3339 (for <time datetime>); empty if unset
.Author       string        — author name; empty string if unset
.Content      template.HTML — fully rendered HTML from Markdown source; never HTML-escaped
```

### List view (`IsListPage = true`)

```
.Posts        []postListItem
  .Slug         string      — URL slug (filename without .md); used in href="/posts/{{.Slug}}"
  .Title        string      — post title; falls back to .Slug if empty
  .Description  string      — empty string if unset
  .DateStr      string      — formatted date ("January 2, 2006"); empty string if unset
  .Author       string      — empty string if unset

.Pagination   paginationData
  .Page         int         — current page (1-based)
  .TotalPages   int
  .TotalPosts   int
  .HasPrev      bool
  .HasNext      bool
  .PrevPage     int         — valid only when HasPrev = true
  .NextPage     int         — valid only when HasNext = true
```

## Notes

- All optional string fields are empty strings when absent — use `{{if .Field}}` to guard them.
- `.NoIndex` is `false` by default; set `noindex: true` in a post's YAML block to suppress indexing.
- `.URL` is the full canonical URL (e.g. `https://example.com/posts/my-post`); empty on pages where `site_url` is not configured.
- `.Pagination` is present on every list render; guard nav rendering with `{{if or .Pagination.HasPrev .Pagination.HasNext}}`.
- `.Plugins` holds the URLs configured in `plugins` (YAML array). Render them as `<script>` tags with `{{range .Plugins}}<script src="{{.}}"></script>{{end}}`. The list is empty when no plugins are configured.
- The `brand.html` file in the same static directory is not injected server-side; the demo template fetches it client-side via `fetch('/static/brand.html')`.
