package review

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// The envelope fixtures live beside the decoder (internal/platform/events).
func envelopeFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "platform", "events", "testdata", "envelope", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

// storeCall is what the practice handler passed to the store.
type storeCall struct {
	method, eventID, accountID, pathSlug, problemID, outcome string
	firstSolve                                               bool
	occurredAt                                               time.Time
}

func recordingStore(calls *[]storeCall) *fakeStore {
	return &fakeStore{
		problemSolved: func(_ context.Context, eventID, accountID, pathSlug, problemID, outcome string, firstSolve bool, occurredAt time.Time) (int, error) {
			*calls = append(*calls, storeCall{"problem_solved", eventID, accountID, pathSlug, problemID, outcome, firstSolve, occurredAt})
			return 5, nil
		},
		revealedEarly: func(_ context.Context, eventID, accountID, pathSlug, problemID string, occurredAt time.Time) (int, error) {
			*calls = append(*calls, storeCall{"revealed_early", eventID, accountID, pathSlug, problemID, "", false, occurredAt})
			return 1, nil
		},
	}
}

func handlePracticeFixture(t *testing.T, subject, name string) ([]storeCall, error) {
	t.Helper()
	var calls []storeCall
	err := testHandler(recordingStore(&calls)).Handle(context.Background(),
		events.Event{ID: "nats-id", Subject: subject, Data: envelopeFixture(t, name)})
	return calls, err
}

// m1-02 (consumers before producers): review's practice consumer accepts the v2
// envelope with the same store call as its v1 twin — course "dsa" — and dead-letters a
// v2 event without path_slug. The store writes pathSlug on every row it creates (the
// store integration test proves the rows).
func TestPracticeHandlerV1V2Twins(t *testing.T) {
	for stem, subject := range map[string]string{
		"problem_solved":          store.SubjectProblemSolved,
		"solution_revealed_early": store.SubjectSolutionRevealedEarly,
	} {
		t.Run(stem, func(t *testing.T) {
			v1, err := handlePracticeFixture(t, subject, stem+".v1.json")
			if err != nil || len(v1) != 1 {
				t.Fatalf("v1: calls=%+v err=%v", v1, err)
			}
			v2, err := handlePracticeFixture(t, subject, stem+".v2.json")
			if err != nil || len(v2) != 1 {
				t.Fatalf("v2: calls=%+v err=%v", v2, err)
			}
			if v1[0] != v2[0] {
				t.Fatalf("v1 %+v\nv2 %+v", v1[0], v2[0])
			}
			if v1[0].pathSlug != "dsa" {
				t.Fatalf("path slug = %q, want dsa", v1[0].pathSlug)
			}

			calls, err := handlePracticeFixture(t, subject, stem+".v2-nopath.json")
			if !errors.Is(err, events.ErrInvalidEnvelope) || len(calls) != 0 {
				t.Fatalf("v2 without path_slug: calls=%+v err=%v, want ErrInvalidEnvelope (dead-letter)", calls, err)
			}
		})
	}
}

// m1-07 (v1.7.0): problem_solved's additive `assist` object (D27) changes nothing for
// review's practice consumer — the same store call as the plain v2 twin.
func TestPracticeHandlerProblemSolvedAssist(t *testing.T) {
	withAssist, err := handlePracticeFixture(t, store.SubjectProblemSolved, "problem_solved.v2-assist.json")
	if err != nil || len(withAssist) != 1 {
		t.Fatalf("v2 + assist: calls=%+v err=%v", withAssist, err)
	}
	v2, _ := handlePracticeFixture(t, store.SubjectProblemSolved, "problem_solved.v2.json")
	if len(v2) != 1 || v2[0] != withAssist[0] {
		t.Fatalf("v2 %+v\nv2+assist %+v", v2, withAssist)
	}
}

// An unknown future version with extra fields is still processed; review's ignored
// subject (attempt_logged) is acked without decoding, whatever its version.
func TestPracticeHandlerUnknownVersionAndIgnored(t *testing.T) {
	v3, err := handlePracticeFixture(t, store.SubjectProblemSolved, "problem_solved.v3-extra.json")
	if err != nil || len(v3) != 1 || v3[0].pathSlug != "dsa" || v3[0].problemID != "16" {
		t.Fatalf("v3: calls=%+v err=%v", v3, err)
	}
	for _, name := range []string{"attempt_logged.v1.json", "attempt_logged.v2.json", "attempt_logged.v2-nopath.json"} {
		calls, err := handlePracticeFixture(t, "xlearn.practice.attempt_logged", name)
		if err != nil || len(calls) != 0 {
			t.Fatalf("%s: calls=%+v err=%v, want ack without a store call", name, calls, err)
		}
	}
}
