//go:build !linux

// Package jail is Linux-only; this stub keeps `go build ./...` and `go test ./...` green on
// macOS (the jail itself only runs in the runner-it lane and in the pod).
package jail

import (
	"errors"
	"fmt"
	"os"
)

// ErrUnsupported is returned off Linux.
var ErrUnsupported = errors.New("jail: Linux only")

// CompileInit is Linux-only.
func CompileInit(args []string) int {
	fmt.Fprintln(os.Stderr, ErrUnsupported)
	return 96
}
