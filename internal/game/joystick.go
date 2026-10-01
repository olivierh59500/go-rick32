package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawJoystick uses bounded vector geometry and the retained input pose; it
// creates no textures or images as the thumb moves.
func (g *Game) drawJoystick(dst *ebiten.Image) {
	j := &g.joystick
	x, y, r := float32(j.CenterX), float32(j.CenterY), float32(j.Radius)
	vector.FillCircle(dst, x, y, r, color.RGBA{24, 37, 53, 255}, true)
	vector.StrokeCircle(dst, x, y, r, 2, color.RGBA{110, 146, 170, 255}, true)
	vector.StrokeCircle(dst, x, y, float32(j.Travel), 1, color.RGBA{53, 75, 97, 255}, true)
	directions := [8][2]int{{1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}}
	for i, d := range directions {
		a := float64(i) * math.Pi / 4
		ux, uy := float32(math.Cos(a)), float32(math.Sin(a))
		px, py := x+ux*r*.81, y+uy*r*.81
		c := color.RGBA{96, 125, 148, 255}
		if j.X == d[0] && j.Y == d[1] {
			c = color.RGBA{133, 209, 246, 255}
		}
		vector.StrokeLine(dst, px-ux*5-uy*4, py-uy*5+ux*4, px, py, 2, c, true)
		vector.StrokeLine(dst, px, py, px-ux*5+uy*4, py-uy*5-ux*4, 2, c, true)
	}
	kx, ky := x+float32(j.OffsetX), y+float32(j.OffsetY)
	radius := r * .31
	vector.FillCircle(dst, kx, ky+3, radius+1, color.RGBA{11, 20, 31, 255}, true)
	c := color.RGBA{59, 81, 104, 255}
	if j.Active() {
		c = color.RGBA{69, 119, 156, 255}
	}
	vector.FillCircle(dst, kx, ky, radius, c, true)
	vector.StrokeCircle(dst, kx, ky, radius, 2, color.RGBA{156, 197, 219, 255}, true)
	vector.FillCircle(dst, kx, ky, 3, color.RGBA{185, 216, 232, 255}, true)
}
