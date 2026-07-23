package gfx

import "testing"

func TestCanvasBoundsAreSafe(t *testing.T) {
	c := NewCanvas(10, 5)
	if c.W != 10 || c.H != 10 {
		t.Fatalf("want 10x10 canvas, got %dx%d", c.W, c.H)
	}
	c.Set(-1, 0, 5)
	c.Set(0, -1, 5)
	c.Set(10, 0, 5)
	c.Set(0, 10, 5)
	c.FillRect(-5, -5, 100, 100, 7)
	if c.At(0, 0) != 7 || c.At(9, 9) != 7 {
		t.Fatal("FillRect did not clip and fill correctly")
	}
	if c.At(-1, 0) != 0 {
		t.Fatal("out-of-bounds At must return 0")
	}
}

func TestSpriteParseAndFlip(t *testing.T) {
	pal := map[rune]uint8{'a': 1, 'b': 2}
	s := MustSprite(pal,
		"ab.",
		"..a")
	if s.W != 3 || s.H != 2 {
		t.Fatalf("bad sprite size %dx%d", s.W, s.H)
	}
	if s.Pix[0] != 1 || s.Pix[1] != 2 || s.Pix[2] != -1 {
		t.Fatal("bad sprite parse")
	}
	f := s.FlipH()
	if f.Pix[0] != -1 || f.Pix[2] != 1 {
		t.Fatal("bad horizontal flip")
	}

	defer func() {
		if recover() == nil {
			t.Fatal("ragged sprite rows must panic")
		}
	}()
	MustSprite(pal, "ab", "a")
}

func TestFontMetrics(t *testing.T) {
	if w := TextPxWidth("CAPPY", 1); w != 5*4-1 {
		t.Fatalf("unexpected text width %d", w)
	}
	c := NewCanvas(40, 5)
	c.DrawTextPx(0, 0, "CAPPY", 1, 220, -1)
	found := false
	for _, p := range c.pix {
		if p == 220 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("DrawTextPx drew nothing")
	}
}
