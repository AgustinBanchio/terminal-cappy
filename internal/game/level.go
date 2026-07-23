package game

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"strings"

	"github.com/AgustinBanchio/terminal-cappy/internal/gfx"
)

// TilePx is the size of one collision tile in canvas pixels.
const TilePx = 4

// The world is four tile layers stored in a plain-text level file
// (see level1.txt), edited with cmd/leveled:
//
//	solid: collision + entities
//	       '.' air  '#' rock  '%' ruin brick  'X' crystal rock
//	       '~' lava (hazard, not solid)  'd' boss door (solid in fights)
//	       'S' spawn  'H' ship  'a' walker  'f' flyer  'P' ship part
//	       'D' boss Dimi  'Q' boss Prisma  'M' boss Magmaw
//	bg:    behind gameplay  't' stalactite  'm' stalagmite  'I' pillar
//	       'c' crystal  'r' ruin column  'b' rubble
//	fg:    in front of gameplay  'g' grass
//	zone:  ambience region, drives the parallax backdrop and weather
//	       's' surface (storm)  'u' cave  'k' crystal  'l' lava
const (
	LayerSolid = iota
	LayerBG
	LayerFG
	LayerZone
	LayerCount
)

var layerNames = [LayerCount]string{"solid", "bg", "fg", "zone"}

// TileOption is one entry of an editor palette.
type TileOption struct {
	Ch   byte
	Name string
}

var palettes = [LayerCount][]TileOption{
	{
		{'#', "rock"},
		{'%', "ruin brick"},
		{'X', "crystal rock"},
		{'~', "lava"},
		{'d', "boss door"},
		{'.', "air"},
		{'S', "spawn"},
		{'H', "ship"},
		{'a', "walker"},
		{'f', "flyer"},
		{'b', "bat"},
		{'u', "lurker"},
		{'z', "shardling"},
		{'e', "magling"},
		{'P', "part"},
		{'D', "boss Dimi"},
		{'Q', "boss Prisma"},
		{'M', "boss Magmaw"},
	},
	{
		{'t', "stalactite"},
		{'m', "stalagmite"},
		{'I', "pillar"},
		{'c', "crystal"},
		{'r', "ruin column"},
		{'b', "rubble"},
		{'.', "none"},
	},
	{
		{'g', "grass"},
		{'.', "none"},
	},
	{
		{'s', "surface"},
		{'u', "cave"},
		{'k', "crystal"},
		{'l', "lava"},
		{'.', "surface (default)"},
	},
}

// Palette returns the valid tiles for a layer.
func Palette(layer int) []TileOption { return palettes[layer] }

func validTile(layer int, ch byte) bool {
	for _, o := range palettes[layer] {
		if o.Ch == ch {
			return true
		}
	}
	return false
}

func isSolidCh(ch byte) bool { return ch == '#' || ch == '%' || ch == 'X' }

// Spawn is an entity placement scanned out of the solid layer.
type Spawn struct {
	Kind rune // 'a' walker, 'f' flyer, 'P' part, 'D'/'Q'/'M' bosses
	X, Y float64
}

// Level is the world: four tile layers plus state derived from them.
type Level struct {
	W, H int
	grid [LayerCount][]byte

	// Derived from the solid layer by refresh().
	tiles  []bool
	Spawns []Spawn
	Doors  [][2]int // all boss door tiles

	SpawnX, SpawnY float64 // player start (pixels)
	ShipX, ShipY   int     // crashed ship sprite position (pixels)

	locked map[int]bool // door tiles currently solid (boss fights)
}

//go:embed level1.txt
var defaultLevelData []byte

// LoadDefault parses the level embedded in the binary.
func LoadDefault() *Level {
	l, err := ParseLevel(defaultLevelData)
	if err != nil {
		panic("embedded level is invalid: " + err.Error())
	}
	return l
}

const levelHeader = "cappy-level v1"

// ParseLevel reads the level text format: a header line, then one
// section per layer ("@solid", "@bg", "@fg", "@zone"), each holding
// H rows of W tiles.
func ParseLevel(data []byte) (*Level, error) {
	sections := map[string][]string{}
	cur := ""
	sawHeader := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if !sawHeader {
			if line != levelHeader {
				return nil, fmt.Errorf("bad header %q, want %q", line, levelHeader)
			}
			sawHeader = true
			continue
		}
		if strings.HasPrefix(line, "@") {
			cur = line[1:]
			continue
		}
		if cur == "" {
			return nil, fmt.Errorf("row outside any @section: %q", line)
		}
		sections[cur] = append(sections[cur], line)
	}

	l := &Level{locked: map[int]bool{}}
	for layer, name := range layerNames {
		rows := sections[name]
		if len(rows) == 0 {
			return nil, fmt.Errorf("missing @%s section", name)
		}
		if layer == LayerSolid {
			l.W, l.H = len(rows[0]), len(rows)
		} else if len(rows) != l.H {
			return nil, fmt.Errorf("@%s has %d rows, want %d", name, len(rows), l.H)
		}
		g := make([]byte, 0, l.W*l.H)
		for y, row := range rows {
			if len(row) != l.W {
				return nil, fmt.Errorf("@%s row %d is %d wide, want %d", name, y, len(row), l.W)
			}
			for x := 0; x < l.W; x++ {
				if !validTile(layer, row[x]) {
					return nil, fmt.Errorf("@%s row %d col %d: invalid tile %q", name, y, x, row[x])
				}
			}
			g = append(g, row...)
		}
		l.grid[layer] = g
	}

	l.refresh()
	if l.SpawnX == 0 && l.SpawnY == 0 {
		return nil, fmt.Errorf("level has no spawn point 'S'")
	}
	if l.ShipX == 0 && l.ShipY == 0 {
		return nil, fmt.Errorf("level has no ship anchor 'H'")
	}
	return l, nil
}

// Marshal serialises the level back to the text format.
func (l *Level) Marshal() []byte {
	var b bytes.Buffer
	b.WriteString(levelHeader + "\n")
	for layer, name := range layerNames {
		fmt.Fprintf(&b, "\n@%s\n", name)
		for y := 0; y < l.H; y++ {
			b.Write(l.grid[layer][y*l.W : (y+1)*l.W])
			b.WriteByte('\n')
		}
	}
	return b.Bytes()
}

// Cell reads one layer tile ('.' when out of bounds).
func (l *Level) Cell(layer, tx, ty int) byte {
	if tx < 0 || tx >= l.W || ty < 0 || ty >= l.H {
		return '.'
	}
	return l.grid[layer][ty*l.W+tx]
}

// SetCell writes one layer tile, refusing invalid tiles for the layer.
// Placing a second spawn or ship anchor moves it: the old one clears.
func (l *Level) SetCell(layer, tx, ty int, ch byte) bool {
	if tx < 0 || tx >= l.W || ty < 0 || ty >= l.H || !validTile(layer, ch) {
		return false
	}
	if layer == LayerSolid && (ch == 'S' || ch == 'H') {
		for i, c := range l.grid[LayerSolid] {
			if c == ch {
				l.grid[LayerSolid][i] = '.'
			}
		}
	}
	l.grid[layer][ty*l.W+tx] = ch
	if layer == LayerSolid {
		l.refresh()
	}
	return true
}

// Zone returns the ambience zone at a tile ('.' maps to surface).
func (l *Level) Zone(tx, ty int) byte {
	tx = clampInt(tx, 0, l.W-1)
	ty = clampInt(ty, 0, l.H-1)
	if z := l.grid[LayerZone][ty*l.W+tx]; z != '.' {
		return z
	}
	return 's'
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// LockDoors makes the given door tiles solid (or open again).
func (l *Level) LockDoors(doors [][2]int, locked bool) {
	for _, d := range doors {
		idx := d[1]*l.W + d[0]
		if locked {
			l.locked[idx] = true
		} else {
			delete(l.locked, idx)
		}
	}
}

// refresh recomputes collision and entity placements from the solid
// layer. The ship anchor 'H' marks the sprite's top-left tile, nudged
// 2px down so the hull sits buried in the ground.
func (l *Level) refresh() {
	l.tiles = make([]bool, l.W*l.H)
	l.Spawns = l.Spawns[:0]
	l.Doors = l.Doors[:0]
	for ty := 0; ty < l.H; ty++ {
		for tx := 0; tx < l.W; tx++ {
			px := float64(tx*TilePx + TilePx/2)
			py := float64(ty*TilePx + TilePx/2)
			ch := l.grid[LayerSolid][ty*l.W+tx]
			switch {
			case isSolidCh(ch):
				l.tiles[ty*l.W+tx] = true
			case ch == 'S':
				l.SpawnX, l.SpawnY = px, py
			case ch == 'H':
				l.ShipX, l.ShipY = tx*TilePx, ty*TilePx+2
			case ch == 'd':
				l.Doors = append(l.Doors, [2]int{tx, ty})
			case ch == 'a' || ch == 'f' || ch == 'b' || ch == 'u' ||
				ch == 'z' || ch == 'e' || ch == 'P' ||
				ch == 'D' || ch == 'Q' || ch == 'M':
				l.Spawns = append(l.Spawns, Spawn{Kind: rune(ch), X: px, Y: py})
			}
		}
	}
}

// PxW and PxH are the world size in pixels.
func (l *Level) PxW() int { return l.W * TilePx }
func (l *Level) PxH() int { return l.H * TilePx }

// SolidTile reports whether a tile blocks movement. The world has
// solid walls beyond its left/right edges, open sky above and an open
// void below. Locked boss doors count as solid.
func (l *Level) SolidTile(tx, ty int) bool {
	if tx < 0 || tx >= l.W {
		return true
	}
	if ty < 0 || ty >= l.H {
		return false
	}
	return l.tiles[ty*l.W+tx] || l.locked[ty*l.W+tx]
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

// LavaAtPx reports whether a pixel-space point is inside lava.
func (l *Level) LavaAtPx(px, py float64) bool {
	return l.Cell(LayerSolid, fdiv(int(px), TilePx), fdiv(int(py), TilePx)) == '~'
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
// open air. Each solid material has its own texture; all speckle comes
// from a position hash so it is stable frame to frame.
func (l *Level) colorAt(wx, wy int) (uint8, bool) {
	tx, ty := fdiv(wx, TilePx), fdiv(wy, TilePx)
	if !l.SolidTile(tx, ty) {
		return 0, false
	}
	ch := l.Cell(LayerSolid, tx, ty)
	h := hash2(wx, wy)
	switch ch {
	case '%': // ruined brickwork with mortar lines
		if wy%TilePx == 0 {
			return 58, true
		}
		if (wx+(wy/TilePx)*2)%6 == 0 {
			return 58, true
		}
		if h%7 == 0 {
			return 95, true
		}
		return 101, true
	case 'X': // crystal rock, glinting
		if !l.SolidTile(tx, ty-1) && wy-ty*TilePx == 0 {
			return 123, true
		}
		if h%17 == 0 {
			return 87, true
		}
		switch h % 3 {
		case 0:
			return 61, true
		case 1:
			return 62, true
		default:
			return 68, true
		}
	default: // '#' planet rock (also locked doors' backing, unseen)
		if !l.SolidTile(tx, ty-1) && wy-ty*TilePx == 0 {
			return 179, true // sunlit crust
		}
		if !l.SolidTile(tx, ty+1) && wy-ty*TilePx == TilePx-1 {
			return 52, true // dark underside of overhangs
		}
		switch {
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
}

// Draw renders the visible slice of terrain, lava and locked doors.
func (l *Level) Draw(c *gfx.Canvas, camX, camY int, t float64) {
	for sy := 0; sy < c.H; sy++ {
		wy := sy + camY
		for sx := 0; sx < c.W; sx++ {
			wx := sx + camX
			if col, ok := l.colorAt(wx, wy); ok {
				c.Set(sx, sy, col)
				continue
			}
			tx, ty := fdiv(wx, TilePx), fdiv(wy, TilePx)
			switch l.Cell(LayerSolid, tx, ty) {
			case '~':
				c.Set(sx, sy, lavaColor(l, wx, wy, tx, ty, t))
			case 'd':
				if l.locked[ty*l.W+tx] {
					c.Set(sx, sy, doorColor(wy, t))
				}
			}
		}
	}
}

// lavaColor animates the molten surface: bright rolling waves on top,
// churning body below.
func lavaColor(l *Level, wx, wy, tx, ty int, t float64) uint8 {
	surface := l.Cell(LayerSolid, tx, ty-1) != '~' && !l.SolidTile(tx, ty-1)
	if surface && wy-ty*TilePx == 0 {
		if (wx+int(t*7))%9 < 3 {
			return 220
		}
		return 208
	}
	switch hash2(wx, wy*31+int(t*4)) % 5 {
	case 0:
		return 202
	case 1:
		return 166
	default:
		return 196
	}
}

// doorColor is the shimmering energy barrier of a locked boss door.
func doorColor(wy int, t float64) uint8 {
	switch (wy + int(t*18)) % 4 {
	case 0:
		return 123
	case 1:
		return 87
	default:
		return 51
	}
}

// --- decoration layers ---------------------------------------------------

// Spike length offsets per pixel column within a tile, so stalactites
// and stalagmites come out pointy instead of square.
var spikeShape = [TilePx]int{-2, 0, -1, -3}

func (l *Level) visibleTiles(c *gfx.Canvas, camX int) (int, int) {
	return fdiv(camX, TilePx) - 1, fdiv(camX+c.W, TilePx) + 1
}

// DrawBackdrop renders the bg layer behind the gameplay layer. Every
// decoration is placed by hand in the level file; pixel shapes still
// come from a position hash so tiles never look stamped.
func (l *Level) DrawBackdrop(c *gfx.Canvas, camX, camY int, t float64) {
	tx0, tx1 := l.visibleTiles(c, camX)
	for tx := tx0; tx <= tx1; tx++ {
		for ty := 0; ty < l.H; ty++ {
			switch l.Cell(LayerBG, tx, ty) {
			case 't':
				drawSpikes(c, camX, camY, tx, ty*TilePx, 3+int(hash2(tx, ty)%5), +1, 52)
			case 'm':
				drawSpikes(c, camX, camY, tx, (ty+1)*TilePx-1, 2+int(hash2(tx, ty)%4), -1, 52)
			case 'I':
				l.drawColumn(c, camX, camY, tx, ty, 'I', 52)
			case 'r':
				l.drawColumn(c, camX, camY, tx, ty, 'r', 59)
			case 'b':
				drawRubble(c, camX, camY, tx, ty)
			case 'c':
				l.drawCrystal(c, camX, camY, tx, ty, t)
			}
		}
	}
}

// drawSpikes draws one tile-column of pointy rock. dir +1 hangs down
// from y0, dir -1 grows up from y0.
func drawSpikes(c *gfx.Canvas, camX, camY, tx, y0, size, dir int, color uint8) {
	for i := 0; i < TilePx; i++ {
		length := size + spikeShape[(tx+i)%TilePx]
		wx := tx*TilePx + i
		for j := 0; j < length; j++ {
			c.Set(wx-camX, y0+j*dir-camY, color)
		}
	}
}

// drawColumn draws one tile-tall segment of a support pillar ('I') or
// weathered ruin column ('r'); paint a vertical run for a full column.
// Ends get flares; ruin columns get chipped edges.
func (l *Level) drawColumn(c *gfx.Canvas, camX, camY, tx, ty int, kind byte, color uint8) {
	x := tx*TilePx + 1 - camX
	y := ty*TilePx - camY
	c.FillRect(x, y, 2, TilePx, color)
	if kind == 'r' && hash2(tx, ty)%3 == 0 {
		c.Set(x+1, y+int(hash2(tx, ty*3)%4), 0) // chipped notch
	}
	if l.Cell(LayerBG, tx, ty-1) != kind {
		c.FillRect(x-1, y, 1, 2, color)
		c.FillRect(x+2, y, 1, 2, color)
	}
	if l.Cell(LayerBG, tx, ty+1) != kind {
		c.FillRect(x-1, y+TilePx-2, 1, 2, color)
		c.FillRect(x+2, y+TilePx-2, 1, 2, color)
	}
}

// drawRubble draws a small pile of broken masonry.
func drawRubble(c *gfx.Canvas, camX, camY, tx, ty int) {
	x := tx*TilePx - camX
	y := (ty+1)*TilePx - camY
	h := hash2(tx, ty)
	c.FillRect(x, y-1, 4, 1, 59)
	c.FillRect(x+1, y-2, 2, 1, 59)
	if h%2 == 0 {
		c.Set(x+int(h%4), y-3, 58)
	}
}

// drawCrystal draws a small glowing gem with a slow twinkle.
func (l *Level) drawCrystal(c *gfx.Canvas, camX, camY, tx, ty int, t float64) {
	x := tx*TilePx + 1 - camX
	y := ty*TilePx + 1 - camY
	c.Set(x+1, y, 97)
	c.Set(x, y+1, 97)
	c.Set(x+1, y+1, 183)
	c.Set(x+2, y+1, 97)
	c.Set(x+1, y+2, 97)
	if (int(t*2)+int(hash2(tx, ty)))%5 == 0 {
		c.Set(x+1, y+1, 231) // shine
	}
}

// DrawForeground renders the fg layer in front of the player: sparse
// tufts of alien grass swaying on 'g' tiles. Blades are single pixels
// with gaps, so Cappy stays visible walking through them.
var grassColors = []uint8{29, 35, 41}

func (l *Level) DrawForeground(c *gfx.Canvas, camX, camY int, t float64) {
	tx0, tx1 := l.visibleTiles(c, camX)
	for tx := tx0; tx <= tx1; tx++ {
		for ty := 0; ty < l.H; ty++ {
			if l.Cell(LayerFG, tx, ty) != 'g' {
				continue
			}
			base := (ty + 1) * TilePx // grass grows up from the tile floor
			for i := 0; i < TilePx; i++ {
				wx := tx*TilePx + i
				if hash2(wx, 0)%5 < 2 {
					continue // gap: keeps the layer see-through
				}
				gh := hash2(wx, ty)
				blade := 2 + int(gh%3)
				sway := int(math.Round(math.Sin(t*1.8 + float64(wx)*0.7)))
				col := grassColors[gh%uint32(len(grassColors))]
				for y := base - blade; y < base; y++ {
					x := wx - camX
					if y == base-blade {
						x += sway // only the tip sways
					}
					c.Set(x, y-camY, col)
				}
			}
		}
	}
}

// DrawMarkers visualises the invisible solid-layer entities (spawn,
// aliens, parts, bosses, ship, doors). The game proper never calls
// this; the editor uses it so placements are visible while editing.
func (l *Level) DrawMarkers(c *gfx.Canvas, camX, camY int) {
	for ty := 0; ty < l.H; ty++ {
		for tx := 0; tx < l.W; tx++ {
			x, y := tx*TilePx-camX, ty*TilePx-camY
			switch l.grid[LayerSolid][ty*l.W+tx] {
			case 'S':
				c.Blit(sprCappyIdle.R, x-3, y-8)
			case 'a':
				c.Blit(sprWalker1.R, x-2, y-1)
			case 'f':
				c.Blit(sprFlyer1.R, x-2, y-1)
			case 'b':
				c.Blit(sprBat1.R, x-2, y)
			case 'u':
				c.Blit(sprLurker1.R, x-2, y-1)
			case 'z':
				c.Blit(sprShard1.R, x-1, y-2)
			case 'e':
				c.Blit(sprMagling1.R, x-2, y-1)
			case 'P':
				c.Blit(sprPart, x, y)
			case 'H':
				c.Blit(sprShip, x, y+2)
			case 'D':
				c.Blit(sprDimi.R, x-11, y-10)
			case 'Q':
				c.Blit(sprPrisma.R, x-7, y-10)
			case 'M':
				c.Blit(sprMagmaw.R, x-9, y-10)
			case 'd':
				c.Rect(x, y, TilePx, TilePx, 51)
			}
		}
	}
}
