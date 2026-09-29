//go:build !linux || !(amd64 || arm64)

package seccomp

// Native has no table off Linux amd64/arm64; Assemble refuses it. The jail itself is
// Linux-only, so nothing here ever loads a filter.
func Native() Arch { return Arch{Name: "unsupported"} }
