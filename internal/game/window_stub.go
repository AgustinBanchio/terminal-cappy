//go:build !window

package game

import "errors"

// RunWindow is only available in builds made with the "window" tag,
// which pulls in Ebitengine. The default build stays pure Go so the
// terminal game keeps building everywhere with no C toolchain.
func RunWindow(scale int) error {
	return errors.New("this build has no window mode\n" +
		"rebuild with it included:  go run -tags window . -window\n" +
		"                     or:   go build -tags window -o cappy . && ./cappy -window")
}
