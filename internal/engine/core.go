// Package engine preserves Rick32 collision, entity and fixed-point movement rules.
// Gameplay derives from Stephane (Bigorno) Gay's XRick engine.
package engine

import (
	"fmt"
	"github.com/olivierh59500/go-rick32/internal/data"
)

const (
	ScreenWidth       = 320
	ScreenHeight      = 240
	DefaultWindowZoom = 2
	MaxWindowZoom     = DefaultWindowZoom * 2

	fbMapLeft = 32
	fbMapTop  = 28

	mapWidthPx     = 0x100
	mapTopHeightPx = 0x40
	mapVisHeightPx = 0xc0

	mapTopHeightTL = 0x08
	mapVisHeightTL = 0x18
	mapBotHeightTL = 0x08

	mapRowScrTop = 0x08
	mapRowScrBot = 0x1f
	mapRowHBTop  = 0x20
	mapRowHBBot  = 0x27
	mapRowHTTop  = 0x00
	mapRowHTBot  = 0x07

	left  = 1
	right = 0

	controlUp    = 0x01
	controlDown  = 0x02
	controlLeft  = 0x04
	controlRight = 0x08
	controlPause = 0x80
	controlEnd   = 0x40
	controlExit  = 0x20
	controlFire  = 0x10

	mapEFlgVert   = 0x80
	mapEFlgSolid  = 0x40
	mapEFlgSPad   = 0x20
	mapEFlgWayUp  = 0x10
	mapEFlgFGnd   = 0x08
	mapEFlgLethal = 0x04
	mapEFlgClimb  = 0x02
	mapEFlg01     = 0x01

	mapMarkNACT = 0x80

	entLethal = 0x80

	entFlgOnce       = 0x01
	entFlgStopRick   = 0x02
	entFlgLethalR    = 0x04
	entFlgLethalI    = 0x08
	entFlgTrigBomb   = 0x10
	entFlgTrigBullet = 0x20
	entFlgTrigStop   = 0x40
	entFlgTrigRick   = 0x80

	eRickNo   = 1
	eBulletNo = 2
	eBombNo   = 3

	eRickStStop   = 0x01
	eRickStShoot  = 0x02
	eRickStClimb  = 0x04
	eRickStJump   = 0x08
	eRickStZombie = 0x10
	eRickStDead   = 0x20
	eRickStCrawl  = 0x40

	gameBombsInit   = 6
	gameBulletsInit = 6
)

type Entity struct {
	N       int
	X       int
	Y       int
	Sprite  int
	W       int
	H       int
	Mark    int
	Flags   int
	TrigX   int
	TrigY   int
	XSave   int
	YSave   int
	SprBase int
	StepNoI int
	StepNo  int
	C1      int
	C2      int
	YLow    int
	OffSY   int
	Latency int
	PrevN   int
	PrevX   int
	PrevY   int
	PrevS   int
	Flip    bool
	Front   bool
	TrigSnd int
}

type Core struct {
	data         *data.Data
	mapMap       [0x2c][0x20]uint8
	mapEFlags    [0x100]uint8
	mapFRow      int
	mapTilesBank int

	envTrainer    bool
	envInvincible bool
	envHighlight  bool
	envDepth      bool
	envLives      int
	envBombs      int
	envBullets    int
	envScore      int
	envMap        int
	envSubmap     int

	controlStatus     uint8
	stepControlStatus uint8
	controlLast       uint8
	prevControl       uint8

	fireJustPressed  bool
	pauseJustPressed bool
	exitJustPressed  bool
	endJustPressed   bool
	f1Held           bool
	f2Held           bool
	f3Held           bool
	f4Held           bool
	f5Held           bool
	f6Held           bool
	f7Held           bool
	f8Held           bool
	f9Held           bool

	firePending        bool
	fireControlPending uint8
	pausePending       bool
	exitPending        bool
	endPending         bool

	gameDir int

	entities [13]Entity

	prevDrawX    [13]int
	prevDrawY    [13]int
	prevDrawN    [13]int
	prevDrawMark [13]int

	rickStopX     int
	rickStopY     int
	rickState     uint8
	rickAtExit    bool
	rickScrawl    bool
	rickStopped   bool
	rickTrigger   bool
	rickOffsX     int
	rickYLow      int
	rickOffsY     int
	rickSeq       int
	rickSaveCrawl bool
	rickSaveX     int
	rickSaveY     int
	rickSaveState uint8
	rickSaveOffsY int
	rickSaveYLow  int
	rickSaveDir   int

	saveMapRow int

	bulletOffsX int
	bulletXC    int
	bulletYC    int

	bombLethal bool
	bombXC     int
	bombYC     int
	bombTicker int

	themRndSeed uint32
	themRndNbr  uint16

	sbonusCounting bool
	sbonusCounter  int
	sbonusBonus    int

	scrollCount int
	CameraShift float64
	Won         bool
	effects     [8]string
	effectCount int
}

func (g *Core) mapResetMarks() {
	for i := range g.data.Marks {
		g.data.Marks[i].Entity &= ^mapMarkNACT
	}
}

func (g *Core) mapInit() {
	copy(g.mapEFlags[:], g.data.Flags)
	g.mapExpand()
	g.entReset()
	g.entActVis(g.mapFRow+8, g.mapFRow+31)
	g.entActVis(g.mapFRow, g.mapFRow+7)
	g.entActVis(g.mapFRow+32, g.mapFRow+39)
}

func (g *Core) mapExpand() {
	pbnum := g.data.Submaps[g.envSubmap].BNum + ((2 * g.mapFRow) & 0xfff8)
	row := 0
	col := 0
	for i := 0; i < 0x0b; i++ {
		for j := 0; j < 0x08; j++ {
			block := g.data.Blocks[g.data.BNums[pbnum]]
			l := 0
			for k := 0; k < 0x04; k++ {
				g.mapMap[row][col] = block[l]
				l++
				g.mapMap[row][col+1] = block[l]
				l++
				g.mapMap[row][col+2] = block[l]
				l++
				g.mapMap[row][col+3] = block[l]
				l++
				row++
			}
			row -= 4
			col += 4
			pbnum++
		}
		row += 4
		col = 0
	}
}

func (g *Core) mapChain() bool {
	start := g.data.Submaps[g.envSubmap].Connect
	row := (g.entities[eRickNo].Y >> 3) + g.mapFRow
	chosen := -1
	for c := start; c < len(g.data.Connects); c++ {
		conn := g.data.Connects[c]
		if conn.Dir == 0xff {
			break
		}
		if conn.Dir != g.gameDir {
			continue
		}
		diff := row - conn.RowOut
		if diff >= 0 && diff < 3 {
			chosen = c
			break
		}
	}
	if chosen < 0 {
		return false
	}
	conn := g.data.Connects[chosen]
	if conn.Submap == 0xff {
		return false
	}
	g.mapFRow = g.mapFRow - conn.RowOut + conn.RowIn
	g.envSubmap = conn.Submap
	return true
}

func (g *Core) entReset() {
	g.rickState &^= eRickStStop
	g.bombLethal = false
	g.entities[0].N = 0
	for i := 2; i < 12; i++ {
		g.entities[i].N = 0
	}
}

func (g *Core) entActVis(frow, lrow int) {
	start := g.data.Submaps[g.envSubmap].Mark
	m := start
	for m < len(g.data.Marks) && g.data.Marks[m].Row != 0xff && g.data.Marks[m].Row < frow {
		m++
	}
	for m < len(g.data.Marks) && g.data.Marks[m].Row != 0xff && g.data.Marks[m].Row < lrow {
		mark := g.data.Marks[m]
		if mark.Entity&mapMarkNACT != 0 {
			m++
			continue
		}
		slot := -1
		if mark.Flags&entFlgStopRick != 0 {
			if g.entities[0].N != 0 {
				m++
				continue
			}
			slot = 0
			g.entities[slot].C1 = 0
		} else if mark.Entity >= 0x10 {
			for i := 0x04; i < 0x09; i++ {
				if g.entities[i].N == 0 {
					slot = i
					g.entities[i].C1 = 0
					break
				}
			}
		} else {
			already := false
			for i := 0x09; i < 0x0c; i++ {
				if g.entities[i].N != 0 && g.entities[i].Mark == m {
					already = true
					break
				}
			}
			if already {
				m++
				continue
			}
			for i := 0x09; i < 0x0c; i++ {
				if g.entities[i].N == 0 {
					slot = i
					g.entities[i].C1 = 2
					break
				}
			}
		}
		if slot < 0 {
			m++
			continue
		}

		e := &g.entities[slot]
		e.Mark = m
		e.Flags = mark.Flags
		e.N = mark.Entity
		if e.Flags&entFlgLethalR != 0 {
			e.N |= entLethal
		}
		e.X = mark.XY & 0xf8
		y := (mark.XY & 0x07) + (mark.Row & 0xf8) - g.mapFRow
		y <<= 3
		if e.Flags&entFlgStopRick == 0 {
			y += 3
		}
		e.Y = y
		e.XSave = e.X
		e.YSave = e.Y
		info := g.data.EntData[mark.Entity]
		e.W = info.W
		e.H = info.H
		e.SprBase = info.Spr
		e.Sprite = info.Spr
		e.StepNoI = info.SNI
		e.TrigSnd = info.Snd
		const triggers = entFlgTrigBomb | entFlgTrigBullet | entFlgTrigStop | entFlgTrigRick
		if e.Flags&triggers == triggers && slot >= 0x09 {
			e.SprBase = info.SNI & 0x00ff
		}
		e.TrigX = mark.LT & 0xf8
		e.Latency = (mark.LT & 0x07) << 5
		e.TrigY = 3 + 8*((mark.Row&0xf8)-g.mapFRow+(mark.LT&0x07))
		e.C2 = 0
		e.OffSY = 0
		e.YLow = 0
		e.Front = false
		m++
	}
}

func (g *Core) entAction() {
	for i := 0; i < 12; i++ {
		n := g.entities[i].N & 127
		if n == 0 {
			continue
		}
		switch {
		case n == 0x47:
			g.themZombieAction(i)
		case n >= 24:
			g.themT3Action(i)
		case n == 1:
			g.rickAction()
			g.advanceCameraAndDoors()
		case n == 2:
			g.bulletAction()
		case n == 3:
			g.bombAction()
		case n == 4 || n == 7 || n == 10 || n == 13:
			g.themT1Action(i, false)
		case n == 5 || n == 8 || n == 11 || n == 14:
			g.themT1Action(i, true)
		case n == 16 || n == 17:
			g.boxAction(i)
		case n >= 18 && n <= 21:
			g.bonusAction(i)
		case n == 22:
			g.sbonusStart(i)
		case n == 23:
			g.sbonusStop(i)
		}
	}
}

func (g *Core) eRick() *Entity   { return &g.entities[eRickNo] }
func (g *Core) eBullet() *Entity { return &g.entities[eBulletNo] }
func (g *Core) eBomb() *Entity   { return &g.entities[eBombNo] }

func (g *Core) bulletInit(x, y int) {
	b := g.eBullet()
	b.N = 0x02
	b.X = x
	b.Y = y + 0x06
	if g.gameDir == left {
		g.bulletOffsX = -0x08
		b.Sprite = 16
		b.Flip = true
	} else {
		g.bulletOffsX = 0x08
		b.Sprite = 16
		b.Flip = false
	}
	g.playEffect("bullet.wav")
}

func (g *Core) bulletAction() {
	b := g.eBullet()
	b.X += g.bulletOffsX
	if b.X <= -0x10 || b.X > 0xe8 {
		b.N = 0
		return
	}
	g.bulletXC = b.X + 0x0c
	g.bulletYC = b.Y + 0x05
	row := g.bulletYC >> 3
	col := g.bulletXC >> 3
	if g.mapEFlags[g.tileAt(row, col)]&mapEFlgSolid != 0 {
		b.N = 0
	}
}

func (g *Core) bombHit(e int) bool {
	ent := &g.entities[e]
	b := g.eBomb()
	if ent.X > b.X+0x20 {
		return false
	}
	if ent.X+ent.W < b.X-0x04 {
		return false
	}
	if ent.Y > b.Y+0x1d {
		return false
	}
	if ent.Y+ent.H < b.Y-0x04 {
		return false
	}
	return true
}

func (g *Core) bombInit(x, y int) {
	b := g.eBomb()
	b.N = 0x03
	b.X = x + 4
	b.Y = y + 5
	g.bombTicker = 0x2d
	g.bombLethal = false
}

func (g *Core) bombAction() {
	b := g.eBomb()
	if g.bombTicker < 10 {
		if g.bombTicker == 9 {
			b.X -= 4
			b.Y -= 5
			g.playEffect("explode")
		}
		b.Sprite = 23 - (g.bombTicker >> 1)
		g.bombLethal = true
		g.bombXC = b.X + 12
		g.bombYC = b.Y + 10
		if g.bombHit(eRickNo) {
			g.rickGoZombie()
		}
	} else {
		b.Sprite = 18 - (g.bombTicker & 1)
	}
	g.bombTicker--
	if g.bombTicker < 0 {
		b.N = 0
		g.bombLethal = false
	}
}

func (g *Core) bonusAction(e int) {
	ent := &g.entities[e]
	if ent.C1 == 0 {
		if g.rickBoxTest(e) {
			g.envScore += 500
			g.playEffect("bonus.wav")
			g.data.Marks[ent.Mark].Entity |= mapMarkNACT
			ent.C1 = 1
			ent.Sprite = 47
			ent.Front = true
			ent.Y -= 0x08
		}
	} else if ent.C1 > 0 && ent.C1 < 10 {
		ent.C1++
		ent.Y -= 2
	} else {
		ent.N = 0
	}
}

func (g *Core) boxAction(e int) {
	ent := &g.entities[e]
	sp := [...]int{19, 20, 21, 22, 23}
	if ent.N&entLethal != 0 {
		idx := ent.C1 >> 2
		if idx >= len(sp) {
			idx = len(sp) - 1
		}
		ent.Sprite = sp[idx]
		ent.C1--
		if ent.C1 == 0 {
			ent.N = 0
			g.data.Marks[ent.Mark].Entity |= mapMarkNACT
		}
		return
	}
	switch {
	case g.rickBoxTest(e):
		g.playEffect("box.wav")
		if ent.N&0x7f == 0x10 {
			g.envBombs = gameBombsInit
		} else {
			g.envBullets = gameBulletsInit
		}
		ent.N = 0
		g.data.Marks[ent.Mark].Entity |= mapMarkNACT
	case g.rickState&eRickStStop != 0 && g.fullBoxTest(e, g.rickStopX, g.rickStopY):
		g.boxExplode(e)
	case g.eBullet().N != 0 && g.fullBoxTest(e, g.bulletXC, g.bulletYC):
		g.eBullet().N = 0
		g.boxExplode(e)
	case g.bombLethal && g.bombHit(e):
		g.boxExplode(e)
	}
}

func (g *Core) boxExplode(e int) {
	g.entities[e].C1 = 0x14
	g.entities[e].N |= entLethal
	g.playEffect("explode.wav")
}

func (g *Core) sbonusStart(e int) {
	ent := &g.entities[e]
	ent.Sprite = 0
	if g.triggerBox(e, g.eRick().X+0x0c, g.eRick().Y+0x0a) {
		ent.N = 0
		g.sbonusCounting = true
		g.sbonusCounter = 0x1e
		g.sbonusBonus = 2000
		g.playEffect("sbonus1.wav")
	}
}

func (g *Core) sbonusStop(e int) {
	ent := &g.entities[e]
	ent.Sprite = 0
	if !g.sbonusCounting {
		return
	}
	if g.triggerBox(e, g.eRick().X+0x0c, g.eRick().Y+0x0a) {
		g.sbonusCounting = false
		ent.N = 0
		g.envScore += g.sbonusBonus
		g.data.Marks[ent.Mark].Entity |= mapMarkNACT
		g.playEffect("sbonus2.wav")
		return
	}
	g.sbonusCounter--
	if g.sbonusCounter == 0 {
		g.sbonusCounter = 0x1e
		if g.sbonusBonus > 0 {
			g.sbonusBonus--
		}
	}
}

func (g *Core) themTest(e int) bool {
	if g.entities[0].N&entLethal != 0 && g.boxTest(e, 0) {
		return true
	}
	for i := 4; i < 9; i++ {
		if g.entities[i].N&entLethal != 0 && g.boxTest(e, i) {
			return true
		}
	}
	return false
}

func (g *Core) themGoZombie(e int) {
	ent := &g.entities[e]
	ent.N = 0x47
	ent.Front = true
	ent.OffSY = -0x0400
	g.playEffect("die.wav")
	g.envScore += 50
	if ent.Flags&entFlgOnce != 0 {
		g.data.Marks[ent.Mark].Entity |= mapMarkNACT
	}
	if ent.X >= 0x80 {
		ent.C1 = -0x02
	} else {
		ent.C1 = 0x02
	}
}

func (g *Core) themT1AAction(e int) { g.themT1Action(e, false) }
func (g *Core) themT1BAction(e int) { g.themT1Action(e, true) }

func (g *Core) themT1Action(e int, towardRick bool) { g.themT1Move(e, towardRick); g.themT1Post(e) }

func (g *Core) themT1Move(e int, towardRick bool) {
	ent := &g.entities[e]
	i := (ent.Y << 8) + ent.OffSY + ent.YLow
	y := i >> 8
	if y > 0x140 {
		ent.N = 0
		return
	}
	_, env1 := g.envTest(ent.X, y, false)
	if env1&(mapEFlgVert|mapEFlgSolid|mapEFlgSPad|mapEFlgWayUp) == 0 {
		if env1&mapEFlgLethal != 0 {
			g.themGoZombie(e)
			return
		}
		ent.Y = y
		ent.YLow = i & 0xff
		ent.OffSY += 0x0080
		if ent.OffSY > 0x0800 {
			ent.OffSY = 0x0800
		}
		return
	}
	ent.Sprite = ent.SprBase + int(g.data.SprSeq[(ent.X>>3)&3])
	ent.Flip = ent.C1 < 0
	ent.OffSY = 0x0080
	ent.Y = (ent.Y & 0xfff8) | 0x0003
	if ent.Latency > 0 {
		ent.Latency--
		return
	}
	if ent.C1 == 0 {
		return
	}
	x := ent.X + ent.C1
	if x < 0 || x > 0xe8 {
		ent.C2 = 0
		ent.C1 = -ent.C1
		return
	}
	_, env1 = g.envTest(x, ent.Y, false)
	if env1&(mapEFlgVert|mapEFlgSolid|mapEFlgSPad|mapEFlgWayUp) != 0 {
		ent.C2 = 0
		ent.C1 = -ent.C1
		return
	}
	if env1&mapEFlgLethal != 0 {
		g.themGoZombie(e)
		return
	}
	ent.X = x
	if towardRick {
		if ent.X&0x1e != 0x10 {
			return
		}
		if ent.X < g.eRick().X {
			ent.C1 = 0x02
		} else {
			ent.C1 = -0x02
		}
		return
	}
	ent.C2++
	if (ent.TrigX >> 1) > ent.C2 {
		return
	}
	ent.C2 = 0
	ent.C1 = -ent.C1
}

func (g *Core) themT1Post(e int) {
	ent := &g.entities[e]
	if g.themTest(e) {
		g.themGoZombie(e)
		return
	}
	if g.eBullet().N != 0 && g.fullBoxTest(e, g.eBullet().X+map[bool]int{true: 0x18, false: 0}[g.bulletOffsX >= 0], g.eBullet().Y) {
		g.eBullet().N = 0
		g.themGoZombie(e)
		return
	}
	if g.bombLethal && g.bombHit(e) {
		g.themGoZombie(e)
		return
	}
	if g.rickState&eRickStStop != 0 && g.fullBoxTest(e, g.rickStopX, g.rickStopY) {
		ent.Latency = 0x14
	}
	if g.rickBoxTest(e) {
		g.rickGoZombie()
	}
}

func (g *Core) themZombieAction(e int) {
	ent := &g.entities[e]
	if ent.Y < 0 || ent.Y > 0x0140 {
		ent.N = 0
		return
	}
	i := (ent.Y << 8) + ent.OffSY + ent.YLow
	ent.OffSY += 0x0080
	ent.YLow = i & 0xff
	ent.Y = i >> 8
	ent.X += ent.C1
	if ent.X < 0 {
		ent.X = 0
	}
	if ent.X > 0xe8 {
		ent.X = 0xe8
	}
}

func (g *Core) themT2Action(e int) {
	ent := &g.entities[e]
	defer g.themT2Post(e)
	if ent.Latency > 0 {
		ent.Latency--
	}
	if ent.C1 == 1 {
		if ent.Latency > 0 {
			return
		}
		ent.Sprite = ent.SprBase + 0x08
		if ((ent.X ^ ent.Y) & 0x04) != 0 {
			ent.Sprite++
		}
		if ent.Y&0xfe != g.eRick().Y&0xfe {
			yd := -0x02
			if ent.Y < g.eRick().Y {
				yd = 0x02
			}
			y := ent.Y + yd
			if y < 0 || y > 0x0140 {
				ent.N = 0
				return
			}
			_, env1 := g.envTest(ent.X, y, false)
			if env1&(mapEFlgSolid|mapEFlgSPad|mapEFlgWayUp) != 0 {
				if yd < 0 {
					goto climbXMove
				}
				ent.C1 = 0
				return
			}
			ent.Y = y
			if env1&(mapEFlgVert|mapEFlgClimb) != 0 {
				return
			}
			ent.C1 = 0
			return
		}
	climbXMove:
		if ent.X < g.eRick().X {
			ent.C2 = 0x02
		} else {
			ent.C2 = -0x02
		}
		x := ent.X + ent.C2
		_, env1 := g.envTest(x, ent.Y, false)
		if env1&(mapEFlgSolid|mapEFlgSPad|mapEFlgWayUp) != 0 {
			return
		}
		if env1&mapEFlgLethal != 0 {
			g.themGoZombie(e)
			return
		}
		ent.X = x
		if env1&(mapEFlgVert|mapEFlgClimb) != 0 {
			return
		}
		ent.C1 = 0
		return
	}

	i := (ent.Y << 8) + ent.OffSY + ent.YLow
	y := i >> 8
	_, env1 := g.envTest(ent.X, y, false)
	if env1&(mapEFlgSolid|mapEFlgSPad|mapEFlgWayUp) == 0 {
		if env1&mapEFlgLethal != 0 {
			g.themGoZombie(e)
			return
		}
		if y > 0x0140 {
			ent.N = 0
			return
		}
		if env1&mapEFlgVert == 0 {
			ent.Y = y
			ent.YLow = i & 0xff
			ent.OffSY += 0x0080
			if ent.OffSY > 0x0800 {
				ent.OffSY = 0x0800
			}
			return
		}
		if ent.X&0x07 == 0x04 && y < g.eRick().Y {
			ent.C1 = 1
			return
		}
	}
	ent.Y = (ent.Y & 0xf8) | 0x03
	ent.OffSY = 0x0100
	if ent.Latency != 0 {
		return
	}
	if env1&mapEFlgClimb != 0 && ent.X&0x0e == 0x04 && ent.Y > g.eRick().Y {
		ent.C1 = 1
		return
	}
	idx := (ent.X & 0x0e) >> 3
	if ent.C2 < 0 {
		idx += 4
	}
	ent.Sprite = ent.SprBase + int(g.data.SprSeq[idx])
	if ent.C2 == 0 {
		ent.C2 = 2
	}
	x := ent.X + ent.C2
	if x < 0xe8 {
		_, env1 = g.envTest(x, ent.Y, false)
		if env1&(mapEFlgVert|mapEFlgSolid|mapEFlgSPad|mapEFlgWayUp) == 0 {
			ent.X = x
			if x&0x1e != 0x08 {
				return
			}
			low := uint16(g.themRndSeed & 0xffff)
			high := uint16((g.themRndSeed >> 16) & 0xffff)
			bx := uint16(g.themRndNbr) + high + low + 0x0d
			bl := byte(bx & 0xff)
			bh := byte((bx >> 8) & 0xff)
			cl := byte(high & 0xff)
			ch := byte((high >> 8) & 0xff)
			bl ^= ch
			bl ^= cl
			bl ^= bh
			g.themRndNbr = bx
			if bl&0x01 != 0 {
				ent.C2 = -0x02
			} else {
				ent.C2 = 0x02
			}
			return
		}
	}
	if ent.C2 == 0 {
		ent.C2 = 2
	} else {
		ent.C2 = -ent.C2
	}
}

func (g *Core) themT2Post(e int) {
	if g.entities[e].N == 0 {
		return
	}
	if g.rickBoxTest(e) {
		g.rickGoZombie()
	}
	if g.themTest(e) {
		g.themGoZombie(e)
		return
	}
	if g.eBullet().N != 0 && g.fullBoxTest(e, g.eBullet().X+map[bool]int{true: 0x18, false: 0}[g.bulletOffsX >= 0], g.eBullet().Y) {
		g.eBullet().N = 0
		g.themGoZombie(e)
		return
	}
	if g.bombLethal && g.bombHit(e) {
		g.themGoZombie(e)
		return
	}
	if g.rickState&eRickStStop != 0 && g.fullBoxTest(e, g.rickStopX, g.rickStopY) {
		g.entities[e].Latency = 0x14
	}
}

func (g *Core) themT3Action(e int) {
	ent := &g.entities[e]
	for {
		i := int(g.data.SprSeq[ent.SprBase+ent.C1])
		if i == 0xff {
			i = int(g.data.SprSeq[ent.SprBase])
		}
		ent.Sprite = i
		if ent.C1 != 0 {
			if g.data.SprSeq[ent.SprBase+ent.C1] != 0xff {
				ent.C1++
			}
			if g.data.SprSeq[ent.SprBase+ent.C1] == 0xff {
				ent.C1 = 1
			}
			if ent.C2 < g.data.MvSteps[ent.StepNo].Count {
				ent.C2++
				x := ent.X + g.data.MvSteps[ent.StepNo].DX
				if x > 0 && x < 0xe8 {
					ent.X = x
					y := ent.Y + g.data.MvSteps[ent.StepNo].DY
					if y > 0 && y < 0x0140 {
						ent.Y = y
						break
					}
				}
			}
			ent.StepNo++
			if g.data.MvSteps[ent.StepNo].Count != 0xff {
				ent.C2 = 0
				continue
			}
			if g.rickState&eRickStZombie == 0 && ent.Flags&entFlgOnce == 0 {
				ent.C1 = 0
				ent.N &^= entLethal
				if ent.Flags&entFlgLethalR != 0 {
					ent.N |= entLethal
				}
				ent.X = ent.XSave
				ent.Y = ent.YSave
				if ent.Y < 0 || ent.Y > 0x140 {
					ent.N = 0
				}
			} else {
				ent.N = 0
			}
			break
		}

		if ent.Flags&entFlgTrigRick != 0 && g.triggerBox(e, g.eRick().X+0x0c, g.eRick().Y+0x0a) {
			goto wakeup
		}
		if ent.Flags&entFlgTrigStop != 0 && g.rickState&eRickStStop != 0 && g.triggerBox(e, g.rickStopX, g.rickStopY) {
			goto wakeup
		}
		if ent.Flags&entFlgTrigBullet != 0 && g.eBullet().N != 0 && g.triggerBox(e, g.bulletXC, g.bulletYC) {
			g.eBullet().N = 0
			goto wakeup
		}
		if ent.Flags&entFlgTrigBomb != 0 && g.bombLethal && g.triggerBox(e, g.bombXC, g.bombYC) {
			goto wakeup
		}
		break
	wakeup:
		if g.rickState&eRickStZombie != 0 {
			break
		}
		if n := (ent.TrigSnd & 0x1f) - 0x14; n >= 0 && n <= 8 {
			g.playEffect(fmt.Sprintf("ent%d.wav", n))
		}
		ent.N &^= entLethal
		if ent.Flags&entFlgLethalI != 0 {
			ent.N |= entLethal
		}
		ent.C1 = 1
		ent.C2 = 0
		ent.StepNo = ent.StepNoI
		break
	}
	if ent.N&entLethal != 0 && g.rickState&eRickStZombie == 0 && g.rickBoxTest(e) {
		g.rickGoZombie()
	}
}

func (g *Core) rickBoxTest(e int) bool {
	rick := g.eRick()
	if rick.X+0x11 < g.entities[e].X ||
		rick.X+0x05 > g.entities[e].X+g.entities[e].W ||
		rick.Y+0x14 < g.entities[e].Y ||
		rick.Y+map[bool]int{true: 0x08, false: 0x00}[g.rickState&eRickStCrawl != 0] > g.entities[e].Y+g.entities[e].H-1 {
		return false
	}
	return true
}

func (g *Core) rickGoZombie() {
	if g.envInvincible {
		return
	}
	if g.rickState&eRickStZombie != 0 {
		return
	}
	g.playEffect("die.wav")
	g.rickState |= eRickStZombie
	g.rickOffsY = -0x0400
	if g.eRick().X > 0x80 {
		g.rickOffsX = -3
	} else {
		g.rickOffsX = 3
	}
	g.rickYLow = 0
	g.eRick().Front = true
}

func (g *Core) rickZombieAction() {
	rick := g.eRick()
	if rick.X&0x04 != 0 {
		rick.Sprite = 15
	} else {
		rick.Sprite = 14
	}
	rick.X += g.rickOffsX
	i := (rick.Y << 8) + g.rickOffsY + g.rickYLow
	rick.Y = i >> 8
	g.rickOffsY += 0x80
	g.rickYLow = i & 0xff
	if rick.Y < 0 || rick.Y > 0x0140 {
		g.rickState |= eRickStDead
	}
}

func (g *Core) rickAction2() {
	rick := g.eRick()
	var x, i, y int
	var mask uint8
	var env0, env1 uint8
	control := g.stepControlStatus
	g.rickState &^= eRickStStop | eRickStShoot
	if g.rickState&eRickStZombie != 0 {
		g.rickZombieAction()
		return
	}
	if g.rickState&eRickStClimb != 0 {
		goto climbing
	}
	g.rickState &^= eRickStJump
	i = (rick.Y << 8) + g.rickOffsY + g.rickYLow
	y = i >> 8
	env0, env1 = g.envTest(rick.X, y, g.rickState&eRickStCrawl != 0)
	if g.rickState&eRickStCrawl != 0 && env0 == 0 {
		g.rickState &^= eRickStCrawl
	}
	mask = uint8(mapEFlgVert | mapEFlgSolid | mapEFlgSPad | mapEFlgWayUp)
	if g.rickOffsY < 0 {
		mask = uint8(mapEFlgVert | mapEFlgSolid | mapEFlgSPad)
	}
	if env1&mask != 0 {
		goto vertNot
	}
	g.rickState |= eRickStJump
	if env1&mapEFlgLethal != 0 {
		g.rickGoZombie()
		return
	}
	rick.Y = y
	g.rickYLow = i & 0xff
	if env1&mapEFlgClimb != 0 && control&(controlUp|controlDown) != 0 {
		g.rickOffsY = 0x0100
		g.rickState |= eRickStClimb
		return
	}
	g.rickOffsY += 0x0080
	if g.rickOffsY > 0x0800 {
		g.rickOffsY = 0x0800
		g.rickYLow = 0
	}
horiz:
	if control&(controlLeft|controlRight) == 0 {
		g.rickSeq = 2
		return
	}
	x = 0
	if control&controlLeft != 0 {
		x = rick.X - 2
		g.gameDir = left
		if x < 0 {
			g.rickAtExit = true
			rick.X = 0xe2
			return
		}
	} else {
		x = rick.X + 2
		g.gameDir = right
		if x >= 0xe8 {
			g.rickAtExit = true
			rick.X = 0x04
			return
		}
	}
	_, env1 = g.envTest(x, rick.Y, g.rickState&eRickStCrawl != 0)
	if env1&(mapEFlgSolid|mapEFlgSPad|mapEFlgWayUp) == 0 {
		rick.X = x
		if env1&mapEFlgLethal != 0 {
			g.rickGoZombie()
		}
	}
	return
vertNot:
	if g.rickOffsY < 0 {
		g.rickState |= eRickStJump
		rick.Y &= 0xf8
		g.rickOffsY = 0
		g.rickYLow = 0
		goto horiz
	}
	if env1&mapEFlgSPad != 0 && g.rickOffsY >= 0x0200 {
		g.playEffect("pad.wav")
		if control&controlUp != 0 {
			g.rickOffsY = 0xf800
		} else {
			g.rickOffsY = 0x00fe - g.rickOffsY
		}
		goto horiz
	}
	g.rickOffsY = 0x0100
	rick.Y &= 0xf8
	rick.Y |= 0x03
	g.rickYLow = 0
	if g.rickScrawl || control&controlFire == 0 {
		goto firingNot
	}
	if control&(controlLeft|controlRight) != 0 {
		if control&controlRight != 0 {
			g.gameDir = right
			g.rickStopX = rick.X + 0x17
		} else {
			g.gameDir = left
			g.rickStopX = rick.X
		}
		g.rickStopY = rick.Y + 0x0e
		g.rickState |= eRickStStop
		return
	}
	if control == (controlFire | controlUp) {
		g.rickState |= eRickStShoot
		if g.rickTrigger {
			return
		}
		g.rickTrigger = true
		if g.eBullet().N != 0 {
			return
		}
		if g.envBullets == 0 {
			return
		}
		if !g.envTrainer {
			g.envBullets--
		}
		g.bulletInit(rick.X, rick.Y)
		return
	}
	g.rickTrigger = false
	g.rickSeq = 0
	if control == (controlFire | controlDown) {
		if g.eBomb().N != 0 {
			return
		}
		if g.envBombs == 0 {
			return
		}
		if !g.envTrainer {
			g.envBombs--
		}
		g.bombInit(rick.X, rick.Y)
		return
	}
	return
firingNot:
	if control&controlUp != 0 {
		if env1&mapEFlgClimb != 0 {
			g.rickState |= eRickStClimb
			return
		}
		g.rickOffsY = -0x0580
		g.rickYLow = 0
		g.playEffect("jump.wav")
		goto horiz
	}
	if control&controlDown != 0 {
		if env1&mapEFlgVert != 0 && control&(controlLeft|controlRight) == 0 && (rick.X&0x1f) < 0x0a {
			rick.X &= 0xf0
			rick.X |= 0x04
			g.rickState |= eRickStClimb
		} else {
			g.rickState |= eRickStCrawl
			goto horiz
		}
	}
	goto horiz
climbing:
	if control&(controlUp|controlDown|controlLeft|controlRight) == 0 {
		g.rickSeq = 0
		return
	}
	if control&(controlUp|controlDown) != 0 {
		y = rick.Y
		if control&controlUp != 0 {
			y -= 2
		} else {
			y += 2
		}
		_, env1 = g.envTest(rick.X, y, g.rickState&eRickStCrawl != 0)
		if env1&(mapEFlgSolid|mapEFlgSPad|mapEFlgWayUp) != 0 && control&controlUp == 0 {
			g.rickState &^= eRickStClimb
			return
		}
		if env1&(mapEFlgSolid|mapEFlgSPad|mapEFlgWayUp) == 0 || env1&mapEFlgWayUp != 0 {
			rick.Y = y
			if env1&mapEFlgLethal != 0 {
				g.rickGoZombie()
				return
			}
			if env1&(mapEFlgVert|mapEFlgClimb) == 0 {
				if control&controlUp != 0 {
					g.rickOffsY = -0x0300
					g.playEffect("jump.wav")
				} else {
					g.rickOffsY = 0x0100
				}
				g.rickState &^= eRickStClimb
				return
			}
		}
	}
	if control&(controlLeft|controlRight) != 0 {
		if control&controlLeft != 0 {
			x = rick.X - 2
			if x < 0 {
				g.rickAtExit = true
				rick.X = 0xe2
				return
			}
		} else {
			x = rick.X + 2
			if x >= 0xe8 {
				g.rickAtExit = true
				rick.X = 0x04
				return
			}
		}
		_, env1 = g.envTest(x, rick.Y, g.rickState&eRickStCrawl != 0)
		if env1&(mapEFlgSolid|mapEFlgSPad) != 0 {
			return
		}
		rick.X = x
		if env1&mapEFlgLethal != 0 {
			g.rickGoZombie()
			return
		}
		if env1&(mapEFlgVert|mapEFlgClimb) != 0 {
			return
		}
		g.rickState &^= eRickStClimb
		if control&controlUp != 0 {
			g.rickOffsY = -0x0300
		}
	}
}

func (g *Core) rickAction() {
	g.rickAction2()
	rick := g.eRick()
	g.rickScrawl = g.rickState&eRickStCrawl != 0
	rick.Flip = g.gameDir == left
	if g.rickState&eRickStZombie != 0 {
		rick.Flip = false
		return
	}
	if g.rickState&eRickStShoot != 0 {
		rick.Sprite = 10
		return
	}
	if g.rickState&eRickStClimb != 0 {
		rick.Flip = false
		rick.Sprite = 12 + (((^rick.X)^rick.Y)&4)>>2
		g.rickSeq = (g.rickSeq + 1) & 7
		if g.rickSeq == 0 {
			g.playEffect("crawl")
		}
		return
	}
	if g.rickState&eRickStCrawl != 0 {
		rick.Sprite = 7 + (rick.X&4)>>2
		g.rickSeq = (g.rickSeq + 1) & 7
		if g.rickSeq == 0 {
			g.playEffect("crawl")
		}
		return
	}
	if g.rickState&eRickStJump != 0 {
		rick.Sprite = 9
		return
	}
	g.rickSeq++
	if g.rickSeq > 11 {
		g.rickSeq = 2
	}
	rick.Sprite = (g.rickSeq >> 1) + 1
}

func (g *Core) rickSave() {
	g.rickSaveX = g.eRick().X
	g.rickSaveY = g.eRick().Y
	g.rickSaveCrawl = g.rickState&eRickStCrawl != 0
	g.rickSaveState = g.rickState & eRickStCrawl
	g.rickSaveOffsY = g.rickOffsY
	g.rickSaveYLow = g.rickYLow
	g.rickSaveDir = g.gameDir
}

func (g *Core) rickRestore() {
	g.eRick().X = g.rickSaveX
	g.eRick().Y = g.rickSaveY
	g.eRick().Front = false
	g.rickState = g.rickSaveState
	if g.rickSaveCrawl {
		g.rickState |= eRickStCrawl
	} else {
		g.rickState &^= eRickStCrawl
	}
	g.rickOffsY = g.rickSaveOffsY
	g.rickYLow = g.rickSaveYLow
	g.rickOffsX = 0
	g.rickSeq = 0
	g.gameDir = g.rickSaveDir
	g.rickScrawl = g.rickState&eRickStCrawl != 0
}

func (g *Core) fullBoxTest(e, x, y int) bool {
	ent := &g.entities[e]
	if ent.X >= x || ent.X+ent.W < x || ent.Y >= y || ent.Y+ent.H < y {
		return false
	}
	return true
}

func (g *Core) boxTest(e1, e2 int) bool {
	if e1 == eRickNo {
		return g.rickBoxTest(e2)
	}
	a := &g.entities[e1]
	b := &g.entities[e2]
	if a.X+0x11 < b.X || a.X+0x05 > b.X+b.W || a.Y+0x14 < b.Y || a.Y > b.Y+b.H-1 {
		return false
	}
	return true
}

func (g *Core) envTest(x, y int, crawl bool) (uint8, uint8) {
	g.entities[12].X = x
	g.entities[12].Y = y
	i := 1
	if !crawl {
		i++
	}
	if y&0x0004 != 0 {
		i++
	}
	x += 4
	xx := x
	tx := x >> 3
	ty := y >> 3
	var rc0, rc1 uint8
	if xx&0x07 != 0 {
		if crawl {
			rc0 |= g.mapEFlags[g.tileAt(ty, tx)] & (mapEFlgVert | mapEFlgSolid | mapEFlgSPad | mapEFlgWayUp)
			rc0 |= g.mapEFlags[g.tileAt(ty, tx+1)] & (mapEFlgVert | mapEFlgSolid | mapEFlgSPad | mapEFlgWayUp)
			rc0 |= g.mapEFlags[g.tileAt(ty, tx+2)] & (mapEFlgVert | mapEFlgSolid | mapEFlgSPad | mapEFlgWayUp)
			ty++
		}
		for {
			rc1 |= g.mapEFlags[g.tileAt(ty, tx)] & (mapEFlgSolid | mapEFlgSPad | mapEFlgFGnd | mapEFlgLethal | mapEFlg01)
			rc1 |= g.mapEFlags[g.tileAt(ty, tx+1)] & (mapEFlgSolid | mapEFlgSPad | mapEFlgFGnd | mapEFlgLethal | mapEFlgClimb | mapEFlg01)
			rc1 |= g.mapEFlags[g.tileAt(ty, tx+2)] & (mapEFlgSolid | mapEFlgSPad | mapEFlgFGnd | mapEFlgLethal | mapEFlg01)
			ty++
			i--
			if i <= 0 {
				break
			}
		}
		rc1 |= g.mapEFlags[g.tileAt(ty, tx)] & (mapEFlgSolid | mapEFlgSPad | mapEFlgWayUp | mapEFlgFGnd | mapEFlgLethal | mapEFlg01)
		rc1 |= g.mapEFlags[g.tileAt(ty, tx+1)]
		rc1 |= g.mapEFlags[g.tileAt(ty, tx+2)] & (mapEFlgSolid | mapEFlgSPad | mapEFlgWayUp | mapEFlgFGnd | mapEFlgLethal | mapEFlg01)
	} else {
		if crawl {
			rc0 |= g.mapEFlags[g.tileAt(ty, tx)] & (mapEFlgVert | mapEFlgSolid | mapEFlgSPad | mapEFlgWayUp)
			rc0 |= g.mapEFlags[g.tileAt(ty, tx+1)] & (mapEFlgVert | mapEFlgSolid | mapEFlgSPad | mapEFlgWayUp)
			ty++
		}
		for {
			rc1 |= g.mapEFlags[g.tileAt(ty, tx)] & (mapEFlgSolid | mapEFlgSPad | mapEFlgFGnd | mapEFlgLethal | mapEFlgClimb | mapEFlg01)
			rc1 |= g.mapEFlags[g.tileAt(ty, tx+1)] & (mapEFlgSolid | mapEFlgSPad | mapEFlgFGnd | mapEFlgLethal | mapEFlgClimb | mapEFlg01)
			ty++
			i--
			if i <= 0 {
				break
			}
		}
		rc1 |= g.mapEFlags[g.tileAt(ty, tx)]
		rc1 |= g.mapEFlags[g.tileAt(ty, tx+1)]
	}
	if rc1&mapEFlgLethal == 0 && g.entities[0].N != 0 && g.boxTest(12, 0) {
		rc1 |= mapEFlgSolid
	}
	if g.envInvincible {
		rc1 &^= mapEFlgLethal
	}
	return rc0, rc1
}

func (g *Core) triggerBox(e, x, y int) bool {
	ent := &g.entities[e]
	info := g.data.EntData[ent.N&0x7f]
	xmax := ent.TrigX + (info.TrigW << 3)
	ymax := ent.TrigY + (info.TrigH << 3)
	if xmax > 0xff {
		xmax = 0xff
	}
	if x <= ent.TrigX || x > xmax || y <= ent.TrigY || y > ymax {
		return false
	}
	return true
}

func (g *Core) tileAt(row, col int) uint8 {
	if row < 0 || row >= len(g.mapMap) || col < 0 || col >= len(g.mapMap[0]) {
		return 0
	}
	return g.mapMap[row][col]
}
