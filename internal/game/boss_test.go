package game

import "testing"

func TestBossChambersSealWhenLocked(t *testing.T) {
	l := LoadDefault()
	found := map[rune]bool{}
	for _, s := range l.Spawns {
		if s.Kind != 'D' && s.Kind != 'Q' && s.Kind != 'M' {
			continue
		}
		found[s.Kind] = true
		b := newBoss(s.Kind, s.X, s.Y, l)

		if len(b.doors) == 0 {
			t.Errorf("boss %c: no door tiles assigned", s.Kind)
			continue
		}
		if b.chamber[2]-b.chamber[0] < 8 || b.chamber[3]-b.chamber[1] < 4 {
			t.Errorf("boss %c: chamber too small: %v", s.Kind, b.chamber)
		}

		// With the doors locked, a flood fill from the boss must stay
		// inside the chamber: the arena is escape-proof.
		l.LockDoors(b.doors, true)
		start := [2]int{fdiv(int(s.X), TilePx), fdiv(int(s.Y), TilePx)}
		seen := map[[2]int]bool{}
		queue := [][2]int{start}
		for len(queue) > 0 {
			q := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			if seen[q] || l.SolidTile(q[0], q[1]) {
				continue
			}
			seen[q] = true
			if q[0] < b.chamber[0]-2 || q[0] > b.chamber[2]+2 ||
				q[1] < b.chamber[1]-2 || q[1] > b.chamber[3]+2 {
				t.Errorf("boss %c: locked chamber leaks at tile %v", s.Kind, q)
				break
			}
			queue = append(queue,
				[2]int{q[0] - 1, q[1]}, [2]int{q[0] + 1, q[1]},
				[2]int{q[0], q[1] - 1}, [2]int{q[0], q[1] + 1})
		}
		l.LockDoors(b.doors, false)
	}
	for _, k := range []rune{'D', 'Q', 'M'} {
		if !found[k] {
			t.Errorf("level has no boss %c", k)
		}
	}
}
