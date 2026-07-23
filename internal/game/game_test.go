package game

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// screenText flattens a simulation screen into plain text rows.
func screenText(s tcell.SimulationScreen) string {
	cells, w, h := s.GetContents()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := cells[y*w+x].Runes
			if len(r) > 0 {
				b.WriteRune(r[0])
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func TestTitleScreenShowsVersion(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	defer s.Fini()
	s.SetSize(80, 24)

	g := New(s)
	g.draw()

	text := screenText(s)
	if !strings.Contains(text, "retro demo "+Version) {
		t.Fatalf("title screen missing version %q:\n%s", Version, text)
	}
	if !strings.Contains(text, "PRESS ANY KEY TO CONTINUE") &&
		!strings.Contains(text, "arrows/AD move") {
		t.Fatalf("title screen missing prompt/help:\n%s", text)
	}
}
