package game

import (
	"math"

	"github.com/AgustinBanchio/terminal-cappy/internal/gfx"
)

// TilePx is the size of one collision tile in canvas pixels.
const TilePx = 4

// chunkRows is the fixed tile height of the world.
const chunkRows = 24

// Spawn is an entity placement scanned out of the map.
type Spawn struct {
	Kind rune // 'a' walker, 'f' flyer, 'P' ship part
	X, Y float64
}

// Level is the world: a tile grid plus entity spawns.
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

// --- decoration layers --------------------------------------------------

// Spike length patterns per pixel column within a tile, so stalactites
// and stalagmites come out pointy instead of square.
var spikeShape = [TilePx]int{-2, 0, -1, -3}

// DrawBackdrop renders non-colliding world-space scenery behind the
// gameplay layer: stalactites under ceilings, stalagmites and tall rock
// pillars in caverns. Everything derives deterministically from the
// tile geometry, so the art always matches the level.
func (l *Level) DrawBackdrop(c *gfx.Canvas, camX, camY int) {
	tx0 := fdiv(camX, TilePx) - 1
	tx1 := fdiv(camX+c.W, TilePx) + 1
	for tx := tx0; tx <= tx1; tx++ {
		for ty := 0; ty < l.H; ty++ {
			solid := l.SolidTile(tx, ty)

			// Ceiling edge: maybe hang a stalactite cluster.
			if solid && !l.SolidTile(tx, ty+1) {
				h := hash2(tx, ty)
				if h%3 != 0 {
					l.drawSpikes(c, camX, camY, tx, (ty+1)*TilePx, 3+int(h%5), +1, 52)
				}
			}
			// Floor edge: maybe a stalagmite, rarely a full pillar
			// reaching a ceiling above (cavern support columns).
			if !solid && l.SolidTile(tx, ty+1) {
				h := hash2(tx, ty+7777)
				if h%7 == 0 {
					l.drawSpikes(c, camX, camY, tx, (ty+1)*TilePx-1, 2+int(h%4), -1, 52)
				}
				if h%29 == 0 {
					if cy, ok := l.ceilingAbove(tx, ty); ok {
						x := tx*TilePx + 1 - camX
						y0 := (cy+1)*TilePx - camY
						c.FillRect(x, y0, 2, (ty+1)*TilePx-(cy+1)*TilePx, 52)
						c.FillRect(x-1, y0, 1, 2, 52) // flared top
						c.FillRect(x+2, y0, 1, 2, 52)
					}
				}
			}
		}
	}
}

// ceilingAbove finds a solid ceiling within pillar range above a floor.
func (l *Level) ceilingAbove(tx, ty int) (int, bool) {
	for cy := ty - 1; cy >= ty-12 && cy >= 0; cy-- {
		if l.SolidTile(tx, cy) {
			return cy, true
		}
	}
	return 0, false
}

// drawSpikes draws one tile-column of pointy rock. dir +1 hangs down
// from y0, dir -1 grows up from y0.
func (l *Level) drawSpikes(c *gfx.Canvas, camX, camY, tx, y0, size, dir int, color uint8) {
	for i := 0; i < TilePx; i++ {
		length := size + spikeShape[(tx+i)%TilePx]
		wx := tx*TilePx + i
		for j := 0; j < length; j++ {
			c.Set(wx-camX, y0+j*dir-camY, color)
		}
	}
}

// DrawForeground renders scenery in front of the player: sparse tufts
// of alien grass swaying on every sunlit crust. The tufts are single
// pixels wide with gaps, so Cappy stays visible walking through them.
var grassColors = []uint8{29, 35, 41}

func (l *Level) DrawForeground(c *gfx.Canvas, camX, camY int, t float64) {
	for sx := 0; sx < c.W; sx++ {
		wx := sx + camX
		tx := fdiv(wx, TilePx)
		h := hash2(wx, 0)
		if h%5 < 2 {
			continue // gap: this blade is missing, keeping it see-through
		}
		for ty := 0; ty < l.H; ty++ {
			if !l.SolidTile(tx, ty) || l.SolidTile(tx, ty-1) {
				continue
			}
			gh := hash2(wx, ty)
			if gh%3 == 0 {
				continue
			}
			blade := 2 + int(gh%3)
			sway := int(math.Round(math.Sin(t*1.8 + float64(wx)*0.7)))
			col := grassColors[gh%uint32(len(grassColors))]
			top := ty*TilePx - blade
			for y := top; y < ty*TilePx; y++ {
				x := sx
				if y == top {
					x += sway // only the tip sways
				}
				c.Set(x, y-camY, col)
			}
		}
	}
}

// --- world building -----------------------------------------------------

// chunk is a hand-authored map segment, chunkRows tall. The contract:
// columns 0-1 and w-2..w-1 have solid floor from row 20 down and open
// air above, so segments join seamlessly.
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

// worldChunks returns the curated world, in order: a difficulty ramp
// from the crash site through caverns and wall-jump climbs to the
// hardest part high on the twin towers, then back on foot.
func worldChunks() []*chunk {
	start := newChunk(20).
		set(18, "..............S.....")

	meadow := newChunk(16).
		set(18, "....##..........").
		set(19, "...####....a....")

	valley := newChunk(16).
		set(16, "........f.......").
		set(20, "####........####").
		set(21, "#####......#####")

	gap := newChunk(16).
		set(14, "........f.......").
		set(18, ".....##...##....").
		setRange(20, 23, "####........####")

	plateau := newChunk(16).
		set(15, ".......a..a.....").
		setRange(16, 17, ".....#######....").
		setRange(18, 19, "...###########..")

	cavern := newChunk(16).
		setRange(0, 15, "....#########...").
		set(18, "........P.......").
		set(19, "......a.........")

	arch := newChunk(16).
		setRange(10, 11, "...##########...").
		setRange(12, 16, "...##......##...").
		set(18, ".......a........")

	chimney := newChunk(16).
		set(6, ".........##.....").
		set(7, ".....P...##.....").
		setRange(8, 16, ".....##..##.....").
		setRange(17, 19, ".........##.....")

	pits := newChunk(16).
		set(14, ".....f....f.....").
		setRange(18, 19, ".......##.......").
		setRange(20, 23, "####...##...####")

	ruins := newChunk(16).
		set(14, "..........P.....").
		set(15, "..........##....").
		setRange(16, 17, "......##..##....").
		set(18, "..##..##..##....").
		set(19, "..##..##..##..a.")

	towers := newChunk(16).
		set(4, "............##..").
		set(5, "........P...##..").
		setRange(6, 16, "........##..##..")

	end := newChunk(12).
		setRange(0, 19, ".........###").
		set(12, "....f....###")

	return []*chunk{start, meadow, valley, gap, plateau, cavern,
		arch, chimney, pits, ruins, towers, end}
}

// Build assembles the curated world. It is the same planet every run:
// a metroidvania map, not a roguelike one.
func Build() *Level {
	chunks := worldChunks()
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
