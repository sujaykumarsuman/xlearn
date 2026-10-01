package store

import "testing"

// TestClampOutcome pins the D27 ceiling (m1-07): coach help during the attempt records a
// self-reported clean or rough as assisted; assisted and miss are unchanged.
func TestClampOutcome(t *testing.T) {
	cases := []struct {
		value       string
		coach       bool
		want, cappd string
	}{
		{OutcomeClean, false, OutcomeClean, ""},
		{OutcomeRough, false, OutcomeRough, ""},
		{OutcomeAssisted, false, OutcomeAssisted, ""},
		{OutcomeMiss, false, OutcomeMiss, ""},
		{OutcomeClean, true, OutcomeAssisted, CappedByCoach},
		{OutcomeRough, true, OutcomeAssisted, CappedByCoach},
		{OutcomeAssisted, true, OutcomeAssisted, ""},
		{OutcomeMiss, true, OutcomeMiss, ""},
	}
	for _, c := range cases {
		got, capped := clampOutcome(c.value, c.coach)
		if got != c.want || capped != c.cappd {
			t.Errorf("clampOutcome(%s, coach=%v) = %s %q, want %s %q", c.value, c.coach, got, capped, c.want, c.cappd)
		}
	}
}

// TestAssistData pins problem_solved's `assist` object: hint = past the statement.
func TestAssistData(t *testing.T) {
	cases := []struct {
		stage        string
		coach        bool
		hint, coach2 bool
	}{
		{StageAttempt, false, false, false},
		{StageAttempt, true, false, true},
		{StageHint, false, true, false},
		{StageSolution, true, true, true},
	}
	for _, c := range cases {
		a := assistData(c.stage, c.coach)
		if len(a) != 2 || a["hint"] != c.hint || a["coach"] != c.coach2 {
			t.Errorf("assistData(%s, %v) = %v", c.stage, c.coach, a)
		}
	}
}
