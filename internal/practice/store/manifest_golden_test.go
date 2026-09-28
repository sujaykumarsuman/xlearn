package store

import (
	"reflect"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// TestManifestGoldenMirror is the practice side of the DSA manifest golden test
// (sprint m1-01; side one is internal/course/golden_test.go): practice's own timers,
// penalty, outcome values and migration CHECK lists must equal the embedded manifest.
// A deliberate change to one side must change the other in the same PR.
func TestManifestGoldenMirror(t *testing.T) {
	m := coursetest.DSA(t)

	secs := func(s int) time.Duration { return time.Duration(s) * time.Second }
	if AttemptTimer != secs(m.Stages.Attempt.DurationS) {
		t.Errorf("AttemptTimer = %v, manifest stages.attempt.duration_s = %ds", AttemptTimer, m.Stages.Attempt.DurationS)
	}
	if HintTimer != secs(m.Stages.Hint.DurationS) {
		t.Errorf("HintTimer = %v, manifest stages.hint.duration_s = %ds", HintTimer, m.Stages.Hint.DurationS)
	}
	if EarlyRevealPenaltyDays != m.Stages.EarlyRevealPenaltyDays {
		t.Errorf("EarlyRevealPenaltyDays = %d, manifest = %d", EarlyRevealPenaltyDays, m.Stages.EarlyRevealPenaltyDays)
	}

	outcomes := []string{OutcomeClean, OutcomeRough, OutcomeAssisted, OutcomeMiss}
	if !reflect.DeepEqual(outcomes, m.GradeIDs()) {
		t.Errorf("outcome constants %v, manifest grades %v", outcomes, m.GradeIDs())
	}
	if got := coursetest.CheckIn(t, migrationsFS, "migrations/00001_init.sql", "last_outcome"); !reflect.DeepEqual(got, m.GradeIDs()) {
		t.Errorf("last_outcome CHECK %v, manifest grades %v", got, m.GradeIDs())
	}

	// The stage roles are universal (the manifest configures them, never renames them).
	if !reflect.DeepEqual(contentStages, course.StageRoles) {
		t.Errorf("contentStages %v, universal stage roles %v", contentStages, course.StageRoles)
	}
	if got := coursetest.CheckIn(t, migrationsFS, "migrations/00001_init.sql", "stage_reached"); !reflect.DeepEqual(got, course.StageRoles) {
		t.Errorf("stage_reached CHECK %v, universal stage roles %v", got, course.StageRoles)
	}
}
