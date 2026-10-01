package engine

import "github.com/olivierh59500/go-rick32/internal/data"

const (
	Up             = controlUp
	Down           = controlDown
	Left           = controlLeft
	Right          = controlRight
	Fire           = controlFire
	TicksPerSecond = 25
)

type Snapshot struct {
	Room, Row, Score, Lives, Bullets, Bombs int
	Flags                                   byte
	Rick                                    Entity
	Dead, Won                               bool
}

func New(d *data.Data) *Core {
	g := &Core{data: d}
	g.Reset()
	return g
}

func (g *Core) Reset() {
	d := g.data
	*g = Core{data: d, envLives: 6, envBullets: 6, envBombs: 6, envDepth: true, mapFRow: 8}
	g.entities[12].N = 255
	g.entities[1] = Entity{N: 1, X: 8, Y: 139, W: 24, H: 21, Sprite: 1}
	g.mapResetMarks()
	g.mapInit()
	g.rickSave()
	g.saveMapRow = g.mapFRow
}

// Step consumes one native 40 ms input sample. Rendering never calls Step.
func (g *Core) Step(input byte) {
	if g.Dead() || g.Won {
		return
	}
	g.effectCount = 0
	g.stepControlStatus = input
	g.entAction()
	g.themRndSeed++
}

func (g *Core) advanceCameraAndDoors() {
	if g.rickAtExit {
		g.rickAtExit = false
		start := g.data.Submaps[g.envSubmap].Connect
		for i := start; i < start+2; i++ {
			c := g.data.Connects[i]
			if c.Dir != g.gameDir {
				continue
			}
			if c.Submap == 255 {
				g.Won = true
				g.entities[1].N = 0
				return
			}
			g.mapFRow += c.RowIn - c.RowOut
			g.envSubmap = c.Submap
			g.mapInit()
			g.rickSave()
			g.saveMapRow = g.mapFRow
			return
		}
	}
	if g.rickState&eRickStZombie != 0 {
		return
	}
	shift := 0
	if g.eRick().Y >= 205 {
		shift = -64
	}
	if g.eRick().Y <= 96 {
		shift = 64
	}
	if shift != 0 {
		g.mapFRow -= shift / 8
		g.mapExpand()
		for i := 0; i < 12; i++ {
			e := &g.entities[i]
			if e.N != 0 {
				e.Y += shift
				e.YSave += shift
				e.TrigY += shift
				if e.Y < 0 || e.Y > 320 {
					e.N = 0
				}
			}
		}
		if shift < 0 {
			g.entActVis(g.mapFRow+32, g.mapFRow+39)
		} else {
			g.entActVis(g.mapFRow, g.mapFRow+7)
		}
		g.CameraShift = float64(shift)
	}
}

func (g *Core) Dead() bool { return g.rickState&eRickStDead != 0 }

func (g *Core) Respawn() {
	g.envLives--
	g.envBullets = 6
	g.envBombs = 6
	g.rickRestore()
	g.mapFRow = g.saveMapRow
	g.rickState = 0
	g.rickAtExit = false
	g.rickStopped = false
	g.mapInit()
}

func (g *Core) State() Snapshot {
	return Snapshot{Room: g.envSubmap, Row: g.mapFRow, Score: g.envScore, Lives: g.envLives, Bullets: g.envBullets, Bombs: g.envBombs, Rick: *g.eRick(), Flags: g.rickState, Dead: g.Dead(), Won: g.Won}
}

func (g *Core) Entities() *[13]Entity  { return &g.entities }
func (g *Core) TileMap() *[44][32]byte { return &g.mapMap }

// playEffect retains bounded event storage; playback belongs to the host.
func (g *Core) playEffect(name string) {
	if g.effectCount < len(g.effects) {
		g.effects[g.effectCount] = name
		g.effectCount++
	}
}

func (g *Core) Effects() []string { return g.effects[:g.effectCount] }
