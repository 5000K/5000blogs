# <img src="./template/icon.png" width="64"/> 5000blogs

> ! 5000blogs is nearly feature-stable, but config and API might still change. In the future, there probably will be manual migrations (mostly regarding your config and template) needed towards version 1.0. We will help with them. But they will probably be needed.

## About
5000blogs is a lightweight platform for blogging and publishing markdown files.

You can use it either as a complete backend to drive your blog, digital garden, or documentation - or you can use it as a library to build something more custom!

This repo contains a full engine capable of driving your markdown-based blog, including a reference implementation of a full blog server that you can use directly and extend if you ever need to.

It was built and tested to be easily able to reach a score of 100 in every category of chrome lighthouse tests, automatically generate social media preview images, rss and atom feeds, sitemaps, robots.txt and anything else you might want for optimal SEO.


## Documentation
Currently, we have full documentation for using [5000blogs as a blog server](./docs/config.md). We will publish documentation for using 5000blogs as a library once our API is stable, probably once we are nearing version `1.0`.

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
5000blogs does not have any runtime dependencies and executes no writing operations, so it would be reasonable to just use a binary build. But it is recommended to use docker(-compose) for this, to keep things clean and reproducible.

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

Or using docker run:
```bash
docker run -d \
  --name 5000blogs \
  -p 8080:8080 \
  -v ./config.yml:/config.yml:ro \
  -v ./posts:/posts:ro \
  ghcr.io/5000k/5000blogs:latest
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

All other defaults should work to get you starteed, you can take a look at everything you can configure and customize [here](./docs/config.md).

For full instructions, read our [Docs](https://5000blogs.5000k.org).
