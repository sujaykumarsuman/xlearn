package store

import (
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
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

// m1-03 (M1b): review emits the v2 envelope, and every review subject is
// course-scoped, so NewEnvelope refuses one without a course (insertEvent can't write a
// course-less event). The integration test checks the rows themselves.
func TestEnvelopeIsV2AndCourseScoped(t *testing.T) {
	if eventVersion != events.EnvelopeV2 {
		t.Fatalf("eventVersion = %d, want %d", eventVersion, events.EnvelopeV2)
	}
	for _, subject := range []string{SubjectRevisionScheduled, SubjectRevisionDue, SubjectMistakeOpened, SubjectMistakeClosed} {
		if !events.CourseScoped(subject) {
			t.Errorf("%s is not course-scoped in topology.go", subject)
		}
		if _, err := events.NewEnvelope(eventVersion, "evt-1", subject, "acct-1", "", time.Now(), map[string]any{}); err == nil {
			t.Errorf("%s: a v2 envelope without path_slug was accepted", subject)
		}
	}
}

func TestIsBelowClean(t *testing.T) {
	for _, o := range []string{"rough", "assisted", "miss"} {
		if !isBelowClean(o) {
			t.Errorf("isBelowClean(%q) = false, want true", o)
		}
	}
	// clean + malformed/empty must NOT open a mistake.
	for _, o := range []string{"clean", "", "CLEAN", "unknown"} {
		if isBelowClean(o) {
			t.Errorf("isBelowClean(%q) = true, want false", o)
		}
	}
}

func TestValidMistakeCategory(t *testing.T) {
	if len(MistakeCategories) != 8 {
		t.Fatalf("MistakeCategories has %d entries, want 8 (R-MJ2)", len(MistakeCategories))
	}
	for _, c := range MistakeCategories {
		if !ValidMistakeCategory(c) {
			t.Errorf("ValidMistakeCategory(%q) = false, want true", c)
		}
	}
	for _, c := range []string{"", "misspelled", "off-by-one"} {
		if ValidMistakeCategory(c) {
			t.Errorf("ValidMistakeCategory(%q) = true, want false", c)
		}
	}
	if MistakeCloseThreshold != 2 {
		t.Errorf("MistakeCloseThreshold = %d, want 2 (R-MJ4)", MistakeCloseThreshold)
	}
}
