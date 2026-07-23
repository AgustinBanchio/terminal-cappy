package game

import "math"

// Camera follows the player loosely: while the player stays inside a
// central dead zone nothing moves; once they push past its edge the
// camera glides to catch up, clamped to the world bounds.
type Camera struct {
	X, Y float64
}

// Center snaps the camera onto a point (used on spawn/restart).
func (c *Camera) Center(px, py, viewW, viewH, worldW, worldH float64) {
	c.X = px - viewW/2
	c.Y = py - viewH/2
	c.clamp(viewW, viewH, worldW, worldH)
}

// Update nudges the camera so the target box stays inside the dead zone.
func (c *Camera) Update(tx, ty, tw, th, viewW, viewH, worldW, worldH, dt float64) {
	dzW := viewW / 4
	dzH := viewH / 3.5

	left := c.X + (viewW-dzW)/2
	right := left + dzW
	top := c.Y + (viewH-dzH)/2
	bottom := top + dzH

	var wantX, wantY float64
	if tx < left {
		wantX = tx - left
	} else if tx+tw > right {
		wantX = tx + tw - right
	}
	if ty < top {
		wantY = ty - top
	} else if ty+th > bottom {
		wantY = ty + th - bottom
	}

	s := math.Min(1, 12*dt)
	c.X += wantX * s
	c.Y += wantY * s
	c.clamp(viewW, viewH, worldW, worldH)
}

func (c *Camera) clamp(viewW, viewH, worldW, worldH float64) {
	if worldW <= viewW {
		c.X = (worldW - viewW) / 2 // world narrower than view: centre it
	} else {
		c.X = math.Max(0, math.Min(c.X, worldW-viewW))
	}
	if worldH <= viewH {
		c.Y = worldH - viewH // align world floor with screen bottom
	} else {
		c.Y = math.Max(0, math.Min(c.Y, worldH-viewH))
	}
}
