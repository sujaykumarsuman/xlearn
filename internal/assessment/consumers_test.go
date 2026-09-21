package assessment

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

func TestProjectionHandlerRecordsAndAcks(t *testing.T) {
	var gotEventID, gotSubject, gotAccount string
	var gotData []byte
	fs := &fakeStore{recordProjection: func(_ context.Context, eventID, subject, accountID string, data []byte) (bool, error) {
		gotEventID, gotSubject, gotAccount, gotData = eventID, subject, accountID, data
		return true, nil
	}}
	h := &projectionHandler{store: fs, log: testLogger()}

	env := `{"event_id":"e1","subject":"xlearn.practice.problem_solved","account_id":"acct-1","data":{"problem_id":"16","outcome":"clean"}}`
	err := h.Handle(context.Background(), events.Event{
		ID:      "e1",
		Subject: "xlearn.practice.problem_solved",
		Data:    []byte(env),
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if gotEventID != "e1" || gotAccount != "acct-1" {
		t.Fatalf("recorded event_id=%q account=%q", gotEventID, gotAccount)
	}
	// The handler passes the NATS subject (not the envelope's), and the raw data.
	if gotSubject != "xlearn.practice.problem_solved" {
		t.Fatalf("subject = %q", gotSubject)
	}
	if !strings.Contains(string(gotData), "problem_id") {
		t.Fatalf("data not forwarded: %s", gotData)
	}
}

func TestProjectionHandlerDuplicateIsAck(t *testing.T) {
	fs := &fakeStore{recordProjection: func(context.Context, string, string, string, []byte) (bool, error) {
		return false, nil // already consumed (inbox conflict)
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
	fs := &fakeStore{recordProjection: func(context.Context, string, string, string, []byte) (bool, error) {
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
	fs := &fakeStore{recordProjection: func(context.Context, string, string, string, []byte) (bool, error) {
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
	var gotEventID string
	fs := &fakeStore{recordProjection: func(_ context.Context, eventID, _, _ string, _ []byte) (bool, error) {
		gotEventID = eventID
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
	if gotEventID != "msg-42" {
		t.Fatalf("event_id = %q, want fallback msg-42", gotEventID)
	}
}

func TestProjectionHandlerNaksOnStoreError(t *testing.T) {
	fs := &fakeStore{recordProjection: func(context.Context, string, string, string, []byte) (bool, error) {
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
