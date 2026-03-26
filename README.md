# <img src="./template/icon.png" width="64"/> 5000blogs

> ! 5000blogs is nearly feature-stable, but config and API might still change. In the future, there probably will be manual migrations (mostly regarding your config and template) needed towards version 1.0. We will help with them. But they will probably be needed.

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

For full instructions, read our [Docs](https://5000blogs.5000k.org).
