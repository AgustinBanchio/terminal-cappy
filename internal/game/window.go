//go:build window

package game

// Window mode: the same game rendered in a desktop window via
// Ebitengine instead of a terminal. Real key press/release events are
// available here, so the input layer runs in direct mode: no key-repeat
// bridging, frame-accurate taps, and variable jump height (release to
// cut the ascent), the parts a terminal fundamentally cannot do.
//
// Each canvas pixel becomes a scale x scale square, so the game looks
// exactly like the terminal version, just crisper. Resizing the window
// grows the canvas (more world on screen), like resizing a terminal.

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"

	"github.com/AgustinBanchio/terminal-cappy/internal/gfx"
)

// RunWindow opens the game in a desktop window. scale is the size of
// one canvas pixel in window pixels.
func RunWindow(scale int) error {
	if scale < 6 {
		scale = 6
	}
	w := &windowRunner{
		g:     newGame(100, 30),
		scale: scale,
		face:  textv2.NewGoXFace(basicfont.Face7x13),
	}
	w.g.in.direct = true
	w.g.audio = newSfxBank()
	ebiten.SetWindowSize(100*scale, 30*2*scale)
	ebiten.SetWindowTitle("Cappy Lost In Space")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return ebiten.RunGame(w)
}

type windowRunner struct {
	g     *Game
	scale int
	face  *textv2.GoXFace
	frame *ebiten.Image
	pix   []byte
	keys  []ebiten.Key
}

// Layout maps the window size to a cell grid, growing the canvas like
// a terminal resize would.
func (w *windowRunner) Layout(outW, outH int) (int, int) {
	cols := max(40, outW/w.scale)
	rows := max(12, outH/(2*w.scale))
	if cols != w.g.canvas.W || rows != w.g.canvas.Rows() {
		w.g.canvas = gfx.NewCanvas(cols, rows)
		w.frame = nil
	}
	return cols * w.scale, rows * 2 * w.scale
}

func (w *windowRunner) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	g := w.g
	now := time.Now()

	bind := func(a action, keys ...ebiten.Key) {
		held, just, released := false, false, false
		for _, k := range keys {
			held = held || ebiten.IsKeyPressed(k)
			just = just || inpututil.IsKeyJustPressed(k)
			released = released || inpututil.IsKeyJustReleased(k)
		}
		g.in.setDirect(a, held, just, released, now)
	}
	bind(actLeft, ebiten.KeyArrowLeft, ebiten.KeyA)
	bind(actRight, ebiten.KeyArrowRight, ebiten.KeyD)
	bind(actJump, ebiten.KeyZ, ebiten.KeyW, ebiten.KeySpace, ebiten.KeyArrowUp)
	bind(actShoot, ebiten.KeyX, ebiten.KeyK)
	bind(actDash, ebiten.KeyC)

	if inpututil.IsKeyJustPressed(ebiten.KeyComma) {
		g.nudge(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) {
		g.nudge(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.reset()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		switch g.state {
		case StatePlaying:
			g.state = StatePaused
		case StatePaused:
			g.state = StatePlaying
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) || inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if g.state == StatePlaying || g.state == StatePaused {
			g.showMap = !g.showMap
		}
	}
	if g.state == StateTitle {
		w.keys = inpututil.AppendJustPressedKeys(w.keys[:0])
		if len(w.keys) > 0 {
			g.startFromTitle()
		}
	}

	g.step(1.0 / 60.0)
	return nil
}

func (w *windowRunner) Draw(screen *ebiten.Image) {
	g := w.g
	g.draw() // fills the canvas and the text overlay queue

	c := g.canvas
	if w.frame == nil {
		w.frame = ebiten.NewImage(c.W, c.H)
		w.pix = make([]byte, c.W*c.H*4)
	}
	i := 0
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			r, gr, b := gfx.PaletteRGB(c.At(x, y))
			w.pix[i], w.pix[i+1], w.pix[i+2], w.pix[i+3] = r, gr, b, 255
			i += 4
		}
	}
	w.frame.WritePixels(w.pix)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(w.scale), float64(w.scale))
	screen.DrawImage(w.frame, op)

	// Cell-text overlays, one glyph per cell so alignment matches the
	// terminal exactly.
	cellH := 2 * w.scale
	for _, t := range g.texts {
		r, gr, b := gfx.PaletteRGB(t.fg)
		fg := color.RGBA{r, gr, b, 255}
		for i, ch := range t.msg {
			px := float64((t.x + i) * w.scale)
			py := float64(t.y*cellH + (cellH-13)/2)
			w.drawGlyph(screen, string(ch), px+1, py+1, color.RGBA{0, 0, 0, 255})
			w.drawGlyph(screen, string(ch), px, py, fg)
		}
	}
}

func (w *windowRunner) drawGlyph(dst *ebiten.Image, s string, x, y float64, col color.RGBA) {
	op := &textv2.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(col)
	textv2.Draw(dst, s, w.face, op)
}
