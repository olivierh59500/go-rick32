package data

import "testing"

func TestNativeTables(t *testing.T) {
	d, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Submaps) != 9 || len(d.Sprites) != 51 || len(d.Tiles) != 141 || len(d.Demo) != 2272 {
		t.Fatal("incomplete Rick32 tables")
	}
	for _, block := range d.Blocks {
		for _, tile := range block {
			if int(tile) >= len(d.Tiles) {
				t.Fatalf("invalid tile %d", tile)
			}
		}
	}
	for _, block := range d.BNums {
		if int(block) >= len(d.Blocks) {
			t.Fatalf("invalid block %d", block)
		}
	}
	for i, room := range d.Submaps {
		if room.BNum+176 > len(d.BNums) || room.Mark >= len(d.Marks) {
			t.Fatalf("invalid room %d", i)
		}
	}
	if d.Pages["complete"].Hold != 43200 {
		t.Fatal("missing completion screen")
	}
}
