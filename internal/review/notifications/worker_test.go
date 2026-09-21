package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

type fakeReminderStore struct {
	calls []reminderCall
	wrote bool
	err   error
}

type reminderCall struct {
	eventID, accountID, kind string
	dueAt                    time.Time
}

func (f *fakeReminderStore) HandleRevisionDue(_ context.Context, eventID, accountID, kind string, dueAt time.Time) (bool, error) {
	f.calls = append(f.calls, reminderCall{eventID, accountID, kind, dueAt})
	if f.err != nil {
		return false, f.err
	}
	return f.wrote, nil
}

type stubResolver struct {
	tz     string
	budget []byte
	err    error
}

func (s stubResolver) ResolveAccount(context.Context, string) (string, []byte, error) {
	return s.tz, s.budget, s.err
}

func testWorker(store ReminderStore, accounts AccountResolver) *Handler {
	h := NewHandler(store, accounts, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	h.now = func() time.Time { return time.Date(2026, 9, 21, 15, 0, 0, 0, time.UTC) }
	return h
}

func dueEvent(t *testing.T, eventID, accountID string) events.Event {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"event_id":   eventID,
		"subject":    "xlearn.review.revision_due",
		"account_id": accountID,
		"data":       map[string]any{"problem_id": "16", "touch_level": 1},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return events.Event{Subject: "xlearn.review.revision_due", Data: data}
}

func TestWorkerWritesReminder(t *testing.T) {
	st := &fakeReminderStore{wrote: true}
	h := testWorker(st, stubResolver{tz: "UTC"}) // no budget → immediate
	if err := h.Handle(context.Background(), dueEvent(t, "evt-1", "acct-1")); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(st.calls) != 1 {
		t.Fatalf("store calls = %d, want 1", len(st.calls))
	}
	c := st.calls[0]
	if c.eventID != "evt-1" || c.accountID != "acct-1" || c.kind != ReminderKind {
		t.Fatalf("call = %+v, want evt-1/acct-1/%s", c, ReminderKind)
	}
	if !c.dueAt.Equal(time.Date(2026, 9, 21, 15, 0, 0, 0, time.UTC)) {
		t.Fatalf("dueAt = %v, want immediate (no budget)", c.dueAt)
	}
}

func TestWorkerDedupeAcks(t *testing.T) {
	st := &fakeReminderStore{wrote: false} // duplicate delivery
	h := testWorker(st, stubResolver{tz: "UTC"})
	if err := h.Handle(context.Background(), dueEvent(t, "evt-1", "acct-1")); err != nil {
		t.Fatalf("a deduped delivery must ack (nil), got %v", err)
	}
}

func TestWorkerStoreErrorNaks(t *testing.T) {
	st := &fakeReminderStore{err: errors.New("db down")}
	h := testWorker(st, stubResolver{tz: "UTC"})
	if err := h.Handle(context.Background(), dueEvent(t, "evt-1", "acct-1")); err == nil {
		t.Fatalf("a store error must propagate for redelivery")
	}
}

func TestWorkerResolveFailureStillWrites(t *testing.T) {
	// An account-resolve failure must not drop the reminder — it schedules immediately.
	st := &fakeReminderStore{wrote: true}
	h := testWorker(st, stubResolver{err: errors.New("identity down")})
	if err := h.Handle(context.Background(), dueEvent(t, "evt-1", "acct-1")); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(st.calls) != 1 {
		t.Fatalf("store calls = %d, want 1 (reminder still written)", len(st.calls))
	}
}

func TestWorkerDropsMalformed(t *testing.T) {
	st := &fakeReminderStore{}
	h := testWorker(st, stubResolver{tz: "UTC"})
	if err := h.Handle(context.Background(), events.Event{Subject: "xlearn.review.revision_due", Data: []byte("not json")}); err != nil {
		t.Fatalf("malformed envelope should be dropped (nil), got %v", err)
	}
	if len(st.calls) != 0 {
		t.Fatalf("store called %d times for malformed event, want 0", len(st.calls))
	}
}

func TestWorkerDropsMissingAccount(t *testing.T) {
	st := &fakeReminderStore{}
	h := testWorker(st, stubResolver{tz: "UTC"})
	data, _ := json.Marshal(map[string]any{"event_id": "evt-1", "subject": "xlearn.review.revision_due", "data": map[string]any{"problem_id": "16"}})
	if err := h.Handle(context.Background(), events.Event{Subject: "xlearn.review.revision_due", Data: data}); err != nil {
		t.Fatalf("missing account_id should drop (nil), got %v", err)
	}
	if len(st.calls) != 0 {
		t.Fatalf("store called for a no-account event, want 0")
	}
}
