# OG Image Generation

Automatically generates a `og:image` PNG for each post at `/posts/{slug}/og-image.png`.

## Configuration (in your main config.yml)

```yaml
og_image:
  enabled: true              # set to false to disable entirely
  blog_name: 'My Blog'       # displayed top-left on the image
  blog_icon: './static/icon.png'  # optional path to a PNG to show next to blog_name
  bg_color: '#111111'        # background color
  text_color: '#f0f0f0'      # post title color
  sub_color: '#999999'       # description + blog name color
  accent_color: '#7eb8f7'    # bottom accent line color
```

These defaults match the built-in template's color scheme.

## Template

When enabled, `.OGImageURL` is set to the absolute image URL for single-post pages. Use it in templates:

```html
{{if .OGImageURL}}<meta property="og:image" content="{{.OGImageURL}}">{{end}}
```

If you are using the built-in `template.html`, you don't need to do anything.

## Caching

Images are generated on first request and cached in memory. The cache for a post is invalidated automatically when the post's content changes.
