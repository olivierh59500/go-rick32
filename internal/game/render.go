package game

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/olivierh59500/democonstructionkit/font"
	"github.com/olivierh59500/democonstructionkit/render"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"github.com/olivierh59500/democonstructionkit/sprites"
	"github.com/olivierh59500/go-rick32/assets"
	"github.com/olivierh59500/go-rick32/internal/data"
	"github.com/olivierh59500/go-rick32/internal/engine"
)

type pageLine struct {
	text     *scrolling.Scrolling
	width, y float64
	tint     color.NRGBA
	index    int
}
type pageView struct {
	hold  float64
	lines []pageLine
}
type renderer struct {
	blocks, tiles, sprites *sprites.Atlas
	batch                  *render.Batch
	fontImage              *ebiten.Image
	labelWidths            map[string]float64
	labels                 map[string]*scrolling.Scrolling
	pages                  map[string]pageView
	filter                 ebiten.Filter
	outline                bool
}

func newRenderer(d *data.Data) (*renderer, error) {
	r := &renderer{batch: render.NewBatch(4096), pages: make(map[string]pageView), labels: make(map[string]*scrolling.Scrolling), labelWidths: make(map[string]float64), filter: ebiten.FilterLinear}
	colors := make([]color.NRGBA, 16)
	for i, c := range d.Palette {
		colors[i] = color.NRGBA{uint8((c >> 10 & 31) * 255 / 31), uint8((c >> 5 & 31) * 255 / 31), uint8((c & 31) * 255 / 31), 255}
	}
	makeAtlas := func(pixels [][]byte, w, h, stride, columns int, transparent bool) (*sprites.Atlas, error) {
		img := image.NewNRGBA(image.Rect(0, 0, columns*stride, ((len(pixels)+columns-1)/columns)*stride))
		for i, frame := range pixels {
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					index := frame[y*w+x]
					if transparent && index == 0 {
						continue
					}
					img.SetNRGBA(i%columns*stride+x, i/columns*stride+y, colors[index])
				}
			}
		}
		return sprites.NewAtlas(sprites.AtlasConfig{Image: ebiten.NewImageFromImage(img), TileW: stride, TileH: stride, Columns: columns, Count: len(pixels)})
	}
	var err error
	if r.tiles, err = makeAtlas(d.Tiles, 8, 8, 8, 16, false); err != nil {
		return nil, err
	}
	if r.sprites, err = makeAtlas(d.Sprites, 32, 21, 32, 8, true); err != nil {
		return nil, err
	}
	blocks := make([][]byte, len(d.Blocks))
	for i, block := range d.Blocks {
		pixels := make([]byte, 32*32)
		for j, t := range block {
			tile := d.Tiles[t]
			for y := 0; y < 8; y++ {
				copy(pixels[(j/4*8+y)*32+j%4*8:], tile[y*8:y*8+8])
			}
		}
		blocks[i] = pixels
	}
	if r.blocks, err = makeAtlas(blocks, 32, 32, 32, 8, false); err != nil {
		return nil, err
	}
	raw, err := assets.Files.ReadFile("font.png")
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	r.fontImage = ebiten.NewImageFromImage(img)
	type metric struct {
		Rune       int `json:"rune"`
		X, Y, W, H int
		Advance    float64
	}
	raw, err = assets.Files.ReadFile("font.json")
	if err != nil {
		return nil, err
	}
	var metrics []metric
	if err = json.Unmarshal(raw, &metrics); err != nil {
		return nil, err
	}
	config := font.Config{Bounds: img.Bounds(), Glyphs: map[rune]font.Glyph{}, LineHeight: 26, SpaceAdvance: 6}
	for _, m := range metrics {
		config.Glyphs[rune(m.Rune)] = font.Glyph{Rect: image.Rect(m.X, m.Y, m.X+m.W, m.Y+m.H), Advance: m.Advance}
	}
	face, err := font.New(config)
	if err != nil {
		return nil, err
	}
	atlas, err := scrolling.NewAtlas(r.fontImage, face)
	if err != nil {
		return nil, err
	}
	for _, label := range []string{"UP", "LEFT", "RIGHT", "DOWN", "FIRE", "PLAY", "DEMO", "PAUSE", "FILTER", "WIRE", "RESET"} {
		text, err := scrolling.New(scrolling.Config{Glyphs: atlas.Glyphs(label)})
		if err != nil {
			return nil, err
		}
		r.labels[label] = text
		for _, glyph := range atlas.Glyphs(label) {
			r.labelWidths[label] += glyph.Advance
		}
	}
	for key, page := range d.Pages {
		view := pageView{hold: page.Hold}
		for i, l := range page.Lines {
			glyphs := atlas.Glyphs(l.Text)
			width := 0.0
			for _, g := range glyphs {
				width += g.Advance
			}
			text, err := scrolling.New(scrolling.Config{Glyphs: glyphs})
			if err != nil {
				return nil, err
			}
			v, err := strconv.ParseUint(l.Color, 16, 32)
			if err != nil {
				return nil, err
			}
			view.lines = append(view.lines, pageLine{text, width, l.Y, color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), uint8(v >> 24)}, i})
		}
		r.pages[key] = view
	}
	return r, nil
}

func (r *renderer) board(dst *ebiten.Image, c *engine.Core, d *data.Data, camera, gamma float64) {
	dst.Fill(color.Black)
	state := c.State()
	room := d.Submaps[state.Room]
	// Cached 32-pixel blocks reproduce the native two-times scale without
	// resampling a complete software framebuffer every displayed frame.
	clip := dst.SubImage(image.Rect(64, 56, 576, 440)).(*ebiten.Image)
	r.batch.Options.Filter = r.filter
	r.batch.Begin(clip, r.blocks.Image)
	scroll := float64(state.Row*8) + camera
	row := int(math.Floor(scroll/32)) + 2
	offset := scroll - float64(row-2)*32
	tint := color.NRGBA{uint8(gamma * 255), uint8(gamma * 255), uint8(gamma * 255), 255}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			n := room.BNum + (row+y)*8 + x
			if n < 0 || n >= len(d.BNums) {
				continue
			}
			r.batch.Rect(float64(64+x*64), 56+float64(y*64)-offset*2, 64, 64, r.blocks.Rect(int(d.BNums[n])), tint)
		}
	}
	r.batch.Flush()
	r.batch.Begin(clip, r.sprites.Image)
	for i, e := range *c.Entities() {
		if i >= 12 || e.N == 0 || e.Sprite < 0 || e.Sprite >= r.sprites.Count() {
			continue
		}
		x := float64(e.X*2 + 64)
		y := float64(e.Y*2-128+56) - camera*2
		src := r.sprites.Rect(e.Sprite)
		src.Max.X = src.Min.X + 31
		src.Max.Y = src.Min.Y + 21
		if !e.Flip {
			r.batch.Rect(x, y, 64, 42, src, tint)
		} else {
			x -= 16
			r.batch.Quad([4]ebiten.Vertex{render.Vertex(x, y, float64(src.Max.X), float64(src.Min.Y), tint), render.Vertex(x+64, y, float64(src.Min.X), float64(src.Min.Y), tint), render.Vertex(x+64, y+42, float64(src.Min.X), float64(src.Max.Y), tint), render.Vertex(x, y+42, float64(src.Max.X), float64(src.Max.Y), tint)})
		}
		if r.outline {
			vector.StrokeRect(clip, float32(x), float32(y), 64, 42, 1, color.RGBA{255, 220, 80, 255}, false)
		}
	}
	r.batch.Flush()
	r.batch.Begin(dst, r.tiles.Image)
	score := state.Score
	for i := 0; i < 6; i++ {
		r.batch.Rect(float64(144-i*16), 40, 16, 16, r.tiles.Rect(105+score%10), tint)
		score /= 10
		if i < state.Bullets {
			r.batch.Rect(float64(208+i*16), 40, 16, 16, r.tiles.Rect(102), tint)
		}
		if i < state.Bombs {
			r.batch.Rect(float64(336+i*16), 40, 16, 16, r.tiles.Rect(103), tint)
		}
		if i < state.Lives {
			r.batch.Rect(float64(480+i*16), 40, 16, 16, r.tiles.Rect(104), tint)
		}
	}
	r.batch.Flush()
	if r.outline {
		for y := 0; y < 7; y++ {
			for x := 0; x < 8; x++ {
				vector.StrokeRect(clip, float32(64+x*64), float32(float64(56+y*64)-offset*2), 64, 64, 1, color.RGBA{80, 160, 255, 255}, false)
			}
		}
	}
}

func (r *renderer) page(dst *ebiten.Image, key string, age, time float64) {
	view, ok := r.pages[key]
	if !ok {
		return
	}
	slide := 0.0
	if age < 1 {
		slide = math.Pow(1-age, 3) * 640
	} else if age > 1+view.hold {
		slide = -math.Pow(math.Min(age-1-view.hold, 1), 3) * 640
	}
	for _, line := range view.lines {
		line := line
		state := scrolling.IdentityState()
		state.X = 320 - line.width/2 + slide
		state.Y = line.y
		state.Options.Filter = ebiten.FilterLinear
		state.Options.ColorScale.Scale(float32(line.tint.R)/255, float32(line.tint.G)/255, float32(line.tint.B)/255, float32(line.tint.A)/255)
		state.Map = func(s scrolling.Sample, op *ebiten.DrawImageOptions) bool {
			op.GeoM.Translate(math.Sin(time*6.1+float64(line.index)*.1+float64(s.Index)*.1)*16, math.Sin(time*6.5+float64(line.index)*.2+float64(s.Index)*.2)*16)
			return true
		}
		if key == "insert" || key == "play-label" {
			state.Map = nil
		}
		line.text.DrawAt(dst, state)
	}
}

func (r *renderer) caption(dst *ebiten.Image, _ string, time float64) {
	// These two native attract-mode labels remain fixed; pages use independent
	// per-character waves through the same DCK font pipeline.
	if int(time/.3)&1 == 0 {
		return
	}
	key := "insert"
	if int(time/.3)&4 != 0 {
		key = "play-label"
	}
	r.page(dst, key, 1, time)
}

func (r *renderer) Close() {
	r.blocks.Image.Deallocate()
	r.tiles.Image.Deallocate()
	r.sprites.Image.Deallocate()
	r.fontImage.Deallocate()
}

func (r *renderer) label(dst *ebiten.Image, label string, x, y float64) {
	text := r.labels[label]
	if text == nil {
		return
	}
	state := scrolling.IdentityState()
	state.ScaleX = .65
	state.ScaleY = .65
	state.X = x - r.labelWidths[label]*.65/2
	state.Y = y
	text.DrawAt(dst, state)
}
