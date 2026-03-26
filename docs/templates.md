---
title: Templates
description: HTML template engine, built-in templates, and data contract.
date: 2025-01-08
tags: [config, templates]
---

5000blogs uses Go's `html/template` engine. A single template file renders all page types (post view, list view, 404).

## Selecting a template

Set `paths.template` to a local file or URL:

```yaml
paths:
  template: "./my-template.html"
```

Default: fetched from the official GitHub repository (`template.html`, dark theme).

Template is parsed once at startup. Changes require a restart.

## Themes

Templates are written against a set of CSS custom properties (variables). A theme is a plain CSS file that defines these properties in a `:root {}` block. It is injected as the first `<style>` block in the page, before any template styles, so every rule in the template that references a variable picks up the theme value automatically.

Set a theme independently of the template:

```yaml
paths:
  template: "./my-template.html"
  theme: "./my-theme.css"
```

`paths.theme` accepts local file paths and HTTP(S) URLs. It will default to theme.base.css in the /template directory of the repo.

For the canonical variable list see [Themes](themes).

## Built-in templates

| Template | Matching theme | Description |
|---|---|---|
| `template.html` | `theme.base.css` | Dark theme, Tachyons CSS. Full-featured default |
| `template.garden.html` | `theme.garden.css` | Warm/earthy theme with serif fonts and card layout |
| `template.docs.html` | `theme.docs.css` | Light documentation theme, clean and minimal |
| `template.raw.html` | - | Unstyled skeleton with all variables. Starting point for custom templates |

You can use a URL to reference built-in variants directly:

```yaml
paths:
  template: "https://raw.githubusercontent.com/5000K/5000blogs/refs/heads/main/template/template.garden.html"
```

## Template data contract

The template receives a single `templateData` struct. All fields:

### Shared fields (all pages)

| Field | Type | Description |
|---|---|---|
| `.Title` | `string` | Page title |
| `.Description` | `string` | Meta description |
| `.URL` | `string` | Canonical page URL |
| `.OGImageURL` | `string` | Absolute og:image URL (empty if disabled) |
| `.OGLogoURL` | `string` | Absolute site logo URL |
| `.Plugins` | `[]string` | JS plugin URLs |
| `.BlogName` | `string` | Blog name from config |
| `.NavLinks` | `[]navLink` | Navigation entries (`.Name`, `.URL`) |
| `.Slug` | `string` | Current post slug (empty on list pages) |
| `.FooterContent` | `template.HTML` | Rendered footer HTML |

### Post view fields

| Field | Type | Description |
|---|---|---|
| `.DateStr` | `string` | Formatted date string |
| `.DateISO` | `string` | RFC 3339 date for `<time datetime>` |
| `.Author` | `string` | Post author |
| `.Tags` | `[]string` | Post tags |
| `.Content` | `template.HTML` | Rendered post HTML |
| `.NoIndex` | `bool` | `true` → add noindex meta tag |

### List view fields

| Field | Type | Description |
|---|---|---|
| `.IsListPage` | `bool` | `true` when rendering a list (not a single post) |
| `.SearchQuery` | `string` | Active search query (empty if none) |
| `.FilterTags` | `[]string` | Active tag filter |
| `.Posts` | `[]postListItem` | Post entries for this page |
| `.Pagination` | `paginationData` | Pagination state |

### `postListItem` fields

| Field | Type |
|---|---|
| `.Slug` | `string` |
| `.Title` | `string` |
| `.Description` | `string` |
| `.DateStr` | `string` |
| `.Author` | `string` |
| `.Tags` | `[]string` |

### `paginationData` fields

| Field | Type | Description |
|---|---|---|
| `.Page` | `int` | Current page number |
| `.TotalPages` | `int` | Total pages |
| `.TotalPosts` | `int` | Total visible posts |
| `.HasPrev` | `bool` | Previous page exists |
| `.HasNext` | `bool` | Next page exists |
| `.PrevPage` | `int` | Previous page number |
| `.NextPage` | `int` | Next page number |
| `.TagParam` | `string` | e.g. `&tags=foo,bar` for pagination links |

## Conditional rendering

Use Go template conditionals to switch between page types:

```html
{{if .IsListPage}}
  <!-- post list -->
{{else}}
  <!-- single post -->
{{end}}
```

Check for optional values:

```html
{{if .Author}}<span>by {{.Author}}</span>{{end}}
{{if .Tags}}
  {{range .Tags}}<span>{{.}}</span>{{end}}
{{end}}
```

## Plugins in templates

Plugin URLs from config are available as `.Plugins`. Load them at the bottom of your template:

```html
{{range .Plugins}}
<script src="{{.}}"></script>
{{end}}
```
