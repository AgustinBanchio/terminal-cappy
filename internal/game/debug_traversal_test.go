package game

// Debug harness for the traversal analyser: prints per-column surface
// reachability and a few underground rows, marking F (reachable) and B
// (reachable with a way back). Useful when a map edit fails
// TestEmbeddedLevelTraversal and you want to see where the frontier is:
//
//	GENMAP_DEBUG_LEVEL=<levelfile> go test -run TestDebugTraversal -v ./internal/game

import (
	"fmt"
	"os"
	"testing"
)

func TestDebugTraversal(t *testing.T) {
	path := os.Getenv("GENMAP_DEBUG_LEVEL")
	if path == "" {
		t.Skip("no GENMAP_DEBUG_LEVEL set")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	l, err := ParseLevel(data)
	if err != nil {
		t.Fatal(err)
	}

	g := newTravGraph(l)
	sx, sy := fdiv(int(l.SpawnX), TilePx), fdiv(int(l.SpawnY), TilePx)
	fwd := g.bfs([]int{g.norm(sx, sy, travRise, travDrift)})
	targetCell := make([]bool, l.W*l.H)
	targetCell[sy*l.W+sx] = true
	bwd := g.bfsBackward(targetCell)

	cellF := func(x, y int) bool {
		for k := 0; k < g.nb; k++ {
			if fwd[(y*l.W+x)*g.nb+k] {
				return true
			}
		}
		return false
	}
	cellB := func(x, y int) bool {
		for k := 0; k < g.nb; k++ {
			if fwd[(y*l.W+x)*g.nb+k] && bwd[(y*l.W+x)*g.nb+k] {
				return true
			}
		}
		return false
	}
	// standing tile per column: first supported fit below the sky
	standAt := func(x int) int {
		for y := 0; y < l.H; y++ {
			if g.fits[y*l.W+x] && g.supported(x, y) {
				return y
			}
		}
		return -1
	}

	fmt.Println("surface forward/back reach per column (F=fwd only, B=both, .=no):")
	line := ""
	for x := 0; x < l.W; x++ {
		y := standAt(x)
		ch := "?"
		switch {
		case y < 0:
			ch = " "
		case cellB(x, y):
			ch = "B"
		case cellF(x, y):
			ch = "F"
		default:
			ch = "."
		}
		line += ch
		if (x+1)%80 == 0 {
			fmt.Printf("x%3d %s\n", x-79, line)
			line = ""
		}
	}
	fmt.Println("underground row 34/55 fwd/bwd:")
	for _, y := range []int{34, 38, 44, 51, 55} {
		line = ""
		for x := 0; x < l.W; x++ {
			ch := "#"
			if g.fits[y*l.W+x] {
				switch {
				case cellB(x, y):
					ch = "B"
				case cellF(x, y):
					ch = "F"
				default:
					ch = "."
				}
			}
			line += ch
		}
		fmt.Printf("row %d:\n%s\n", y, line)
	}
}
