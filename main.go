// Cappy Lost In Space (Retro Demo)
//
// A terminal metroidvania sidescroller: half-block pixels, 256 colours,
// a curated planet. Runs on the default terminals of Windows, macOS and
// Linux.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/gdamore/tcell/v2"

	"github.com/AgustinBanchio/terminal-cappy/internal/game"
)

func main() {
	fps := flag.Int("fps", 60, "terminal mode: simulation and render rate")
	window := flag.Bool("window", false, "run in a desktop window instead of the terminal (needs a build with -tags window)")
	scale := flag.Int("scale", 8, "window mode: size of one game pixel")
	flag.Parse()

	if *window {
		if err := game.RunWindow(*scale); err != nil {
			fmt.Fprintln(os.Stderr, "cappy:", err)
			os.Exit(1)
		}
		return
	}

	if *fps < 10 || *fps > 120 {
		fmt.Fprintln(os.Stderr, "fps must be between 10 and 120")
		os.Exit(2)
	}

	if err := run(*fps); err != nil {
		fmt.Fprintln(os.Stderr, "cappy:", err)
		os.Exit(1)
	}
}

// run owns the screen lifecycle: the terminal is always restored, even
// on a crash, before anything is printed to stderr.
func run(fps int) error {
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

	return game.New(screen).Run(fps)
}
