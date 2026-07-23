package game

import "fmt"

// Traversal analysis: simulates simplified platformer movement over the
// tile grid and reports places Cappy can get into but not back out of,
// plus any points of interest that are out of reach.
//
// The model, in tiles: Cappy is 2 tall; a jump rises up to 4; air drift
// is 6 sideways per flight; falls are unlimited; chimneys up to 4 tiles
// wide are climbable with wall jumps (budgets reset inside them); lava
// counts as bouncy support. Boss doors are analysed unlocked. The model
// is a little conservative, so anything it passes is comfortably
// traversable in game.

const (
	travRise  = 4 // max tiles gained per jump
	travDrift = 6 // max sideways tiles per flight
)

type travGraph struct {
	l    *Level
	nb   int // budget combinations per cell
	n    int // total states
	fits []bool
}

func newTravGraph(l *Level) *travGraph {
	g := &travGraph{l: l, nb: (travRise + 1) * (travDrift + 1)}
	g.n = l.W * l.H * g.nb
	g.fits = make([]bool, l.W*l.H)
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			// Feet tile plus head tile must be open.
			g.fits[y*l.W+x] = !l.SolidTile(x, y) && !l.SolidTile(x, y-1)
		}
	}
	return g
}

func (g *travGraph) id(x, y, up, drift int) int {
	return (y*g.l.W+x)*g.nb + up*(travDrift+1) + drift
}

func (g *travGraph) supported(x, y int) bool {
	if g.l.SolidTile(x, y+1) {
		return true
	}
	c := g.l.Cell(LayerSolid, x, y)
	below := g.l.Cell(LayerSolid, x, y+1)
	return c == '~' || below == '~'
}

// chimney reports whether walls close enough on both sides allow
// wall-jump climbing through this cell.
func (g *travGraph) chimney(x, y int) bool {
	dl, dr := 5, 5
	for d := 1; d <= 4; d++ {
		if g.l.SolidTile(x-d, y) {
			dl = d
			break
		}
	}
	for d := 1; d <= 4; d++ {
		if g.l.SolidTile(x+d, y) {
			dr = d
			break
		}
	}
	return dl+dr <= 5
}

// norm assigns the budgets gained by arriving in a cell: full on
// support, full on chimney walls, otherwise whatever was carried in.
func (g *travGraph) norm(x, y, up, drift int) int {
	if g.supported(x, y) || g.chimney(x, y) {
		return g.id(x, y, travRise, travDrift)
	}
	return g.id(x, y, up, drift)
}

// succs appends the reachable neighbour states of state s to out.
func (g *travGraph) succs(s int, out []int) []int {
	drift := s % (travDrift + 1)
	up := (s / (travDrift + 1)) % (travRise + 1)
	cell := s / g.nb
	x, y := cell%g.l.W, cell/g.l.W

	try := func(nx, ny, nup, ndrift int) {
		if nx >= 0 && nx < g.l.W && ny >= 0 && ny < g.l.H && g.fits[ny*g.l.W+nx] {
			out = append(out, g.norm(nx, ny, nup, ndrift))
		}
	}
	try(x, y+1, 0, drift) // fall: no more rising this flight
	if up > 0 {
		try(x, y-1, up-1, drift)
	}
	if drift > 0 {
		try(x-1, y, up, drift-1)
		try(x+1, y, up, drift-1)
	}
	return out
}

func (g *travGraph) bfs(starts []int) []bool {
	seen := make([]bool, g.n)
	queue := make([]int, 0, len(starts))
	for _, s := range starts {
		if !seen[s] {
			seen[s] = true
			queue = append(queue, s)
		}
	}
	buf := make([]int, 0, 4)
	for len(queue) > 0 {
		s := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		buf = g.succs(s, buf[:0])
		for _, t := range buf {
			if !seen[t] {
				seen[t] = true
				queue = append(queue, t)
			}
		}
	}
	return seen
}

// bfsBackward computes which states can reach any of the target cells,
// by exploring the reversed edge set (materialised on the fly).
func (g *travGraph) bfsBackward(targets []bool) []bool {
	// Build reverse adjacency with two counting passes.
	counts := make([]int32, g.n+1)
	buf := make([]int, 0, 4)
	for s := 0; s < g.n; s++ {
		if !g.fits[s/g.nb] {
			continue
		}
		buf = g.succs(s, buf[:0])
		for _, t := range buf {
			counts[t+1]++
		}
	}
	for i := 1; i <= g.n; i++ {
		counts[i] += counts[i-1]
	}
	edges := make([]int32, counts[g.n])
	fill := make([]int32, g.n)
	for s := 0; s < g.n; s++ {
		if !g.fits[s/g.nb] {
			continue
		}
		buf = g.succs(s, buf[:0])
		for _, t := range buf {
			edges[counts[t]+fill[t]] = int32(s)
			fill[t]++
		}
	}

	seen := make([]bool, g.n)
	queue := []int{}
	for s := 0; s < g.n; s++ {
		if targets[s/g.nb] && g.fits[s/g.nb] {
			seen[s] = true
			queue = append(queue, s)
		}
	}
	for len(queue) > 0 {
		t := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		for _, p := range edges[counts[t]:counts[t+1]] {
			if !seen[p] {
				seen[p] = true
				queue = append(queue, int(p))
			}
		}
	}
	return seen
}

// AnalyzeTraversal returns human-readable problems, empty if the level
// is fully traversable: every point of interest reachable from spawn,
// and every reachable standing spot has a way back to spawn.
func (l *Level) AnalyzeTraversal() []string {
	g := newTravGraph(l)

	sx, sy := fdiv(int(l.SpawnX), TilePx), fdiv(int(l.SpawnY), TilePx)
	fwd := g.bfs([]int{g.norm(sx, sy, travRise, travDrift)})

	targetCell := make([]bool, l.W*l.H)
	targetCell[sy*l.W+sx] = true
	bwd := g.bfsBackward(targetCell)

	var issues []string
	cellReach := func(x, y int) (bool, bool) {
		f, b := false, false
		for k := 0; k < g.nb; k++ {
			s := (y*l.W+x)*g.nb + k
			f = f || fwd[s]
			b = b || (fwd[s] && bwd[s])
		}
		return f, b
	}

	pois := []struct {
		name string
		x, y int
	}{{"spawn", sx, sy}, {"ship", l.ShipX/TilePx + 3, l.ShipY / TilePx}}
	for _, s := range l.Spawns {
		if s.Kind == 'P' || s.Kind == 'D' || s.Kind == 'Q' || s.Kind == 'M' {
			pois = append(pois, struct {
				name string
				x, y int
			}{fmt.Sprintf("%c", s.Kind), fdiv(int(s.X), TilePx), fdiv(int(s.Y), TilePx)})
		}
	}
	for _, p := range pois {
		f, b := cellReach(p.x, p.y)
		if !f {
			issues = append(issues, fmt.Sprintf("%s@%d,%d unreachable from spawn", p.name, p.x, p.y))
		} else if !b {
			issues = append(issues, fmt.Sprintf("%s@%d,%d cannot return to spawn", p.name, p.x, p.y))
		}
	}

	// Any standing spot you can reach must offer a way back.
	stuck := 0
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			if !g.fits[y*l.W+x] || !g.supported(x, y) {
				continue
			}
			s := g.id(x, y, travRise, travDrift)
			if fwd[s] && !bwd[s] {
				stuck++
				if stuck <= 8 {
					issues = append(issues, fmt.Sprintf("stuck spot: standing tile %d,%d has no way back", x, y))
				}
			}
		}
	}
	if stuck > 8 {
		issues = append(issues, fmt.Sprintf("...and %d more stuck spots", stuck-8))
	}
	return issues
}
