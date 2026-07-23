package gfx

// PaletteRGB converts an xterm-256 palette index to RGB, for renderers
// that draw real pixels instead of terminal cells.
func PaletteRGB(idx uint8) (r, g, b uint8) {
	c := palette256[idx]
	return c[0], c[1], c[2]
}

var palette256 = buildPalette()

func buildPalette() [256][3]uint8 {
	var p [256][3]uint8

	// 0-15: the standard system colours.
	sys := [16][3]uint8{
		{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0},
		{0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
		{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
		{0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
	}
	copy(p[:16], sys[:])

	// 16-231: the 6x6x6 colour cube.
	levels := [6]uint8{0, 95, 135, 175, 215, 255}
	for i := 0; i < 216; i++ {
		p[16+i] = [3]uint8{levels[i/36], levels[(i/6)%6], levels[i%6]}
	}

	// 232-255: the grayscale ramp.
	for i := 0; i < 24; i++ {
		v := uint8(8 + 10*i)
		p[232+i] = [3]uint8{v, v, v}
	}
	return p
}
