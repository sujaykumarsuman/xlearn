package review

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

func testHandler(st store.Store) *practiceHandler {
	return &practiceHandler{store: st, log: slog.New(slog.NewJSONHandler(io.Discard, nil))}
}

func envJSON(t *testing.T, eventID, subject, accountID, occurredAt string, data map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"event_id":    eventID,
		"subject":     subject,
		"occurred_at": occurredAt,
		"version":     1,
		"account_id":  accountID,
		"data":        data,
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return b
}

func TestConsumerRoutesProblemSolved(t *testing.T) {
	var got struct {
		eventID, accountID, problemID, outcome string
		firstSolve                             bool
		occurredAt                             time.Time
	}
	st := &fakeStore{
		problemSolved: func(_ context.Context, eventID, accountID, problemID, outcome string, firstSolve bool, occurredAt time.Time) (int, error) {
			got.eventID, got.accountID, got.problemID, got.outcome, got.firstSolve, got.occurredAt = eventID, accountID, problemID, outcome, firstSolve, occurredAt
			return 5, nil
		},
	}
	data := envJSON(t, "evt-1", store.SubjectProblemSolved, "acct-1", "2026-09-21T12:00:00Z", map[string]any{
		"problem_id": "16", "outcome": "clean", "first_solve": true, "below_clean": false,
	})
	if err := testHandler(st).Handle(context.Background(), events.Event{Subject: store.SubjectProblemSolved, Data: data}); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if got.eventID != "evt-1" || got.accountID != "acct-1" || got.problemID != "16" || got.outcome != "clean" || !got.firstSolve {
		t.Fatalf("dispatched fields = %+v", got)
	}
	if want := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC); !got.occurredAt.Equal(want) {
		t.Fatalf("occurredAt = %v, want %v", got.occurredAt, want)
	}
}

func TestConsumerRoutesSolutionRevealedEarly(t *testing.T) {
	called := false
	st := &fakeStore{
		revealedEarly: func(_ context.Context, eventID, accountID, problemID string, _ time.Time) (int, error) {
			called = true
			if eventID != "evt-2" || accountID != "acct-1" || problemID != "42" {
				t.Fatalf("fields = %s/%s/%s", eventID, accountID, problemID)
			}
			return 1, nil
		},
	}
	data := envJSON(t, "evt-2", store.SubjectSolutionRevealedEarly, "acct-1", "2026-09-21T12:00:00Z", map[string]any{"problem_id": "42"})
	if err := testHandler(st).Handle(context.Background(), events.Event{Subject: store.SubjectSolutionRevealedEarly, Data: data}); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if !called {
		t.Fatalf("revealedEarly handler not called")
	}
}

func TestConsumerIgnoresUnhandledSubject(t *testing.T) {
	st := &fakeStore{
		problemSolved: func(context.Context, string, string, string, string, bool, time.Time) (int, error) {
			t.Fatalf("problemSolved should not be called for attempt_logged")
			return 0, nil
		},
	}
	data := envJSON(t, "evt-3", "xlearn.practice.attempt_logged", "acct-1", "2026-09-21T12:00:00Z", map[string]any{
		"problem_id": "16", "stage_reached": "attempt", "duration_s": 42,
	})
	if err := testHandler(st).Handle(context.Background(), events.Event{Subject: "xlearn.practice.attempt_logged", Data: data}); err != nil {
		t.Fatalf("unhandled subject should ack (nil), got %v", err)
	}
}

func TestConsumerDropsMalformedEnvelope(t *testing.T) {
	st := &fakeStore{}
	if err := testHandler(st).Handle(context.Background(), events.Event{Subject: store.SubjectProblemSolved, Data: []byte("not json")}); err != nil {
		t.Fatalf("malformed envelope should be dropped (nil), got %v", err)
	}
}

func TestConsumerPropagatesStoreError(t *testing.T) {
	st := &fakeStore{
		problemSolved: func(context.Context, string, string, string, string, bool, time.Time) (int, error) {
			return 0, errors.New("db down")
		},
	}
	data := envJSON(t, "evt-4", store.SubjectProblemSolved, "acct-1", "2026-09-21T12:00:00Z", map[string]any{
		"problem_id": "16", "outcome": "clean", "first_solve": true,
	})
	// A store error must surface so the consumer naks → redelivery (at-least-once).
	if err := testHandler(st).Handle(context.Background(), events.Event{Subject: store.SubjectProblemSolved, Data: data}); err == nil {
		t.Fatalf("store error should propagate for redelivery")
	}
}

func TestConsumerFallsBackToEventIDHeader(t *testing.T) {
	var gotID string
	st := &fakeStore{
		problemSolved: func(_ context.Context, eventID, _, _, _ string, _ bool, _ time.Time) (int, error) {
			gotID = eventID
			return 5, nil
		},
	}
	// Envelope with no event_id in the body; the consumer supplies e.ID (Nats-Msg-Id).
	data := envJSON(t, "", store.SubjectProblemSolved, "acct-1", "2026-09-21T12:00:00Z", map[string]any{
		"problem_id": "16", "outcome": "clean", "first_solve": true,
	})
	if err := testHandler(st).Handle(context.Background(), events.Event{ID: "hdr-id", Subject: store.SubjectProblemSolved, Data: data}); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if gotID != "hdr-id" {
		t.Fatalf("eventID = %q, want the header fallback hdr-id", gotID)
	}
}
