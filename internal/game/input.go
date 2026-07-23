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

// Terminals deliver key presses plus OS auto-repeat, never key releases
// (tcell has no release events, even with the kitty protocol), so a key
// counts as "held" for a window after each press event.
//
// The tricky part is the OS initial repeat delay: after the first press
// there is a long silent gap (~300-800ms) before repeats start. A short
// window would expire inside that gap and cause a visible hitch. So:
//
//   - a first press is held for repeatDelay, long enough to bridge the
//     gap; repeatDelay starts pessimistic and is calibrated down from
//     the first-repeat gaps actually observed on this machine
//   - once repeats stream in, the window shrinks to a few multiples of
//     the measured repeat interval, so releases register quickly
//   - pressing one direction instantly releases the opposite one
const (
	defaultRepeatDelay = 550 * time.Millisecond
	repeatStreamMax    = 200 * time.Millisecond // gaps below this are repeats
)

type input struct {
	pressed   [actCount]bool
	heldUntil [actCount]time.Time
	lastEvent [actCount]time.Time
	lastPress [actCount]time.Time

	repeatDelay time.Duration

	// Direct mode (window build): the renderer reports true key state
	// every frame, so all the hold-window emulation above is bypassed,
	// and real release events become available.
	direct     bool
	directHeld [actCount]bool
	released   [actCount]bool
}

func newInput() input {
	return input{repeatDelay: defaultRepeatDelay}
}

func (in *input) press(a action, now time.Time) {
	gap := now.Sub(in.lastEvent[a])
	in.lastEvent[a] = now
	in.lastPress[a] = now
	in.pressed[a] = true

	var window time.Duration
	if gap < repeatStreamMax {
		// Inside an auto-repeat stream: stay held for a few missed
		// repeats, then release promptly.
		window = clampDur(3*gap, 90*time.Millisecond, 240*time.Millisecond)
	} else {
		if gap < 1200*time.Millisecond {
			// Likely the first repeat after the OS initial delay:
			// calibrate so future first presses bridge it exactly.
			in.repeatDelay = clampDur(gap+80*time.Millisecond,
				300*time.Millisecond, 1100*time.Millisecond)
		}
		window = in.repeatDelay
	}
	in.heldUntil[a] = now.Add(window)

	// Direction changes must be instant: a new direction press releases
	// the opposite one instead of fighting its hold window.
	switch a {
	case actLeft:
		in.heldUntil[actRight] = now
	case actRight:
		in.heldUntil[actLeft] = now
	}
}

// setDirect feeds real per-frame key state (window mode).
func (in *input) setDirect(a action, held, justPressed, justReleased bool, now time.Time) {
	in.direct = true
	in.directHeld[a] = held
	if justPressed {
		in.pressed[a] = true
		in.lastPress[a] = now
	}
	if justReleased {
		in.released[a] = true
	}
}

// held reports whether the action counts as held right now.
func (in *input) held(a action, now time.Time) bool {
	if in.direct {
		return in.directHeld[a]
	}
	return now.Before(in.heldUntil[a])
}

// consumeRelease returns and clears the edge-triggered release state.
// Only direct mode ever sets it: terminals have no release events.
func (in *input) consumeRelease(a action) bool {
	r := in.released[a]
	in.released[a] = false
	return r
}

// dir returns the horizontal input direction; when both directions are
// held the most recent press wins (last-input priority).
func (in *input) dir(now time.Time) int {
	l, r := in.held(actLeft, now), in.held(actRight, now)
	switch {
	case l && r:
		if in.lastPress[actRight].After(in.lastPress[actLeft]) {
			return 1
		}
		return -1
	case l:
		return -1
	case r:
		return 1
	}
	return 0
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
		in.released[i] = false
	}
}

func clampDur(d, lo, hi time.Duration) time.Duration {
	if d < lo {
		return lo
	}
	if d > hi {
		return hi
	}
	return d
}
