package gateway

import (
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// TestManifestGoldenMirror is the gateway side of the DSA manifest golden test (sprint
// m1-01; side one is internal/course/golden_test.go): the Today "Mock best" tile checks
// against the manifest's W13 readiness target. Since m1-03 the gateway READS it from the
// course's manifest, so this pins v1's value (24) for DSA and the no-mock case.
func TestManifestGoldenMirror(t *testing.T) {
	dsa := coursetest.DSA(t)
	if got := mockW13Target(dsa); got != 24 || got != dsa.Mock.Targets.W13 {
		t.Errorf("mockW13Target(dsa) = %d, want v1's 24 (manifest w13 = %d)", got, dsa.Mock.Targets.W13)
	}
	fx, _ := coursetest.Registry(t).Lookup(coursetest.FixtureActive)
	if got := mockW13Target(fx); got != 0 {
		t.Errorf("a course without a mock has no target, got %d", got)
	}
}
