package assessment

import (
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
)

func TestComputeRailPhaseBoundaries(t *testing.T) {
	start := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name        string
		elapsedSecs int
		wantPhase   int
		wantRemain  int
		wantOver    bool
	}{
		{"start of clarify", 0, 0, 2700, false},
		{"mid clarify", 60, 0, 2640, false},
		{"exactly 5m -> brute force", 5 * 60, 1, 2400, false},
		{"7m in brute force", 7 * 60, 1, 2280, false},
		{"exactly 10m -> observation", 10 * 60, 2, 2100, false},
		{"12m observation", 12 * 60, 2, 2700 - 720, false},
		{"exactly 18m -> code", 18 * 60, 3, 2700 - 1080, false},
		{"exactly 33m -> trace", 33 * 60, 4, 2700 - 1980, false},
		{"exactly 40m -> complexity", 40 * 60, 5, 2700 - 2400, false},
		{"44m still complexity", 44 * 60, 5, 60, false},
		{"exactly 45m clamps + overtime", 45 * 60, 5, 0, true},
		{"past deadline clamps at 45", 60 * 60, 5, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			now := start.Add(time.Duration(tc.elapsedSecs) * time.Second)
			rail := computeRail(start, now)
			if rail.PhaseIndex != tc.wantPhase {
				t.Fatalf("phase = %d, want %d", rail.PhaseIndex, tc.wantPhase)
			}
			if rail.RemainingSeconds != tc.wantRemain {
				t.Fatalf("remaining = %d, want %d", rail.RemainingSeconds, tc.wantRemain)
			}
			if rail.Overtime != tc.wantOver {
				t.Fatalf("overtime = %v, want %v", rail.Overtime, tc.wantOver)
			}
			if rail.PhaseLabel != phases[tc.wantPhase].Label {
				t.Fatalf("label = %q, want %q", rail.PhaseLabel, phases[tc.wantPhase].Label)
			}
			// The phase states must be past for < idx, current at idx, upcoming after.
			for i, ps := range rail.Phases {
				want := "upcoming"
				switch {
				case i < tc.wantPhase:
					want = "past"
				case i == tc.wantPhase:
					want = "current"
				}
				if ps.State != want {
					t.Fatalf("phase[%d].State = %q, want %q", i, ps.State, want)
				}
			}
		})
	}
}

func TestComputeRailClampsNegativeElapsed(t *testing.T) {
	start := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	// A clock skew where now < start must not produce a negative elapsed / >45:00 remaining.
	rail := computeRail(start, start.Add(-30*time.Second))
	if rail.ElapsedSeconds != 0 {
		t.Fatalf("elapsed = %d, want 0", rail.ElapsedSeconds)
	}
	if rail.RemainingSeconds != 2700 {
		t.Fatalf("remaining = %d, want 2700", rail.RemainingSeconds)
	}
	if rail.PhaseIndex != 0 {
		t.Fatalf("phase = %d, want 0", rail.PhaseIndex)
	}
}

func TestPhasesRailIsSixPhasesAndFullWindow(t *testing.T) {
	ps := Phases()
	if len(ps) != 6 {
		t.Fatalf("phases = %d, want 6", len(ps))
	}
	if ps[0].StartMin != 0 || ps[len(ps)-1].EndMin != 45 {
		t.Fatalf("rail must span 0..45 minutes, got %d..%d", ps[0].StartMin, ps[len(ps)-1].EndMin)
	}
	grow := 0
	for i, p := range ps {
		if p.Index != i {
			t.Fatalf("phase %d has index %d", i, p.Index)
		}
		if p.EndMin <= p.StartMin {
			t.Fatalf("phase %d has non-positive span", i)
		}
		grow += p.Grow
	}
	// Grow weights are the minute spans; they must sum to the full 45-minute window.
	if grow != 45 {
		t.Fatalf("grow weights sum = %d, want 45", grow)
	}
}

func TestValidateRubric(t *testing.T) {
	full := func() map[string]int {
		m := map[string]int{}
		for _, d := range store.Dimensions {
			m[d] = 3
		}
		return m
	}

	if err := store.ValidateRubric(full()); err != nil {
		t.Fatalf("full valid rubric rejected: %v", err)
	}

	// Missing a dimension.
	m := full()
	delete(m, "complexity")
	if err := store.ValidateRubric(m); err == nil {
		t.Fatal("missing dimension accepted")
	}

	// Extra/unknown dimension (7 keys but one unknown).
	m = full()
	delete(m, "complexity")
	m["unknown_dim"] = 3
	if err := store.ValidateRubric(m); err == nil {
		t.Fatal("unknown dimension accepted")
	}

	// Out-of-range scores.
	for _, bad := range []int{0, 6, -1} {
		m = full()
		m["communication"] = bad
		if err := store.ValidateRubric(m); err == nil {
			t.Fatalf("score %d accepted", bad)
		}
	}

	// Too many keys.
	m = full()
	m["eighth"] = 4
	if err := store.ValidateRubric(m); err == nil {
		t.Fatal("eight keys accepted")
	}
}

func TestTotalScore(t *testing.T) {
	m := map[string]int{
		"communication":         4,
		"problem_understanding": 4,
		"brute_force":           3,
		"optimisation":          3,
		"code_quality":          4,
		"edge_cases":            3,
		"complexity":            3,
	}
	if got := store.TotalScore(m); got != 24 {
		t.Fatalf("total = %d, want 24", got)
	}
	// All fives is the /35 ceiling.
	for _, d := range store.Dimensions {
		m[d] = 5
	}
	if got := store.TotalScore(m); got != 35 {
		t.Fatalf("total = %d, want 35", got)
	}
}
