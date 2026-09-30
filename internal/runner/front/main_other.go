//go:build !linux

package front

import (
	"fmt"
	"os"
)

// Main is Linux-only (the front's fds come from the Linux spawner).
func Main() int {
	fmt.Fprintln(os.Stderr, "front: Linux only")
	return 1
}
