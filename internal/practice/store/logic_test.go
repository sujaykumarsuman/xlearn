package store

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/practice/store/gen"
)

// The gating ladder is the product's core invariant (R-PF1); these pure tests pin
// it without a database so CI covers it.

func TestNextStage(t *testing.T) {
	cases := []struct {
		current string
		want    string
		wantErr error
	}{
		{StageAttempt, StageHint, nil},
		{StageHint, StageSolution, nil},
		{StageSolution, "", ErrNothingToReveal},
	}
	for _, c := range cases {
		got, err := nextStage(c.current)
		if !errors.Is(err, c.wantErr) {
			t.Errorf("nextStage(%q) err = %v, want %v", c.current, err, c.wantErr)
		}
		if got != c.want {
			t.Errorf("nextStage(%q) = %q, want %q", c.current, got, c.want)
		}
	}
}

func TestUnlockedFromEvents(t *testing.T) {
	ev := func(stage string) gen.PracticeStageEvent { return gen.PracticeStageEvent{Stage: stage} }

	cases := []struct {
		name   string
		events []gen.PracticeStageEvent
		want   []string
	}{
		{"no events still unlocks the statement", nil, []string{StageAttempt}},
		{"attempt only", []gen.PracticeStageEvent{ev(StageAttempt)}, []string{StageAttempt}},
		{"through hint", []gen.PracticeStageEvent{ev(StageAttempt), ev(StageHint)}, []string{StageAttempt, StageHint}},
		{"through solution", []gen.PracticeStageEvent{ev(StageAttempt), ev(StageHint), ev(StageSolution)}, []string{StageAttempt, StageHint, StageSolution}},
		{"ordered regardless of event order", []gen.PracticeStageEvent{ev(StageSolution), ev(StageAttempt)}, []string{StageAttempt, StageSolution}},
	}
	for _, c := range cases {
		got := unlockedFromEvents(c.events)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: unlockedFromEvents = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestValidOutcome(t *testing.T) {
	for _, v := range []string{OutcomeClean, OutcomeRough, OutcomeAssisted, OutcomeMiss} {
		if !validOutcome(v) {
			t.Errorf("validOutcome(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"", "great", "CLEAN", "done"} {
		if validOutcome(v) {
			t.Errorf("validOutcome(%q) = true, want false", v)
		}
	}
}

func TestDefaultState(t *testing.T) {
	st := defaultState("16")
	if st.ProblemID != "16" || st.Status != "available" {
		t.Fatalf("defaultState = %+v", st)
	}
	if !reflect.DeepEqual(st.UnlockedStages, []string{StageAttempt}) {
		t.Errorf("default unlocked = %v, want [attempt]", st.UnlockedStages)
	}
	if st.Timer != nil {
		t.Errorf("default state should have no timer")
	}
}

// m1-03 (M1b): practice emits the v2 envelope, and every practice subject is
// course-scoped, so NewEnvelope refuses one without a course (insertEvent can't write a
// course-less event). The integration test checks the rows themselves.
func TestEnvelopeIsV2AndCourseScoped(t *testing.T) {
	if eventVersion != events.EnvelopeV2 {
		t.Fatalf("eventVersion = %d, want %d", eventVersion, events.EnvelopeV2)
	}
	for _, subject := range []string{SubjectProblemSolved, SubjectAttemptLogged, SubjectSolutionRevealedEarly} {
		if !events.CourseScoped(subject) {
			t.Errorf("%s is not course-scoped in topology.go", subject)
		}
		if _, err := events.NewEnvelope(eventVersion, "evt-1", subject, "acct-1", "", time.Now(), map[string]any{}); err == nil {
			t.Errorf("%s: a v2 envelope without path_slug was accepted", subject)
		}
	}
}
