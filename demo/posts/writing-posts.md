---
title: Writing posts
description: How to create a post — valid front matter fields, visibility controls, and markdown basics.
date: 2026-03-07
author: 5000k
tags:
  - guide
  - writing
---

# Writing posts

A post is a single markdown file dropped into the posts directory. The filename becomes the URL slug: `my-post.md` is served at `/posts/my-post`.

## Front matter

Every post starts with a YAML front matter block between `---` delimiters. It must be the very first thing in the file.

```yaml
---
title: My Post Title
description: A short sentence shown in lists and used as the meta description.
date: 2026-03-07
author: Alice
tags:
  - go
  - backend
---
```

### Supported fields

| Field | Type | Required | Description |
|---|---|---|---|
| `title` | string | yes | Displayed in the page heading and browser tab |
| `description` | string | recommended | Used in post lists, RSS, and `<meta name="description">` |
| `date` | `YYYY-MM-DD` | recommended | Publication date; posts are sorted by this field |
| `author` | string | no | Shown on the post page |
| `tags` | list of strings | no | Used for categorisation; visible in templates |
| `visible` | bool | no | Set to `false` to hide from all lists and RSS (default: `true`) |
| `rss-visible` | bool | no | Set to `false` to exclude from RSS only (default: `true`) |
| `noindex` | bool | no | Set to `true` to add `<meta name="robots" content="noindex">` |

Any extra field you add lands in the raw metadata map and is accessible in custom templates.

### Hiding a post

```yaml
---
title: Draft post
visible: false
---
```

To publish the page but keep it out of the RSS feed:

```yaml
---
title: Unlisted post
rss-visible: false
---
```

## Writing the content

Everything after the closing `---` is standard markdown. A few notes:

- Use a single `# H1` heading at the top; it doubles as the visible page title.
- Fenced code blocks with a language tag get syntax-highlighted by whatever stylesheet your template loads.
- Relative links between posts work fine: `[see also](/posts/introduction)`.

For a full markdown reference see the [CommonMark spec](https://spec.commonmark.org/current/) — that is the dialect 5000blogs uses, with automatic heading IDs added.
