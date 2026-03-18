---
title: Markdown
description: CommonMark support and available extensions.
date: 2025-01-07
tags: [markdown]
---

5000blogs uses [goldmark](https://github.com/yuin/goldmark), a CommonMark-compliant markdown parser. All standard CommonMark syntax is supported. Extensions are toggled via config.

## CommonMark basics

### Headings

```markdown
# Heading 1
## Heading 2
### Heading 3
```

Auto-generated `id` attributes for anchor linking (e.g. `<h2 id="heading-2">`).

### Emphasis

```markdown
*italic* or _italic_
**bold** or __bold__
***bold italic***
```

### Links and images

```markdown
[Link text](https://example.com)
![Alt text](image.jpg)
```

Relative `.md` links are rewritten to post URLs. Relative non-markdown links become `/media/...` URLs. See [Writing Posts: Media files](writing-posts#media-files).

### Code

Inline: `` `code` ``

Fenced blocks:

````markdown
```go
fmt.Println("hello")
```
````

### Blockquotes

```markdown
> This is a blockquote.
> It can span multiple lines.
```

### Lists

```markdown
- Unordered item
- Another item

1. Ordered item
2. Another item
```

### Horizontal rules

```markdown
---
```

(Use blank lines around it to avoid confusion with front matter.)

### HTML

Raw HTML in markdown is passed through as-is.

---

## Extensions

Toggled in config under `features`. See [Configuration: Features](configuration#features).

### Tables

**Config:** `features.tables: true` (default: on)

GFM-style pipe tables:

```markdown
| Name  | Value |
|-------|-------|
| Alpha | 1     |
| Beta  | 2     |
```

Alignment with `:`:

```markdown
| Left | Center | Right |
|:-----|:------:|------:|
| a    | b      | c     |
```

### Strikethrough

**Config:** `features.strikethrough: true` (default: on)

```markdown
~~deleted text~~
```

### Wikilinks

**Config:** `features.wiki_links: true` (default: on)

Link to posts by title using `[[Title]]` syntax:

```markdown
See [[Setup: Docker]] for container instructions.
```

Resolution: the title is matched against all loaded posts. If found, links to that post's slug. If not found, falls back to a URL-encoded path.

Compatible with Obsidian-style wikilinks.

### Embedded posts

Inline another post's rendered HTML at any point in a post. Two syntaxes are supported:

**Wikilink-style** (requires `features.wiki_links: true`):

```markdown
![[Post Title]]
```

The title is resolved against all loaded posts. If the title matches a known post, its HTML is inlined. If not, renders as a regular image (fallback to `WikiImage`).

**Relative-path image syntax** (always available):

```markdown
![](./relative/path/to/post.md)
```

Any `![...](path.md)` where the destination ends in `.md` is treated as an embed. The path is resolved relative to the current post's directory.

If the target post is not yet rendered (e.g. during initial load), it is rendered on demand. Recursive embeds (A embeds B which embeds A) are detected; the cycle is broken with an HTML comment in place of the nested embed.

### Autolinks

**Config:** `features.autolinks: false` (default: off)

Bare URLs are automatically converted to clickable links:

```markdown
Visit https://example.com for details.
```

### Task lists

**Config:** `features.task_list: false` (default: off)

```markdown
- [x] Completed task
- [ ] Pending task
```

Renders as checkboxes.

### Footnotes

**Config:** `features.footnotes: false` (default: off)

```markdown
This has a footnote[^1].

[^1]: Footnote content here.
```

Renders as superscript references with a footnote section at the bottom.

---

## Link rewriting

Relative links in markdown are rewritten automatically:

| Markdown | Rendered `href` |
|---|---|
| `[text](other.md)` | `/other` |
| `[text](sub/page.md)` | `/sub/page` |
| `[text](image.png)` | `/media/image.png` |
| `[text](#section)` | `#section` (unchanged) |
| `[text](https://...)` | `https://...` (unchanged) |

For posts in subdirectories, relative paths resolve relative to that subdirectory.
