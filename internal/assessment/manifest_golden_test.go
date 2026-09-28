package assessment

import (
	"fmt"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// TestManifestGoldenMirror is the assessment side of the DSA manifest golden test
// (sprint m1-01; side one is internal/course/golden_test.go): the six-phase rail, the
// dimension labels, the readiness targets and the mock window must equal the manifest.
func TestManifestGoldenMirror(t *testing.T) {
	m := coursetest.DSA(t)
	mk := m.Mock

	if mockDurationSecs != mk.DurationS {
		t.Errorf("mockDurationSecs = %d, manifest mock.duration_s = %d", mockDurationSecs, mk.DurationS)
	}

	if len(phases) != len(mk.Rail) {
		t.Fatalf("%d rail phases, manifest has %d", len(phases), len(mk.Rail))
	}
	for i, p := range phases {
		r := mk.Rail[i]
		if p.Label != r.Label || p.StartMin != r.StartMin || p.EndMin != r.EndMin || p.Prompt != r.Prompt {
			t.Errorf("phase %d = {%q %d–%d %q}, manifest rail = {%q %d–%d %q}",
				i, p.Label, p.StartMin, p.EndMin, p.Prompt, r.Label, r.StartMin, r.EndMin, r.Prompt)
		}
		// The derived fields stay consistent with the manifest's window.
		if want := fmt.Sprintf("%d–%d", r.StartMin, r.EndMin); p.Range != want || p.Grow != r.EndMin-r.StartMin || p.Index != i {
			t.Errorf("phase %d: Range %q Grow %d Index %d, want %q %d %d", i, p.Range, p.Grow, p.Index, want, r.EndMin-r.StartMin, i)
		}
	}

	if len(dimensionLabels) != len(mk.Rubric.Dims) {
		t.Errorf("%d dimension labels, manifest rubric has %d dims", len(dimensionLabels), len(mk.Rubric.Dims))
	}
	for _, d := range mk.Rubric.Dims {
		if got := dimensionLabels[d.ID]; got != d.Label {
			t.Errorf("dimensionLabels[%q] = %q, manifest label %q", d.ID, got, d.Label)
		}
	}

	want := Targets{W13: mk.Targets.W13, W15: mk.Targets.W15, Pre: mk.Targets.Pre}
	if readinessTargets != want {
		t.Errorf("readinessTargets %+v, manifest mock.targets %+v", readinessTargets, mk.Targets)
	}
}
