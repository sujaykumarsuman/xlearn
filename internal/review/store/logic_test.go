package store

import (
	"testing"
	"time"
)

// The auto-score rule and the five-touch day mapping are the product's core
// invariants (R-SR2 / R-SR1); these pure tests pin them without a database so CI
// covers them.

func TestAutoPass(t *testing.T) {
	cases := []struct {
		name string
		in   ScoreInput
		want bool
	}{
		{"all three pass (pattern < 2 min)", ScoreInput{NamedPatternSecs: 119, SolvedInTimer: true, StatedComplexity: true}, true},
		{"pattern exactly 2 min fails", ScoreInput{NamedPatternSecs: 120, SolvedInTimer: true, StatedComplexity: true}, false},
		{"pattern over 2 min fails", ScoreInput{NamedPatternSecs: 300, SolvedInTimer: true, StatedComplexity: true}, false},
		{"not solved in timer fails", ScoreInput{NamedPatternSecs: 30, SolvedInTimer: false, StatedComplexity: true}, false},
		{"complexity not stated fails", ScoreInput{NamedPatternSecs: 30, SolvedInTimer: true, StatedComplexity: false}, false},
		{"named instantly passes", ScoreInput{NamedPatternSecs: 0, SolvedInTimer: true, StatedComplexity: true}, true},
	}
	for _, c := range cases {
		if got := AutoPass(c.in); got != c.want {
			t.Errorf("%s: AutoPass(%+v) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}

func TestTouchDueDate(t *testing.T) {
	anchor := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	wantDays := map[int]int{1: 1, 2: 3, 3: 7, 4: 21, 5: 45}
	for level, days := range wantDays {
		got := touchDueDate(level, anchor)
		want := anchor.AddDate(0, 0, days)
		if !got.Equal(want) {
			t.Errorf("touchDueDate(%d) = %v, want %v (anchor + %dd)", level, got, want, days)
		}
	}
	// Out-of-range levels return the anchor unchanged (defensive).
	if got := touchDueDate(0, anchor); !got.Equal(anchor) {
		t.Errorf("touchDueDate(0) = %v, want anchor", got)
	}
	if got := touchDueDate(6, anchor); !got.Equal(anchor) {
		t.Errorf("touchDueDate(6) = %v, want anchor", got)
	}
}

func TestIsMockTouch(t *testing.T) {
	for _, level := range []int{1, 2, 3} {
		if isMockTouch(level) {
			t.Errorf("isMockTouch(%d) = true, want false", level)
		}
	}
	for _, level := range []int{4, 5} {
		if !isMockTouch(level) {
			t.Errorf("isMockTouch(%d) = false, want true", level)
		}
	}
}

func TestMarshalEnvelope(t *testing.T) {
	payload, err := marshalEnvelope("evt-1", SubjectRevisionScheduled, "acct-1", map[string]any{
		"problem_id": "16", "touch_level": 2, "due_date": "2026-09-24T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{
		`"event_id":"evt-1"`,
		`"subject":"xlearn.review.revision_scheduled"`,
		`"account_id":"acct-1"`,
		`"problem_id":"16"`,
		`"touch_level":2`,
		`"version":1`,
	} {
		if !contains(payload, want) {
			t.Errorf("payload missing %q\ngot %s", want, payload)
		}
	}
}

func contains(b []byte, sub string) bool {
	s := string(b)
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return len(sub) == 0
}
