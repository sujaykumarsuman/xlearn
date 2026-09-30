//go:build !runner_it

package main

import (
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
)

// TestReleaseRegistryHasNoTestProfile: the release build's profile registry has no test-only
// profile (testgo@0 exists only under the runner_it tag).
func TestReleaseRegistryHasNoTestProfile(t *testing.T) {
	for _, p := range profile.All() {
		if p.TestOnly {
			t.Errorf("release build registers the test-only profile %s", p.Name)
		}
	}
	if _, _, ok := profile.Lookup("testgo@0"); ok {
		t.Error("release build registers testgo@0")
	}
}
