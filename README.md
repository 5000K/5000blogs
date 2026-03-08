# <img src="./template/icon.png" width="64"/> 5000blogs

## About
5000blogs is a lightweight platform for blogging and publishing markdown files.

It is intended to be

 - Easy to host - up and running without in-depth configuring
 - Minimal - only includes what you need to host a blog driven by a bunch of markdown-files
 - Modern - uses features of the modern web (e.g. social media previews, intelligent caching, automatic atom and rss feeds)
 - Customizable - the powerful template allows you to make your blog look however you like. Extend, script and style your blog to fit what you want.

# Run a demo

> assumes a terminal opened within the cloned repository, with go or docker set up

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

## Post sources

By default, 5000blogs reads posts from the `paths.posts` directory (or the `POSTS_PATH` env var).

You can instead — or in combination — pull posts from one or more **sources** declared in your config. Sources are layered in order: if two sources expose a post with the same slug, the first one wins.

**Posts from a local directory:**
```yml
sources:
  - type: filesystem
    path: ./posts
```

**Posts from a public git repository** (cloned in-memory, no checkout written to disk):
```yml
sources:
  - type: git
    url: https://github.com/yourname/blog-posts
    dir: posts  # subdirectory within the repo; omit for repo root
```

**Private repo via HTTPS token** (e.g. a GitHub personal access token):
```yml
sources:
  - type: git
    url: https://github.com/yourname/private-blog
    auth_token: ghp_yourpersonalaccesstoken
```

**Private repo via SSH key:**
```yml
sources:
  - type: git
    url: git@github.com:yourname/private-blog.git
    ssh_key_path: /home/user/.ssh/id_ed25519
    ssh_key_passphrase: optional-passphrase  # omit if key is unencrypted
```

The git source is re-pulled on every rescan, so new commits are picked up automatically without restarting 5000blogs.

<img width="931" height="602" alt="image" src="https://github.com/user-attachments/assets/4318ea7e-b160-4b03-9411-ec699383c9ad" />


# Writing posts

A post is a single markdown file dropped into the posts directory. The filename becomes the URL slug: `my-post.md` is served at `/posts/my-post`.

## Slugs

The "slug" is a name for a file derived from the file path **relative to the source root**, with directory segments joined by `+`. The server supports paths like /posts/directory/hello but internally, it uses these slugs - and it's good to know how they work for some more in-depth configuration.

| File path (relative to source root) | Slug | URL |
|---|---|---|
| `hello.md` | `hello` | `/posts/hello` |
| `more/hello.md` | `more+hello` | `/posts/more+hello` |
| `more/things/hello-world.md` | `more+things+hello-world` | `/posts/more+things+hello-world` |

Any literal `+` character in a filename or directory name is replaced with `-` to keep slugs unambiguous.

When multiple sources are configured, the first source that provides a given slug wins. This lets you override individual posts from a git source by placing a file with the same name in a local filesystem source listed earlier.

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
