package service

import (
	"5000blogs/config"
	"bytes"
	"image/png"
	"testing"
)

func defaultOGConfig() config.OGImageConfig {
	return config.OGImageConfig{
		Enabled:     true,
		BgColor:     "#111111",
		TextColor:   "#f0f0f0",
		SubColor:    "#999999",
		AccentColor: "#7eb8f7",
		CacheSize:   64,
	}
}

func TestOGImageGenerator_GeneratesPNG(t *testing.T) {
	gen, err := NewOGImageGenerator(defaultOGConfig(), "Test Blog", nil)
	if err != nil {
		t.Fatalf("NewOGImageGenerator: %v", err)
	}

	post := NewPost("hello.md", &Metadata{Title: "Hello World", Description: "A short description"}, []byte("<p>content</p>"))
	data, err := gen.Generate(post)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty PNG data")
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("result is not valid PNG: %v", err)
	}
	if bounds := img.Bounds(); bounds.Dx() != ogWidth || bounds.Dy() != ogHeight {
		t.Errorf("image size: want %dx%d got %dx%d", ogWidth, ogHeight, bounds.Dx(), bounds.Dy())
	}
}

func TestOGImageGenerator_Cache(t *testing.T) {
	gen, err := NewOGImageGenerator(defaultOGConfig(), "Test Blog", nil)
	if err != nil {
		t.Fatalf("NewOGImageGenerator: %v", err)
	}

	post := NewPost("cached.md", &Metadata{Title: "Cached Post"}, []byte("<p>hi</p>"))
	// Force a hash so cache key is deterministic
	post.hash = 12345

	first, err := gen.Generate(post)
	if err != nil {
		t.Fatalf("Generate first: %v", err)
	}
	second, err := gen.Generate(post)
	if err != nil {
		t.Fatalf("Generate second: %v", err)
	}
	// Same pointer means cached
	if &first[0] != &second[0] {
		t.Error("expected cached result (same slice) on second call")
	}
}

func TestOGImageGenerator_Invalidate(t *testing.T) {
	gen, err := NewOGImageGenerator(defaultOGConfig(), "Test Blog", nil)
	if err != nil {
		t.Fatalf("NewOGImageGenerator: %v", err)
	}

	post := NewPost("inv.md", &Metadata{Title: "Invalidated"}, []byte("<p>hi</p>"))
	post.hash = 99999

	if _, err := gen.Generate(post); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !gen.cache.has(post.hash) {
		t.Fatal("expected entry in cache after generate")
	}

	gen.Invalidate(post.hash)

	if gen.cache.has(post.hash) {
		t.Fatal("expected cache entry removed after Invalidate")
	}
}

func TestOGImageGenerator_NoTitle(t *testing.T) {
	gen, err := NewOGImageGenerator(defaultOGConfig(), "Test Blog", nil)
	if err != nil {
		t.Fatalf("NewOGImageGenerator: %v", err)
	}

	post := NewPost("sluggy.md", &Metadata{}, []byte{})
	data, err := gen.Generate(post)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("result is not valid PNG: %v", err)
	}
}

func TestOGImageGenerator_CacheEviction(t *testing.T) {
	cap := 5
	cfg := defaultOGConfig()
	cfg.CacheSize = cap
	gen, err := NewOGImageGenerator(cfg, "", nil)
	if err != nil {
		t.Fatalf("NewOGImageGenerator: %v", err)
	}

	// Fill the cache beyond capacity to trigger eviction.
	for i := 0; i < cap+10; i++ {
		post := NewPost("p.md", &Metadata{Title: "T"}, []byte("<p>x</p>"))
		post.hash = uint64(i)
		if _, err := gen.Generate(post); err != nil {
			t.Fatalf("Generate(%d): %v", i, err)
		}
	}

	if got := gen.cache.len(); got > cap {
		t.Errorf("cache size %d exceeds cap %d", got, cap)
	}
}

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		r, g, b uint8
	}{
		{"#111111", false, 0x11, 0x11, 0x11},
		{"#7eb8f7", false, 0x7e, 0xb8, 0xf7},
		{"111111", false, 0x11, 0x11, 0x11}, // without #
		{"#gg0000", true, 0, 0, 0},
		{"#12345", true, 0, 0, 0},
	}
	for _, tc := range tests {
		c, err := parseHexColor(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseHexColor(%q): expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseHexColor(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if c.R != tc.r || c.G != tc.g || c.B != tc.b {
			t.Errorf("parseHexColor(%q): got rgba(%d,%d,%d) want (%d,%d,%d)",
				tc.input, c.R, c.G, c.B, tc.r, tc.g, tc.b)
		}
	}
}

func TestWrapText(t *testing.T) {
	gen, err := NewOGImageGenerator(defaultOGConfig(), "", nil)
	if err != nil {
		t.Fatalf("NewOGImageGenerator: %v", err)
	}

	lines := wrapText(gen.boldFace, "short", 1040)
	if len(lines) != 1 {
		t.Errorf("short text: want 1 line got %d", len(lines))
	}

	longText := "This is a significantly longer title that should definitely wrap across multiple lines when constrained to a reasonable width"
	lines = wrapText(gen.boldFace, longText, 1040)
	if len(lines) < 2 {
		t.Errorf("long text: want >=2 lines got %d", len(lines))
	}

	if got := wrapText(gen.boldFace, "", 1040); len(got) != 0 {
		t.Errorf("empty text: want 0 lines got %d", len(got))
	}
}
