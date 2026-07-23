package gfx

import "fmt"

// Sprite is a small image of 256-colour palette indices.
// -1 marks a transparent pixel.
type Sprite struct {
	W, H int
	Pix  []int16
}

// MustSprite parses ASCII art rows into a sprite using a rune palette.
// '.' (and any rune missing from the palette) is transparent.
// It panics on ragged rows so bad art fails loudly at startup.
func MustSprite(pal map[rune]uint8, rows ...string) *Sprite {
	if len(rows) == 0 {
		panic("gfx: sprite with no rows")
	}
	w := len(rows[0])
	s := &Sprite{W: w, H: len(rows), Pix: make([]int16, w*len(rows))}
	for y, row := range rows {
		if len(row) != w {
			panic(fmt.Sprintf("gfx: ragged sprite row %d: %q (want width %d)", y, row, w))
		}
		for x, r := range row {
			idx := int16(-1)
			if r != '.' {
				if col, ok := pal[r]; ok {
					idx = int16(col)
				}
			}
			s.Pix[y*w+x] = idx
		}
	}
	return s
}

// FlipH returns a horizontally mirrored copy of the sprite.
func (s *Sprite) FlipH() *Sprite {
	out := &Sprite{W: s.W, H: s.H, Pix: make([]int16, len(s.Pix))}
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			out.Pix[y*s.W+x] = s.Pix[y*s.W+(s.W-1-x)]
		}
	}
	return out
}

// Frames pairs the right-facing and left-facing variants of a sprite.
type Frames struct {
	R, L *Sprite
}

// NewFrames builds Frames from right-facing art.
func NewFrames(right *Sprite) Frames { return Frames{R: right, L: right.FlipH()} }

// Facing returns the sprite for a facing direction (>= 0 is right).
func (f Frames) Facing(dir int) *Sprite {
	if dir < 0 {
		return f.L
	}
	return f.R
}
