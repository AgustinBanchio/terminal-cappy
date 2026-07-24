package game

import (
	"fmt"
	"math"
	"math/rand"
	"runtime/debug"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"

	"github.com/AgustinBanchio/terminal-cappy/internal/gfx"
)

// Version is the fallback shown on the title screen for local builds.
// Releases are tagged to match; module-installed builds (go run/install
// @vX.Y.Z) display the exact version stamped by the toolchain instead.
const Version = "v0.1.2"

func displayVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok &&
		bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return Version
}

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
	bosses    []*Boss
	bullets   []Bullet
	shots     []Shot
	pickups   []*Pickup
	particles []Particle

	activeBoss *Boss
	bossTitleT float64

	// Exploration map: tiles that have been on screen. The map overlay
	// only reveals what Cappy has actually seen.
	seen    []bool
	showMap bool

	partsGot, partsTotal int
	partSeq              int // next ship-part sprite variant to hand out

	// Weather (surface zone): rain comes and goes; lightning whites
	// out the whole screen for a moment.
	raining bool
	rainT   float64
	boltT   float64
	flashT  float64
	emberT  float64

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

// New builds a game rendering to a terminal screen.
func New(screen tcell.Screen) *Game {
	cols, rows := screen.Size()
	g := newGame(cols, rows)
	g.screen = screen
	return g
}

// newGame builds a game with no output backend; the window runner (and
// tests) drive step/draw themselves and read the canvas.
func newGame(cols, rows int) *Game {
	g := &Game{canvas: gfx.NewCanvas(cols, rows)}
	g.in = newInput()
	g.reset()
	g.state = StateTitle
	return g
}

// reset rebuilds the world and respawns everything.
func (g *Game) reset() {
	g.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	g.level = LoadDefault()
	g.bg = NewBackground()
	g.player = NewPlayer(g.level.SpawnX, g.level.SpawnY)

	g.aliens = g.aliens[:0]
	g.bosses = g.bosses[:0]
	g.bullets = g.bullets[:0]
	g.shots = g.shots[:0]
	g.pickups = g.pickups[:0]
	g.particles = g.particles[:0]
	g.activeBoss = nil
	g.bossTitleT = 0
	g.partsGot = 0
	g.partsTotal = 0
	g.partSeq = 0
	for _, s := range g.level.Spawns {
		switch s.Kind {
		case 'a':
			g.aliens = append(g.aliens, newAlien(alienWalker, s.X, s.Y))
		case 'f':
			g.aliens = append(g.aliens, newAlien(alienFlyer, s.X, s.Y))
		case 'b':
			g.aliens = append(g.aliens, newAlien(alienBat, s.X, s.Y))
		case 'u':
			g.aliens = append(g.aliens, newAlien(alienLurker, s.X, s.Y))
		case 'z':
			g.aliens = append(g.aliens, newAlien(alienShard, s.X, s.Y))
		case 'e':
			g.aliens = append(g.aliens, newAlien(alienMagling, s.X, s.Y))
		case 'P':
			g.pickups = append(g.pickups, &Pickup{Kind: pickupPart, Variant: g.nextPartVariant(), X: s.X, Y: s.Y})
			g.partsTotal++
		case 'D', 'Q', 'M':
			g.bosses = append(g.bosses, newBoss(s.Kind, s.X, s.Y, g.level))
			g.partsTotal++ // each boss guards a part
		}
	}

	g.time, g.shake, g.lift, g.deadT = 0, 0, 0, 0
	g.raining, g.rainT, g.boltT, g.flashT = false, 14, 0, 0
	g.seen = make([]bool, g.level.W*g.level.H)
	g.showMap = false
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
			if ev.Modifiers()&tcell.ModShift != 0 {
				g.nudge(-1)
			} else {
				g.in.press(actLeft, now)
			}
		case tcell.KeyRight:
			if ev.Modifiers()&tcell.ModShift != 0 {
				g.nudge(1)
			} else {
				g.in.press(actRight, now)
			}
		case tcell.KeyUp:
			g.in.press(actJump, now)
		case tcell.KeyRune:
			switch unicode.ToLower(ev.Rune()) {
			case 'z', 'w', ' ':
				g.in.press(actJump, now)
			case 'x', 'k':
				g.in.press(actShoot, now)
			case 'c':
				g.in.press(actDash, now)
			case 'a':
				g.in.press(actLeft, now)
			case 'd':
				g.in.press(actRight, now)
			case ',':
				g.nudge(-1)
			case '.':
				g.nudge(1)
			case 'r':
				g.reset()
			case 'p':
				if g.state == StatePlaying {
					g.state = StatePaused
				} else if g.state == StatePaused {
					g.state = StatePlaying
				}
			case 'm':
				if g.state == StatePlaying || g.state == StatePaused {
					g.showMap = !g.showMap
				}
			}
		case tcell.KeyTab:
			if g.state == StatePlaying || g.state == StatePaused {
				g.showMap = !g.showMap
			}
		}
		// The title screen starts on any key that is not a quit key.
		if g.state == StateTitle {
			g.startFromTitle()
		}
	}
	return false
}

func (g *Game) startFromTitle() {
	g.state = StatePlaying
	g.in.endFrame()
	g.sayf("FIND %d SHIP PARTS TO FIX YOUR SHIP", 4, g.partsTotal)
}

func (g *Game) step(dt float64) {
	g.time += dt
	switch g.state {
	case StatePlaying:
		if g.showMap {
			break // studying the map freezes the world, like pausing
		}
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

	for _, b := range g.bosses {
		b.update(g, dt)
	}
	g.bossTitleT = math.Max(0, g.bossTitleT-dt)

	g.updateBullets(dt)
	g.updateShots(dt)
	g.updateWeather(dt)

	// Lava burns: damage plus a hard bounce out of the pool.
	p := g.player
	if g.level.LavaAtPx(p.X+playerW/2, p.Y+playerH-1) {
		p.Hurt(g, -float64(p.Facing))
		p.VY = -140
	}

	kept := g.pickups[:0]
	for _, pk := range g.pickups {
		if pk.update(g, dt) {
			kept = append(kept, pk)
		}
	}
	g.pickups = kept

	g.crashSmoke(dt)
	g.updateParticles(dt)
	g.markSeen()

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

func (g *Game) playerZone() byte {
	p := g.player
	return g.level.Zone(fdiv(int(p.X+playerW/2), TilePx), fdiv(int(p.Y+playerH/2), TilePx))
}

// zoneColumns samples the ambience zone per screen column (at the
// camera's vertical centre), which drives the parallax backdrop.
func (g *Game) zoneColumns(camX, camY int) func(int) byte {
	l := g.level
	cy := fdiv(camY+g.canvas.H/2, TilePx)
	return func(sx int) byte { return l.Zone(fdiv(sx+camX, TilePx), cy) }
}

// updateWeather runs the surface storm cycle and per-zone ambience.
func (g *Game) updateWeather(dt float64) {
	g.flashT = math.Max(0, g.flashT-dt)
	g.rainT -= dt
	if g.rainT <= 0 {
		g.raining = !g.raining
		if g.raining {
			g.rainT = 12 + g.rng.Float64()*14
			g.boltT = 3 + g.rng.Float64()*5
		} else {
			g.rainT = 10 + g.rng.Float64()*15
		}
	}
	zone := g.playerZone()
	if g.raining && zone == 's' {
		g.boltT -= dt
		if g.boltT <= 0 {
			g.boltT = 4 + g.rng.Float64()*6
			g.flashT = 0.13 // lightning: whole-screen white flash
			g.shake = math.Max(g.shake, 1.5)
		}
	}

	g.emberT -= dt
	if g.emberT > 0 {
		return
	}
	switch zone {
	case 'l': // embers rise from the deep fire
		g.emberT = 0.09
		life := 1 + g.rng.Float64()
		g.particles = append(g.particles, Particle{
			X:    g.cam.X + g.rng.Float64()*float64(g.canvas.W),
			Y:    g.cam.Y + float64(g.canvas.H) - g.rng.Float64()*8,
			VX:   (g.rng.Float64() - 0.5) * 8,
			VY:   -14 - g.rng.Float64()*14,
			Life: life, Max: life, Grav: -4,
			Colors: []uint8{208, 202, 130},
		})
	case 'k': // drifting glitter in the crystal caves
		g.emberT = 0.28
		life := 0.5 + g.rng.Float64()*0.5
		g.particles = append(g.particles, Particle{
			X:    g.cam.X + g.rng.Float64()*float64(g.canvas.W),
			Y:    g.cam.Y + g.rng.Float64()*float64(g.canvas.H),
			VY:   4,
			Life: life, Max: life,
			Colors: []uint8{231, 51, 183},
		})
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

// nextPartVariant deals out ship-part sprite shapes so every part in a
// run looks like a different piece of the rocket.
func (g *Game) nextPartVariant() int {
	v := g.partSeq
	g.partSeq++
	return v
}

// nudge queues a 1px micro-step. Unlike held movement, each keypress
// maps to exactly one pixel, so precision never depends on key-repeat
// timing; holding the key gives a slow creep via auto-repeat.
func (g *Game) nudge(dir int) {
	if g.state == StatePlaying {
		g.player.NudgeX += dir
	}
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

	zone := g.zoneColumns(camX, camY)
	g.bg.Draw(c, camX, camY, g.time, zone, g.raining)
	g.level.DrawBackdrop(c, camX, camY, g.time)
	g.level.Draw(c, camX, camY, g.time)

	c.Blit(sprShip, g.level.ShipX-camX, g.level.ShipY-g.liftOffset()-camY)

	for _, pk := range g.pickups {
		pk.draw(c, camX, camY)
	}
	for _, a := range g.aliens {
		a.draw(c, camX, camY)
	}
	for _, b := range g.bosses {
		b.draw(c, camX, camY, g.time)
	}
	if g.state != StateDead && g.state != StateWon {
		g.player.Draw(c, camX, camY)
	}
	for _, b := range g.bullets {
		drawBullet(c, b, camX, camY)
	}
	for _, s := range g.shots {
		drawShot(c, s, camX, camY)
	}
	for _, p := range g.particles {
		drawParticle(c, p, camX, camY)
	}
	g.level.DrawForeground(c, camX, camY, g.time)
	if g.raining {
		g.drawRain(zone, camX, camY)
	}
	if g.flashT > 0.06 {
		c.Clear(231) // lightning whiteout
	}

	switch g.state {
	case StateTitle:
		g.drawTitle()
	case StatePlaying:
		g.drawDialogue()
		g.drawHUD()
		g.drawBossHUD()
		if g.showMap {
			g.drawMap()
		}
	case StatePaused:
		g.drawHUD()
		if g.showMap {
			g.drawMap()
		} else {
			g.panel([]string{"PAUSED", "press P to resume"}, 220)
		}
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
	if g.screen == nil {
		return // window mode: the runner reads canvas + texts itself
	}
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

// drawRain streaks world-locked rain down surface-zone columns, in
// front of the scenery. The streak pattern lives at (world y - phase),
// so on screen it moves downward as the phase grows; the modulo must
// stay positive once the phase overtakes the world coordinate.
func (g *Game) drawRain(zone func(int) byte, camX, camY int) {
	c := g.canvas
	ph := int(g.time * 130)
	for sx := 0; sx < c.W; sx++ {
		if zone(sx) != 's' {
			continue
		}
		wx := sx + camX
		for sy := 0; sy < c.H; sy++ {
			wy := sy + camY - ph
			if ((wy%12)+12)%12 < 5 && hash2(wx, fdiv(wy, 12))%17 == 0 {
				c.Set(sx, sy, 67)
			}
		}
	}
}

// markSeen records every tile currently on screen for the map overlay.
func (g *Game) markSeen() {
	l := g.level
	tx0 := clampInt(fdiv(int(g.cam.X), TilePx), 0, l.W-1)
	tx1 := clampInt(fdiv(int(g.cam.X)+g.canvas.W, TilePx), 0, l.W-1)
	ty0 := clampInt(fdiv(int(g.cam.Y), TilePx), 0, l.H-1)
	ty1 := clampInt(fdiv(int(g.cam.Y)+g.canvas.H, TilePx), 0, l.H-1)
	for ty := ty0; ty <= ty1; ty++ {
		for tx := tx0; tx <= tx1; tx++ {
			g.seen[ty*l.W+tx] = true
		}
	}
}

// drawMap renders the exploration map overlay: one map pixel per block
// of tiles, showing only blocks Cappy has had on screen. Unexplored
// regions stay dark.
func (g *Game) drawMap() {
	c := g.canvas
	l := g.level

	scale := 1
	for l.W/scale > c.W-8 || l.H/scale > c.H-14 {
		scale++
	}
	mw, mh := (l.W+scale-1)/scale, (l.H+scale-1)/scale
	x0, y0 := (c.W-mw)/2, (c.H-mh)/2

	c.FillRect(x0-3, y0-5, mw+6, mh+9, 16)
	c.Rect(x0-3, y0-5, mw+6, mh+9, 96)

	for my := 0; my < mh; my++ {
		for mx := 0; mx < mw; mx++ {
			c.Set(x0+mx, y0+my, g.mapBlockColor(mx, my, scale))
		}
	}

	// Live markers on explored ground: uncollected parts and the ship.
	blockSeen := func(tx, ty int) bool {
		return g.seen[clampInt(ty, 0, l.H-1)*l.W+clampInt(tx, 0, l.W-1)]
	}
	for _, pk := range g.pickups {
		tx, ty := int(pk.X)/TilePx, int(pk.Y)/TilePx
		if pk.Kind == pickupPart && blockSeen(tx, ty) {
			c.Set(x0+tx/scale, y0+ty/scale, 220)
		}
	}
	if blockSeen(l.ShipX/TilePx+3, l.ShipY/TilePx) {
		c.Set(x0+(l.ShipX/TilePx+3)/scale, y0+(l.ShipY/TilePx)/scale, 160)
	}
	if int(g.time*3)%2 == 0 { // Cappy, blinking
		px, py := int(g.player.X+playerW/2)/TilePx, int(g.player.Y+playerH/2)/TilePx
		c.Set(x0+px/scale, y0+py/scale, 231)
	}

	row := (y0 - 5) / 2
	g.textCentered(row+1, "MAP", 220)
	g.textCentered((y0+mh+3)/2, "M/TAB close   white: you   yellow: parts", 245)
}

// mapBlockColor summarises one scale x scale block of tiles.
func (g *Game) mapBlockColor(mx, my, scale int) uint8 {
	l := g.level
	seen := false
	var solid byte
	lava := false
	zone := byte('s')
	for ty := my * scale; ty < (my+1)*scale && ty < l.H; ty++ {
		for tx := mx * scale; tx < (mx+1)*scale && tx < l.W; tx++ {
			if !g.seen[ty*l.W+tx] {
				continue
			}
			seen = true
			switch ch := l.Cell(LayerSolid, tx, ty); ch {
			case '#', '%', 'X':
				if solid == 0 {
					solid = ch
				}
			case '~':
				lava = true
			}
			zone = l.Zone(tx, ty)
		}
	}
	switch {
	case !seen:
		return 233 // unexplored: fog
	case lava:
		return 202
	case solid == '%':
		return 101
	case solid == 'X':
		return 61
	case solid == '#':
		return 95
	default: // open air, tinted by region
		switch zone {
		case 'u':
			return 237
		case 'k':
			return 24
		case 'l':
			return 52
		default:
			return 238
		}
	}
}

// drawBossHUD renders the boss health bar and, on fight start, the big
// name card.
func (g *Game) drawBossHUD() {
	b := g.activeBoss
	if b == nil || b.Dead {
		return
	}
	c := g.canvas

	w := c.W / 2
	x := (c.W - w) / 2
	c.FillRect(x-1, 7, w+2, 5, 16)
	c.Rect(x-1, 7, w+2, 5, 88)
	fill := int(float64(w) * float64(b.HP) / float64(b.info.hp))
	c.FillRect(x, 8, fill, 3, 196)
	g.textCentered(2, b.info.name, 210)

	if g.bossTitleT > 0 {
		scale := 2
		tw := gfx.TextPxWidth(b.info.name, scale)
		c.DrawTextPx((c.W-tw)/2, c.H/3, b.info.name, scale, 196, 52)
		g.textCentered(c.Rows()/3+gfx.TextPxHeight(scale)/2+2, b.info.sub, 250)
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
	g.textCentered(rows-3, "arrows/AD move  Z jump  X shoot  C dash  ,/. step  M map", 245)
	g.textCentered(rows-2, "hold into a wall to slide, Z to wall jump", 245)
	g.text(1, rows-1, "retro demo "+displayVersion(), 240)
	g.text(c.W-13, rows-1, "ESC to quit", 240)
}
