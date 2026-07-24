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
	case x >= 294 && x <= 297:
		return 19 // step up to the lookout
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
	// Broken structures along the sunken shelf: low open-topped tower
	// pairs (walls 3-4 tall are always escapable with a single jump,
	// and no lintel means nothing to get trapped under). Clear of the
	// shaft at x96 so nothing overhangs the way down.
	for _, gx := range []int{54, 70, 86, 110} {
		// Base each tower on the deepest nearby ground so it never
		// rises more than 3 tiles above either approach side.
		base := 0
		for x := gx - 1; x <= gx+10; x++ {
			if gr := groundRow(x); gr > base {
				base = gr
			}
		}
		g.rect(g.solid, gx, base-3, gx+1, base-1, '%')
		g.rect(g.solid, gx+8, base-2, gx+9, base-1, '%')
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
	shaft(262, 30) // reaches the upper tunnel; go deeper via connectors

	// --- cave tunnels --------------------------------------------------------
	// A tile taller than they used to be, so the underground breathes.
	for x := 14; x <= 300; x++ { // upper tunnel
		floor := 36 + int(math.Round(math.Sin(float64(x)*0.15)))
		g.rect(g.solid, x, 30, x, floor, '.')
	}
	for x := 14; x <= 292; x++ { // lower tunnel
		floor := 45 + int(math.Round(math.Sin(float64(x)*0.11+2)))
		g.rect(g.solid, x, 40, x, floor, '.')
	}
	// connectors between tunnels
	g.rect(g.solid, 56, 35, 59, 40, '.')
	g.rect(g.solid, 150, 35, 153, 40, '.')
	g.rect(g.solid, 236, 35, 239, 40, '.')
	// carved chambers, roomier than before. The west chamber keeps a
	// floor at row 39 (the lower tunnel runs right below) and keeps its
	// distance from the x56 connector so the chimney walls survive.
	g.rect(g.solid, 62, 31, 84, 38, '.')
	g.rect(g.solid, 150, 28, 174, 36, '.')
	g.rect(g.solid, 220, 40, 242, 46, '.')
	// Climbing stairs at each shaft and connector: low tiers on BOTH
	// sides of the mouth plus a tall tier hugging the mouth's left
	// column, so the stairs can be climbed AND crossed from either
	// direction (the tall tier would otherwise wall the tunnel off).
	// The fall channel (the other three tiles) stays clear. The middle
	// shaft (x186) deliberately gets none: one-way drop.
	for _, s := range []int{96, 262} {
		g.rect(g.solid, s-3, 35, s-2, 38, '#') // west tier
		g.rect(g.solid, s+5, 35, s+6, 38, '#') // east tier
		g.rect(g.solid, s, 32, s, 38, '#')     // tall tier, in-mouth
	}
	for _, c := range []int{56, 150, 236} {
		g.rect(g.solid, c-3, 44, c-2, 47, '#')
		g.rect(g.solid, c+5, 44, c+6, 47, '#')
		g.rect(g.solid, c, 41, c, 47, '#')
	}

	// --- crystal region (south-east) ----------------------------------------
	// The grand cavern is terraced: broad 2-tile steps rise eastward
	// from the deep floor to the exit shaft and Prisma's door, so any
	// drop can be walked back up.
	g.rect(g.zone, 194, 42, W-1, H-1, 'k')
	terrace := func(x int) int {
		switch {
		case x < 215:
			return 58
		case x < 231:
			return 56
		case x < 247:
			return 54
		default:
			return 52
		}
	}
	for x := 200; x <= 269; x++ {
		g.rect(g.solid, x, 46, x, terrace(x)-1, '.') // high vaulted ceiling
	}
	// remnant columns standing on the terraces (short: jump over them)
	g.rect(g.solid, 218, 53, 219, 55, '#')
	g.rect(g.solid, 236, 51, 237, 53, '#')
	g.rect(g.solid, 246, 42, 249, 48, '.') // way in from the lower tunnel
	// Climb back out: a step up from the western terrace, then a tall
	// tier hugging the mouth's left column (fall channel x247-249).
	g.rect(g.solid, 244, 51, 245, 53, '#')
	g.rect(g.solid, 246, 49, 246, 53, '#')
	// Prisma's chamber, entered at terrace level
	g.rect(g.solid, 270, 40, 308, 41, '#') // ceiling
	g.rect(g.solid, 270, 42, 271, 51, '#') // west wall
	g.rect(g.solid, 307, 42, 308, 51, '#') // east wall
	g.rect(g.solid, 272, 42, 306, 51, '.') // interior, floor at 52
	g.rect(g.solid, 270, 48, 271, 51, 'd')
	g.set(g.solid, 289, 49, 'Q')
	// crystal rock conversion
	for y := 38; y < H; y++ {
		for x := 194; x < W; x++ {
			if g.get(g.solid, x, y) == '#' {
				g.set(g.solid, x, y, 'X')
			}
		}
	}

	// --- lava region (south-west) --------------------------------------------
	g.rect(g.zone, 0, 44, 142, H-1, 'l')
	g.rect(g.solid, 12, 48, 136, 57, '.') // great burning cavern, high roof
	g.rect(g.solid, 16, 42, 19, 50, '.')  // west way down: one-way plunge
	g.rect(g.solid, 84, 42, 87, 50, '.')  // east way down
	// Ledge stairs up to the east way out only, offset BESIDE the
	// chimney so the drop stays clear; from the top step you hop left
	// into the chimney. The west way is a deliberate one-way drop into
	// the pocket; leaving it means finding the crawl corridor below.
	g.rect(g.solid, 91, 55, 92, 55, '#')
	g.rect(g.solid, 88, 52, 89, 52, '#')
	// Magmaw's chamber inside the cavern, taller than before
	g.rect(g.solid, 28, 46, 74, 47, '#') // ceiling
	g.rect(g.solid, 28, 48, 29, 57, '#') // west wall
	g.rect(g.solid, 73, 48, 74, 57, '#') // east wall
	g.rect(g.solid, 30, 48, 72, 57, '.') // interior
	g.rect(g.solid, 73, 54, 74, 57, 'd')
	g.set(g.solid, 50, 55, 'M')
	g.rect(g.solid, 46, 57, 51, 57, '~') // a molten puddle to hop
	// The crawl corridor: a dark passage dug beneath Magmaw's chamber,
	// the only way out of the west pocket. Drop through the hole in the
	// pocket floor, cross under the boss, climb the chimney hole into
	// the east cavern. The chamber floor above stays intact.
	g.rect(g.solid, 22, 59, 79, 61, '.')
	g.rect(g.solid, 22, 58, 25, 58, '.') // pocket floor opening (drop in)
	g.rect(g.solid, 76, 58, 79, 58, '.') // east exit hole (climbable)
	g.set(g.solid, 40, 61, 'a')
	g.set(g.solid, 60, 61, 'a')
	// pools outside, with easy jumps between islands
	for _, p := range [][2]int{{88, 93}, {104, 109}, {122, 127}} {
		g.rect(g.solid, p[0], 58, p[1], 59, '~')
	}

	// --- ship parts (non-boss) -----------------------------------------------
	// Four parts are free pickups in far corners (the other three are
	// boss drops); each free part gets explicit guards below.
	g.set(g.solid, 305, 15, 'P') // eastern lookout
	g.set(g.solid, 161, 33, 'P') // upper cave chamber
	g.set(g.solid, 206, 56, 'P') // crystal cavern, deep end
	g.set(g.solid, 115, 55, 'P') // lava island between pools
	g.set(g.bg, 22, 54, 'c')     // the west lava pocket keeps a secret glow

	// --- decorations, by zone -------------------------------------------------
	for y := 0; y < H-1; y++ {
		for x := 0; x < W; x++ {
			zone := g.get(g.zone, x, y)
			// under-ceiling air: stalactites, vines, crystal roofs
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
					} else if h%9 == 1 {
						g.set(g.bg, x, y+1, 'v')
					}
				case 'u':
					switch h % 5 {
					case 0, 1, 4:
						g.set(g.bg, x, y+1, 't')
					case 2:
						g.set(g.bg, x, y+1, 'v')
					}
				default: // lava
					if h%3 != 0 {
						g.set(g.bg, x, y+1, 't')
					}
				}
			}
			// above-floor air: each region gets its own ground cover
			if !g.solidAt(x, y) && g.solidAt(x, y+1) && g.get(g.solid, x, y) == '.' {
				h := hash2(x, y+7777)
				switch zone {
				case 's':
					if hash2(x, 12345)%4 != 0 {
						g.set(g.fg, x, y, 'g')
					}
					switch {
					case x >= 118 && x <= 175 && h%6 == 0:
						g.set(g.bg, x, y, 'w') // debris field around the crash
					case x < 124 && h%9 == 0:
						g.set(g.bg, x, y, 'b')
					case h%11 == 0:
						g.set(g.bg, x, y, 'n')
					}
				case 'u':
					switch {
					case h%7 == 0:
						g.set(g.bg, x, y, 'q')
					case h%6 == 0:
						g.set(g.bg, x, y, 'm')
					case h%29 == 0:
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
	// Walkers only spawn on floors at least 3 tiles wide, so none end
	// up perched on thin pillars or ruin walls.
	placeOnFloor := func(kind byte, x0, x1, y0, y1, spacing int) {
		for x := x0; x <= x1; x += spacing {
			for y := y0; y <= y1; y++ {
				if g.get(g.solid, x, y) == '.' && g.get(g.solid, x, y-1) == '.' &&
					g.solidAt(x, y+1) && g.solidAt(x-1, y+1) && g.solidAt(x+1, y+1) {
					g.set(g.solid, x, y, kind)
					break
				}
			}
		}
	}
	// placeOnCeiling hangs lurkers under cave ceilings with room to
	// drop below them.
	placeOnCeiling := func(kind byte, x0, x1, y0, y1, spacing int) {
		for x := x0; x <= x1; x += spacing {
			for y := y0; y <= y1; y++ {
				if g.get(g.solid, x, y) == '.' && g.solidAt(x, y-1) &&
					g.get(g.solid, x, y+1) == '.' && g.get(g.solid, x, y+2) == '.' {
					g.set(g.solid, x, y, kind)
					break
				}
			}
		}
	}

	placeOnFloor('a', 60, 120, 8, 24, 17)    // ruins shelf: walkers
	placeOnFloor('a', 168, 296, 8, 24, 19)   // eastern plains: walkers
	placeOnFloor('a', 20, 296, 29, 45, 37)   // caves: a few walkers...
	placeOnCeiling('u', 34, 290, 29, 44, 31) // ...plus ceiling lurkers
	placeOnFloor('z', 204, 258, 48, 58, 21)  // crystal terraces: shardlings
	placeOnFloor('e', 78, 134, 50, 58, 13)   // lava islands: maglings

	for _, e := range []struct {
		x, y int
		k    byte
	}{
		{70, 14, 'f'}, {105, 13, 'f'}, {180, 13, 'f'}, {230, 12, 'f'}, {286, 12, 'f'},
		{70, 34, 'b'}, {230, 42, 'b'}, {130, 32, 'b'}, {202, 32, 'b'}, // cave bats
		{212, 54, 'f'}, {244, 50, 'f'}, // crystal drifters
		{22, 53, 'f'}, // the west pocket keeps one
		// guards for the free ship parts
		{301, 13, 'f'}, {309, 13, 'f'}, // eastern lookout
		{157, 30, 'b'}, {165, 30, 'b'}, // upper cave chamber
		{203, 53, 'z'}, {209, 53, 'z'}, // crystal deep end
		{112, 52, 'e'}, {118, 53, 'f'}, // lava island
	} {
		if g.get(g.solid, e.x, e.y) == '.' {
			g.set(g.solid, e.x, e.y, e.k)
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
	if issues := lvl.AnalyzeTraversal(); len(issues) > 0 {
		fmt.Fprintln(os.Stderr, "traversal problems:")
		for _, s := range issues {
			fmt.Fprintln(os.Stderr, " -", s)
		}
		if dump := os.Getenv("GENMAP_DEBUG_DUMP"); dump != "" {
			_ = os.WriteFile(dump, []byte(out), 0o644)
			fmt.Fprintln(os.Stderr, "debug dump written to", dump)
		}
		os.Exit(1)
	}

	if err := os.WriteFile("internal/game/level1.txt", []byte(out), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	counts := map[rune]int{}
	for _, s := range lvl.Spawns {
		counts[s.Kind]++
	}
	fmt.Printf("wrote level1.txt: %dx%d, %d parts + %d bosses, enemies: %d walkers %d flyers %d bats %d lurkers %d shardlings %d maglings, %d door tiles\n",
		lvl.W, lvl.H, counts['P'], counts['D']+counts['Q']+counts['M'],
		counts['a'], counts['f'], counts['b'], counts['u'], counts['z'], counts['e'],
		len(lvl.Doors))
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
