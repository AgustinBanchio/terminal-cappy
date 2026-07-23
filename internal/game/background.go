package game

import (
	"math"
	"math/rand"

	"cappy/internal/gfx"
)

// Background renders the multi-layer parallax space backdrop:
// a banded sky, a hashed starfield, a cratered moon, and two silhouette
// mountain ridges scrolling at different fractions of the camera speed.
type Background struct {
	p1, p2, p3, p4 float64 // ridge phase offsets, derived from the seed
	moonX          float64
}

func NewBackground(rng *rand.Rand) *Background {
	return &Background{
		p1:    rng.Float64() * 100,
		p2:    rng.Float64() * 100,
		p3:    rng.Float64() * 100,
		p4:    rng.Float64() * 100,
		moonX: 40 + rng.Float64()*30,
	}
}

// skyBands runs top (deep space) to bottom (a faint purple horizon glow).
var skyBands = []uint8{16, 16, 232, 232, 233, 233, 234, 234, 53, 54, 96, 54}

var starColors = []uint8{244, 250, 255, 111, 229}

func (b *Background) farRidge(x float64) float64 {
	return 8 + 5*math.Sin(x*0.045+b.p1) + 3*math.Sin(x*0.11+b.p2)
}

func (b *Background) nearRidge(x float64) float64 {
	return 6 + 4*math.Sin(x*0.06+b.p3) + 2.5*math.Sin(x*0.15+b.p4)
}

// Draw fills the whole canvas: every pixel gets painted, so no Clear is
// needed before it.
func (b *Background) Draw(c *gfx.Canvas, camX, camY int, t float64) {
	frame := int(t * 8)

	// Sky bands + starfield (stars barely parallax: factor 0.08).
	for sy := 0; sy < c.H; sy++ {
		band := (sy + camY/6) * len(skyBands) / max(1, c.H)
		if band < 0 {
			band = 0
		}
		if band >= len(skyBands) {
			band = len(skyBands) - 1
		}
		sky := skyBands[band]
		wy := sy + camY/8
		for sx := 0; sx < c.W; sx++ {
			wx := sx + camX/12
			col := sky
			if h := hash2(wx, wy); h%53 == 0 {
				col = starColors[(h>>8)%uint32(len(starColors))]
				if (uint32(frame)+h>>16)%37 < 4 {
					col = 240 // twinkle off
				}
			}
			c.Set(sx, sy, col)
		}
	}

	b.drawMoon(c, camX, camY)

	// Far ridge (deep purple), then near ridge (dark red), each anchored
	// a little above the terrain line and scrolling slower than it.
	farBase := float64(c.H)*0.62 - float64(camY)*0.3
	nearBase := float64(c.H)*0.78 - float64(camY)*0.55
	for sx := 0; sx < c.W; sx++ {
		fx := float64(sx) + float64(camX)*0.25
		top := int(farBase - b.farRidge(fx))
		for sy := top; sy < c.H; sy++ {
			c.Set(sx, sy, 53)
		}
		nx := float64(sx) + float64(camX)*0.5
		top = int(nearBase - b.nearRidge(nx))
		for sy := top; sy < c.H; sy++ {
			c.Set(sx, sy, 89)
		}
	}
}

func (b *Background) drawMoon(c *gfx.Canvas, camX, camY int) {
	cx := b.moonX - float64(camX)*0.03
	cy := 9.0 - float64(camY)*0.1
	const r = 6.5
	for sy := int(cy - r); sy <= int(cy+r)+1; sy++ {
		for sx := int(cx - r); sx <= int(cx+r)+1; sx++ {
			dx, dy := float64(sx)-cx, float64(sy)-cy
			if dx*dx+dy*dy > r*r {
				continue
			}
			col := uint8(146)
			if dx+dy > r*0.55 {
				col = 103 // shaded limb
			}
			if h := hash2(sx*7, sy*7); h%9 == 0 {
				col = 139 // craters
			}
			c.Set(sx, sy, col)
		}
	}
}
