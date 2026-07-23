// Cappy Lost In Space (Retro Demo)
//
// A terminal sidescroller: half-block pixels, 256 colours, procedural
// planets. Runs on the default terminals of Windows, macOS and Linux.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/gdamore/tcell/v2"

	"cappy/internal/game"
)

func main() {
	seed := flag.Int64("seed", 0, "world seed (0 = random)")
	fps := flag.Int("fps", 30, "simulation and render rate")
	flag.Parse()

	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}
	if *fps < 10 || *fps > 120 {
		fmt.Fprintln(os.Stderr, "fps must be between 10 and 120")
		os.Exit(2)
	}

	if err := run(*seed, *fps); err != nil {
		fmt.Fprintln(os.Stderr, "cappy:", err)
		os.Exit(1)
	}
}

// run owns the screen lifecycle: the terminal is always restored, even
// on a crash, before anything is printed to stderr.
func run(seed int64, fps int) error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("cannot open terminal: %w", err)
	}
	if err := screen.Init(); err != nil {
		return fmt.Errorf("cannot init terminal: %w", err)
	}
	screen.SetStyle(tcell.StyleDefault)
	screen.HideCursor()

	defer func() {
		r := recover()
		screen.Fini()
		if r != nil {
			fmt.Fprintf(os.Stderr, "cappy crashed: %v\n%s", r, debug.Stack())
			os.Exit(1)
		}
	}()

	return game.New(screen, seed).Run(fps)
}
