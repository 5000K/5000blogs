---
title: Themes
description: CSS variable reference for the 5000blogs theming system.
date: 2026-03-26
tags: [config, templates, themes]
---

A theme is a `:root { }` CSS block injected into the page before any template styles. Templates are written against these variables, so swapping the theme block changes the look without touching the template.

## Applying a theme

Set `paths.theme` to a local CSS file or URL:

```yaml
paths:
  theme: "./my-theme.css"
```

The file should contain a single `:root { }` block that defines the variables below. It should define all variables to be portable across different templates. For dynamic light/dark mode support, multiple `:root` blocks are reasonable.

## Built-in themes

| Theme file | Description |
|---|---|
| `theme.base.css` | Adaptive light/dark - blue accent, matches `template.html` |
| `theme.garden.css` | Warm earthy dark theme - serif headings, amber accent, matches `template.garden.html` |
| `theme.docs.css` | Light documentation theme - indigo accent, matches `template.docs.html` |

Reference a built-in theme by URL:

```yaml
paths:
  theme: "https://raw.githubusercontent.com/5000K/5000blogs/refs/heads/main/template/theme.base.css"
```

## Variable reference

### Colors

| Variable | Purpose |
|---|---|
| `--color-bg` | Main page/body background |
| `--color-surface` | Slightly elevated surface - card backgrounds, banded sections, table header cells |
| `--color-border` | Dividers, rule lines, table and code block outlines |
| `--color-text` | Default body and prose text |
| `--color-text-heading` | Headings and high-emphasis text |
| `--color-text-muted` | Secondary labels - dates, author lines, descriptions |
| `--color-text-dim` | Tertiary / very subtle text - page counters, section separators, tag row labels |
| `--color-header-bg` | Site header bar background |
| `--color-header-border` | Header bottom border |
| `--color-header-brand` | Site name / brand text in the header |
| `--color-header-nav` | Navigation link text in the header |
| `--color-accent` | Primary interactive color - links, blockquote accents, focus rings |
| `--color-accent-hover` | Hover / active state of accent elements |
| `--color-tag-bg` | Tag badge background |
| `--color-tag-text` | Tag badge text |
| `--color-code-bg` | Inline code and code block background |
| `--color-code-text` | Code text |

### Layout

| Variable | Purpose |
|---|---|
| `--content-width` | Max width of the main content column (e.g. `52rem`) |
| `--spacing-page-h` | Horizontal page padding at narrow viewports |
| `--spacing-page-h-wide` | Horizontal page padding at wider viewports |
| `--spacing-page-v` | Vertical padding at the top and bottom of the content area |
| `--gap-base` | Minimum meaningful gap between elements. Used directly or as a multiplier (e.g. `calc(var(--gap-base) * 2)`) for spacing between nav items, form controls, list rows, and similar compound layouts |

### Shape

| Variable | Purpose |
|---|---|
| `--radius-sm` | Corner radius for small elements - inline code, tag badges, inputs |
| `--radius-md` | Corner radius for mid-size elements - buttons, search box, post cards |
| `--radius-lg` | Corner radius for large elements - code blocks, image frames |

### Typography

| Variable | Purpose |
|---|---|
| `--font-body` | Font stack for body text and UI elements |
| `--font-heading` | Font stack for headings (`h1`-`h4`) |
| `--font-mono` | Font stack for code and pre blocks |
| `--font-size-base` | Root font size (cascades via `rem`). Default `1rem` / `16px` |

## Example theme file

```css
/* my-theme.css - minimal light theme */
:root {
  /* Colors */
  --color-bg:             #ffffff;
  --color-surface:        #f9fafb;
  --color-border:         #e5e7eb;
  --color-text:           #374151;
  --color-text-heading:   #111827;
  --color-text-muted:     #6b7280;
  --color-text-dim:       #9ca3af;
  --color-header-bg:      #111827;
  --color-header-border:  #374151;
  --color-header-brand:   #f9fafb;
  --color-header-nav:     #9ca3af;
  --color-accent:         #4f46e5;
  --color-accent-hover:   #4338ca;
  --color-tag-bg:         #e0e7ff;
  --color-tag-text:       #4f46e5;
  --color-code-bg:        #f3f4f6;
  --color-code-text:      #1f2937;

  /* Layout */
  --content-width:        52rem;
  --spacing-page-h:       1rem;
  --spacing-page-h-wide:  1.5rem;
  --spacing-page-v:       2.5rem;
  --gap-base:             0.5rem;

  /* Shape */
  --radius-sm:            0.25rem;
  --radius-md:            0.375rem;
  --radius-lg:            0.5rem;

  /* Typography */
  --font-body:            system-ui, Helvetica, Arial, sans-serif;
  --font-heading:         system-ui, Helvetica, Arial, sans-serif;
  --font-mono:            'SFMono-Regular', Menlo, Consolas, monospace;
  --font-size-base:       1rem;
}
```
