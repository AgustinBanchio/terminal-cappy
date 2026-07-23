package game

import "time"

type action int

const (
	actLeft action = iota
	actRight
	actJump
	actShoot
	actCount
)

// Terminals only deliver key presses (plus OS auto-repeat), never key
// releases, so "holding" a key is emulated: each press event keeps the
// action held for a short window, and auto-repeat refreshes it. The
// window must outlast typical repeat intervals without making single
// taps feel sticky.
const holdWindow = 260 * time.Millisecond

type input struct {
	heldUntil [actCount]time.Time
	pressed   [actCount]bool
}

func (in *input) press(a action, now time.Time) {
	in.pressed[a] = true
	in.heldUntil[a] = now.Add(holdWindow)
}

// held reports whether the action counts as held right now.
func (in *input) held(a action, now time.Time) bool {
	return now.Before(in.heldUntil[a])
}

// consume returns and clears the edge-triggered press state, so an
// action fires once per key event rather than once per frame.
func (in *input) consume(a action) bool {
	p := in.pressed[a]
	in.pressed[a] = false
	return p
}

func (in *input) endFrame() {
	for i := range in.pressed {
		in.pressed[i] = false
	}
}
