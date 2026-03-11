# Plugins

Plugins are extensions attached to the page served via the template, that extend the website in some kind of way.

The official plugins contain a few helpers.

## link-preview.js

Attaches hover previews to all links in the page.

- **Internal links** (same origin): fetches `/api/v1/post/{slug}` and shows title, date, and description.
- **External links**: shows the base domain only.

The preview box uses `<pre>` (styled by the template) for the container and `<strong>`/`<em>`/`<code>` for content — no extra CSS needed.

## sort-tables.js

Makes all tables on the page sortable by clicking header cells. There are classes (.sort-tables-plugin-asc and .sort-tables-plugin-desc) to allow for styling, the official templates don't yet have any styling though.
