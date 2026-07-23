package game

import "testing"

func TestBuildWorld(t *testing.T) {
	l := Build()

	if l.W <= 0 || l.H != chunkRows {
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
	if parts != 4 {
		t.Fatalf("want 4 ship parts in the curated world, got %d", parts)
	}

	// The ship must sit inside the world with its landing spot solid.
	if l.ShipX < 0 || l.ShipX+sprShip.W > l.PxW() {
		t.Fatalf("ship out of bounds at x=%d", l.ShipX)
	}
	if !l.SolidAtPx(float64(l.ShipX+sprShip.W/2), float64(l.ShipY+sprShip.H+2)) {
		t.Fatal("no ground under the ship")
	}
}

func TestWorldBounds(t *testing.T) {
	l := Build()
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

func TestChunkEdgeContract(t *testing.T) {
	// Every segment must have solid floor and open air at both edges so
	// the curated sequence stays traversable on foot. The final chunk is
	// exempt: its far edge is the deliberate wall capping the world.
	chunks := worldChunks()
	for i, c := range chunks {
		edges := []int{0, 1, c.w - 2, c.w - 1}
		if i == len(chunks)-1 {
			edges = edges[:2]
		}
		for _, tx := range edges {
			for ty := 20; ty < chunkRows; ty++ {
				if c.rows[ty][tx] != '#' {
					t.Fatalf("chunk %d: edge column %d row %d is not floor", i, tx, ty)
				}
			}
			for ty := 14; ty < 20; ty++ {
				if c.rows[ty][tx] == '#' {
					t.Fatalf("chunk %d: edge column %d row %d blocks walking", i, tx, ty)
				}
			}
		}
	}
}
