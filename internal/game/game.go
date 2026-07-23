package game

import (
	"fmt"
	"math"
	"math/rand"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"

	"github.com/AgustinBanchio/terminal-cappy/internal/gfx"
)

type State int

const (
	StateTitle State = iota
	StatePlaying
	StatePaused
	StateDead
	StateWon
)

type textCmd struct {
	x, y int // terminal cells
	msg  string
	fg   uint8
}

// Game owns the whole demo: world, entities, camera, and the tcell
// screen it renders to.
type Game struct {
	screen tcell.Screen
	canvas *gfx.Canvas
	in     input
	rng    *rand.Rand
	state  State

	level  *Level
	bg     *Background
	player *Player
	cam    Camera

	aliens    []*Alien
	bullets   []Bullet
	pickups   []*Pickup
	particles []Particle

	partsGot, partsTotal int

	time    float64
	shake   float64
	lift    float64 // ship liftoff progress after winning
	deadT   float64
	smokeT  float64
	deathBy string

	msg  string
	msgT float64

	texts []textCmd
}

func New(screen tcell.Screen) *Game {
	cols, rows := screen.Size()
	g := &Game{screen: screen, canvas: gfx.NewCanvas(cols, rows)}
	g.in = newInput()
	g.reset()
	g.state = StateTitle
	return g
}

// reset rebuilds the world and respawns everything.
func (g *Game) reset() {
	g.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	g.level = Build()
	g.bg = NewBackground()
	g.player = NewPlayer(g.level.SpawnX, g.level.SpawnY)

	g.aliens = g.aliens[:0]
	g.bullets = g.bullets[:0]
	g.pickups = g.pickups[:0]
	g.particles = g.particles[:0]
	g.partsGot = 0
	g.partsTotal = 0
	for _, s := range g.level.Spawns {
		switch s.Kind {
		case 'a':
			g.aliens = append(g.aliens, newAlien(alienWalker, s.X, s.Y))
		case 'f':
			g.aliens = append(g.aliens, newAlien(alienFlyer, s.X, s.Y))
		case 'P':
			g.pickups = append(g.pickups, &Pickup{Kind: pickupPart, X: s.X, Y: s.Y})
			g.partsTotal++
		}
	}

	g.time, g.shake, g.lift, g.deadT = 0, 0, 0, 0
	g.msg, g.msgT = "", 0
	g.state = StatePlaying
	g.cam.Center(g.player.X, g.player.Y,
		float64(g.canvas.W), float64(g.canvas.H),
		float64(g.level.PxW()), float64(g.level.PxH()))
	g.sayf("FIND %d SHIP PARTS TO FIX YOUR SHIP", 4, g.partsTotal)
}

// Run drives the fixed-timestep game loop until the player quits.
func (g *Game) Run(fps int) error {
	events := make(chan tcell.Event, 64)
	go func() {
		for {
			ev := g.screen.PollEvent()
			if ev == nil {
				return
			}
			events <- ev
		}
	}()

	dt := 1.0 / float64(fps)
	tick := time.NewTicker(time.Duration(float64(time.Second) / float64(fps)))
	defer tick.Stop()

	for {
		select {
		case ev := <-events:
			if g.handleEvent(ev) {
				return nil
			}
		case <-tick.C:
			g.step(dt)
			g.draw()
		}
	}
}

func (g *Game) handleEvent(ev tcell.Event) (quit bool) {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		cols, rows := ev.Size()
		g.canvas = gfx.NewCanvas(cols, rows)
		g.screen.Sync()
	case *tcell.EventKey:
		now := time.Now()
		switch ev.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			return true
		case tcell.KeyLeft:
			g.in.press(actLeft, now)
		case tcell.KeyRight:
			g.in.press(actRight, now)
		case tcell.KeyUp:
			g.in.press(actJump, now)
		case tcell.KeyRune:
			switch unicode.ToLower(ev.Rune()) {
			case 'z', 'w', ' ':
				g.in.press(actJump, now)
			case 'x', 'k':
				g.in.press(actShoot, now)
			case 'a':
				g.in.press(actLeft, now)
			case 'd':
				g.in.press(actRight, now)
			case 'r':
				g.reset()
			case 'p':
				if g.state == StatePlaying {
					g.state = StatePaused
				} else if g.state == StatePaused {
					g.state = StatePlaying
				}
			}
		}
		// The title screen starts on any key that is not a quit key.
		if g.state == StateTitle {
			g.state = StatePlaying
			g.in.endFrame()
			g.sayf("FIND %d SHIP PARTS TO FIX YOUR SHIP", 4, g.partsTotal)
		}
	}
	return false
}

func (g *Game) step(dt float64) {
	g.time += dt
	switch g.state {
	case StatePlaying:
		g.updateWorld(dt)
	case StateTitle:
		// Keep the crash site smouldering behind the logo.
		g.crashSmoke(dt)
		g.updateParticles(dt)
	case StateDead:
		g.deadT += dt
		g.updateParticles(dt)
	case StateWon:
		g.lift += dt
		off := g.liftOffset()
		g.emitFlame(float64(g.level.ShipX+22), float64(g.level.ShipY+10-off))
		g.emitFlame(float64(g.level.ShipX+8), float64(g.level.ShipY+12-off))
		g.updateParticles(dt)
	}
	g.msgT = math.Max(0, g.msgT-dt)
	g.shake = math.Max(0, g.shake-6*dt)
	g.in.endFrame()
}

func (g *Game) updateWorld(dt float64) {
	now := time.Now()
	g.player.Update(g, dt, now)

	for _, a := range g.aliens {
		if a.HP > 0 {
			a.update(g, dt)
		}
	}
	alive := g.aliens[:0]
	for _, a := range g.aliens {
		if a.HP > 0 {
			alive = append(alive, a)
		}
	}
	g.aliens = alive

	g.updateBullets(dt)

	kept := g.pickups[:0]
	for _, pk := range g.pickups {
		if pk.update(g, dt) {
			kept = append(kept, pk)
		}
	}
	g.pickups = kept

	g.crashSmoke(dt)
	g.updateParticles(dt)

	p := g.player
	g.cam.Update(p.X, p.Y, playerW, playerH,
		float64(g.canvas.W), float64(g.canvas.H),
		float64(g.level.PxW()), float64(g.level.PxH()), dt)

	// Falling out of the world is fatal.
	if p.Y > float64(g.level.PxH())+20 {
		g.kill("CAPPY FELL INTO THE VOID")
	}

	// Win: all parts collected and back at the ship.
	if g.state == StatePlaying && g.partsGot == g.partsTotal &&
		aabb(p.X, p.Y, playerW, playerH,
			float64(g.level.ShipX-4), float64(g.level.ShipY-4), float64(sprShip.W+8), float64(sprShip.H+8)) {
		g.state = StateWon
		g.lift = 0
	}
}

// crashSmoke keeps the damaged ship smouldering until it is repaired.
func (g *Game) crashSmoke(dt float64) {
	g.smokeT += dt
	if g.smokeT > 0.15 {
		g.smokeT = 0
		g.emitSmoke(float64(g.level.ShipX+15), float64(g.level.ShipY+6))
	}
}

func (g *Game) kill(reason string) {
	if g.state != StatePlaying {
		return
	}
	g.state = StateDead
	g.deadT = 0
	g.deathBy = reason
	g.shake = 3
	p := g.player
	g.emitBurst(p.X+playerW/2, p.Y+playerH/2, 20, []uint8{255, 160, 152, 240}, 70, 120)
}

func (g *Game) liftOffset() int {
	t := math.Max(0, g.lift-0.4)
	return int(t * t * 14)
}

func (g *Game) say(msg string, secs float64) {
	g.msg, g.msgT = msg, secs
}

func (g *Game) sayf(format string, secs float64, args ...any) {
	g.say(fmt.Sprintf(format, args...), secs)
}

// --- rendering ----------------------------------------------------------

func (g *Game) draw() {
	c := g.canvas
	g.texts = g.texts[:0]

	if c.W < 40 || c.H < 24 {
		c.Clear(16)
		g.text((c.W-18)/2, c.Rows()/2, "TERMINAL TOO SMALL", 196)
		g.flush()
		return
	}

	sx, sy := 0, 0
	if g.shake > 0 {
		sx = g.rng.Intn(3) - 1
		sy = g.rng.Intn(3) - 1
	}
	camX, camY := int(g.cam.X)+sx, int(g.cam.Y)+sy

	g.bg.Draw(c, camX, camY, g.time)
	g.level.DrawBackdrop(c, camX, camY)
	g.level.Draw(c, camX, camY)

	c.Blit(sprShip, g.level.ShipX-camX, g.level.ShipY-g.liftOffset()-camY)

	for _, pk := range g.pickups {
		pk.draw(c, camX, camY)
	}
	for _, a := range g.aliens {
		a.draw(c, camX, camY)
	}
	if g.state != StateDead && g.state != StateWon {
		g.player.Draw(c, camX, camY)
	}
	for _, b := range g.bullets {
		drawBullet(c, b, camX, camY)
	}
	for _, p := range g.particles {
		drawParticle(c, p, camX, camY)
	}
	g.level.DrawForeground(c, camX, camY, g.time)

	switch g.state {
	case StateTitle:
		g.drawTitle()
	case StatePlaying:
		g.drawDialogue()
		g.drawHUD()
	case StatePaused:
		g.drawHUD()
		g.panel([]string{"PAUSED", "press P to resume"}, 220)
	case StateDead:
		g.drawHUD()
		if g.deadT > 0.8 {
			g.panel([]string{
				"CAPPY WAS LOST IN SPACE...",
				g.deathBy,
				"",
				"press R to try again",
			}, 196)
		}
	case StateWon:
		g.drawHUD()
		if g.lift > 2.2 {
			g.panel([]string{
				"SHIP REPAIRED!",
				"CAPPY ESCAPES THE PLANET",
				"",
				"press R to play again, ESC to quit",
			}, 220)
		}
	}

	g.flush()
}

func (g *Game) flush() {
	g.canvas.Flush(g.screen)
	for _, t := range g.texts {
		for i, r := range t.msg {
			x := t.x + i
			if x < 0 || x >= g.canvas.W || t.y < 0 || t.y >= g.canvas.Rows() {
				continue
			}
			bg := g.canvas.At(x, t.y*2)
			st := tcell.StyleDefault.
				Foreground(tcell.PaletteColor(int(t.fg))).
				Background(tcell.PaletteColor(int(bg)))
			g.screen.SetContent(x, t.y, r, nil, st)
		}
	}
	g.screen.Show()
}

// text queues a terminal-cell text overlay drawn after the pixel flush.
func (g *Game) text(x, y int, msg string, fg uint8) {
	g.texts = append(g.texts, textCmd{x: x, y: y, msg: msg, fg: fg})
}

func (g *Game) textCentered(y int, msg string, fg uint8) {
	g.text((g.canvas.W-len(msg))/2, y, msg, fg)
}

func (g *Game) drawHUD() {
	c := g.canvas
	for i := 0; i < maxHP; i++ {
		spr := sprHeartEmpty
		if i < g.player.HP {
			spr = sprHeart
		}
		c.Blit(spr, 2+i*6, 2)
	}
	parts := fmt.Sprintf("PARTS %d/%d", g.partsGot, g.partsTotal)
	g.text(c.W-len(parts)-1, 0, parts, 220)
	if g.msgT > 0 {
		g.textCentered(2, g.msg, 231)
	}
}

// drawDialogue shows a speech box with Cappy's avatar while she stands
// near the crashed ship.
func (g *Game) drawDialogue() {
	shipCX := float64(g.level.ShipX + sprShip.W/2)
	px := g.player.X + playerW/2
	if math.Abs(px-shipCX) > 46 {
		return
	}
	var lines []string
	if g.partsGot < g.partsTotal {
		lines = []string{
			"I NEED TO FIX MY SHIP.",
			fmt.Sprintf("%d PARTS STILL OUT THERE...", g.partsTotal-g.partsGot),
		}
	} else {
		lines = []string{"THAT'S ALL OF THEM!", "TIME TO GO HOME."}
	}

	c := g.canvas
	maxLen := 0
	for _, s := range lines {
		if len(s) > maxLen {
			maxLen = len(s)
		}
	}
	w := 2 + sprPortrait.W + 2 + maxLen + 2
	const hRows = 8
	x := (c.W - w) / 2
	yRow := c.Rows() - hRows - 1

	c.FillRect(x, yRow*2, w, hRows*2, 16)
	c.Rect(x, yRow*2, w, hRows*2, 96)
	c.Blit(sprPortrait, x+2, yRow*2+2)
	for i, s := range lines {
		g.text(x+2+sprPortrait.W+2, yRow+3+i, s, 250)
	}
}

// panel draws a bordered dialog box with centred lines of cell text.
func (g *Game) panel(lines []string, accent uint8) {
	c := g.canvas
	w := 0
	for _, s := range lines {
		if len(s) > w {
			w = len(s)
		}
	}
	w += 6
	h := len(lines) + 2
	x := (c.W - w) / 2
	y := (c.Rows()-h)/2 - 1

	c.FillRect(x, y*2, w, h*2, 16)
	c.Rect(x, y*2, w, h*2, 96)
	for i, s := range lines {
		fg := uint8(250)
		if i == 0 {
			fg = accent
		}
		g.textCentered(y+1+i, s, fg)
	}
}

// drawTitle renders the landing screen: pixel-font logo over the crash
// site scene, plus a blinking prompt and the controls.
func (g *Game) drawTitle() {
	c := g.canvas

	scale := 3
	if gfx.TextPxWidth("CAPPY", 3) > c.W-8 {
		scale = 2
	}
	title := "CAPPY"
	sub := "LOST IN SPACE"

	ty := c.H / 8
	c.DrawTextPx((c.W-gfx.TextPxWidth(title, scale))/2, ty, title, scale, 220, 130)
	sy := ty + gfx.TextPxHeight(scale) + 4
	c.DrawTextPx((c.W-gfx.TextPxWidth(sub, 1))/2, sy, sub, 1, 217, 53)

	rows := c.Rows()
	if int(g.time*2)%2 == 0 {
		g.textCentered(rows-5, "PRESS ANY KEY TO CONTINUE", 231)
	}
	g.textCentered(rows-3, "arrows/AD move   Z jump   X shoot", 245)
	g.textCentered(rows-2, "hold into a wall to slide, Z to wall jump", 245)
	g.text(1, rows-1, "retro demo", 240)
	g.text(c.W-13, rows-1, "ESC to quit", 240)
}
