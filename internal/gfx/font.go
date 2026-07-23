package gfx

// A tiny 3x5 pixel font used for the title logo and in-canvas labels.
// Only the glyphs the game needs are defined; unknown runes render as
// a blank advance.
var font3x5 = map[rune][5]string{
	'A': {".#.", "#.#", "###", "#.#", "#.#"},
	'C': {".##", "#..", "#..", "#..", ".##"},
	'E': {"###", "#..", "##.", "#..", "###"},
	'I': {"###", ".#.", ".#.", ".#.", "###"},
	'L': {"#..", "#..", "#..", "#..", "###"},
	'N': {"#.#", "###", "###", "#.#", "#.#"},
	'O': {"###", "#.#", "#.#", "#.#", "###"},
	'P': {"##.", "#.#", "##.", "#..", "#.."},
	'S': {".##", "#..", ".#.", "..#", "##."},
	'T': {"###", ".#.", ".#.", ".#.", ".#."},
	'Y': {"#.#", "#.#", ".#.", ".#.", ".#."},
	' ': {"...", "...", "...", "...", "..."},
}

const (
	glyphW = 3
	glyphH = 5
)

// TextPxWidth returns the pixel width of msg at a given scale.
func TextPxWidth(msg string, scale int) int {
	n := len([]rune(msg))
	if n == 0 {
		return 0
	}
	return (n*(glyphW+1) - 1) * scale
}

// TextPxHeight returns the pixel height of the font at a given scale.
func TextPxHeight(scale int) int { return glyphH * scale }

// DrawTextPx renders msg into the canvas with the 3x5 pixel font,
// scaled up by an integer factor, with an optional 1px drop shadow.
func (c *Canvas) DrawTextPx(x, y int, msg string, scale int, color uint8, shadow int16) {
	if scale < 1 {
		scale = 1
	}
	cx := x
	for _, r := range msg {
		g, ok := font3x5[r]
		if ok {
			for gy := 0; gy < glyphH; gy++ {
				for gx := 0; gx < glyphW; gx++ {
					if g[gy][gx] != '#' {
						continue
					}
					px := cx + gx*scale
					py := y + gy*scale
					if shadow >= 0 {
						c.FillRect(px+scale, py+scale, scale, scale, uint8(shadow))
					}
					c.FillRect(px, py, scale, scale, color)
				}
			}
		}
		cx += (glyphW + 1) * scale
	}
}
