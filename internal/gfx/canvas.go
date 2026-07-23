// Package gfx implements a tiny retro renderer on top of tcell.
//
// The screen is treated as a pixel canvas using unicode half blocks:
// every terminal cell shows two vertically stacked "pixels" by drawing
// U+2580 (upper half block) with the foreground colour as the top pixel
// and the background colour as the bottom pixel. A Cols x Rows terminal
// therefore gives a Cols x (2*Rows) canvas in 256 colours, which works
// on the default terminals of Windows, macOS and Linux.
package gfx

import "github.com/gdamore/tcell/v2"

// Canvas is a W x H buffer of 256-colour palette indices.
// H is always twice the number of terminal rows it flushes to.
type Canvas struct {
	W, H int
	pix  []uint8
}

// NewCanvas builds a canvas for a terminal of cols x rows cells.
func NewCanvas(cols, rows int) *Canvas {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return &Canvas{W: cols, H: rows * 2, pix: make([]uint8, cols*rows*2)}
}

// Rows returns the terminal row count this canvas flushes to.
func (c *Canvas) Rows() int { return c.H / 2 }

// Clear fills the whole canvas with one colour.
func (c *Canvas) Clear(color uint8) {
	for i := range c.pix {
		c.pix[i] = color
	}
}

// Set writes one pixel, ignoring out-of-bounds coordinates.
func (c *Canvas) Set(x, y int, color uint8) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.pix[y*c.W+x] = color
}

// At reads one pixel; out-of-bounds returns 0 (palette black).
func (c *Canvas) At(x, y int) uint8 {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return 0
	}
	return c.pix[y*c.W+x]
}

// FillRect fills an axis-aligned rectangle, clipped to the canvas.
func (c *Canvas) FillRect(x, y, w, h int, color uint8) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			c.Set(px, py, color)
		}
	}
}

// Rect draws a 1px rectangle outline.
func (c *Canvas) Rect(x, y, w, h int, color uint8) {
	for px := x; px < x+w; px++ {
		c.Set(px, y, color)
		c.Set(px, y+h-1, color)
	}
	for py := y; py < y+h; py++ {
		c.Set(x, py, color)
		c.Set(x+w-1, py, color)
	}
}

// Blit draws a sprite with transparency at (x, y).
func (c *Canvas) Blit(s *Sprite, x, y int) {
	for sy := 0; sy < s.H; sy++ {
		for sx := 0; sx < s.W; sx++ {
			if col := s.Pix[sy*s.W+sx]; col >= 0 {
				c.Set(x+sx, y+sy, uint8(col))
			}
		}
	}
}

// BlitTinted draws every opaque sprite pixel in a single colour.
// Used for hit flashes.
func (c *Canvas) BlitTinted(s *Sprite, x, y int, tint uint8) {
	for sy := 0; sy < s.H; sy++ {
		for sx := 0; sx < s.W; sx++ {
			if s.Pix[sy*s.W+sx] >= 0 {
				c.Set(x+sx, y+sy, tint)
			}
		}
	}
}

// Flush pushes the pixel buffer to the terminal. tcell diffs cells
// internally, so only changed cells hit the wire.
func (c *Canvas) Flush(s tcell.Screen) {
	rows := c.Rows()
	for row := 0; row < rows; row++ {
		for x := 0; x < c.W; x++ {
			top := c.pix[(row*2)*c.W+x]
			bot := c.pix[(row*2+1)*c.W+x]
			st := tcell.StyleDefault.
				Foreground(tcell.PaletteColor(int(top))).
				Background(tcell.PaletteColor(int(bot)))
			s.SetContent(x, row, '▀', nil, st)
		}
	}
}
