package game

import (
	"math"

	"github.com/AgustinBanchio/terminal-cappy/internal/gfx"
)

// --- aliens ------------------------------------------------------------

type alienKind int

const (
	alienWalker alienKind = iota
	alienFlyer
)

type Alien struct {
	Kind   alienKind
	X, Y   float64
	W, H   float64
	VX, VY float64
	HP     int
	dir    float64
	homeY  float64
	t      float64
	hurt   float64
}

func newAlien(kind alienKind, x, y float64) *Alien {
	a := &Alien{Kind: kind, dir: 1, t: 0}
	switch kind {
	case alienWalker:
		a.W, a.H, a.HP = 8, 5, 2
	case alienFlyer:
		a.W, a.H, a.HP = 7, 6, 1
	}
	a.X, a.Y = x-a.W/2, y-a.H/2
	a.homeY = a.Y
	return a
}

func (a *Alien) update(g *Game, dt float64) {
	a.t += dt
	a.hurt = math.Max(0, a.hurt-dt)
	switch a.Kind {
	case alienWalker:
		a.updateWalker(g.level, dt)
	case alienFlyer:
		a.updateFlyer(g.player, dt)
	}

	// Contact damage.
	p := g.player
	if aabb(p.X, p.Y, playerW, playerH, a.X, a.Y, a.W, a.H) {
		from := -1.0
		if a.X+a.W/2 > p.X+playerW/2 {
			from = 1.0
		}
		p.Hurt(g, from)
	}
}

func (a *Alien) updateWalker(l *Level, dt float64) {
	// Gravity + floor collision.
	a.VY = math.Min(a.VY+fallGrav*dt, maxFall)
	ny := a.Y + a.VY*dt
	grounded := false
	if l.SolidBox(a.X, ny, a.W, a.H) {
		a.Y = math.Floor((ny+a.H)/TilePx)*TilePx - a.H
		a.VY = 0
		grounded = true
	} else {
		a.Y = ny
	}

	// Patrol: turn at walls and at ledges.
	const speed = 14.0
	nx := a.X + a.dir*speed*dt
	footX := nx + a.W + 1
	if a.dir < 0 {
		footX = nx - 1
	}
	blocked := l.SolidBox(nx, a.Y, a.W, a.H)
	ledge := grounded && !l.SolidBox(footX, a.Y+a.H+1, 1, 2)
	if blocked || ledge {
		a.dir = -a.dir
	} else {
		a.X = nx
	}
}

func (a *Alien) updateFlyer(p *Player, dt float64) {
	// Bob on a sine wave around a home point that slowly stalks Cappy
	// when she gets close.
	dx := (p.X + playerW/2) - (a.X + a.W/2)
	dy := (p.Y + playerH/2) - (a.homeY + a.H/2)
	if math.Abs(dx) < 50 && math.Abs(dy) < 40 {
		a.dir = sign(dx)
		a.X += sign(dx) * 11 * dt
		a.homeY += sign(dy) * 7 * dt
	}
	a.Y = a.homeY + math.Sin(a.t*2.4)*3
}

func (a *Alien) damage(g *Game, dmg int, fromVX float64) {
	a.HP -= dmg
	a.hurt = 0.1
	a.X += sign(fromVX) * 1.5
	if a.HP > 0 {
		return
	}
	colors := []uint8{40, 118, 231}
	if a.Kind == alienFlyer {
		colors = []uint8{165, 201, 231}
	}
	g.emitBurst(a.X+a.W/2, a.Y+a.H/2, 14, colors, 55, 160)
	g.shake = math.Max(g.shake, 1)
	if g.rng.Float64() < 0.25 {
		g.pickups = append(g.pickups, &Pickup{Kind: pickupHeart, X: a.X + a.W/2, Y: a.Y})
	}
}

func (a *Alien) draw(c *gfx.Canvas, camX, camY int) {
	var f gfx.Frames
	switch a.Kind {
	case alienWalker:
		if int(a.t*6)%2 == 0 {
			f = sprWalker1
		} else {
			f = sprWalker2
		}
	case alienFlyer:
		if int(a.t*4)%2 == 0 {
			f = sprFlyer1
		} else {
			f = sprFlyer2
		}
	}
	spr := f.Facing(int(a.dir))
	x, y := int(a.X)-camX, int(a.Y)-camY
	if a.hurt > 0 {
		c.BlitTinted(spr, x, y, 231)
	} else {
		c.Blit(spr, x, y)
	}
}

// --- bullets -----------------------------------------------------------

type Bullet struct {
	X, Y, VX float64
	Life     float64
}

func (g *Game) spawnBullet(x, y, vx float64) {
	g.bullets = append(g.bullets, Bullet{X: x, Y: y, VX: vx, Life: 0.6})
}

func (g *Game) updateBullets(dt float64) {
	alive := g.bullets[:0]
bullets:
	for _, b := range g.bullets {
		b.Life -= dt
		if b.Life <= 0 {
			continue
		}
		// Two substeps so fast bolts cannot tunnel through 4px tiles.
		for s := 0; s < 2; s++ {
			b.X += b.VX * dt / 2
			if g.level.SolidAtPx(b.X, b.Y) {
				g.emitBurst(b.X, b.Y, 4, []uint8{226, 208, 240}, 30, 120)
				continue bullets
			}
			for _, a := range g.aliens {
				if a.HP > 0 && aabb(b.X-1, b.Y-1, 2, 2, a.X, a.Y, a.W, a.H) {
					a.damage(g, 1, b.VX)
					g.emitBurst(b.X, b.Y, 5, []uint8{226, 231, 208}, 35, 120)
					continue bullets
				}
			}
		}
		alive = append(alive, b)
	}
	g.bullets = alive
}

func drawBullet(c *gfx.Canvas, b Bullet, camX, camY int) {
	x, y := int(b.X)-camX, int(b.Y)-camY
	d := 1
	if b.VX < 0 {
		d = -1
	}
	c.Set(x, y, 231)
	c.Set(x-d, y, 214)
	c.Set(x-2*d, y, 196)
}

// --- pickups -----------------------------------------------------------

type pickupKind int

const (
	pickupPart pickupKind = iota
	pickupHeart
)

type Pickup struct {
	Kind pickupKind
	X, Y float64 // centre
	VY   float64
	t    float64
}

func (pk *Pickup) update(g *Game, dt float64) bool {
	pk.t += dt
	if pk.Kind == pickupHeart {
		// Dropped hearts fall until they land.
		pk.VY = math.Min(pk.VY+fallGrav*dt, maxFall)
		ny := pk.Y + pk.VY*dt
		if g.level.SolidBox(pk.X-2, ny-2, 4, 4) {
			pk.VY = 0
		} else {
			pk.Y = ny
		}
	}
	if g.rng.Float64() < 0.06 {
		g.emitBurst(pk.X, pk.Y-2, 1, []uint8{230, 221}, 8, -10)
	}

	p := g.player
	if !aabb(p.X, p.Y, playerW, playerH, pk.X-3, pk.Y-3, 6, 6) {
		return true
	}
	switch pk.Kind {
	case pickupPart:
		g.partsGot++
		g.emitBurst(pk.X, pk.Y, 12, []uint8{220, 226, 231}, 45, 60)
		if g.partsGot == g.partsTotal {
			g.say("ALL PARTS FOUND! GET BACK TO THE SHIP", 4)
		} else {
			g.sayf("SHIP PART FOUND (%d/%d)", 3, g.partsGot, g.partsTotal)
		}
	case pickupHeart:
		if p.HP < maxHP {
			p.HP++
		}
		g.emitBurst(pk.X, pk.Y, 8, []uint8{196, 210}, 35, 60)
	}
	return false
}

func (pk *Pickup) draw(c *gfx.Canvas, camX, camY int) {
	bob := int(math.Sin(pk.t*3) * 1.5)
	switch pk.Kind {
	case pickupPart:
		c.Blit(sprPart, int(pk.X)-2-camX, int(pk.Y)-2+bob-camY)
	case pickupHeart:
		c.Blit(sprHeart, int(pk.X)-2-camX, int(pk.Y)-2+bob-camY)
	}
}

// --- particles ----------------------------------------------------------

type Particle struct {
	X, Y, VX, VY float64
	Life, Max    float64
	Grav         float64
	Colors       []uint8
}

func (g *Game) emitBurst(x, y float64, n int, colors []uint8, speed, grav float64) {
	for i := 0; i < n; i++ {
		ang := g.rng.Float64() * 2 * math.Pi
		v := speed * (0.4 + g.rng.Float64()*0.6)
		life := 0.25 + g.rng.Float64()*0.4
		g.particles = append(g.particles, Particle{
			X: x, Y: y,
			VX: math.Cos(ang) * v, VY: math.Sin(ang) * v,
			Life: life, Max: life, Grav: grav, Colors: colors,
		})
	}
}

func (g *Game) emitDust(x, y float64, n int) {
	for i := 0; i < n; i++ {
		life := 0.15 + g.rng.Float64()*0.25
		g.particles = append(g.particles, Particle{
			X: x + (g.rng.Float64()-0.5)*4, Y: y,
			VX: (g.rng.Float64() - 0.5) * 30, VY: -g.rng.Float64() * 20,
			Life: life, Max: life, Grav: 120, Colors: []uint8{250, 247, 243},
		})
	}
}

func (g *Game) emitSmoke(x, y float64) {
	life := 0.8 + g.rng.Float64()*0.6
	g.particles = append(g.particles, Particle{
		X: x + (g.rng.Float64()-0.5)*3, Y: y,
		VX: (g.rng.Float64() - 0.5) * 6, VY: -12 - g.rng.Float64()*8,
		Life: life, Max: life, Colors: []uint8{240, 242, 245, 247},
	})
}

func (g *Game) emitFlame(x, y float64) {
	life := 0.2 + g.rng.Float64()*0.3
	g.particles = append(g.particles, Particle{
		X: x + (g.rng.Float64()-0.5)*6, Y: y,
		VX: (g.rng.Float64() - 0.5) * 15, VY: 30 + g.rng.Float64()*40,
		Life: life, Max: life, Colors: []uint8{231, 226, 208, 196},
	})
}

func (g *Game) updateParticles(dt float64) {
	alive := g.particles[:0]
	for _, p := range g.particles {
		p.Life -= dt
		if p.Life <= 0 {
			continue
		}
		p.VY += p.Grav * dt
		p.X += p.VX * dt
		p.Y += p.VY * dt
		alive = append(alive, p)
	}
	g.particles = alive
}

func drawParticle(c *gfx.Canvas, p Particle, camX, camY int) {
	frac := 1 - p.Life/p.Max
	idx := int(frac * float64(len(p.Colors)))
	if idx >= len(p.Colors) {
		idx = len(p.Colors) - 1
	}
	c.Set(int(p.X)-camX, int(p.Y)-camY, p.Colors[idx])
}

// --- helpers ------------------------------------------------------------

func aabb(x1, y1, w1, h1, x2, y2, w2, h2 float64) bool {
	return x1 < x2+w2 && x2 < x1+w1 && y1 < y2+h2 && y2 < y1+h1
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}
