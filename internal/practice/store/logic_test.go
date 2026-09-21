package store

import (
	"errors"
	"reflect"
	"testing"

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

func TestMarshalEnvelope(t *testing.T) {
	payload, err := marshalEnvelope("evt-1", SubjectProblemSolved, "acct-1", map[string]any{
		"problem_id": "16", "outcome": "clean", "first_solve": true, "below_clean": false,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"event_id":"evt-1"`, `"subject":"xlearn.practice.problem_solved"`, `"account_id":"acct-1"`, `"problem_id":"16"`, `"first_solve":true`, `"version":1`} {
		if !contains(payload, want) {
			t.Errorf("payload missing %q\ngot %s", want, payload)
		}
	}
}

func contains(b []byte, sub string) bool {
	return len(sub) == 0 || (len(b) >= len(sub) && indexOf(string(b), sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
