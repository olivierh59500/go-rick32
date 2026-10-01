package engine

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"github.com/olivierh59500/go-rick32/internal/data"
	"io"
	"os"
	"testing"
)

func TestRecordedAttractRoute(t *testing.T) {
	d, err := data.Load()
	if err != nil {
		t.Fatal(err)
	}
	g := New(d)
	reference, err := os.ReadFile("testdata/native-demo.bin")
	if err != nil {
		t.Fatal(err)
	}
	if len(reference) != len(d.Demo)*12 {
		t.Fatal("incomplete native replay fixture")
	}
	rooms := map[int]bool{}
	for i, input := range d.Demo {
		g.Step(input)
		s := g.State()
		rooms[s.Room] = true
		got := [6]int{s.Room, s.Row, s.Rick.X, s.Rick.Y, s.Rick.Sprite, int(s.Flags)}
		for k, v := range got {
			want := int(int16(binary.LittleEndian.Uint16(reference[i*12+k*2:])))
			if v != want {
				t.Fatalf("native sample %d field %d: got %d want %d", i, k, v, want)
			}
		}
		for n, e := range *g.Entities() {
			if e.N != 0 && n < 12 && e.Y >= 64 && e.Y < 256 && (e.Sprite < 0 || e.Sprite >= len(d.Sprites)) {
				t.Fatalf("sample %d: entity %d has sprite %d: %+v; state=%+v", i, n, e.Sprite, e, g.State())
			}
		}
		if s.Dead && i < 2238 {
			t.Fatalf("recorded route died too early at sample %d", i)
		}
	}
	if len(rooms) != 4 || !g.Dead() || g.State().Score != 4395 {
		t.Fatalf("unexpected attract ending: %+v", g.State())
	}
}

func TestJumpAndRestart(t *testing.T) {
	d, err := data.Load()
	if err != nil {
		t.Fatal(err)
	}
	g := New(d)
	for i := 0; i < 10; i++ {
		g.Step(Up)
	}
	if g.State().Rick.Y >= 139 {
		t.Fatal("Rick did not jump")
	}
	g.Reset()
	if s := g.State(); s.Rick.X != 8 || s.Rick.Y != 139 || s.Lives != 6 || s.Score != 0 {
		t.Fatalf("reset: %+v", s)
	}
}

func TestEveryNativeRoom(t *testing.T) {
	d, err := data.Load()
	if err != nil {
		t.Fatal(err)
	}
	for room := 0; room < 9; room++ {
		g := New(d)
		g.envSubmap = room
		g.mapFRow = 8
		for _, conn := range d.Connects {
			if conn.Submap == room && conn.Dir == right {
				g.mapFRow = conn.RowIn - 16
				break
			}
		}
		g.mapInit()
		g.rickSave()
		g.saveMapRow = g.mapFRow
		for i := 0; i < 200; i++ {
			if g.Dead() {
				g.Respawn()
			}
			g.Step(0)
		}
	}
}

// This fixture comes from executing the original x86 entity routines, rather
// than from the Go implementation. It includes both guard banks and deaths.
func TestNativeEnemyAndItemSprites(t *testing.T) {
	d, err := data.Load()
	if err != nil {
		t.Fatal(err)
	}
	compressed, err := os.ReadFile("testdata/native-entities.bin.gz")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	reference, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	g := New(d)
	offset := 0
	checked := 0
	for tick, input := range d.Demo[:2238] {
		g.Step(input)
		for offset < len(reference) && int(int16(binary.LittleEndian.Uint16(reference[offset:]))) == tick {
			var expected [9]int
			for k := range expected {
				expected[k] = int(int16(binary.LittleEndian.Uint16(reference[offset+k*2:])))
			}
			slot := expected[1]
			entity := g.entities[slot]
			flip := 0
			if entity.Flip {
				flip = 1
			}
			got := [9]int{tick, slot, entity.N, entity.Mark, entity.X, entity.Y, entity.Sprite, entity.SprBase, flip}
			if got != expected {
				t.Fatalf("native entity sample: got %v want %v", got, expected)
			}
			offset += 18
			checked++
		}
	}
	if offset != len(reference) || checked != 7261 {
		t.Fatalf("checked %d samples, expected 7261", checked)
	}
}
