---
title: Customizing your blog
description: How the templating system works and what you can change to make the site your own.
date: 2026-03-04T11:00:00Z
author: 5000k
tags:
  - customization
  - about
---

# Customizing your blog

Every page on a 5000blogs site is produced by a single HTML template. Understanding that one file is all you need to make the site look exactly the way you want.

## The template

The template lives in your `static` folder — by default `static/template.html`. It is a standard [Go `html/template`](https://pkg.go.dev/html/template) file, which means:

- `{{.Title}}`, `{{.Content}}`, `{{.DateStr}}` etc. are replaced with real values at render time
- All output is HTML-escaped automatically unless you use `template.HTML`
- You can use `{{if}}`, `{{range}}`, and any other template action

The same file drives every page type. A boolean field — `IsListPage` — lets you branch between the post view and the post-list view inside a single template.

## Fields available in the template

**Every page:**

| Field | Type | Notes |
|---|---|---|
| `Title` | string | Post title or page title |
| `Description` | string | From the post's YAML front matter |
| `IsListPage` | bool | `true` on the `/posts` index |

**Post view** (`IsListPage` is false):

| Field | Type | Notes |
|---|---|---|
| `DateStr` | string | Formatted date, e.g. `March 4, 2026` |
| `Content` | HTML | Rendered markdown — already safe to output with `{{.Content}}` |

**List view** (`IsListPage` is true):

| Field | Type | Notes |
|---|---|---|
| `Posts` | slice | Each item has `Slug`, `Title`, `Description`, `DateStr` |
| `Pagination` | struct | `Page`, `TotalPages`, `HasPrev`, `HasNext`, `PrevPage`, `NextPage` |

## Post front matter

Every markdown post can include a YAML front matter block at the top:

```
---
title: My post title
description: A short summary shown in the post list.
date: 2026-03-04T11:00:00Z
---
```

Place it at the very start of the file, enclosed by `---` delimiters. The parser extracts and removes it from the rendered output automatically.

## Styling

The demo uses [Tachyons](https://tachyons.io) for layout and spacing utilities, plus a small `<style>` block in the template for prose typography (headings, code blocks, blockquotes). Swap in any CSS framework or roll your own — the template is yours.
