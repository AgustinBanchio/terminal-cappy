# Cappy Lost In Space (Retro Demo)

A terminal sidescroller platformer/roguelike. Cappy, a bipedal border
collie in a red astronaut suit, crash-lands on an alien planet. Find the
scattered ship parts, fend off the local aliens with your laser
revolver, and get back to the ship to escape.

Every run generates a new planet from a seed: the world is stitched
together from hand-authored chunks in a shuffled (and randomly mirrored)
order. Death means a fresh planet.

## Running

```sh
go build -o cappy .
./cappy
```

Flags:

- `-seed N` play a specific planet (0 = random)
- `-fps N` simulation/render rate (default 30)

## Controls

- Left/Right arrows (or A/D): move
- Z (or W, Space, Up): jump
- X (or K): shoot
- Hold into a wall while airborne: wall slide; Z while sliding: wall jump
- P: pause
- R: restart on a new planet
- Esc / Ctrl+C: quit

## Portability

The renderer targets the default terminals of Windows, macOS and Linux:

- Unicode half blocks (`▀`): each terminal cell is two vertical pixels,
  so an 80x24 terminal is an 80x48 pixel canvas. Bigger or resized
  terminals show more of the world before the camera scrolls.
- 256-colour palette indices only, no truecolour required.
- Pure Go, no cgo. Cross-compile with the usual
  `GOOS=windows GOARCH=amd64 go build` etc. [tcell](https://github.com/gdamore/tcell)
  is the single dependency and handles both the Windows console API and
  Unix termios.

One inherent terminal limitation: terminals report key presses (plus OS
auto-repeat) but never key releases, so "holding" a key is emulated with
a short hold window refreshed by auto-repeat. If movement feels sticky
or stuttery, your OS key-repeat delay/rate settings are the knob to
tweak.

## Architecture

- `internal/gfx`: half-block pixel canvas on tcell, sprite parsing from
  ASCII art with rune palettes, and a 3x5 pixel font for the title logo.
- `internal/game`:
  - `level.go`: chunk templates, seeded world generation, tile collision
  - `player.go`: movement physics (coyote time, jump buffering, wall
    slide/jump), shooting
  - `entities.go`: aliens (patrolling walkers, drifting flyers),
    bullets, pickups, particles
  - `background.go`: multi-layer parallax (sky, starfield, moon, two
    mountain ridges)
  - `camera.go`: dead-zone follow camera clamped to the world
  - `game.go`: state machine (title/playing/paused/dead/won), fixed
    timestep loop, HUD and overlays
