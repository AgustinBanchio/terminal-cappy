// genmap is a one-off builder for the world in internal/game/level1.txt.
// The layout is curated in code here; fine detail is then hand-edited
// with cmd/leveled. Regenerating OVERWRITES hand edits.
package main

import (
	"fmt"
	"math"
	"os"

	"github.com/AgustinBanchio/terminal-cappy/internal/game"
)

const (
	W = 320
	H = 64
)

type gridSet struct {
	solid, bg, fg, zone []byte
}

func newGrid() *gridSet {
	g := &gridSet{
		solid: make([]byte, W*H),
		bg:    make([]byte, W*H),
		fg:    make([]byte, W*H),
		zone:  make([]byte, W*H),
	}
	for i := range g.solid {
		g.solid[i], g.bg[i], g.fg[i], g.zone[i] = '.', '.', '.', '.'
	}
	return g
}

func (g *gridSet) set(l []byte, x, y int, ch byte) {
	if x >= 0 && x < W && y >= 0 && y < H {
		l[y*W+x] = ch
	}
}

func (g *gridSet) get(l []byte, x, y int) byte {
	if x < 0 || x >= W || y < 0 || y >= H {
		return '#'
	}
	return l[y*W+x]
}

func (g *gridSet) rect(l []byte, x0, y0, x1, y1 int, ch byte) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			g.set(l, x, y, ch)
		}
	}
}

func (g *gridSet) solidAt(x, y int) bool {
	c := g.get(g.solid, x, y)
	return c == '#' || c == '%' || c == 'X'
}

func hash2(x, y int) uint32 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	return h ^ (h >> 16)
}

// groundRow is the curated surface line: sunken ruins west, a flat
// crash site in the middle, rolling hills east.
func groundRow(x int) int {
	switch {
	case x < 58:
		return 24 // sunken ruins shelf
	case x < 70:
		return 24 - (x-58)/3 // slope back up to the plains
	case x >= 124 && x <= 158:
		return 20 // crash site, flat
	case x >= 298 && x <= 312:
		return 17 // eastern lookout plateau
	default:
		w := 2*math.Sin(float64(x)*0.07) + 1.5*math.Sin(float64(x)*0.023+1.3)
		return 20 + int(math.Round(w/1.4))
	}
}

func main() {
	g := newGrid()

	// --- terrain shell ---------------------------------------------------
	for x := 0; x < W; x++ {
		for y := groundRow(x); y <= 25; y++ {
			g.set(g.solid, x, y, '#')
		}
	}
	g.rect(g.solid, 0, 26, W-1, H-1, '#') // underground mass
	// zones: surface above, cave below; overridden per region later
	g.rect(g.zone, 0, 0, W-1, 27, 's')
	g.rect(g.zone, 0, 28, W-1, H-1, 'u')

	// --- crash site --------------------------------------------------------
	g.set(g.solid, 130, 17, 'H')
	g.set(g.solid, 146, 18, 'S')

	// --- west: the ruins + Dimi's chamber ----------------------------------
	// Broken structures along the sunken shelf, clear of the shaft at
	// x96 so nothing overhangs the way down.
	for _, gx := range []int{54, 70, 86, 110} {
		base := groundRow(gx)
		g.rect(g.solid, gx, base-5, gx+1, base-1, '%')
		g.rect(g.solid, gx+8, base-4, gx+9, base-1, '%')
		for x := gx; x <= gx+6; x++ { // broken lintel
			if hash2(x, 77)%3 != 0 {
				g.set(g.solid, x, base-6, '%')
			}
		}
		g.rect(g.bg, gx+4, base-3, gx+5, base-1, 'r')
		g.set(g.bg, gx+11, base-1, 'b')
		g.set(g.bg, gx-2, base-1, 'b')
	}
	// Dimi's chamber: a great ruined hall.
	g.rect(g.solid, 8, 6, 46, 7, '%')   // ceiling
	g.rect(g.solid, 8, 8, 9, 25, '%')   // west wall
	g.rect(g.solid, 45, 8, 46, 25, '%') // east wall
	g.rect(g.solid, 8, 24, 46, 25, '%') // floor
	g.rect(g.solid, 10, 8, 44, 23, '.') // interior
	g.rect(g.solid, 45, 20, 46, 23, 'd')
	g.set(g.solid, 26, 22, 'D')
	for x := 12; x <= 42; x += 6 { // ruined colonnade backdrop
		g.rect(g.bg, x, 18, x, 23, 'r')
	}

	// --- surface shafts down -----------------------------------------------
	// Open wells, four tiles wide: an unobstructed drop going down and a
	// comfortable wall-jump chimney coming back up. The mouth opens from
	// the highest ground across the shaft so no lip juts into it.
	shaft := func(x0, depth int) {
		top := H
		for x := x0; x < x0+4; x++ {
			if gr := groundRow(x); gr < top {
				top = gr
			}
		}
		g.rect(g.solid, x0, top, x0+3, depth, '.')
	}
	shaft(96, 30)
	shaft(186, 30)
	shaft(262, 40)

	// --- cave tunnels --------------------------------------------------------
	for x := 14; x <= 300; x++ { // upper tunnel
		floor := 35 + int(math.Round(math.Sin(float64(x)*0.15)))
		g.rect(g.solid, x, 30, x, floor, '.')
	}
	for x := 14; x <= 292; x++ { // lower tunnel
		floor := 44 + int(math.Round(math.Sin(float64(x)*0.11+2)))
		g.rect(g.solid, x, 40, x, floor, '.')
	}
	// connectors between tunnels
	g.rect(g.solid, 56, 35, 59, 40, '.')
	g.rect(g.solid, 150, 35, 153, 40, '.')
	g.rect(g.solid, 236, 35, 239, 40, '.')
	// carved chambers
	g.rect(g.solid, 62, 32, 80, 38, '.')
	g.rect(g.solid, 152, 29, 170, 35, '.')
	g.rect(g.solid, 222, 41, 238, 45, '.')

	// --- crystal region (south-east) ----------------------------------------
	g.rect(g.zone, 194, 42, W-1, H-1, 'k')
	g.rect(g.solid, 200, 48, 262, 58, '.') // grand cavern
	for _, px := range []int{214, 232, 248} {
		g.rect(g.solid, px, 52, px+1, 58, '#') // remnant pillars
	}
	g.rect(g.solid, 246, 42, 249, 48, '.') // way in from the lower tunnel
	// Prisma's chamber
	g.rect(g.solid, 270, 46, 308, 47, '#') // ceiling
	g.rect(g.solid, 270, 48, 271, 58, '#') // west wall
	g.rect(g.solid, 307, 48, 308, 58, '#') // east wall
	g.rect(g.solid, 272, 48, 306, 57, '.') // interior
	g.rect(g.solid, 270, 53, 271, 56, 'd')
	g.rect(g.solid, 262, 52, 269, 56, '.') // approach corridor
	g.set(g.solid, 289, 54, 'Q')
	// crystal rock conversion
	for y := 42; y < H; y++ {
		for x := 194; x < W; x++ {
			if g.get(g.solid, x, y) == '#' {
				g.set(g.solid, x, y, 'X')
			}
		}
	}

	// --- lava region (south-west) --------------------------------------------
	g.rect(g.zone, 0, 44, 142, H-1, 'l')
	g.rect(g.solid, 12, 50, 136, 57, '.') // great burning cavern
	g.rect(g.solid, 16, 42, 19, 50, '.')  // west way down
	g.rect(g.solid, 84, 42, 87, 50, '.')  // east way down
	// Magmaw's chamber inside the cavern
	g.rect(g.solid, 28, 48, 74, 49, '#') // ceiling
	g.rect(g.solid, 28, 50, 29, 57, '#') // west wall
	g.rect(g.solid, 73, 50, 74, 57, '#') // east wall
	g.rect(g.solid, 30, 50, 72, 57, '.') // interior
	g.rect(g.solid, 73, 54, 74, 57, 'd')
	g.set(g.solid, 50, 55, 'M')
	g.rect(g.solid, 46, 57, 51, 57, '~') // a molten puddle to hop
	// pools outside, with easy jumps between islands
	for _, p := range [][2]int{{88, 93}, {104, 109}, {122, 127}} {
		g.rect(g.solid, p[0], 58, p[1], 59, '~')
	}

	// --- ship parts (non-boss) -----------------------------------------------
	g.set(g.solid, 305, 15, 'P') // eastern lookout
	g.set(g.solid, 161, 31, 'P') // upper cave chamber
	g.set(g.solid, 208, 50, 'P') // crystal cavern
	g.set(g.solid, 20, 54, 'P')  // lava west pocket

	// --- decorations, by zone -------------------------------------------------
	for y := 0; y < H-1; y++ {
		for x := 0; x < W; x++ {
			zone := g.get(g.zone, x, y)
			// under-ceiling air
			if g.solidAt(x, y) && !g.solidAt(x, y+1) && g.get(g.solid, x, y+1) == '.' {
				h := hash2(x, y)
				switch zone {
				case 'k':
					if h%2 == 0 {
						g.set(g.bg, x, y+1, 'c')
					} else if h%3 != 0 {
						g.set(g.bg, x, y+1, 't')
					}
				case 's':
					if h%4 == 0 {
						g.set(g.bg, x, y+1, 't')
					}
				default:
					if h%3 != 0 {
						g.set(g.bg, x, y+1, 't')
					}
				}
			}
			// above-floor air
			if !g.solidAt(x, y) && g.solidAt(x, y+1) && g.get(g.solid, x, y) == '.' {
				h := hash2(x, y+7777)
				switch zone {
				case 's':
					if hash2(x, 12345)%4 != 0 {
						g.set(g.fg, x, y, 'g')
					}
					if x < 124 && h%9 == 0 {
						g.set(g.bg, x, y, 'b')
					}
				case 'u':
					if h%6 == 0 {
						g.set(g.bg, x, y, 'm')
					} else if h%29 == 0 {
						g.set(g.bg, x, y, 'c')
					}
				case 'k':
					if h%2 == 0 {
						g.set(g.bg, x, y, 'c')
					} else if h%9 == 0 {
						g.set(g.bg, x, y, 'm')
					}
				case 'l':
					if h%8 == 0 {
						g.set(g.bg, x, y, 'm')
					}
				}
			}
		}
	}

	// --- enemies: the challenge is combat, not jumps ---------------------------
	placeOnFloor := func(kind byte, x0, x1, y0, y1, spacing int) {
		for x := x0; x <= x1; x += spacing {
			for y := y0; y <= y1; y++ {
				if g.get(g.solid, x, y) == '.' && g.solidAt(x, y+1) &&
					g.get(g.solid, x, y-1) == '.' {
					g.set(g.solid, x, y, kind)
					break
				}
			}
		}
	}
	placeOnFloor('a', 60, 120, 8, 24, 17)   // ruins shelf
	placeOnFloor('a', 168, 296, 8, 24, 19)  // eastern plains
	placeOnFloor('a', 20, 296, 29, 45, 23)  // cave tunnels
	placeOnFloor('a', 202, 260, 48, 58, 22) // crystal cavern
	placeOnFloor('a', 78, 134, 50, 58, 15)  // lava islands
	for _, f := range [][2]int{
		{70, 14}, {105, 13}, {180, 13}, {230, 12}, {286, 12}, // surface
		{70, 34}, {160, 31}, {230, 42}, // cave chambers
		{210, 51}, {226, 52}, {244, 50}, {256, 53}, // crystal
		{94, 53}, {118, 52}, // lava
	} {
		if g.get(g.solid, f[0], f[1]) == '.' {
			g.set(g.solid, f[0], f[1], 'f')
		}
	}

	write(g)
}

func write(g *gridSet) {
	out := "cappy-level v1\n"
	for _, sec := range []struct {
		name string
		l    []byte
	}{{"solid", g.solid}, {"bg", g.bg}, {"fg", g.fg}, {"zone", g.zone}} {
		out += "\n@" + sec.name + "\n"
		for y := 0; y < H; y++ {
			out += string(sec.l[y*W:(y+1)*W]) + "\n"
		}
	}

	lvl, err := game.ParseLevel([]byte(out))
	if err != nil {
		fmt.Fprintln(os.Stderr, "generated level invalid:", err)
		os.Exit(1)
	}
	if bad := checkConnectivity(lvl); len(bad) > 0 {
		fmt.Fprintln(os.Stderr, "unreachable from spawn:", bad)
		os.Exit(1)
	}

	if err := os.WriteFile("internal/game/level1.txt", []byte(out), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	parts, walkers, flyers, bosses := 0, 0, 0, 0
	for _, s := range lvl.Spawns {
		switch s.Kind {
		case 'P':
			parts++
		case 'a':
			walkers++
		case 'f':
			flyers++
		default:
			bosses++
		}
	}
	fmt.Printf("wrote level1.txt: %dx%d, %d parts + %d bosses, %d walkers, %d flyers, %d door tiles\n",
		lvl.W, lvl.H, parts, bosses, walkers, flyers, len(lvl.Doors))
}

// checkConnectivity flood-fills open air from the spawn and reports any
// part, boss or the ship that cannot possibly be reached. (Optimistic:
// it ignores gravity, so it catches sealed rooms, not hard jumps.)
func checkConnectivity(l *game.Level) []string {
	visited := make([]bool, l.W*l.H)
	queue := [][2]int{{int(l.SpawnX) / game.TilePx, int(l.SpawnY) / game.TilePx}}
	for len(queue) > 0 {
		q := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		x, y := q[0], q[1]
		if x < 0 || x >= l.W || y < 0 || y >= l.H || visited[y*l.W+x] || l.SolidTile(x, y) {
			continue
		}
		visited[y*l.W+x] = true
		queue = append(queue,
			[2]int{x - 1, y}, [2]int{x + 1, y},
			[2]int{x, y - 1}, [2]int{x, y + 1})
	}

	var bad []string
	for _, s := range l.Spawns {
		switch s.Kind {
		case 'P', 'D', 'Q', 'M':
			tx, ty := int(s.X)/game.TilePx, int(s.Y)/game.TilePx
			if !visited[ty*l.W+tx] {
				bad = append(bad, fmt.Sprintf("%c@%d,%d", s.Kind, tx, ty))
			}
		}
	}
	if !visited[(l.ShipY/game.TilePx)*l.W+l.ShipX/game.TilePx+3] {
		bad = append(bad, "ship")
	}
	return bad
}
