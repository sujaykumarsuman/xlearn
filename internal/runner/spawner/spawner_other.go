//go:build !linux

// Package spawner is the runner's capability-holding half; it only runs on Linux. This stub
// keeps macOS builds and tests green.
package spawner

import (
	"errors"
	"fmt"
	"os"
)

// ErrUnsupported is returned off Linux.
var ErrUnsupported = errors.New("spawner: Linux only")

// Main refuses to run off Linux.
func Main(version string) int {
	fmt.Fprintln(os.Stderr, ErrUnsupported)
	return 1
}
