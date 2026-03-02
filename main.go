package main

import (
	"5000blogs/render"
	"fmt"
	"strings"
)

func main() {
	input := `---
title: My First Post
date: 2024-01-15
tags: [golang, markdown]
---

# Hello **World**

This is a paragraph with *italic* and **bold** text.

Here's a [link to Google](https://google.com) and a [[Wiki Link]].

## Code Example

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

Inline ` + "`code`" + ` works too.
`

	engine := render.NewEngine()

	// Optional: Set up wiki link resolver
	engine.SetWikiLinkResolver(render.WikiLinkResolverFunc(func(title string) (string, bool) {
		// Convert "Wiki Link" to "/blog/wiki-link"
		slug := strings.ToLower(strings.ReplaceAll(title, " ", "-"))
		return "/blog/" + slug, true
	}))

	html, fm, err := engine.ParseAndRender(input)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Title: %s\n", fm.Title)
	fmt.Printf("HTML:\n%s\n", html)
}
