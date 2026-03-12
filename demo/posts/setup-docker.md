---
title: "Setup: Docker"
description: Run 5000blogs using Docker or Docker Compose.
date: 2025-01-01
tags: [setup, docker]
---

## Docker image

```
ghcr.io/5000k/5000blogs:latest
```

The container expects:

- A config file mounted at the path set by `CONFIG_PATH` (default: `/config.yml`)
- A posts directory mounted at the path set in `paths.posts` (default: `/posts/`)

## Minimal run

```sh
docker run -p 8080:8080 \
  -v ./config.yml:/config.yml:ro \
  -v ./posts:/posts:ro \
  ghcr.io/5000k/5000blogs:latest
```

## Docker Compose

```yaml
services:
  blog:
    image: ghcr.io/5000k/5000blogs:latest
    ports:
      - "8080:8080"
    environment:
      CONFIG_PATH: /config.yml
    volumes:
      - ./config.yml:/config.yml:ro
      - ./posts:/posts:ro
```

## Example config for Docker

When running in Docker, paths refer to container paths:

```yaml
address: ":8080"
blog_name: "My Blog"
site_url: "https://example.com"
paths:
  posts: "/posts"
page_size: 10
feed_size: 20
feed_description: "Latest posts"
rss_content: "none"
```

Template and icon default to fetching from the official GitHub repository. Override with local paths if you mount your own:

```yaml
paths:
  template: "/static/template.html"
  icon: "/static/icon.png"
```

## Environment variable overrides

Every config key can be set via environment variables. Useful in Docker:

```yaml
services:
  blog:
    image: ghcr.io/5000k/5000blogs:latest
    ports:
      - "8080:8080"
    environment:
      CONFIG_PATH: /config.yml
      BLOG_NAME: "My Blog"
      LOG_LEVEL: "debug"
      SITE_URL: "https://example.com"
    volumes:
      - ./posts:/posts:ro
```

See [Configuration](configuration) for the full list of env vars.

## Health check

The `/health` endpoint returns `200 ok`. Use it in Compose:

```yaml
services:
  blog:
    image: ghcr.io/5000k/5000blogs:latest
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
```

## Git sources in Docker

When using git sources with SSH keys, mount the key file:

```yaml
services:
  blog:
    image: ghcr.io/5000k/5000blogs:latest
    volumes:
      - ./config.yml:/config.yml:ro
      - ~/.ssh/id_ed25519:/keys/id_ed25519:ro
```

Then reference it in config:

```yaml
sources:
  - type: git
    url: "git@github.com:user/posts.git"
    ssh_key_path: "/keys/id_ed25519"
```
