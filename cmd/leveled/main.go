// leveled is the level editor for Cappy Lost In Space.
//
// It edits the layered level text file (solid / background decor /
// foreground decor) that the game embeds. Layers are edited one at a
// time, with a composite view that renders exactly what the game shows.
//
//	go run ./cmd/leveled            # edits internal/game/level1.txt
//	go run ./cmd/leveled -file p.txt
//
// Rebuild the game after saving: the level is embedded at compile time.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/AgustinBanchio/terminal-cappy/internal/game"
	"github.com/AgustinBanchio/terminal-cappy/internal/gfx"
)

func main() {
	file := flag.String("file", "internal/game/level1.txt", "level file to edit")
	flag.Parse()

	if err := run(*file); err != nil {
		fmt.Fprintln(os.Stderr, "leveled:", err)
		os.Exit(1)
	}
}

func run(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lvl, err := game.ParseLevel(data)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("cannot open terminal: %w", err)
	}
	if err := screen.Init(); err != nil {
		return fmt.Errorf("cannot init terminal: %w", err)
	}
	screen.EnableMouse()
	screen.HideCursor()

	defer func() {
		r := recover()
		screen.Fini()
		if r != nil {
			fmt.Fprintf(os.Stderr, "leveled crashed: %v\n%s", r, debug.Stack())
			os.Exit(1)
		}
	}()

	ed := &editor{
		screen: screen,
		lvl:    lvl,
		path:   path,
		bg:     game.NewBackground(),
	}
	cols, rows := screen.Size()
	ed.canvas = gfx.NewCanvas(cols, rows)
	return ed.run()
}

type editor struct {
	screen tcell.Screen
	canvas *gfx.Canvas
	lvl    *game.Level
	bg     *game.Background
	path   string

	curX, curY int // cursor, in tiles
	camX, camY int // camera, in pixels

	layer     int // game.LayerSolid / LayerBG / LayerFG
	sel       [game.LayerCount]int
	composite bool

	dirty     bool
	quitArmed bool
	status    string
	statusT   float64
	t         float64
}

func (ed *editor) run() error {
	events := make(chan tcell.Event, 64)
	go func() {
		for {
			ev := ed.screen.PollEvent()
			if ev == nil {
				return
			}
			events <- ev
		}
	}()

	const fps = 30
	tick := time.NewTicker(time.Second / fps)
	defer tick.Stop()

	for {
		select {
		case ev := <-events:
			if ed.handleEvent(ev) {
				return nil
			}
		case <-tick.C:
			ed.t += 1.0 / fps
			ed.statusT -= 1.0 / fps
			ed.draw()
		}
	}
}

func (ed *editor) handleEvent(ev tcell.Event) (quit bool) {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		cols, rows := ev.Size()
		ed.canvas = gfx.NewCanvas(cols, rows)
		ed.screen.Sync()

	case *tcell.EventMouse:
		cx, cy := ev.Position()
		tx := (cx + ed.camX) / game.TilePx
		ty := (cy*2 + ed.camY) / game.TilePx
		if ev.Buttons()&tcell.Button1 != 0 {
			ed.curX, ed.curY = tx, ty
			ed.paint(ed.selected())
		} else if ev.Buttons()&(tcell.Button2|tcell.Button3) != 0 {
			ed.curX, ed.curY = tx, ty
			ed.paint('.')
		}

	case *tcell.EventKey:
		ed.quitArmedTick(ev)
		switch ev.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			if ed.dirty && !ed.quitArmed {
				ed.quitArmed = true
				ed.say("UNSAVED CHANGES: Esc again to discard, Ctrl-S to save")
				return false
			}
			return true
		case tcell.KeyCtrlS:
			ed.save()
		case tcell.KeyLeft:
			ed.moveCursor(-1, 0, ev.Modifiers())
		case tcell.KeyRight:
			ed.moveCursor(1, 0, ev.Modifiers())
		case tcell.KeyUp:
			ed.moveCursor(0, -1, ev.Modifiers())
		case tcell.KeyDown:
			ed.moveCursor(0, 1, ev.Modifiers())
		case tcell.KeyTab:
			ed.layer = (ed.layer + 1) % game.LayerCount
		case tcell.KeyEnter:
			ed.paint(ed.selected())
		case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete:
			ed.paint('.')
		case tcell.KeyRune:
			switch ev.Rune() {
			case ' ':
				ed.paint(ed.selected())
			case 'x', 'X':
				ed.paint('.')
			case '1':
				ed.layer = game.LayerSolid
			case '2':
				ed.layer = game.LayerBG
			case '3':
				ed.layer = game.LayerFG
			case 'v', 'V', '4':
				ed.composite = !ed.composite
			case '[':
				ed.cycle(-1)
			case ']':
				ed.cycle(1)
			}
		}
	}
	return false
}

// quitArmedTick disarms the pending quit on any key other than Esc.
func (ed *editor) quitArmedTick(ev *tcell.EventKey) {
	if ev.Key() != tcell.KeyEscape && ev.Key() != tcell.KeyCtrlC {
		ed.quitArmed = false
	}
}

func (ed *editor) selected() byte {
	return game.Palette(ed.layer)[ed.sel[ed.layer]].Ch
}

func (ed *editor) cycle(d int) {
	n := len(game.Palette(ed.layer))
	ed.sel[ed.layer] = (ed.sel[ed.layer] + d + n) % n
}

func (ed *editor) paint(ch byte) {
	if ed.lvl.SetCell(ed.layer, ed.curX, ed.curY, ch) {
		ed.dirty = true
	}
}

func (ed *editor) moveCursor(dx, dy int, mods tcell.ModMask) {
	step := 1
	if mods&tcell.ModShift != 0 {
		step = 4
	}
	ed.curX = clamp(ed.curX+dx*step, 0, ed.lvl.W-1)
	ed.curY = clamp(ed.curY+dy*step, 0, ed.lvl.H-1)
}

func (ed *editor) save() {
	if err := os.WriteFile(ed.path, ed.lvl.Marshal(), 0o644); err != nil {
		ed.say("SAVE FAILED: " + err.Error())
		return
	}
	ed.dirty = false
	ed.say("saved " + ed.path + " (rebuild the game to embed it)")
}

func (ed *editor) say(msg string) {
	ed.status, ed.statusT = msg, 4
}

// --- rendering -----------------------------------------------------------

func (ed *editor) draw() {
	c := ed.canvas
	ed.followCursor()

	if ed.composite {
		// Exactly the game's draw order.
		ed.bg.Draw(c, ed.camX, ed.camY, ed.t)
		ed.lvl.DrawBackdrop(c, ed.camX, ed.camY, ed.t)
		ed.lvl.Draw(c, ed.camX, ed.camY)
		ed.lvl.DrawMarkers(c, ed.camX, ed.camY)
		ed.lvl.DrawForeground(c, ed.camX, ed.camY, ed.t)
	} else {
		c.Clear(233)
		ed.drawGrid()
		switch ed.layer {
		case game.LayerSolid:
			ed.lvl.Draw(c, ed.camX, ed.camY)
			ed.lvl.DrawMarkers(c, ed.camX, ed.camY)
		case game.LayerBG:
			ed.lvl.Draw(c, ed.camX, ed.camY)
			ed.lvl.DrawBackdrop(c, ed.camX, ed.camY, ed.t)
		case game.LayerFG:
			ed.lvl.Draw(c, ed.camX, ed.camY)
			ed.lvl.DrawForeground(c, ed.camX, ed.camY, ed.t)
		}
	}

	ed.drawCursor()
	c.Flush(ed.screen)
	ed.drawStatus()
	ed.screen.Show()
}

// followCursor keeps the cursor comfortably inside the viewport.
func (ed *editor) followCursor() {
	c := ed.canvas
	px, py := ed.curX*game.TilePx, ed.curY*game.TilePx
	const margin = 12
	if px < ed.camX+margin {
		ed.camX = px - margin
	}
	if px+game.TilePx > ed.camX+c.W-margin {
		ed.camX = px + game.TilePx - c.W + margin
	}
	if py < ed.camY+margin {
		ed.camY = py - margin
	}
	if py+game.TilePx > ed.camY+c.H-margin {
		ed.camY = py + game.TilePx - c.H + margin
	}
	ed.camX = clamp(ed.camX, 0, max(0, ed.lvl.PxW()-c.W))
	ed.camY = clamp(ed.camY, 0, max(0, ed.lvl.PxH()-c.H))
}

func (ed *editor) drawGrid() {
	c := ed.canvas
	for sy := 0; sy < c.H; sy++ {
		wy := sy + ed.camY
		if wy > ed.lvl.PxH() {
			continue
		}
		for sx := 0; sx < c.W; sx++ {
			wx := sx + ed.camX
			if wx > ed.lvl.PxW() {
				continue
			}
			if wx%game.TilePx == 0 && wy%game.TilePx == 0 {
				c.Set(sx, sy, 237)
			}
		}
	}
}

func (ed *editor) drawCursor() {
	col := uint8(226)
	if int(ed.t*3)%2 == 0 {
		col = 231
	}
	ed.canvas.Rect(ed.curX*game.TilePx-ed.camX, ed.curY*game.TilePx-ed.camY,
		game.TilePx, game.TilePx, col)
}

func (ed *editor) drawStatus() {
	mode := "layer: " + [...]string{"solid", "background", "foreground"}[ed.layer]
	if ed.composite {
		mode += " +full view"
	}
	dirty := ""
	if ed.dirty {
		dirty = " [modified]"
	}
	pal := game.Palette(ed.layer)
	opt := pal[ed.sel[ed.layer]]
	top := fmt.Sprintf(" %s%s   %s   tile: %c %s   cursor: %d,%d ",
		ed.path, dirty, mode, opt.Ch, opt.Name, ed.curX, ed.curY)
	ed.line(0, top, 231)

	// Palette strip with the selected tile highlighted.
	strip := " "
	for i, o := range pal {
		if i == ed.sel[ed.layer] {
			strip += fmt.Sprintf(">%c %s<  ", o.Ch, o.Name)
		} else {
			strip += fmt.Sprintf(" %c %s   ", o.Ch, o.Name)
		}
	}
	ed.line(1, strip, 250)

	help := " arrows/click move+paint  space paint  x erase  tab/123 layer  v full  [ ] tile  ctrl-s save  esc quit "
	if ed.statusT > 0 {
		help = " " + ed.status + " "
	}
	ed.line(ed.canvas.Rows()-1, help, 226)
}

func (ed *editor) line(row int, msg string, fg uint8) {
	st := tcell.StyleDefault.
		Foreground(tcell.PaletteColor(int(fg))).
		Background(tcell.PaletteColor(235))
	for x := 0; x < ed.canvas.W; x++ {
		r := ' '
		if x < len(msg) {
			r = rune(msg[x])
		}
		ed.screen.SetContent(x, row, r, nil, st)
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
