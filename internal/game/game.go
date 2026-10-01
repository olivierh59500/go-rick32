// Package game composes Rick32's simulation, DCK renderer and desktop/touch input.
package game

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/olivierh59500/democonstructionkit/render"
	"github.com/olivierh59500/go-rick32/internal/controls"
	"github.com/olivierh59500/go-rick32/internal/data"
	"github.com/olivierh59500/go-rick32/internal/engine"
)

const (
	Width  = 640
	Height = 480
	FPS    = 60
)

type Config struct{ Mute, Mobile, Recording bool }
type mode uint8

const (
	attract mode = iota
	starting
	playing
	respawning
	ending
	completed
)

type button struct {
	label, action string
	rect          image.Rectangle
	mask          byte
	held          bool
}
type Game struct {
	data                                              *data.Data
	core                                              *engine.Core
	renderer                                          *renderer
	audio                                             *soundtrack
	canvas                                            *ebiten.Image
	config                                            Config
	mode                                              mode
	tick, logicalTick, accumulator, demoIndex         int
	elapsed, modeAge, pageAge, cameraAge, cameraStart float64
	pageIndex                                         int
	paused                                            bool
	joystick                                          controls.Joystick
	touchSamples                                      []controls.Touch
	controlsWidth                                     int
	buttons                                           []button
	touches                                           []ebiten.TouchID
	layoutWidth                                       int
}

func New(c Config) (*Game, error) {
	d, err := data.Load()
	if err != nil {
		return nil, err
	}
	d.Pages["insert"] = data.Page{Hold: 1e9, Lines: []data.Line{{Y: 400, Text: "INSERT COIN(S)", Color: "ffffffff"}}}
	d.Pages["play-label"] = data.Page{Hold: 1e9, Lines: []data.Line{{Y: 400, Text: "OR PRESS CONTROL TO PLAY", Color: "ffffffff"}}}
	r, err := newRenderer(d)
	if err != nil {
		return nil, err
	}
	g := &Game{data: d, core: engine.New(d), renderer: r, config: c, canvas: render.NewSurface(Width, Height), layoutWidth: Width}
	if !c.Mute {
		g.audio, err = newSoundtrack()
		if err != nil {
			g.Close()
			return nil, err
		}
		if err = g.audio.track("demo.ym"); err != nil {
			g.Close()
			return nil, err
		}
	}
	return g, nil
}
func (g *Game) Update() error {
	input, action := g.input()
	switch action {
	case "quit":
		return ebiten.Termination
	case "filter":
		if g.renderer.filter == ebiten.FilterLinear {
			g.renderer.filter = ebiten.FilterNearest
		} else {
			g.renderer.filter = ebiten.FilterLinear
		}
	case "wire":
		g.renderer.outline = !g.renderer.outline
	case "pause":
		g.paused = !g.paused
		g.audio.pause(g.paused)
	case "mute":
		g.audio.mute()
	case "reset", "demo":
		g.restartAttract()
	case "start":
		g.startPlaying()
	}
	if g.mode == ending && input&engine.Fire != 0 {
		g.restartAttract()
		input = 0
	}
	if g.mode == attract && input&engine.Fire != 0 {
		g.startPlaying()
	}
	if g.paused {
		return nil
	}
	g.tick++
	dt := 1.0 / FPS
	g.elapsed += dt
	g.modeAge += dt
	g.pageAge += dt
	g.cameraAge += dt
	if g.mode == starting && g.modeAge >= .5 {
		g.core.Reset()
		g.mode = playing
		g.modeAge = 0
		g.pageAge = 0
		g.cameraStart = 0
		if err := g.audio.track("game.ym"); err != nil {
			return err
		}
	}
	if g.mode == respawning && g.modeAge >= .5 {
		g.core.Respawn()
		g.mode = playing
		g.modeAge = 0
		g.cameraStart = 0
	}
	if (g.mode == ending && g.modeAge >= 20) || (g.mode == completed && input&engine.Fire != 0) {
		g.restartAttract()
	}
	g.accumulator += engine.TicksPerSecond
	if g.accumulator >= FPS {
		g.accumulator -= FPS
		g.logicalTick++
		if g.mode == attract {
			if g.demoIndex < len(g.data.Demo) {
				input = g.data.Demo[g.demoIndex]
				g.demoIndex++
			} else {
				input = 0
			}
			if g.core.Dead() {
				g.core.Reset()
				g.demoIndex = 0
				g.cameraStart = 0
			}
			g.core.Step(input)
		} else if g.mode == playing {
			g.core.Step(input)
		}
		if g.core.CameraShift != 0 {
			g.cameraStart = g.core.CameraShift
			g.cameraAge = 0
			g.core.CameraShift = 0
		}
		for _, event := range g.core.Effects() {
			g.audio.effect(event)
		}
		if g.mode == playing {
			s := g.core.State()
			if s.Won {
				g.mode = completed
				g.modeAge = 0
				g.pageAge = 0
				if err := g.audio.track("complete.ym"); err != nil {
					return err
				}
			}
			if s.Dead {
				g.modeAge = 0
				if s.Lives > 1 {
					g.mode = respawning
				} else {
					g.mode = ending
					g.pageAge = 0
					if err := g.audio.track("game-over.ym"); err != nil {
						return err
					}
				}
			}
		}
	}
	if g.mode == attract {
		pages := []string{"title", "concept", "commands", "features", "sound", "credits"}
		key := pages[g.pageIndex]
		if g.pageAge >= g.renderer.pages[key].hold+2 {
			g.pageAge = 0
			g.pageIndex = (g.pageIndex + 1) % len(pages)
		}
	}
	return nil
}
func (g *Game) restartAttract() {
	g.core.Reset()
	g.mode = attract
	g.modeAge = 0
	g.pageAge = 0
	g.pageIndex = 0
	g.demoIndex = 0
	g.cameraStart = 0
	g.paused = false
	g.audio.restart("demo.ym")
}
func (g *Game) startPlaying() {
	if g.mode == starting {
		return
	}
	g.mode = starting
	g.modeAge = 0
	g.pageAge = 0
	g.paused = false
	g.audio.pause(false)
}
func (g *Game) Draw(dst *ebiten.Image) {
	camera := g.cameraStart * math.Pow(math.Max(0, 1-g.cameraAge/.7), 3)
	gamma := 1.0
	if g.mode == starting || g.mode == respawning {
		gamma = math.Max(0, 1-g.modeAge/.5)
	}
	g.renderer.board(g.canvas, g.core, g.data, camera, gamma)
	switch g.mode {
	case attract:
		pages := []string{"title", "concept", "commands", "features", "sound", "credits"}
		g.renderer.page(g.canvas, pages[g.pageIndex], g.pageAge, g.elapsed)
		g.renderer.caption(g.canvas, "", g.elapsed)
	case playing:
		if g.modeAge < 3 {
			g.renderer.page(g.canvas, "start", g.modeAge, g.elapsed)
		}
	case ending:
		g.renderer.page(g.canvas, "game-over", g.pageAge, g.elapsed)
	case completed:
		g.renderer.page(g.canvas, "complete", g.pageAge, g.elapsed)
	}
	dst.Fill(color.RGBA{9, 13, 22, 255})
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64((g.layoutWidth-Width)/2), 0)
	dst.DrawImage(g.canvas, &op)
	if g.config.Mobile {
		g.drawControls(dst)
	}
}
func (g *Game) Layout(w, h int) (int, int) {
	if !g.config.Mobile {
		g.layoutWidth = Width
	} else if w > 0 && h > 0 {
		g.layoutWidth = max(960, int(float64(Height)*float64(w)/float64(h)))
	} else if g.layoutWidth < 960 {
		g.layoutWidth = 960
	}
	return g.layoutWidth, Height
}

// VerificationState reports input/layout diagnostics without changing the game.
func (g *Game) VerificationState() string {
	s := g.core.State()
	return fmt.Sprintf("mobile=%v width=%d mode=%d room=%d x=%d y=%d", g.config.Mobile, g.layoutWidth, g.mode, s.Room, s.Rick.X, s.Rick.Y)
}

func (g *Game) input() (byte, string) {
	if g.config.Recording {
		return 0, ""
	}
	var input byte
	action := ""
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		input |= engine.Up
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		input |= engine.Down
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		input |= engine.Left
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		input |= engine.Right
	}
	if ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight) || ebiten.IsKeyPressed(ebiten.KeySpace) {
		input |= engine.Fire
	}
	for _, key := range []struct {
		k ebiten.Key
		a string
	}{{ebiten.KeyEnter, "start"}, {ebiten.KeyF, "filter"}, {ebiten.KeyW, "wire"}, {ebiten.KeyP, "pause"}, {ebiten.KeyR, "reset"}, {ebiten.KeyD, "demo"}, {ebiten.KeyM, "mute"}} {
		if inpututil.IsKeyJustPressed(key.k) {
			action = key.a
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.mode == attract {
			action = "quit"
		} else {
			action = "demo"
		}
	}
	if !g.config.Mobile {
		return input, action
	}
	g.ensureControls()
	g.touches = ebiten.AppendTouchIDs(g.touches[:0])
	g.touchSamples = g.touchSamples[:0]
	for _, id := range g.touches {
		x, y := ebiten.TouchPosition(id)
		g.touchSamples = append(g.touchSamples, controls.Touch{ID: int(id), X: float64(x), Y: float64(y), Pressed: inpututil.TouchPressDuration(id) == 1})
	}
	g.joystick.Update(g.touchSamples)
	if g.joystick.X < 0 {
		input |= engine.Left
	} else if g.joystick.X > 0 {
		input |= engine.Right
	}
	if g.joystick.Y < 0 {
		input |= engine.Up
	} else if g.joystick.Y > 0 {
		input |= engine.Down
	}
	for i := range g.buttons {
		b := &g.buttons[i]
		held := false
		for _, id := range g.touches {
			if g.joystick.Owns(int(id)) {
				continue
			}
			x, y := ebiten.TouchPosition(id)
			if image.Pt(x, y).In(b.rect) {
				held = true
				break
			}
		}
		if held {
			input |= b.mask
			if !b.held && b.action != "" {
				action = b.action
			}
		}
		b.held = held
	}
	return input, action
}
func (g *Game) ensureControls() {
	if g.controlsWidth == g.layoutWidth && len(g.buttons) > 0 {
		return
	}
	g.controlsWidth = g.layoutWidth
	side := (g.layoutWidth - Width) / 2
	right := g.layoutWidth - side
	g.joystick.Place(float64(side)/2, 360, math.Min(84, float64(side)/2-12))
	g.buttons = []button{
		{"FIRE", "", image.Rect(right+side/2-55, 326, right+side/2+55, 402), engine.Fire, false},
		{"PLAY", "start", image.Rect(right+18, 60, g.layoutWidth-18, 105), 0, false},
		{"DEMO", "demo", image.Rect(right+18, 116, g.layoutWidth-18, 161), 0, false},
		{"PAUSE", "pause", image.Rect(right+18, 172, g.layoutWidth-18, 217), 0, false},
		{"FILTER", "filter", image.Rect(18, 60, side-18, 105), 0, false},
		{"WIRE", "wire", image.Rect(18, 116, side-18, 161), 0, false},
		{"RESET", "reset", image.Rect(18, 172, side-18, 217), 0, false},
	}
}
func (g *Game) drawControls(dst *ebiten.Image) {
	g.ensureControls()
	g.drawJoystick(dst)
	for _, b := range g.buttons {
		c := color.RGBA{28, 43, 63, 255}
		if b.held {
			c = color.RGBA{58, 99, 136, 255}
		}
		vector.FillRect(dst, float32(b.rect.Min.X), float32(b.rect.Min.Y), float32(b.rect.Dx()), float32(b.rect.Dy()), c, false)
		vector.StrokeRect(dst, float32(b.rect.Min.X), float32(b.rect.Min.Y), float32(b.rect.Dx()), float32(b.rect.Dy()), 1, color.RGBA{110, 146, 170, 255}, false)
		g.renderer.label(dst, b.label, float64(b.rect.Min.X+b.rect.Dx()/2), float64(b.rect.Min.Y+b.rect.Dy()/2)-9)
	}
}
func (g *Game) Tick() int { return g.tick }
func (g *Game) Close() {
	if g == nil {
		return
	}
	g.audio.Close()
	if g.renderer != nil {
		g.renderer.Close()
	}
	if g.canvas != nil {
		g.canvas.Deallocate()
	}
}
