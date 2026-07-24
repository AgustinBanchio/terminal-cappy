package game

import (
	"math"

	"github.com/AgustinBanchio/terminal-cappy/internal/gfx"
)

// Bosses are big, room-locked fights. Each boss idles in its chamber
// until Cappy walks fully inside; the chamber's energy doors then seal
// until the boss dies, at which point it drops a ship part.

type bossKind int

const (
	bossDimi bossKind = iota
	bossPrisma
	bossMagmaw
)

type bossInfo struct {
	kind      bossKind
	name, sub string
	frames    gfx.Frames
	w, h      float64
	hp        int
}

var bossInfos = map[rune]bossInfo{
	'D': {bossDimi, "DIMI", "WARDEN OF THE RUINS", sprDimi, 24, 14, 18},
	'Q': {bossPrisma, "PRISMA", "THE CRYSTAL QUEEN", sprPrisma, 16, 12, 14},
	'M': {bossMagmaw, "MAGMAW", "LORD OF THE DEEP FIRE", sprMagmaw, 20, 13, 17},
}

type Boss struct {
	info   bossInfo
	X, Y   float64
	VX, VY float64
	HP     int
	dir    float64

	Active bool // fight running: doors sealed
	Dead   bool

	chamber [4]int // interior tile bounds x0,y0,x1,y1
	doors   [][2]int

	t, hurt  float64
	cool     float64 // main attack cooldown
	tele     float64 // telegraph before a charge
	dash     float64 // remaining charge/dash time
	jumpCool float64
	volley   float64
	onGround bool
	wasAir   bool
	anchorX  float64
	anchorY  float64
}

func newBoss(marker rune, x, y float64, l *Level) *Boss {
	info := bossInfos[marker]
	b := &Boss{
		info: info,
		X:    x - info.w/2, Y: y - info.h,
		HP:  info.hp,
		dir: -1,
	}
	b.anchorX, b.anchorY = b.X, b.Y
	tx, ty := fdiv(int(x), TilePx), fdiv(int(y), TilePx)
	b.chamber = chamberBounds(l, tx, ty)
	// Chamber walls are up to two tiles thick, so doors can sit two
	// tiles outside the interior bounds.
	for _, d := range l.Doors {
		if d[0] >= b.chamber[0]-2 && d[0] <= b.chamber[2]+2 &&
			d[1] >= b.chamber[1]-2 && d[1] <= b.chamber[3]+2 {
			b.doors = append(b.doors, d)
		}
	}
	return b
}

// chamberBounds finds the room around a boss marker by scanning until
// walls. Door tiles count as walls so open doors do not leak the room
// bounds outside the chamber.
func chamberBounds(l *Level, tx, ty int) [4]int {
	blocked := func(x, y int) bool {
		return l.SolidTile(x, y) || l.Cell(LayerSolid, x, y) == 'd'
	}
	x0, x1, y0, y1 := tx, tx, ty, ty
	for x0 > 0 && !blocked(x0-1, ty) {
		x0--
	}
	for x1 < l.W-1 && !blocked(x1+1, ty) {
		x1++
	}
	for y0 > 0 && !blocked(tx, y0-1) {
		y0--
	}
	for y1 < l.H-1 && !blocked(tx, y1+1) {
		y1++
	}
	return [4]int{x0, y0, x1, y1}
}

// playerInChamber reports whether Cappy is fully inside the room.
func (b *Boss) playerInChamber(p *Player) bool {
	x0 := float64(b.chamber[0]*TilePx) + 4
	y0 := float64(b.chamber[1] * TilePx)
	x1 := float64((b.chamber[2]+1)*TilePx) - 4
	y1 := float64((b.chamber[3] + 1) * TilePx)
	return p.X > x0 && p.X+playerW < x1 && p.Y > y0 && p.Y+playerH < y1
}

func (b *Boss) update(g *Game, dt float64) {
	if b.Dead {
		return
	}
	p := g.player

	if !b.Active {
		if b.playerInChamber(p) {
			b.Active = true
			g.level.LockDoors(b.doors, true)
			g.activeBoss = b
			g.bossTitleT = 2.6
			g.shake = math.Max(g.shake, 1.5)
		}
		return
	}

	b.t += dt
	b.hurt = math.Max(0, b.hurt-dt)

	// Hold still while the name card shows, then fight.
	if g.bossTitleT <= 0 {
		switch b.info.kind {
		case bossDimi:
			b.updateDimi(g, dt)
		case bossPrisma:
			b.updatePrisma(g, dt)
		case bossMagmaw:
			b.updateMagmaw(g, dt)
		}
	} else if b.info.kind != bossPrisma {
		b.VX = 0
		b.moveGrounded(g, dt) // grounded bosses still settle
	}

	if aabb(p.X, p.Y, playerW, playerH, b.X, b.Y, b.info.w, b.info.h) {
		from := -1.0
		if b.X+b.info.w/2 > p.X+playerW/2 {
			from = 1.0
		}
		p.Hurt(g, from)
	}
}

func (b *Boss) dirToPlayer(p *Player) float64 {
	if p.X+playerW/2 > b.X+b.info.w/2 {
		return 1
	}
	return -1
}

// --- Dimi: ground bruiser. Stalks, telegraphs, then charges; leaps
// when Cappy takes to the air.
func (b *Boss) updateDimi(g *Game, dt float64) {
	p := g.player
	b.cool -= dt
	b.jumpCool -= dt

	switch {
	case b.tele > 0:
		b.tele -= dt
		b.VX = 0
		if b.tele <= 0 {
			b.dash = 0.9
			b.dir = b.dirToPlayer(p)
			b.VX = b.dir * 95
			g.shake = math.Max(g.shake, 1)
		}
	case b.dash > 0:
		b.dash -= dt
	default:
		b.dir = b.dirToPlayer(p)
		b.VX = b.dir * 18
		if b.cool <= 0 {
			b.tele = 0.5
			b.cool = 3.2
		}
	}

	if b.onGround && b.jumpCool <= 0 && p.Y+playerH < b.Y-10 {
		b.VY = -150
		b.jumpCool = 2.4
	}
	b.moveGrounded(g, dt)
}

// --- Prisma: floats around the room, dashes at Cappy and fires shard
// volleys.
func (b *Boss) updatePrisma(g *Game, dt float64) {
	p := g.player
	b.cool -= dt
	b.volley -= dt

	if b.dash > 0 {
		b.dash -= dt
	} else {
		// Drift on a slow lissajous around the chamber anchor.
		tx := b.anchorX + math.Sin(b.t*0.9)*24
		ty := b.anchorY + math.Sin(b.t*1.7)*10 - 8
		b.VX = (tx - b.X) * 1.6
		b.VY = (ty - b.Y) * 1.6
		if b.cool <= 0 {
			dx := (p.X + playerW/2) - (b.X + b.info.w/2)
			dy := (p.Y + playerH/2) - (b.Y + b.info.h/2)
			d := math.Hypot(dx, dy)
			if d > 1 {
				b.VX, b.VY = dx/d*110, dy/d*110
				b.dash = 0.5
			}
			b.cool = 2.6
		}
	}
	if b.volley <= 0 {
		b.volley = 4
		cx, cy := b.X+b.info.w/2, b.Y+b.info.h/2
		dx := (p.X + playerW/2) - cx
		dy := (p.Y + playerH/2) - cy
		base := math.Atan2(dy, dx)
		for _, off := range []float64{-0.35, 0, 0.35} {
			g.spawnShot(cx, cy, math.Cos(base+off)*65, math.Sin(base+off)*65, 0, 51, 38)
		}
	}
	b.dir = b.dirToPlayer(p)
	b.X += b.VX * dt
	b.Y += b.VY * dt
	b.clampToChamber()
}

func (b *Boss) clampToChamber() {
	x0 := float64(b.chamber[0]*TilePx) + 1
	y0 := float64(b.chamber[1]*TilePx) + 1
	x1 := float64((b.chamber[2]+1)*TilePx) - b.info.w - 1
	y1 := float64((b.chamber[3]+1)*TilePx) - b.info.h - 1
	b.X = math.Max(x0, math.Min(b.X, x1))
	b.Y = math.Max(y0, math.Min(b.Y, y1))
}

// --- Magmaw: hops at Cappy in heavy arcs; every landing spits molten
// blobs that arc back down.
func (b *Boss) updateMagmaw(g *Game, dt float64) {
	p := g.player
	b.cool -= dt
	if b.onGround && b.cool <= 0 {
		b.dir = b.dirToPlayer(p)
		b.VY = -135
		b.VX = b.dir * 42
		b.cool = 1.9
	}
	if b.onGround {
		b.VX = 0
	}
	landed := b.moveGrounded(g, dt)
	if landed {
		g.shake = math.Max(g.shake, 1.2)
		cx, cy := b.X+b.info.w/2, b.Y
		for _, vx := range []float64{-32, 34} {
			g.spawnShot(cx, cy, vx+b.dir*12, -95, 260, 208, 202)
		}
		g.emitBurst(cx, b.Y+b.info.h, 6, []uint8{202, 208, 240}, 30, 120)
	}
}

// moveGrounded applies gravity and tile collision; returns true on the
// frame the boss lands.
func (b *Boss) moveGrounded(g *Game, dt float64) (landed bool) {
	l := g.level
	b.VY = math.Min(b.VY+fallGrav*dt, maxFall)

	nx := b.X + b.VX*dt
	if !l.SolidBox(nx, b.Y, b.info.w, b.info.h) {
		b.X = nx
	} else {
		b.VX = 0
		b.dash = 0
	}
	ny := b.Y + b.VY*dt
	if !l.SolidBox(b.X, ny, b.info.w, b.info.h) {
		b.Y = ny
	} else {
		if b.VY > 0 {
			b.Y = math.Floor((ny+b.info.h)/TilePx)*TilePx - b.info.h
		}
		b.VY = 0
	}
	b.onGround = l.SolidBox(b.X, b.Y+b.info.h+0.05, b.info.w, 0.1)
	landed = b.onGround && b.wasAir
	b.wasAir = !b.onGround
	return landed
}

func (b *Boss) damage(g *Game, dmg int) {
	if b.Dead || !b.Active {
		return
	}
	b.HP -= dmg
	b.hurt = 0.12
	if b.HP > 0 {
		return
	}
	b.Dead = true
	g.level.LockDoors(b.doors, false)
	if g.activeBoss == b {
		g.activeBoss = nil
	}
	cx, cy := b.X+b.info.w/2, b.Y+b.info.h/2
	g.emitBurst(cx, cy, 30, []uint8{231, 226, 196, 240}, 80, 120)
	g.shake = 4
	g.pickups = append(g.pickups, &Pickup{Kind: pickupPart, Variant: g.nextPartVariant(), X: cx, Y: cy - 4})
	g.sayf("%s DEFEATED! IT WAS GUARDING A SHIP PART", 4, b.info.name)
}

func (b *Boss) draw(c *gfx.Canvas, camX, camY int, t float64) {
	if b.Dead {
		return
	}
	spr := b.info.frames.Facing(int(b.dir))
	x := int(b.X) - int(float64(spr.W)-b.info.w)/2 - camX
	y := int(b.Y+b.info.h) - spr.H - camY
	if b.info.kind == bossPrisma {
		y += int(math.Sin(t*3) * 1.5)
	}
	switch {
	case b.hurt > 0:
		c.BlitTinted(spr, x, y, 231)
	case b.tele > 0 && int(b.tele*14)%2 == 0:
		c.BlitTinted(spr, x, y, 255) // charge telegraph blink
	default:
		c.Blit(spr, x, y)
	}
}

// --- enemy projectiles ----------------------------------------------------

// Shot is a boss projectile: Prisma's shards fly straight, Magmaw's
// lava blobs arc under gravity.
type Shot struct {
	X, Y, VX, VY float64
	Grav         float64
	Life         float64
	Color, Tail  uint8
}

func (g *Game) spawnShot(x, y, vx, vy, grav float64, color, tail uint8) {
	g.shots = append(g.shots, Shot{X: x, Y: y, VX: vx, VY: vy, Grav: grav,
		Life: 3, Color: color, Tail: tail})
}

func (g *Game) updateShots(dt float64) {
	p := g.player
	alive := g.shots[:0]
	for _, s := range g.shots {
		s.Life -= dt
		s.VY += s.Grav * dt
		s.X += s.VX * dt
		s.Y += s.VY * dt
		if s.Life <= 0 || g.level.SolidAtPx(s.X, s.Y) {
			g.emitBurst(s.X, s.Y, 4, []uint8{s.Color, s.Tail, 240}, 25, 100)
			continue
		}
		if aabb(s.X-1, s.Y-1, 2, 2, p.X, p.Y, playerW, playerH) {
			p.Hurt(g, -sign(s.VX))
			continue
		}
		alive = append(alive, s)
	}
	g.shots = alive
}

func drawShot(c *gfx.Canvas, s Shot, camX, camY int) {
	x, y := int(s.X)-camX, int(s.Y)-camY
	c.Set(x, y, s.Color)
	c.Set(x-1, y, s.Tail)
	c.Set(x, y-1, s.Tail)
}
