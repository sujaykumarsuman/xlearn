//go:build !runner_it

package seccomp

import "testing"

// TestReleaseBuildHasNoLOGDefault guards the dev-only switch: a release build (no runner_it
// tag) never gives an exec filter a LOG default, on any arch or in any mode.
func TestReleaseBuildHasNoLOGDefault(t *testing.T) {
	if devLogBuild {
		t.Fatal("devLogBuild is set in a release build")
	}
	for _, arch := range []string{"amd64", "arm64", "riscv64"} {
		for _, dev := range []bool{false, true} {
			if got := ExecDefault(arch, dev); got != ActKillProcess {
				t.Errorf("ExecDefault(%s, dev=%v) = %#x, want KILL_PROCESS", arch, dev, uint32(got))
			}
		}
	}
}
