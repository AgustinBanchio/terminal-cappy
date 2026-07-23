package game

import (
	"math"

	"github.com/AgustinBanchio/terminal-cappy/internal/gfx"
)

// Background renders the parallax backdrop. Each ambience zone has its
// own scenery, sampled per screen column so regions blend as the
// camera moves between them:
//
//	's' surface: storm sky, stars, moon, two mountain ridges
//	'u' cave:    dark rock walls, hanging and mounded silhouettes
//	'k' crystal: deep blue gloom, glints, jagged shard silhouettes
//	'l' lava:    ember glow rising from below, dark basalt silhouettes
type Background struct {
	p1, p2, p3, p4 float64 // ridge phase offsets
	moonX          float64
}

func NewBackground() *Background {
	return &Background{p1: 0.7, p2: 2.3, p3: 1.1, p4: 4.2, moonX: 58}
}

var (
	skyBands     = []uint8{16, 16, 232, 232, 233, 233, 234, 234, 53, 54, 96, 54}
	stormBands   = []uint8{16, 16, 232, 232, 233, 233, 234, 234, 235, 236, 59, 235}
	caveBands    = []uint8{232, 232, 233, 233, 233, 234}
	crystalBands = []uint8{232, 233, 17, 17, 17, 233}
	lavaBands    = []uint8{234, 233, 233, 52, 52, 88, 124}
	starColors   = []uint8{244, 250, 255, 111, 229}
	glintColors  = []uint8{30, 38, 45, 183}
	emberColors  = []uint8{202, 208, 130}
)

func (b *Background) farRidge(x float64) float64 {
	return 8 + 5*math.Sin(x*0.045+b.p1) + 3*math.Sin(x*0.11+b.p2)
}

func (b *Background) nearRidge(x float64) float64 {
	return 6 + 4*math.Sin(x*0.06+b.p3) + 2.5*math.Sin(x*0.15+b.p4)
}

// zig is a jagged triangle wave for crystal shard silhouettes.
func zig(x float64, period, amp float64) float64 {
	m := math.Mod(x, period)
	if m < 0 {
		m += period
	}
	return math.Abs(m-period/2) / (period / 2) * amp
}

func bandAt(bands []uint8, sy, h, camY, vshift int) uint8 {
	i := (sy + camY/vshift) * len(bands) / max(1, h)
	return bands[clampInt(i, 0, len(bands)-1)]
}

// Draw fills the whole canvas: every pixel gets painted, so no Clear
// is needed before it. zoneAt gives the ambience zone per screen
// column; raining darkens the surface sky.
func (b *Background) Draw(c *gfx.Canvas, camX, camY int, t float64, zoneAt func(sx int) byte, raining bool) {
	frame := int(t * 8)
	for sx := 0; sx < c.W; sx++ {
		switch zoneAt(sx) {
		case 'u':
			b.caveColumn(c, sx, camX, camY, frame)
		case 'k':
			b.crystalColumn(c, sx, camX, camY, frame)
		case 'l':
			b.lavaColumn(c, sx, camX, camY, frame)
		default:
			b.surfaceColumn(c, sx, camX, camY, frame, raining)
		}
	}
	b.drawMoon(c, camX, camY, zoneAt)
}

func (b *Background) surfaceColumn(c *gfx.Canvas, sx, camX, camY, frame int, raining bool) {
	bands := skyBands
	if raining {
		bands = stormBands
	}
	wx := sx + camX/12
	for sy := 0; sy < c.H; sy++ {
		col := bandAt(bands, sy, c.H, camY, 6)
		if h := hash2(wx, sy+camY/8); h%53 == 0 {
			col = starColors[(h>>8)%uint32(len(starColors))]
			if (uint32(frame)+h>>16)%37 < 4 {
				col = 240 // twinkle off
			}
		}
		c.Set(sx, sy, col)
	}
	farBase := float64(c.H)*0.62 - float64(camY)*0.3
	top := int(farBase - b.farRidge(float64(sx)+float64(camX)*0.25))
	for sy := top; sy < c.H; sy++ {
		c.Set(sx, sy, 53)
	}
	nearBase := float64(c.H)*0.78 - float64(camY)*0.55
	top = int(nearBase - b.nearRidge(float64(sx)+float64(camX)*0.5))
	for sy := top; sy < c.H; sy++ {
		c.Set(sx, sy, 89)
	}
}

func (b *Background) caveColumn(c *gfx.Canvas, sx, camX, camY, frame int) {
	wx := sx + camX/6
	for sy := 0; sy < c.H; sy++ {
		col := bandAt(caveBands, sy, c.H, 0, 8)
		if h := hash2(wx, sy+camY/6); h%23 == 0 {
			col = 234 // distant wall texture
		} else if h%701 == 0 && (uint32(frame/3)+h)%5 < 3 {
			col = 29 // glow worms
		}
		c.Set(sx, sy, col)
	}
	fx := float64(sx) + float64(camX)*0.3
	hang := int(6 + 4*math.Sin(fx*0.08+b.p1) + 2*math.Sin(fx*0.21+b.p2))
	for sy := 0; sy < hang; sy++ {
		c.Set(sx, sy, 235)
	}
	mound := int(5 + 3*math.Sin(fx*0.07+b.p3) + 2*math.Sin(fx*0.19+b.p4))
	for sy := c.H - mound; sy < c.H; sy++ {
		c.Set(sx, sy, 235)
	}
}

func (b *Background) crystalColumn(c *gfx.Canvas, sx, camX, camY, frame int) {
	wx := sx + camX/6
	for sy := 0; sy < c.H; sy++ {
		col := bandAt(crystalBands, sy, c.H, 0, 8)
		if h := hash2(wx, sy+camY/6); h%89 == 0 {
			col = glintColors[(h>>8)%uint32(len(glintColors))]
			if (uint32(frame)+h>>16)%23 < 8 {
				col = 24 // glint fades
			}
		}
		c.Set(sx, sy, col)
	}
	fx := float64(sx) + float64(camX)*0.35
	up := int(4 + zig(fx+b.p1*10, 14, 9) + zig(fx, 5, 3))
	for sy := c.H - up; sy < c.H; sy++ {
		c.Set(sx, sy, 61)
	}
	down := int(3 + zig(fx+b.p2*10, 17, 7) + zig(fx+3, 6, 3))
	for sy := 0; sy < down; sy++ {
		c.Set(sx, sy, 54)
	}
}

func (b *Background) lavaColumn(c *gfx.Canvas, sx, camX, camY, frame int) {
	wx := sx + camX/6
	for sy := 0; sy < c.H; sy++ {
		col := bandAt(lavaBands, sy, c.H, 0, 8)
		if h := hash2(wx, sy+camY/6); h%97 == 0 && sy > c.H/3 {
			col = emberColors[(h>>8)%uint32(len(emberColors))]
			if (uint32(frame)+h>>16)%13 < 6 {
				col = 52 // ember fades
			}
		}
		c.Set(sx, sy, col)
	}
	fx := float64(sx) + float64(camX)*0.35
	mound := int(6 + 4*math.Sin(fx*0.09+b.p3) + 2.5*math.Sin(fx*0.23+b.p4))
	for sy := c.H - mound; sy < c.H; sy++ {
		c.Set(sx, sy, 232)
	}
	if hash2(wx, frame/2)%11 == 0 {
		c.Set(sx, c.H-mound-1, 208) // molten rim licking the silhouette
	}
}

func (b *Background) drawMoon(c *gfx.Canvas, camX, camY int, zoneAt func(sx int) byte) {
	cx := b.moonX - float64(camX)*0.03
	cy := 9.0 - float64(camY)*0.1
	const r = 6.5
	for sy := int(cy - r); sy <= int(cy+r)+1; sy++ {
		for sx := int(cx - r); sx <= int(cx+r)+1; sx++ {
			if sx < 0 || sx >= c.W || zoneAt(sx) != 's' {
				continue
			}
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
