# <img src="./template/icon.png" width="64"/> 5000blogs

## About
5000blogs is a lightweight platform for blogging and publishing markdown files.

It is intended to be

 - Easy to host - up and running without in-depth configuring
 - Minimal - only includes what you need to host a blog driven by a bunch of markdown-files
 - Modern - uses features of the modern web (e.g. social media previews, intelligent caching, automatic atom and rss feeds)
 - Customizable - uses a powerful template to allow you to make your blog look however you like. Extend, script and style your blog to fit what you want.

# Run a demo

## Run the demo locally
```bash
CONFIG_PATH=./config.demo.yml go run .
```

## Run the demo with Docker
```bash
docker compose up --build
```


# Deploy your own blog
5000blogs does not have any runtime dependencies, so you could also just use a binary build. But it is highly recommended to use docker-compose for this, to keep things clean and reproducible.

Here is a simple docker-compose.yml, that simply mounts a config and a posts folder:
```yml
services:
  5000blogs:
    image: ghcr.io/5000k/5000blogs:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yml:/config.yml:ro
      - ./posts:/posts:ro
```

And here is a simple config.yml that you can tweak:
```yml
blog_name: Blog

site_url: http://localhost:8080 # the URL your blog will be available under. Very relevant for a functional RSS-feed!

# These are quick links that will be shown in the top bar.
# Use this to link to whatever are the most important pages on your site.
# These could also be links to other websites.
nav_links:
  - name: 'Posts'
    url: '/posts'
```

All other defaults should work for the vast majority of blogs, but you can take a look at everything you can configure [here](./docs/config.md).

<img width="931" height="602" alt="image" src="https://github.com/user-attachments/assets/4318ea7e-b160-4b03-9411-ec699383c9ad" />


# Writing posts

A post is a single markdown file dropped into the posts directory. The filename becomes the URL slug: `my-post.md` is served at `/posts/my-post`.

Simply add a new file in your posts folder ending in `.md`. 5000blogs rescans your folder every minute, so it will automatically update changes and detect new posts without you needing to restart it!

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
| `date` | `YYYY-MM-DD` or `YYYY-MM-DDTHH:MM:SSZ` | recommended | Publication date (and optional time); posts are sorted by this field. Add a time component to order multiple posts on the same day. |
| `author` | string | no | Shown on the post page |
| `tags` | list of strings | no | Used for categorisation; visible in templates |
| `visible` | bool | no | Set to `false` to hide from all lists (including RSS/Atom) (default: `true`) |
| `rss-visible` | bool | no | Set to `false` to exclude from RSS/Atom specifically (default: `true`) |
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

## Special posts
There are a few posts that are kind of *special*. They are called *well-known posts*, and here they are:

 - home.md - The content of this post will be shown on the main page of your blog. It's your landing page.
 - 404.md - The content of this post will be shown if the user tries to access a post that doesn't exist.
 - footer.md - The content of the footer shown on every page.

There are defaults for these pages, but if you want to fill them with your own content, just add a file with the correct name in your posts folder.

# Roadmap
 - Better image support
 - CLI-version allowing pre-rendering a full blog into a static webpage
 - Write additional templates to cater towards different styles of blogs
 - Rework post indexing to be more performant (probably by utilizing bleve)
 - Write a few basic client-plugins:
   - Make table-columns sortable
   - Mermaid-support using codeblocks
   - Code-highlighting
