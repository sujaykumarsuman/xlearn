package gateway

import (
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// TestManifestGoldenMirror is the gateway side of the DSA manifest golden test (sprint
// m1-01; side one is internal/course/golden_test.go): the Today "Mock best" tile checks
// against the manifest's W13 readiness target.
func TestManifestGoldenMirror(t *testing.T) {
	if w13 := coursetest.DSA(t).Mock.Targets.W13; mockW13Target != w13 {
		t.Errorf("mockW13Target = %d, manifest mock.targets.w13 = %d", mockW13Target, w13)
	}
}
