package game

import "testing"

func TestGenerateIsPlayableAndDeterministic(t *testing.T) {
	for seed := int64(1); seed <= 25; seed++ {
		l := Generate(seed)

		if l.W <= 0 || l.H != chunkRows {
			t.Fatalf("seed %d: bad dimensions %dx%d", seed, l.W, l.H)
		}

		// The player must spawn in open air with ground below.
		if l.SolidAtPx(l.SpawnX, l.SpawnY) {
			t.Fatalf("seed %d: spawn inside solid terrain", seed)
		}
		if !l.SolidBox(l.SpawnX-1, l.SpawnY, 2, float64(l.PxH())-l.SpawnY) {
			t.Fatalf("seed %d: no ground below spawn", seed)
		}

		parts := 0
		for _, s := range l.Spawns {
			if s.Kind == 'P' {
				parts++
			}
			if l.SolidAtPx(s.X, s.Y) {
				t.Fatalf("seed %d: spawn %q inside solid terrain at %.0f,%.0f", seed, s.Kind, s.X, s.Y)
			}
		}
		if parts < 3 {
			t.Fatalf("seed %d: want at least 3 ship parts, got %d", seed, parts)
		}

		// Same seed must give the same planet.
		l2 := Generate(seed)
		for i := range l.tiles {
			if l.tiles[i] != l2.tiles[i] {
				t.Fatalf("seed %d: generation is not deterministic", seed)
			}
		}
	}
}

func TestWorldBounds(t *testing.T) {
	l := Generate(42)
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
	// Every middle chunk must have solid floor and open air at both
	// edges so any chunk order (and mirroring) stays traversable.
	for i, c := range middleChunks() {
		for _, tx := range []int{0, 1, c.w - 2, c.w - 1} {
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
