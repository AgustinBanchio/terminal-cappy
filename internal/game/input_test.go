package game

import (
	"testing"
	"time"
)

func TestInputBridgesInitialRepeatDelay(t *testing.T) {
	in := newInput()
	t0 := time.Unix(1000, 0)

	// First press must stay held across a typical OS initial repeat
	// delay (~500ms) so holding a key has no hitch before repeats start.
	in.press(actRight, t0)
	if !in.held(actRight, t0.Add(450*time.Millisecond)) {
		t.Fatal("first press must bridge the OS initial repeat delay")
	}

	// First repeat arrives after 480ms: calibrates repeatDelay, and the
	// repeat stream keeps the key held with much shorter windows.
	in.press(actRight, t0.Add(480*time.Millisecond))
	if in.repeatDelay != 560*time.Millisecond {
		t.Fatalf("repeatDelay not calibrated, got %v", in.repeatDelay)
	}
	last := t0.Add(480 * time.Millisecond)
	for i := 0; i < 5; i++ {
		last = last.Add(50 * time.Millisecond)
		in.press(actRight, last)
		if !in.held(actRight, last.Add(30*time.Millisecond)) {
			t.Fatal("repeat stream must keep the key held")
		}
	}

	// Once repeats stop, the key releases quickly (a few intervals).
	if in.held(actRight, last.Add(300*time.Millisecond)) {
		t.Fatal("key must release shortly after repeats stop")
	}
}

func TestOppositeDirectionCancels(t *testing.T) {
	in := newInput()
	t0 := time.Unix(1000, 0)

	in.press(actRight, t0)
	in.press(actLeft, t0.Add(100*time.Millisecond))

	at := t0.Add(101 * time.Millisecond)
	if in.held(actRight, at) {
		t.Fatal("pressing left must instantly release right")
	}
	if in.dir(at) != -1 {
		t.Fatal("direction must switch to left")
	}
}
