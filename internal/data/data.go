// Package data reads Rick32's compact, native tables.
package data

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/olivierh59500/go-rick32/assets"
)

type Map struct {
	X, Y, Row, Submap int
	Tune              string
}
type Submap struct{ Page, BNum, Connect, Mark int }
type Connect struct{ Dir, RowOut, Submap, RowIn int }
type Mark struct{ Row, Entity, Flags, XY, LT int }
type EntityData struct{ W, H, Spr, SNI, TrigW, TrigH, Snd int }
type MoveStep struct{ Count, DX, DY int }
type Line struct {
	Y     float64 `json:"y"`
	Text  string  `json:"text"`
	Color string  `json:"color"`
}
type Page struct {
	Hold  float64 `json:"hold"`
	Lines []Line  `json:"lines"`
}

type Data struct {
	Maps           []Map
	Submaps        []Submap
	Connects       []Connect
	BNums          []byte
	Blocks         [][]byte
	Marks          []Mark
	EntData        []EntityData
	SprSeq         []byte
	MvSteps        []MoveStep
	Tiles, Sprites [][]byte
	Flags          []byte
	Demo           []byte
	Palette        []uint16
	Pages          map[string]Page
}

func Load() (*Data, error) {
	read := func(name string, size int) ([]byte, error) {
		b, err := assets.Files.ReadFile(name + ".bin")
		if err != nil {
			return nil, err
		}
		if len(b) != size {
			return nil, fmt.Errorf("data: %s contains %d bytes, expected %d", name, len(b), size)
		}
		return b, nil
	}
	tables := map[string]int{"blocks": 58 * 16, "demo": 2272, "sprites": 51 * 32 * 21, "map": 2048, "tiles": 141 * 64, "marks": 508, "entities": 74 * 8, "sequences": 136, "moves": 76 * 3, "flags": 141, "connections": 9 * 8, "map-starts": 18, "mark-starts": 9, "palette": 32}
	bank := make(map[string][]byte, len(tables))
	for name, size := range tables {
		b, err := read(name, size)
		if err != nil {
			return nil, err
		}
		bank[name] = b
	}
	d := &Data{BNums: bank["map"], SprSeq: bank["sequences"], Flags: bank["flags"], Demo: bank["demo"], Maps: []Map{{8, 139, 8, 0, "game.ym"}, {8, 139, 8, 0, "complete.ym"}}}
	for i := 0; i < 9; i++ {
		d.Submaps = append(d.Submaps, Submap{BNum: int(binary.LittleEndian.Uint16(bank["map-starts"][i*2:])), Connect: i * 3, Mark: int(bank["mark-starts"][i])})
		for j := 0; j < 2; j++ {
			b := bank["connections"][i*8+j*4:]
			d.Connects = append(d.Connects, Connect{int(b[0]), int(b[1]), int(b[2]), int(b[3])})
		}
		d.Connects = append(d.Connects, Connect{Dir: 255})
	}
	for i := 0; i+5 <= len(bank["marks"]); i += 5 {
		b := bank["marks"][i:]
		d.Marks = append(d.Marks, Mark{int(b[0]), int(b[1]), int(b[2]), int(b[3]), int(b[4])})
	}
	for i := 0; i < 74; i++ {
		b := bank["entities"][i*8:]
		d.EntData = append(d.EntData, EntityData{W: int(b[0]), H: int(b[1]), Spr: int(binary.LittleEndian.Uint16(b[2:])), SNI: int(binary.LittleEndian.Uint16(b[4:])), TrigW: int(b[6]), TrigH: int(b[7])})
	}
	for i := 0; i < 76; i++ {
		b := bank["moves"][i*3:]
		d.MvSteps = append(d.MvSteps, MoveStep{int(b[0]), int(int8(b[1])), int(int8(b[2]))})
	}
	slice := func(b []byte, stride int) [][]byte {
		out := make([][]byte, len(b)/stride)
		for i := range out {
			out[i] = b[i*stride : (i+1)*stride]
		}
		return out
	}
	d.Blocks = slice(bank["blocks"], 16)
	d.Tiles = slice(bank["tiles"], 64)
	d.Sprites = slice(bank["sprites"], 32*21)
	for i := 0; i < 16; i++ {
		d.Palette = append(d.Palette, binary.LittleEndian.Uint16(bank["palette"][i*2:]))
	}
	p, err := assets.Files.ReadFile("pages.json")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(p, &d.Pages); err != nil {
		return nil, err
	}
	return d, nil
}
