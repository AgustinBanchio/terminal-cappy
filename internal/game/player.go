package game

import (
	"math"
	"time"

	"github.com/AgustinBanchio/terminal-cappy/internal/gfx"
)

// Movement tuning, in pixels and seconds, modelled on Hollow Knight's
// controller feel translated to our scale (the canvas is ~48px tall in
// a default 80x24 terminal):
//
//   - horizontal velocity is set directly to a locked run speed: no
//     acceleration ramp, no skid, identical control on ground and air
//   - gravity is asymmetric: falling is ~1.6x heavier than rising, so
//     jumps peak floaty and land snappy
//   - coyote time and jump buffering keep inputs forgiving
//   - wall jumps (and knockback) briefly lock horizontal control so
//     the kick away from the wall cannot be instantly cancelled
const (
	playerW = 6
	playerH = 10

	runSpeed  = 52.0
	riseGrav  = 330.0
	fallGrav  = 540.0
	maxFall   = 150.0
	jumpVel   = -112.0
	slideFall = 26.0 // fixed fall speed while wall sliding

	wallJumpX    = 62.0
	wallJumpY    = -102.0
	wallJumpLock = 0.14
	hurtLock     = 0.25

	coyoteTime = 0.09
	jumpBuffer = 0.10
	shootEvery = 0.18
	bulletVel  = 160.0
	maxHP      = 5

	// Dash, Hollow Knight rules: a horizontal burst in the facing
	// direction with gravity suspended, usable in the air (once per
	// airtime, restored by landing or grabbing a wall), on a short
	// cooldown.
	dashSpeed    = 130.0
	dashTime     = 0.18
	dashCooldown = 0.55
)

type Player struct {
	X, Y   float64 // hitbox top-left
	VX, VY float64
	Facing int // 1 right, -1 left
	HP     int

	OnGround bool
	Sliding  bool
	wallDir  int // -1 wall on the left, 1 wall on the right

	NudgeX int // queued 1px micro-steps from the precision keys

	dashT     float64 // remaining dash time
	dashCD    float64
	dashDir   int
	airDashed bool // air dash spent; restored on landing / wall grab

	lock    float64 // horizontal control lock (wall jump, knockback)
	coyote  float64
	jumpBuf float64
	shootCD float64
	HurtCD  float64
	muzzle  float64
	landT   float64 // landing squash pose timer
	anim    float64
	idleT   float64
}

func NewPlayer(x, y float64) *Player {
	return &Player{X: x - playerW/2, Y: y - playerH, Facing: 1, HP: maxHP}
}

func (p *Player) Update(g *Game, dt float64, now time.Time) {
	l := g.level

	dir := g.in.dir(now)

	// Locked run speed: velocity is set, not accelerated, unless a wall
	// jump or knockback temporarily owns the horizontal axis.
	if p.lock <= 0 {
		p.VX = float64(dir) * runSpeed
		if dir != 0 {
			p.Facing = dir
		}
	}

	// Precision micro-steps: exactly 1px per keypress, pixel-collided,
	// ignored while running so they never fight held movement.
	if p.NudgeX != 0 {
		if dir == 0 && p.lock <= 0 {
			step := 1.0
			if p.NudgeX < 0 {
				step = -1.0
			}
			for i := 0; i < p.NudgeX*int(step); i++ {
				if l.SolidBox(p.X+step, p.Y, playerW, playerH) {
					break
				}
				p.X += step
			}
			p.Facing = int(step)
		}
		p.NudgeX = 0
	}

	// Dash: starts in the facing direction; grounded dashes are always
	// available (off cooldown), airborne ones once per airtime.
	if g.in.consume(actDash) && p.dashCD <= 0 && p.dashT <= 0 &&
		(p.OnGround || p.Sliding || !p.airDashed) {
		p.dashT = dashTime
		p.dashCD = dashCooldown
		p.dashDir = p.Facing
		if p.Sliding {
			p.dashDir = -p.wallDir // dash away from the wall
		}
		if !p.OnGround {
			p.airDashed = true
		}
		g.emitDust(p.X+playerW/2, p.Y+playerH/2, 3)
		g.play(SfxDash)
	}
	if p.dashT > 0 {
		p.VX = float64(p.dashDir) * dashSpeed
		p.VY = 0
		p.Facing = p.dashDir
		// Afterimage trail.
		g.emitBurst(p.X+playerW/2, p.Y+playerH/2, 1, []uint8{195, 117, 111}, 8, 0)
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

	// Asymmetric gravity, suspended entirely while dashing.
	if p.dashT <= 0 {
		grav := riseGrav
		if p.VY >= 0 {
			grav = fallGrav
		}
		p.VY += grav * dt
		if p.Sliding {
			p.VY = math.Min(p.VY, slideFall)
		} else {
			p.VY = math.Min(p.VY, maxFall)
		}
	}

	// Jumping: buffered presses, coyote time, and wall jumps.
	if g.in.consume(actJump) {
		p.jumpBuf = jumpBuffer
	}
	if p.jumpBuf > 0 {
		switch {
		case p.OnGround || p.coyote > 0:
			p.VY = jumpVel
			p.jumpBuf, p.coyote, p.dashT = 0, 0, 0 // jump cancels a dash
			g.emitDust(p.X+playerW/2, p.Y+playerH, 4)
			g.play(SfxJump)
		case p.wallDir != 0:
			p.VY = wallJumpY
			p.VX = float64(-p.wallDir) * wallJumpX
			p.Facing = -p.wallDir
			p.lock = wallJumpLock
			p.jumpBuf, p.dashT = 0, 0
			g.emitDust(p.X+playerW/2+float64(p.wallDir*3), p.Y+playerH/2, 4)
			g.play(SfxJump)
		}
	}

	// Variable jump height: releasing jump while rising cuts the ascent
	// short. Only window mode has real release events; in the terminal
	// this never fires and jumps stay fixed-height.
	if g.in.consumeRelease(actJump) && p.VY < -30 {
		p.VY *= 0.45
	}

	// Shooting. Holding X keeps firing via terminal auto-repeat.
	if g.in.consume(actShoot) && p.shootCD <= 0 {
		mx, my := p.muzzlePos()
		g.spawnBullet(mx, my, float64(p.Facing)*bulletVel)
		p.shootCD = shootEvery
		p.muzzle = 0.07
		g.play(SfxShoot)
	}

	p.moveX(l, dt)
	if p.dashT > 0 && p.VX == 0 {
		p.dashT = 0 // dashed into a wall
	}
	wasAirborne := !p.OnGround
	p.moveY(l, dt)
	p.OnGround = p.VY >= 0 && l.SolidBox(p.X, p.Y+playerH+0.05, playerW, 0.1)
	if p.OnGround {
		p.coyote = coyoteTime
		p.airDashed = false
		if wasAirborne {
			p.landT = 0.12
			g.emitDust(p.X+playerW/2, p.Y+playerH, 3)
			g.play(SfxLand)
		}
	}
	if p.Sliding {
		p.airDashed = false
		g.emitDust(p.X+playerW/2+float64(p.wallDir*3), p.Y+playerH, 1)
	}

	p.dashT = math.Max(0, p.dashT-dt)
	p.dashCD = math.Max(0, p.dashCD-dt)
	p.lock = math.Max(0, p.lock-dt)
	p.coyote = math.Max(0, p.coyote-dt)
	p.jumpBuf = math.Max(0, p.jumpBuf-dt)
	p.shootCD = math.Max(0, p.shootCD-dt)
	p.HurtCD = math.Max(0, p.HurtCD-dt)
	p.muzzle = math.Max(0, p.muzzle-dt)
	p.landT = math.Max(0, p.landT-dt)
	p.anim += dt
	if p.OnGround && math.Abs(p.VX) <= 5 {
		p.idleT += dt
	} else {
		p.idleT = 0
	}
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
	p.lock = hurtLock
	g.shake = 2
	g.play(SfxHurt)
	g.emitBurst(p.X+playerW/2, p.Y+playerH/2, 8, []uint8{160, 196, 255}, 50, 200)
	if p.HP <= 0 {
		g.kill("CAPPY'S SUIT GAVE OUT")
	}
}

func (p *Player) sprite() *gfx.Sprite {
	var f gfx.Frames
	switch {
	case p.dashT > 0:
		return sprCappyDash.Facing(p.dashDir)
	case p.Sliding:
		return sprCappySlide.Facing(-p.wallDir) // face away from the wall
	case !p.OnGround && p.VY < 0:
		f = sprCappyJump
	case !p.OnGround: // ears fluttering on the way down
		if int(p.anim*8)%2 == 0 {
			f = sprCappyFall1
		} else {
			f = sprCappyFall2
		}
	case p.landT > 0:
		f = sprCappySlide // landing squash, knees bent
	case math.Abs(p.VX) > 5:
		f = sprCappyRun[int(p.anim*12)%len(sprCappyRun)]
	default: // idle: slow tail wag, ears settle, occasional blink
		switch {
		case math.Mod(p.idleT, 3.4) > 3.25:
			f = sprCappyBlink
		case int(p.idleT*1.6)%2 == 1:
			f = sprCappyIdle2
		default:
			f = sprCappyIdle1
		}
	}
	return f.Facing(p.Facing)
}

// Draw renders Cappy (blinking while invulnerable) plus the muzzle flash.
func (p *Player) Draw(c *gfx.Canvas, camX, camY int) {
	if p.HurtCD > 0 && int(p.HurtCD*12)%2 == 0 {
		return
	}
	// Sprites are centred on the hitbox and aligned at the feet, so the
	// wide dash pose still stands on the ground.
	spr := p.sprite()
	x := int(p.X) + playerW/2 - spr.W/2
	y := int(p.Y) + playerH - spr.H
	c.Blit(spr, x-camX, y-camY)
	if p.muzzle > 0 {
		mx, my := p.muzzlePos()
		c.Blit(sprMuzzle, int(mx)-1-camX, int(my)-1-camY)
	}
}
