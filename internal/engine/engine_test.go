package engine

import (
	"encoding/binary"
	"github.com/olivierh59500/go-rick32/internal/data"
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
