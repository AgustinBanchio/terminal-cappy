package game

import (
	"math/rand"

	"cappy/internal/gfx"
)

// TilePx is the size of one collision tile in canvas pixels.
const TilePx = 4

// chunkRows is the fixed tile height of the world.
const chunkRows = 24

// Spawn is an entity placement scanned out of the generated map.
type Spawn struct {
	Kind rune // 'a' walker, 'f' flyer, 'P' ship part
	X, Y float64
}

// Level is the generated world: a tile grid plus entity spawns.
type Level struct {
	W, H   int // in tiles
	tiles  []bool
	Spawns []Spawn

	SpawnX, SpawnY float64 // player start (pixels)
	ShipX, ShipY   int     // crashed ship sprite position (pixels)
}

// PxW and PxH are the world size in pixels.
func (l *Level) PxW() int { return l.W * TilePx }
func (l *Level) PxH() int { return l.H * TilePx }

// SolidTile reports whether a tile blocks movement. The world has
// solid walls beyond its left/right edges, open sky above and an open
// void below (falling out is fatal).
func (l *Level) SolidTile(tx, ty int) bool {
	if tx < 0 || tx >= l.W {
		return true
	}
	if ty < 0 || ty >= l.H {
		return false
	}
	return l.tiles[ty*l.W+tx]
}

func fdiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// SolidAtPx reports whether a pixel-space point is inside a solid tile.
func (l *Level) SolidAtPx(px, py float64) bool {
	return l.SolidTile(fdiv(int(px), TilePx), fdiv(int(py), TilePx))
}

// SolidBox reports whether an AABB (pixel space) overlaps any solid tile.
func (l *Level) SolidBox(x, y, w, h float64) bool {
	x0 := fdiv(int(x), TilePx)
	x1 := fdiv(int(x+w-0.001), TilePx)
	y0 := fdiv(int(y), TilePx)
	y1 := fdiv(int(y+h-0.001), TilePx)
	for ty := y0; ty <= y1; ty++ {
		for tx := x0; tx <= x1; tx++ {
			if l.SolidTile(tx, ty) {
				return true
			}
		}
	}
	return false
}

func hash2(x, y int) uint32 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	return h ^ (h >> 16)
}

// colorAt returns the terrain colour for a world pixel, or ok=false for
// open air. The rock gets a deterministic speckle from a position hash
// and a sunlit crust on exposed top edges.
func (l *Level) colorAt(wx, wy int) (uint8, bool) {
	tx, ty := fdiv(wx, TilePx), fdiv(wy, TilePx)
	if !l.SolidTile(tx, ty) {
		return 0, false
	}
	if !l.SolidTile(tx, ty-1) && wy-ty*TilePx == 0 {
		return 179, true // sunlit crust
	}
	if !l.SolidTile(tx, ty+1) && wy-ty*TilePx == TilePx-1 {
		return 52, true // dark underside of overhangs
	}
	switch h := hash2(wx, wy); {
	case h%11 == 0:
		return 131, true
	case h%5 == 0:
		return 95, true
	case h%3 == 0:
		return 58, true
	default:
		return 94, true
	}
}

// Draw renders the visible slice of terrain.
func (l *Level) Draw(c *gfx.Canvas, camX, camY int) {
	for sy := 0; sy < c.H; sy++ {
		wy := sy + camY
		for sx := 0; sx < c.W; sx++ {
			if col, ok := l.colorAt(sx+camX, wy); ok {
				c.Set(sx, sy, col)
			}
		}
	}
}

// --- world generation -------------------------------------------------

// chunk is a hand-authored template, chunkRows tall. The contract for
// middle chunks: columns 0-1 and w-2..w-1 have solid floor from row 20
// down and open air above, so any chunk can neighbour any other.
type chunk struct {
	w    int
	rows []string
}

func newChunk(w int) *chunk {
	c := &chunk{w: w, rows: make([]string, chunkRows)}
	blank := make([]byte, w)
	ground := make([]byte, w)
	for i := 0; i < w; i++ {
		blank[i] = '.'
		ground[i] = '#'
	}
	for r := 0; r < chunkRows; r++ {
		if r >= 20 {
			c.rows[r] = string(ground)
		} else {
			c.rows[r] = string(blank)
		}
	}
	return c
}

func (c *chunk) set(row int, s string) *chunk {
	if len(s) != c.w {
		panic("chunk row width mismatch: " + s)
	}
	c.rows[row] = s
	return c
}

func (c *chunk) setRange(r0, r1 int, s string) *chunk {
	for r := r0; r <= r1; r++ {
		c.set(r, s)
	}
	return c
}

func (c *chunk) mirrored() *chunk {
	out := &chunk{w: c.w, rows: make([]string, chunkRows)}
	for i, row := range c.rows {
		b := []byte(row)
		for l, r := 0, len(b)-1; l < r; l, r = l+1, r-1 {
			b[l], b[r] = b[r], b[l]
		}
		out.rows[i] = string(b)
	}
	return out
}

// startChunk: the crash site. Flat and safe; the ship position and the
// player spawn 'S' live here.
func startChunk() *chunk {
	return newChunk(20).
		set(18, "..............S.....")
}

// endChunk: a sheer cliff capping the far side of the world.
func endChunk() *chunk {
	return newChunk(12).
		setRange(0, 19, ".........###").
		set(12, "....f....###")
}

// middleChunks returns the pool of 16-wide templates. Four of them
// carry a ship part 'P'; all are traversable in both directions with
// run + jump + wall jump.
func middleChunks() []*chunk {
	meadow := newChunk(16).
		set(18, "....##..........").
		set(19, "...####....a....")

	gap := newChunk(16).
		set(14, "........f.......").
		set(18, ".....##...##....").
		setRange(20, 23, "####........####")

	chimney := newChunk(16).
		set(6, ".........##.....").
		set(7, ".....P...##.....").
		setRange(8, 16, ".....##..##.....").
		setRange(17, 19, ".........##.....")

	plateau := newChunk(16).
		set(15, ".......a..a.....").
		setRange(16, 17, ".....#######....").
		setRange(18, 19, "...###########..")

	cavern := newChunk(16).
		setRange(0, 15, "....#########...").
		set(18, "........P.......").
		set(19, "......a.........")

	pits := newChunk(16).
		set(14, ".....f....f.....").
		setRange(18, 19, ".......##.......").
		setRange(20, 23, "####...##...####")

	towers := newChunk(16).
		setRange(4, 4, "............##..").
		set(5, "........P...##..").
		setRange(6, 16, "........##..##..")

	ruins := newChunk(16).
		set(14, "..........P.....").
		set(15, "..........##....").
		setRange(16, 17, "......##..##....").
		set(18, "..##..##..##....").
		set(19, "..##..##..##..a.")

	return []*chunk{meadow, gap, chimney, plateau, cavern, pits, towers, ruins}
}

// Generate builds a world from a seed: crash site, all middle chunk
// templates in a shuffled order (each possibly mirrored), and a cliff
// at the end. Same seed, same planet.
func Generate(seed int64) *Level {
	rng := rand.New(rand.NewSource(seed))

	middles := middleChunks()
	rng.Shuffle(len(middles), func(i, j int) {
		middles[i], middles[j] = middles[j], middles[i]
	})
	chunks := []*chunk{startChunk()}
	for _, m := range middles {
		if rng.Intn(2) == 0 {
			m = m.mirrored()
		}
		chunks = append(chunks, m)
	}
	chunks = append(chunks, endChunk())

	total := 0
	for _, c := range chunks {
		total += c.w
	}

	l := &Level{W: total, H: chunkRows, tiles: make([]bool, total*chunkRows)}

	xOff := 0
	for _, c := range chunks {
		for ty := 0; ty < chunkRows; ty++ {
			for tx := 0; tx < c.w; tx++ {
				wx := xOff + tx
				px := float64(wx*TilePx + TilePx/2)
				py := float64(ty*TilePx + TilePx/2)
				switch c.rows[ty][tx] {
				case '#':
					l.tiles[ty*l.W+wx] = true
				case 'S':
					l.SpawnX, l.SpawnY = px, py
				case 'a', 'f', 'P':
					l.Spawns = append(l.Spawns, Spawn{Kind: rune(c.rows[ty][tx]), X: px, Y: py})
				}
			}
		}
		xOff += c.w
	}

	// The ship sits near the left edge of the crash site, nose buried
	// a couple of pixels into the ground (ground top is row 20).
	l.ShipX = 4
	l.ShipY = 20*TilePx - sprShip.H + 2
	return l
}
