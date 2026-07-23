package game

import (
	"math"
	"time"

	"cappy/internal/gfx"
)

// Movement tuning, in pixels and seconds. The canvas is ~48px tall in a
// default 80x24 terminal, so a full jump (~17px) is about a third of it.
const (
	playerW = 6
	playerH = 10

	runSpeed  = 46.0
	runAccel  = 300.0
	friction  = 260.0
	gravity   = 340.0
	maxFall   = 130.0
	jumpVel   = -108.0
	slideFall = 28.0 // max fall speed while wall sliding
	wallJumpX = 60.0
	wallJumpY = -95.0

	coyoteTime = 0.10
	jumpBuffer = 0.12
	shootEvery = 0.18
	bulletVel  = 140.0
	maxHP      = 5
)

type Player struct {
	X, Y   float64 // hitbox top-left
	VX, VY float64
	Facing int // 1 right, -1 left
	HP     int

	OnGround bool
	Sliding  bool
	wallDir  int // -1 wall on the left, 1 wall on the right

	coyote  float64
	jumpBuf float64
	shootCD float64
	HurtCD  float64
	muzzle  float64
	anim    float64
}

func NewPlayer(x, y float64) *Player {
	return &Player{X: x - playerW/2, Y: y - playerH, Facing: 1, HP: maxHP}
}

func (p *Player) Update(g *Game, dt float64, now time.Time) {
	l := g.level

	dir := 0.0
	if g.in.held(actLeft, now) {
		dir -= 1
	}
	if g.in.held(actRight, now) {
		dir += 1
	}
	if dir != 0 {
		p.Facing = int(dir)
	}

	// Horizontal acceleration / friction.
	if dir != 0 {
		p.VX += dir * runAccel * dt
		p.VX = math.Max(-runSpeed, math.Min(runSpeed, p.VX))
	} else if p.OnGround {
		if p.VX > 0 {
			p.VX = math.Max(0, p.VX-friction*dt)
		} else {
			p.VX = math.Min(0, p.VX+friction*dt)
		}
	}

	// Wall contact probes (a sliver just outside each side of the hitbox).
	touchL := l.SolidBox(p.X-0.6, p.Y+1, 0.5, playerH-2)
	touchR := l.SolidBox(p.X+playerW+0.1, p.Y+1, 0.5, playerH-2)
	p.wallDir = 0
	if touchL {
		p.wallDir = -1
	} else if touchR {
		p.wallDir = 1
	}

	// Wall slide: airborne, falling, and pushing into the wall.
	p.Sliding = !p.OnGround && p.VY > 0 &&
		((touchL && dir < 0) || (touchR && dir > 0))

	// Gravity.
	p.VY += gravity * dt
	limit := maxFall
	if p.Sliding {
		limit = slideFall
	}
	p.VY = math.Min(p.VY, limit)

	// Jumping: buffered presses, coyote time, and wall jumps.
	if g.in.consume(actJump) {
		p.jumpBuf = jumpBuffer
	}
	if p.jumpBuf > 0 {
		switch {
		case p.OnGround || p.coyote > 0:
			p.VY = jumpVel
			p.jumpBuf, p.coyote = 0, 0
			g.emitDust(p.X+playerW/2, p.Y+playerH, 4)
		case p.wallDir != 0:
			p.VY = wallJumpY
			p.VX = float64(-p.wallDir) * wallJumpX
			p.Facing = -p.wallDir
			p.jumpBuf = 0
			g.emitDust(p.X+playerW/2+float64(p.wallDir*3), p.Y+playerH/2, 4)
		}
	}

	// Shooting. Holding X keeps firing via terminal auto-repeat.
	if g.in.consume(actShoot) && p.shootCD <= 0 {
		mx, my := p.muzzlePos()
		g.spawnBullet(mx, my, float64(p.Facing)*bulletVel)
		p.shootCD = shootEvery
		p.muzzle = 0.07
		p.VX -= float64(p.Facing) * 6 // a little recoil
	}

	p.moveX(l, dt)
	wasAirborne := !p.OnGround
	p.moveY(l, dt)
	p.OnGround = p.VY >= 0 && l.SolidBox(p.X, p.Y+playerH+0.05, playerW, 0.1)
	if p.OnGround {
		p.coyote = coyoteTime
		if wasAirborne {
			g.emitDust(p.X+playerW/2, p.Y+playerH, 3)
		}
	}
	if p.Sliding {
		g.emitDust(p.X+playerW/2+float64(p.wallDir*3), p.Y+playerH, 1)
	}

	p.coyote = math.Max(0, p.coyote-dt)
	p.jumpBuf = math.Max(0, p.jumpBuf-dt)
	p.shootCD = math.Max(0, p.shootCD-dt)
	p.HurtCD = math.Max(0, p.HurtCD-dt)
	p.muzzle = math.Max(0, p.muzzle-dt)
	p.anim += dt
}

func (p *Player) moveX(l *Level, dt float64) {
	nx := p.X + p.VX*dt
	if !l.SolidBox(nx, p.Y, playerW, playerH) {
		p.X = nx
		return
	}
	if p.VX > 0 {
		p.X = math.Floor((nx+playerW)/TilePx)*TilePx - playerW - 0.001
	} else {
		p.X = math.Floor(nx/TilePx+1) * TilePx
	}
	p.VX = 0
}

func (p *Player) moveY(l *Level, dt float64) {
	ny := p.Y + p.VY*dt
	if !l.SolidBox(p.X, ny, playerW, playerH) {
		p.Y = ny
		return
	}
	if p.VY > 0 {
		p.Y = math.Floor((ny+playerH)/TilePx)*TilePx - playerH - 0.001
	} else {
		p.Y = math.Floor(ny/TilePx+1) * TilePx
	}
	p.VY = 0
}

// muzzlePos is the world position of the revolver tip.
func (p *Player) muzzlePos() (float64, float64) {
	if p.Facing >= 0 {
		return p.X + playerW + 3, p.Y + 5
	}
	return p.X - 4, p.Y + 5
}

// Hurt applies contact damage with knockback. fromDir is the x direction
// the hit came from (-1: from the left).
func (p *Player) Hurt(g *Game, fromDir float64) {
	if p.HurtCD > 0 {
		return
	}
	p.HP--
	p.HurtCD = 1.2
	p.VX = -fromDir * 60
	p.VY = -60
	g.shake = 2
	g.emitBurst(p.X+playerW/2, p.Y+playerH/2, 8, []uint8{160, 196, 255}, 50, 200)
	if p.HP <= 0 {
		g.kill("CAPPY'S SUIT GAVE OUT")
	}
}

func (p *Player) sprite() *gfx.Sprite {
	var f gfx.Frames
	switch {
	case p.Sliding:
		f = sprCappySlide
	case !p.OnGround && p.VY < 0:
		f = sprCappyJump
	case !p.OnGround:
		f = sprCappyFall
	case math.Abs(p.VX) > 5:
		if int(p.anim*10)%2 == 0 {
			f = sprCappyRun1
		} else {
			f = sprCappyRun2
		}
	default:
		f = sprCappyIdle
	}
	facing := p.Facing
	if p.Sliding {
		facing = -p.wallDir // face away from the wall
	}
	return f.Facing(facing)
}

// Draw renders Cappy (blinking while invulnerable) plus the muzzle flash.
func (p *Player) Draw(c *gfx.Canvas, camX, camY int) {
	if p.HurtCD > 0 && int(p.HurtCD*12)%2 == 0 {
		return
	}
	// The 10x12 sprite is centred on the 6x10 hitbox.
	c.Blit(p.sprite(), int(p.X)-2-camX, int(p.Y)-2-camY)
	if p.muzzle > 0 {
		mx, my := p.muzzlePos()
		c.Blit(sprMuzzle, int(mx)-1-camX, int(my)-1-camY)
	}
}
