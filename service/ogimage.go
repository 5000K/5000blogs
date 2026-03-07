package service

import (
	"5000blogs/config"
	"bytes"
	"container/list"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"os"
	"strconv"
	"strings"
	"sync"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	ogWidth  = 1200
	ogHeight = 630
)

// ogLRUCache is a fixed-capacity LRU cache keyed by post hash.
type ogLRUCache struct {
	mu    sync.Mutex
	cap   int
	items map[uint64]*list.Element
	list  *list.List
}

type ogLRUEntry struct {
	key  uint64
	data []byte
}

func newOGLRUCache(cap int) *ogLRUCache {
	return &ogLRUCache{
		cap:   cap,
		items: make(map[uint64]*list.Element, cap),
		list:  list.New(),
	}
}

func (c *ogLRUCache) get(key uint64) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.list.MoveToFront(el)
	return el.Value.(*ogLRUEntry).data, true
}

func (c *ogLRUCache) set(key uint64, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.list.MoveToFront(el)
		el.Value.(*ogLRUEntry).data = data
		return
	}
	if c.list.Len() >= c.cap {
		oldest := c.list.Back()
		if oldest != nil {
			c.list.Remove(oldest)
			delete(c.items, oldest.Value.(*ogLRUEntry).key)
		}
	}
	el := c.list.PushFront(&ogLRUEntry{key: key, data: data})
	c.items[key] = el
}

func (c *ogLRUCache) delete(key uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.list.Remove(el)
		delete(c.items, key)
	}
}

func (c *ogLRUCache) has(key uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.items[key]
	return ok
}

func (c *ogLRUCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.list.Len()
}

// OGImageGenerator generates og:image PNGs for posts and caches them by post hash.
type OGImageGenerator struct {
	cfg         config.OGImageConfig
	blogName    string
	boldFace    font.Face
	regularFace font.Face
	smallFace   font.Face
	icon        image.Image // optional, pre-loaded

	cache *ogLRUCache
}

// NewOGImageGenerator creates a generator from the given config.
func NewOGImageGenerator(cfg config.OGImageConfig, blogName, iconPath string) (*OGImageGenerator, error) {
	boldFont, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return nil, fmt.Errorf("ogimage: parse bold font: %w", err)
	}
	regularFont, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("ogimage: parse regular font: %w", err)
	}

	boldFace, err := opentype.NewFace(boldFont, &opentype.FaceOptions{Size: 52, DPI: 72})
	if err != nil {
		return nil, fmt.Errorf("ogimage: create bold face: %w", err)
	}
	regularFace, err := opentype.NewFace(regularFont, &opentype.FaceOptions{Size: 22, DPI: 72})
	if err != nil {
		return nil, fmt.Errorf("ogimage: create regular face: %w", err)
	}
	smallFace, err := opentype.NewFace(regularFont, &opentype.FaceOptions{Size: 32, DPI: 72})
	if err != nil {
		return nil, fmt.Errorf("ogimage: create small face: %w", err)
	}

	g := &OGImageGenerator{
		cfg:         cfg,
		blogName:    blogName,
		boldFace:    boldFace,
		regularFace: regularFace,
		smallFace:   smallFace,
		cache:       newOGLRUCache(cfg.CacheSize),
	}

	if iconPath != "" {
		g.icon = loadIcon(iconPath)
	}

	return g, nil
}

// Generate returns a PNG for the post, using a cache keyed by post hash.
// The cache is invalidated automatically when the post's content hash changes.
func (g *OGImageGenerator) Generate(post *Post) ([]byte, error) {
	if data, ok := g.cache.get(post.hash); ok {
		return data, nil
	}

	data, err := g.generate(post)
	if err != nil {
		return nil, err
	}

	g.cache.set(post.hash, data)
	return data, nil
}

func (g *OGImageGenerator) generate(post *Post) ([]byte, error) {
	bgColor, err := parseHexColor(g.cfg.BgColor)
	if err != nil {
		return nil, fmt.Errorf("ogimage: bg_color: %w", err)
	}
	textColor, err := parseHexColor(g.cfg.TextColor)
	if err != nil {
		return nil, fmt.Errorf("ogimage: text_color: %w", err)
	}
	subColor, err := parseHexColor(g.cfg.SubColor)
	if err != nil {
		return nil, fmt.Errorf("ogimage: sub_color: %w", err)
	}
	accentColor, err := parseHexColor(g.cfg.AccentColor)
	if err != nil {
		return nil, fmt.Errorf("ogimage: accent_color: %w", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, ogWidth, ogHeight))
	draw.Draw(img, img.Bounds(), image.NewUniform(bgColor), image.Point{}, draw.Src)

	// bottom accent line
	accentUniform := image.NewUniform(accentColor)
	draw.Draw(img, image.Rect(80, ogHeight-42, ogWidth-80, ogHeight-39), accentUniform, image.Point{}, draw.Src)

	const padX = 80
	const usableW = ogWidth - padX*2 // 1040

	title := ""
	description := ""
	if post.metadata != nil {
		title = post.metadata.Title
		description = post.metadata.Description
	}
	if title == "" {
		title = slugFromPath(post.path)
	}

	// Top-left: icon + blog name
	iconX := padX
	nameY := 92
	if g.icon != nil {
		const iconSize = 56
		dst := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), g.icon, g.icon.Bounds(), xdraw.Over, nil)
		draw.Draw(img, image.Rect(iconX, nameY-iconSize+8, iconX+iconSize, nameY+8), dst, image.Point{}, draw.Over)
		iconX += iconSize + 16
	}
	if g.blogName != "" {
		drawText(img, g.smallFace, g.blogName, iconX, nameY, subColor)
	}

	// Compute title + description lines, then vertically centre in body area [110, 550]
	titleLines := wrapText(g.boldFace, title, usableW)
	if len(titleLines) > 4 {
		titleLines = titleLines[:4]
	}
	var descLines []string
	if description != "" {
		descLines = wrapText(g.regularFace, description, usableW)
		if len(descLines) > 3 {
			descLines = descLines[:3]
		}
	}

	const titleLineH = 66
	const descLineH = 30
	const sectionGap = 20

	totalH := len(titleLines) * titleLineH
	if len(descLines) > 0 {
		totalH += sectionGap + len(descLines)*descLineH
	}

	const bodyTop = 130
	const bodyBottom = 550
	centerY := (bodyTop + bodyBottom) / 2
	startY := centerY - totalH/2

	titleMetrics := g.boldFace.Metrics()
	titleAscent := titleMetrics.Ascent.Ceil()

	d := &font.Drawer{Dst: img, Src: image.NewUniform(textColor), Face: g.boldFace}
	y := startY + titleAscent
	for _, line := range titleLines {
		d.Dot = fixed.P(padX, y)
		d.DrawString(line)
		y += titleLineH
	}

	if len(descLines) > 0 {
		descMetrics := g.regularFace.Metrics()
		descAscent := descMetrics.Ascent.Ceil()
		dy := startY + len(titleLines)*titleLineH + sectionGap + descAscent
		d.Src = image.NewUniform(subColor)
		d.Face = g.regularFace
		for _, line := range descLines {
			d.Dot = fixed.P(padX, dy)
			d.DrawString(line)
			dy += descLineH
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("ogimage: encode png: %w", err)
	}
	return buf.Bytes(), nil
}

// Invalidate removes the cached image for the given post hash, called on post change/removal.
func (g *OGImageGenerator) Invalidate(hash uint64) {
	g.cache.delete(hash)
}

func wrapText(face font.Face, text string, maxWidth int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if font.MeasureString(face, candidate).Ceil() > maxWidth {
			lines = append(lines, current)
			current = word
		} else {
			current = candidate
		}
	}
	lines = append(lines, current)
	return lines
}

func drawText(img *image.RGBA, face font.Face, text string, x, y int, col color.RGBA) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

func loadIcon(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}

func parseHexColor(s string) (color.RGBA, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("expected 6-digit hex, got %q", s)
	}
	rv, err := strconv.ParseUint(s[0:2], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}
	gv, err := strconv.ParseUint(s[2:4], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}
	bv, err := strconv.ParseUint(s[4:6], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}
	return color.RGBA{R: uint8(rv), G: uint8(gv), B: uint8(bv), A: 255}, nil
}
