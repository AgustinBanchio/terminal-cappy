# Cappy Lost In Space (Retro Demo)

A terminal metroidvania sidescroller. Cappy, a bipedal border collie in
a red astronaut suit, crash-lands on an alien planet. Find the four
ship parts scattered across a curated, hand-authored world, fend off
the local aliens with your laser revolver, and get back to the ship to
escape.

## Running

```sh
go build -o cappy .
./cappy
```

Or straight from GitHub (needs Go installed, and the repo to be
public/accessible):

```sh
go run github.com/AgustinBanchio/terminal-cappy@latest
```

Flags:

- `-fps N` simulation/render rate (default 60)

## Controls

- Left/Right arrows (or A/D): move
- `,` / `.` (or Shift+arrows): micro-step exactly 1px, for precise
  positioning; hold for a slow creep
- Z (or W, Space, Up): jump
- X (or K): shoot
- Hold into a wall while airborne: wall slide; Z while sliding: wall jump
- P: pause
- R: restart
- Esc / Ctrl+C: quit

Movement is modelled on Hollow Knight's controller: instant locked run
speed with no acceleration ramp, identical control on ground and in the
air, asymmetric gravity (snappy falls), coyote time, jump buffering,
and a brief control lock on wall jumps so the kick away from the wall
lands.

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

### Input handling

Terminals report key presses (plus OS auto-repeat) but never key
releases, so held keys are emulated: the first press is held long
enough to bridge the OS initial repeat delay (no hitch when you hold a
direction), the game calibrates that delay from your actual keystream,
and once repeats arrive the hold window tightens so releases register
quickly. Pressing the opposite direction always switches instantly.
The one trade-off: a single arrow tap moves Cappy for roughly the
initial repeat delay, because the game cannot know it was a tap until
that window passes. That is what the micro-step keys are for: `,` and
`.` move exactly 1px per keypress, so precision never depends on
timing.

## Level editor

The world lives in `internal/game/level1.txt`, a plain-text file with
three layers, embedded into the game binary at compile time:

- `@solid`: collision and entities (`#` rock, `S` spawn, `a` walker,
  `f` flyer, `P` ship part, `H` ship anchor)
- `@bg`: decoration behind gameplay (`t` stalactite, `m` stalagmite,
  `I` pillar, `c` crystal)
- `@fg`: decoration in front of gameplay (`g` grass)

Edit it with the bundled editor (run from the repo root):

```sh
go run ./cmd/leveled
```

- Arrows or mouse click/drag: move the cursor and paint
- Space/Enter: paint the selected tile; X/Backspace: erase
- Tab or 1/2/3: switch the edited layer (solid/background/foreground)
- V: toggle the full composite view (exactly what the game renders)
- `[` `]`: cycle the tile palette for the current layer
- Ctrl+S: save; Esc: quit (asks twice if unsaved)

Decoration tiles keep their pixel shapes from a position hash, so
hand-placed stalactites and grass still look organic rather than
stamped. After saving, rebuild the game to embed the new level.

## Architecture

- `internal/gfx`: half-block pixel canvas on tcell, sprite parsing from
  ASCII art with rune palettes, and a 3x5 pixel font for the title logo.
- `internal/game`:
  - `level.go`: the layered level format (parse/serialise/edit API),
    tile collision, and rendering for the decoration layers:
    background stalactites/stalagmites/pillars/crystals and swaying
    see-through foreground grass
  - `player.go`: the Hollow Knight-style movement controller, shooting
  - `entities.go`: aliens (patrolling walkers, drifting flyers),
    bullets, pickups, particles
  - `background.go`: multi-layer parallax (sky, starfield, moon, two
    mountain ridges)
  - `camera.go`: dead-zone follow camera clamped to the world
  - `input.go`: key-repeat bridging and calibration (see above)
  - `game.go`: state machine (title/playing/paused/dead/won), fixed
    timestep loop, HUD, ship dialogue, overlays
- `cmd/leveled`: the level editor (see above)
