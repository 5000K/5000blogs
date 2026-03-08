package service

import (
	"testing"
	"time"
)

func boolPtr(v bool) *bool { return &v }

// --- slugFromSegments ---

func TestSlugFromSegments(t *testing.T) {
	cases := []struct {
		root string
		path string
		want string
	}{
		// flat file at root
		{"/posts", "/posts/hello.md", "hello"},
		// one subdirectory
		{"/posts", "/posts/more/hello.md", "more+hello"},
		// two subdirectories
		{"/posts", "/posts/more/things/hello-world.md", "more+things+hello-world"},
		// + in filename becomes -
		{"/posts", "/posts/a+b.md", "a-b"},
		// + in directory segment also becomes -
		{"/posts", "/posts/a+b/hello.md", "a-b+hello"},
		// root "."
		{".", "sub/hello.md", "sub+hello"},
		// no subdir, root == dir
		{"./demo/posts", "demo/posts/simple.md", "simple"},
	}
	for _, tc := range cases {
		if got := slugFromSegments(tc.root, tc.path); got != tc.want {
			t.Errorf("slugFromSegments(%q, %q) = %q, want %q", tc.root, tc.path, got, tc.want)
		}
	}
}

// --- IsVisible / IsRSSVisible ---

func TestIsVisible_NilMetadata(t *testing.T) {
	p := &Post{}
	if !p.IsVisible() {
		t.Error("want visible=true when metadata is nil")
	}
}

func TestIsVisible_NilField(t *testing.T) {
	p := &Post{metadata: &Metadata{}}
	if !p.IsVisible() {
		t.Error("want visible=true when Visible field is nil")
	}
}

func TestIsVisible_ExplicitTrue(t *testing.T) {
	p := &Post{metadata: &Metadata{Visible: boolPtr(true)}}
	if !p.IsVisible() {
		t.Error("want visible=true")
	}
}

func TestIsVisible_ExplicitFalse(t *testing.T) {
	p := &Post{metadata: &Metadata{Visible: boolPtr(false)}}
	if p.IsVisible() {
		t.Error("want visible=false")
	}
}

func TestIsRSSVisible_DefaultTrue(t *testing.T) {
	p := &Post{metadata: &Metadata{}}
	if !p.IsRSSVisible() {
		t.Error("want rss-visible=true by default")
	}
}

func TestIsRSSVisible_InheritHidden(t *testing.T) {
	// visible:false implies rss-visible:false regardless of RSSVisible field
	p := &Post{metadata: &Metadata{
		Visible:    boolPtr(false),
		RSSVisible: boolPtr(true),
	}}
	if p.IsRSSVisible() {
		t.Error("want rss-visible=false when visible=false")
	}
}

func TestIsRSSVisible_ExplicitFalse(t *testing.T) {
	p := &Post{metadata: &Metadata{RSSVisible: boolPtr(false)}}
	if p.IsRSSVisible() {
		t.Error("want rss-visible=false when explicitly set to false")
	}
}

// --- Post.Data() ---

func TestPostData_Fields(t *testing.T) {
	date := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	content := []byte("<p>hello</p>")
	p := NewPost("posts/my-slug.md", &Metadata{
		Title:       "My Title",
		Description: "A description",
		Date:        date,
		Author:      "Alice",
	}, content)

	d := p.Data()

	if d.Slug != "my-slug" {
		t.Errorf("Slug: got %q, want %q", d.Slug, "my-slug")
	}
	if d.Title != "My Title" {
		t.Errorf("Title: got %q", d.Title)
	}
	if d.Description != "A description" {
		t.Errorf("Description: got %q", d.Description)
	}
	if d.Author != "Alice" {
		t.Errorf("Author: got %q", d.Author)
	}
	if !d.Date.Equal(date) {
		t.Errorf("Date: got %v, want %v", d.Date, date)
	}
	if d.DateISO != "2025-06-01T00:00:00Z" {
		t.Errorf("DateISO: got %q", d.DateISO)
	}
	if string(d.Content) != "<p>hello</p>" {
		t.Errorf("Content: got %q", string(d.Content))
	}
	if !d.Visible {
		t.Error("want Visible=true by default")
	}
	if !d.RSSVisible {
		t.Error("want RSSVisible=true by default")
	}
}

func TestPostData_DateFallsBackToModTime(t *testing.T) {
	mod := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	p := &Post{
		path:     "posts/no-date.md",
		modTime:  mod,
		metadata: &Metadata{}, // no date set
	}
	d := p.Data()
	if !d.Date.Equal(mod) {
		t.Errorf("Date fallback: got %v, want %v", d.Date, mod)
	}
}

func TestPostData_NoDateISO_WhenDateZero(t *testing.T) {
	p := &Post{path: "posts/empty.md", metadata: &Metadata{}}
	d := p.Data()
	if d.DateISO != "" {
		t.Errorf("DateISO should be empty when date is zero, got %q", d.DateISO)
	}
}

func TestPostData_NoIndex(t *testing.T) {
	p := NewPost("posts/secret.md", &Metadata{NoIndex: boolPtr(true)}, nil)
	d := p.Data()
	if !d.NoIndex {
		t.Error("want NoIndex=true")
	}
}

func TestPostData_NilContents(t *testing.T) {
	p := NewPost("posts/empty.md", &Metadata{}, nil)
	d := p.Data()
	if d.Content != nil {
		t.Errorf("want nil Content for post with no contents, got %v", d.Content)
	}
}

func TestPostData_NilMetadata(t *testing.T) {
	p := &Post{path: "posts/bare.md"}
	d := p.Data()
	if d.Slug != "bare" {
		t.Errorf("Slug: got %q", d.Slug)
	}
	if !d.Visible {
		t.Error("want Visible=true when metadata is nil")
	}
}
