package assessment

import (
	"context"
	"errors"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

func TestProjectionHandlerDecodesAndApplies(t *testing.T) {
	var got store.ProjectionEvent
	fs := &fakeStore{applyProjection: func(_ context.Context, ev store.ProjectionEvent) (bool, error) {
		got = ev
		return true, nil
	}}
	h := &projectionHandler{store: fs, log: testLogger()}

	env := `{"event_id":"e1","subject":"xlearn.practice.problem_solved","occurred_at":"2026-09-20T10:00:00Z","account_id":"acct-1","data":{"problem_id":"16","outcome":"clean","first_solve":true}}`
	err := h.Handle(context.Background(), events.Event{
		ID:      "e1",
		Subject: "xlearn.practice.problem_solved",
		Data:    []byte(env),
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	// The handler forwards the NATS subject (not the envelope's) and the decoded fields.
	if got.EventID != "e1" || got.AccountID != "acct-1" || got.Subject != "xlearn.practice.problem_solved" {
		t.Fatalf("event ids wrong: %+v", got)
	}
	if got.ProblemID != "16" || got.Outcome != "clean" || !got.FirstSolve {
		t.Fatalf("decoded data wrong: %+v", got)
	}
	if got.OccurredAt.IsZero() || got.OccurredAt.Year() != 2026 {
		t.Fatalf("occurred_at not parsed: %v", got.OccurredAt)
	}
}

func TestProjectionHandlerDecodesRevisionTouch(t *testing.T) {
	var got store.ProjectionEvent
	fs := &fakeStore{applyProjection: func(_ context.Context, ev store.ProjectionEvent) (bool, error) {
		got = ev
		return true, nil
	}}
	h := &projectionHandler{store: fs, log: testLogger()}
	env := `{"event_id":"e2","account_id":"a","occurred_at":"2026-09-20T10:00:00Z","data":{"problem_id":"3","touch_level":1,"due_date":"2026-09-21T10:00:00Z"}}`
	if err := h.Handle(context.Background(), events.Event{ID: "e2", Subject: "xlearn.review.revision_scheduled", Data: []byte(env)}); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if got.TouchLevel != 1 || got.ProblemID != "3" {
		t.Fatalf("revision touch not decoded: %+v", got)
	}
}

func TestProjectionHandlerDuplicateIsAck(t *testing.T) {
	fs := &fakeStore{applyProjection: func(context.Context, store.ProjectionEvent) (bool, error) {
		return false, nil // already applied (inbox conflict)
	}}
	h := &projectionHandler{store: fs, log: testLogger()}
	err := h.Handle(context.Background(), events.Event{
		ID: "e1", Subject: "xlearn.review.mistake_opened",
		Data: []byte(`{"event_id":"e1","account_id":"a","data":{}}`),
	})
	if err != nil {
		t.Fatalf("duplicate delivery must ack (nil), got %v", err)
	}
}

func TestProjectionHandlerDropsMalformedEnvelope(t *testing.T) {
	fs := &fakeStore{applyProjection: func(context.Context, store.ProjectionEvent) (bool, error) {
		t.Fatal("store must not be called for a malformed envelope")
		return false, nil
	}}
	h := &projectionHandler{store: fs, log: testLogger()}
	err := h.Handle(context.Background(), events.Event{ID: "x", Subject: "xlearn.practice.x", Data: []byte("not json")})
	if err != nil {
		t.Fatalf("malformed envelope must ack (drop), got %v", err)
	}
}

func TestProjectionHandlerDropsMissingAccount(t *testing.T) {
	fs := &fakeStore{applyProjection: func(context.Context, store.ProjectionEvent) (bool, error) {
		t.Fatal("store must not be called when account_id is missing")
		return false, nil
	}}
	h := &projectionHandler{store: fs, log: testLogger()}
	err := h.Handle(context.Background(), events.Event{
		ID: "e1", Subject: "xlearn.practice.problem_solved",
		Data: []byte(`{"event_id":"e1","data":{}}`), // no account_id
	})
	if err != nil {
		t.Fatalf("missing account must ack (drop), got %v", err)
	}
}

func TestProjectionHandlerFallsBackToMsgID(t *testing.T) {
	var got store.ProjectionEvent
	fs := &fakeStore{applyProjection: func(_ context.Context, ev store.ProjectionEvent) (bool, error) {
		got = ev
		return true, nil
	}}
	h := &projectionHandler{store: fs, log: testLogger()}
	// Envelope has no event_id; the consumer-resolved Event.ID (Nats-Msg-Id) is used.
	err := h.Handle(context.Background(), events.Event{
		ID: "msg-42", Subject: "xlearn.review.mistake_opened",
		Data: []byte(`{"account_id":"acct","data":{}}`),
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if got.EventID != "msg-42" {
		t.Fatalf("event_id = %q, want fallback msg-42", got.EventID)
	}
}

func TestProjectionHandlerNaksOnStoreError(t *testing.T) {
	fs := &fakeStore{applyProjection: func(context.Context, store.ProjectionEvent) (bool, error) {
		return false, errors.New("db unavailable")
	}}
	h := &projectionHandler{store: fs, log: testLogger()}
	err := h.Handle(context.Background(), events.Event{
		ID: "e1", Subject: "xlearn.practice.problem_solved",
		Data: []byte(`{"event_id":"e1","account_id":"a","data":{}}`),
	})
	if err == nil {
		t.Fatal("a store error must return non-nil (nak for redelivery)")
	}
}
