---
title: "Setup: Binary"
description: Build from source or run the Go binary directly.
date: 2025-01-02
tags: [setup]
---

## Requirements

- Go 1.26+ (for building from source)

## Build from source

```sh
git clone https://github.com/5000K/5000blogs.git
cd 5000blogs
go build -o 5000blogs .
```

## Run

```sh
CONFIG_PATH=./config.yml ./5000blogs
```

If no config file exists at the path, 5000blogs starts with defaults (posts from `./posts/`, listening on `:8080`).

## Config file location

Set via environment variable:

```sh
CONFIG_PATH=/etc/5000blogs/config.yml ./5000blogs
```

Or use the default `config.yml` in the working directory.

## Example config

```yaml
address: ":8080"
blog_name: "My Blog"
site_url: "http://localhost:8080"
paths:
  posts: "./posts"
  template: "./template/template.html"
  icon: "./template/icon.png"
log_level: "info"
page_size: 10
feed_size: 20
feed_description: "Latest posts"
rss_content: "none"
nav_links:
  - name: "Posts"
    url: "/posts"
```

## All environment variables

Every YAML key has a corresponding env var. Env vars override config file values. See [Configuration](configuration) for the complete reference.

## Debug mode

```sh
LOG_LEVEL=debug CONFIG_PATH=./config.yml ./5000blogs
```

Produces detailed logs for post rescans, source syncs, and request handling.

## Running as a systemd service

```ini
[Unit]
Description=5000blogs
After=network.target

[Service]
ExecStart=/usr/local/bin/5000blogs
Environment=CONFIG_PATH=/etc/5000blogs/config.yml
Restart=on-failure
User=blog
Group=blog

[Install]
WantedBy=multi-user.target
```
