---
title: Markdown Embeds
description: CommonMark support and available extensions.
date: 2025-01-07
tags: [markdown]
visible: false
---

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