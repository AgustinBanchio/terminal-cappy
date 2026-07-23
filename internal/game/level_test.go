package game

import (
	"bytes"
	"testing"
)

func TestEmbeddedLevelIsPlayable(t *testing.T) {
	l := LoadDefault()

	if l.W <= 0 || l.H <= 0 {
		t.Fatalf("bad dimensions %dx%d", l.W, l.H)
	}

	// The player must spawn in open air with ground below.
	if l.SolidAtPx(l.SpawnX, l.SpawnY) {
		t.Fatal("spawn inside solid terrain")
	}
	if !l.SolidBox(l.SpawnX-1, l.SpawnY, 2, float64(l.PxH())-l.SpawnY) {
		t.Fatal("no ground below spawn")
	}

	parts := 0
	for _, s := range l.Spawns {
		if s.Kind == 'P' {
			parts++
		}
		if l.SolidAtPx(s.X, s.Y) {
			t.Fatalf("spawn %q inside solid terrain at %.0f,%.0f", s.Kind, s.X, s.Y)
		}
	}
	if parts < 1 {
		t.Fatal("level has no ship parts, cannot be won")
	}

	// The ship must sit inside the world with ground under it.
	if l.ShipX < 0 || l.ShipX+sprShip.W > l.PxW() {
		t.Fatalf("ship out of bounds at x=%d", l.ShipX)
	}
	if !l.SolidAtPx(float64(l.ShipX+sprShip.W/2), float64(l.ShipY+sprShip.H+2)) {
		t.Fatal("no ground under the ship")
	}
}

func TestEmbeddedLevelTraversal(t *testing.T) {
	// The movement-model analysis must stay clean: every part, boss and
	// the ship reachable from spawn, and no reachable spot without a
	// way back. Guards hand edits to level1.txt as well as genmap.
	if issues := LoadDefault().AnalyzeTraversal(); len(issues) > 0 {
		for _, s := range issues {
			t.Error(s)
		}
	}
}

func TestLevelRoundTrip(t *testing.T) {
	l := LoadDefault()
	data := l.Marshal()
	l2, err := ParseLevel(data)
	if err != nil {
		t.Fatalf("re-parse failed: %v", err)
	}
	if !bytes.Equal(l2.Marshal(), data) {
		t.Fatal("marshal/parse round trip is not stable")
	}
}

func TestWorldBounds(t *testing.T) {
	l := LoadDefault()
	if !l.SolidTile(-1, 5) || !l.SolidTile(l.W, 5) {
		t.Fatal("world edges must be solid walls")
	}
	if l.SolidTile(3, -1) {
		t.Fatal("sky above the world must be open")
	}
	if l.SolidTile(3, l.H) {
		t.Fatal("void below the world must be open (falling is fatal)")
	}
}

func TestSetCell(t *testing.T) {
	l := LoadDefault()

	if l.SetCell(LayerBG, 0, 0, '#') {
		t.Fatal("rock must not be a valid bg tile")
	}
	if !l.SetCell(LayerBG, 0, 0, 'c') || l.Cell(LayerBG, 0, 0) != 'c' {
		t.Fatal("valid bg tile not applied")
	}

	// Solid edits must refresh collision immediately.
	if !l.SetCell(LayerSolid, 2, 2, '#') || !l.SolidTile(2, 2) {
		t.Fatal("solid edit did not refresh collision")
	}

	// Placing a second spawn moves it: exactly one 'S' remains.
	l.SetCell(LayerSolid, 5, 5, 'S')
	count := 0
	for ty := 0; ty < l.H; ty++ {
		for tx := 0; tx < l.W; tx++ {
			if l.Cell(LayerSolid, tx, ty) == 'S' {
				count++
			}
		}
	}
	if count != 1 {
		t.Fatalf("want exactly 1 spawn after move, got %d", count)
	}
	if l.SpawnX != float64(5*TilePx+TilePx/2) {
		t.Fatal("spawn position not updated after move")
	}
}

func TestParseRejectsBadLevels(t *testing.T) {
	cases := []string{
		"",
		"not-a-level",
		"cappy-level v1\n@solid\n##\n#\n@bg\n..\n..\n@fg\n..\n..\n",  // ragged
		"cappy-level v1\n@solid\n##\n##\n@bg\n..\n..\n",              // missing fg
		"cappy-level v1\n@solid\nZ#\n##\n@bg\n..\n..\n@fg\n..\n..\n", // bad tile
		"cappy-level v1\n@solid\n..\n##\n@bg\n..\n..\n@fg\n..\n..\n", // no spawn/ship
	}
	for i, c := range cases {
		if _, err := ParseLevel([]byte(c)); err == nil {
			t.Fatalf("case %d: want parse error, got none", i)
		}
	}
}
