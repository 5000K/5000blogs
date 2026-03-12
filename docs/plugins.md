---
title: Plugins
description: JavaScript plugins for enhanced functionality.
date: 2025-01-11
tags: [config, plugins]
---

Plugins are client-side JavaScript files loaded on every page. Configure them as a list of URLs:

```yaml
plugins:
  - "https://example.com/my-plugin.js"
```

Each URL is injected as a `<script>` tag in the template.

## Official plugins

### Link Preview

Hover previews for internal and external links.

- **Internal links:** Fetches post metadata via the API (`/api/v1/post/{slug}`) and shows a tooltip with title and description
- **External links:** Shows the link URL in a preview tooltip

```yaml
plugins:
  - "https://github.com/5000K/5000blogs/releases/latest/download/link-preview.js"
```

### Sort Tables

Click any table header cell to sort the table by that column. Supports ascending/descending toggle. Sorts numerically when column values are numbers.

```yaml
plugins:
  - "https://github.com/5000K/5000blogs/releases/latest/download/sort-tables.js"
```

## Using both

```yaml
plugins:
  - "https://github.com/5000K/5000blogs/releases/latest/download/sort-tables.js"
  - "https://github.com/5000K/5000blogs/releases/latest/download/link-preview.js"
```

## Custom plugins

Any JavaScript file served over HTTP can be used as a plugin. The script runs in the context of the rendered page with full DOM access. Use the [API](api) for server-side data access.

## Template integration

Plugin URLs are available in the template as `.Plugins`:

```html
{{range .Plugins}}
<script src="{{.}}"></script>
{{end}}
```

All built-in templates support this. Custom templates should include the same loop.
