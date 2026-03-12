---
title: OG Images
description: Auto-generated Open Graph images for social previews.
date: 2025-01-09
tags: [config, og-image]
---

5000blogs generates `og:image` PNGs on-the-fly for every post. These appear as preview cards on social media, messaging apps, and link previews.

## How it works

Each post has an og:image URL at `/{slug}/og-image.png`. The image is:

- 1200×630 pixels (standard Open Graph dimensions)
- Generated server-side using Go fonts (Go Bold for titles, Go Regular for descriptions)
- Cached in an LRU cache keyed by post content hash — regenerated only when content changes

The template automatically includes the og:image URL in `<meta property="og:image">` and Twitter Card tags.

## Configuration

| Key | Default | Description |
|---|---|---|
| `og_image.enabled` | `true` | Enable/disable og:image generation |
| `og_image.bg_color` | `#111111` | Background color |
| `og_image.text_color` | `#f0f0f0` | Title text color |
| `og_image.sub_color` | `#999999` | Description text color |
| `og_image.accent_color` | `#7eb8f7` | Bottom accent line color |
| `og_image.cache_size` | `128` | LRU cache capacity (number of images) |

All colors are hex values.

## Layout

The generated image contains:

1. **Top-left:** Site icon (if configured via `paths.icon`) + blog name
2. **Center:** Post title (bold, up to 4 lines, auto-wrapped)
3. **Below title:** Post description (regular, up to 3 lines)
4. **Bottom:** Accent-colored horizontal line

## Disabling

```yaml
og_image:
  enabled: false
```

When disabled, no og:image URLs are generated and the `/{slug}/og-image.png` endpoint returns 404.

## Caching

Images are cached in memory using an LRU eviction policy. The cache is keyed by the FNV-64a hash of the raw post content, so images are regenerated only when the post file changes.

Set `og_image.cache_size` based on your total post count and available memory. Each image is roughly 20-50 KB.

## HTTP caching

The og:image endpoint sets `Cache-Control: public, max-age=86400` (24 hours), so CDNs and browsers cache the image externally.
