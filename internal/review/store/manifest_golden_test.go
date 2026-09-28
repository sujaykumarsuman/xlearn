package store

import (
	"reflect"
	"sort"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// TestManifestGoldenMirror is the review side of the DSA manifest golden test (sprint
// m1-01; side one is internal/course/golden_test.go): the ladder, the R-SR2 pattern
// threshold, the mock-conditions levels, the owed re-attempt, the outcome values and
// the mistake categories (constants and migration CHECK) must equal the manifest.
func TestManifestGoldenMirror(t *testing.T) {
	m := coursetest.DSA(t)

	if !reflect.DeepEqual(touchDays[:], m.Revision.LadderDays) {
		t.Errorf("touchDays %v, manifest revision.ladder_days %v", touchDays, m.Revision.LadderDays)
	}
	if MaxTouchLevel != len(m.Revision.LadderDays) {
		t.Errorf("MaxTouchLevel = %d, manifest ladder has %d levels", MaxTouchLevel, len(m.Revision.LadderDays))
	}

	for level := 1; level <= MaxTouchLevel; level++ {
		band, ok := m.Revision.BandForLevel(level)
		if !ok {
			t.Fatalf("manifest has no band for level %d", level)
		}
		if isMockTouch(level) != band.MockMode {
			t.Errorf("isMockTouch(%d) = %v, manifest band %q mock_mode = %v", level, isMockTouch(level), band.Format, band.MockMode)
		}
		c, ok := band.Criterion("pattern_named_fast")
		if !ok || c.ThresholdS != NamePatternMaxSecs {
			t.Errorf("level %d: NamePatternMaxSecs = %d, manifest pattern_named_fast threshold_s = %d (present %v)",
				level, NamePatternMaxSecs, c.ThresholdS, ok)
		}
		for _, key := range []string{"correct_in_timer", "complexity_stated"} {
			if _, ok := band.Criterion(key); !ok {
				t.Errorf("level %d: manifest band lacks AutoPass criterion %q", level, key)
			}
		}
	}

	if OwedAttemptDays != m.Stages.EarlyRevealPenaltyDays {
		t.Errorf("OwedAttemptDays = %d, manifest early_reveal_penalty_days = %d", OwedAttemptDays, m.Stages.EarlyRevealPenaltyDays)
	}
	if got := m.Revision.LadderDays[OwedAttemptTouchLevel-1]; got != OwedAttemptDays {
		t.Errorf("OwedAttemptTouchLevel %d is Day %d on the manifest ladder, want Day %d", OwedAttemptTouchLevel, got, OwedAttemptDays)
	}

	grades := m.GradeIDs()
	if OutcomeClean != grades[0] {
		t.Errorf("OutcomeClean = %q, manifest best grade %q", OutcomeClean, grades[0])
	}
	var below []string
	for o := range belowCleanOutcomes {
		below = append(below, o)
	}
	sort.Strings(below)
	want := append([]string(nil), grades[1:]...)
	sort.Strings(want)
	if !reflect.DeepEqual(below, want) {
		t.Errorf("belowCleanOutcomes %v, manifest grades below clean %v", below, want)
	}

	if !reflect.DeepEqual(MistakeCategories, m.CategoryIDs()) {
		t.Errorf("MistakeCategories %v, manifest mistakes.categories %v", MistakeCategories, m.CategoryIDs())
	}
	if got := coursetest.CheckIn(t, migrationsFS, "migrations/00002_mistakes_notifications.sql", "category"); !reflect.DeepEqual(got, m.CategoryIDs()) {
		t.Errorf("mistake_entry.category CHECK %v, manifest mistakes.categories %v", got, m.CategoryIDs())
	}
}
