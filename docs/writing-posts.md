---
title: Writing Posts
description: Front matter, slugs, visibility, and special posts.
date: 2025-01-06
tags: [posts]
---

## Creating a post

Add a `.md` file to your posts directory. The file is detected on the next rescan (default: every minute).

## Front matter

YAML front matter between `---` delimiters at the top of the file:

```yaml
---
title: My Post
description: A short summary.
date: 2025-06-15
author: Jane
tags: [go, tutorial]
visible: true
rss-visible: true
noindex: false
---

Post content starts here.
```

All fields are optional. Behavior when omitted:

| Field | Default | Description |
|---|---|---|
| `title` | `""` | Page title. Displayed in lists, feeds, og:image |
| `description` | `""` | Summary. Used in feed entries and `<meta>` tags |
| `date` | file mod time | Publication date. Determines sort order |
| `author` | `""` | Post author. Shown in post view and Atom entries |
| `tags` | `[]` | Categorization. Used for filtering in lists and feeds |
| `visible` | `true` | `false` hides from post list, search, feeds, and sitemap |
| `rss-visible` | `true` | `false` hides from RSS/Atom feeds only |
| `noindex` | `false` | `true` adds `<meta name="robots" content="noindex">` |

Front matter also supports arbitrary keys via the `Raw` map, accessible in custom tooling.

## Slugs

The URL slug is derived from the file path relative to the source root, without the `.md` extension:

| File path | Slug | URL |
|---|---|---|
| `hello.md` | `hello` | `/hello` |
| `guides/setup.md` | `guides/setup` | `/guides/setup` |
| `2025/my-post.md` | `2025/my-post` | `/2025/my-post` |

Slugs are the URL identity of a post. Renaming a file changes its URL.

## Special posts

Three slugs have special behavior. They have a default page built in, but can be overwritten. For this, simply define your own and the default will no longer be used. Typically, you would set Set `visible: false` on these, so that they don't appear in feeds and lists.

### `index.md`

Rendered at `/` (the home page). When no `index.md` exists, `/` shows a placeholder.

### `404.md`

Rendered when a requested URL doesn't match any post.

### `footer.md`

Content injected as the page footer on every page. Set `visible: false` and `rss-visible: false`. The built-in footer links to RSS and Atom feeds.

## Visibility rules

| `visible` | `rss-visible` | In post list | In feeds | Directly accessible |
|---|---|---|---|---|
| `true` | `true` | yes | yes | yes |
| `true` | `false` | yes | no | yes |
| `false` | any | no | no | yes |

Hidden posts (`visible: false`) are still accessible by direct URL. They are excluded from search, sitemap, and the post list.

## Plain text

Every post is available as plain text at `/plain/{slug}`. HTML tags are stripped, block elements become newlines.

## Media files

Non-markdown files in the source directory are served at `/media/{path}`. Relative links in markdown (images, downloads) are automatically rewritten to `/media/...` URLs.

```markdown
![Photo](photo.jpg)
```

Becomes `<img src="/media/photo.jpg">` (or `/media/subdir/photo.jpg` for posts in subdirectories).

## Linking between posts

Relative `.md` links are rewritten to post URLs:

```markdown
[See setup](setup.md)
```

Becomes `<a href="/setup">See setup</a>`.

Use [wikilinks](markdown#wikilinks) for title-based linking: `[[Setup: Docker]]`.
