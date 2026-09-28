package notifications

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// The notifications durable is declared in topology.go with the worker's filter, and
// every Handles subject reaches the store (never the default branch); anything else
// that reaches the handler is acked with an ERROR, never silently.
func TestNotificationsSubjectRegistry(t *testing.T) {
	d, ok := events.LookupDurable(events.StreamReview, Durable)
	if !ok || d.Filter != SubjectRevisionDue {
		t.Fatalf("durable %s declared=%v filter=%q, want %q", Durable, ok, d.Filter, SubjectRevisionDue)
	}

	st := &fakeReminderStore{wrote: true}
	var logs bytes.Buffer
	h := NewHandler(st, nil, slog.New(slog.NewJSONHandler(&logs, nil)))
	h.now = func() time.Time { return time.Date(2026, 9, 21, 15, 0, 0, 0, time.UTC) }

	for i, s := range d.Handles {
		e := dueEvent(t, "evt-h", "acct-1")
		e.Subject = s
		if err := h.Handle(context.Background(), e); err != nil {
			t.Fatalf("handle %s: %v", s, err)
		}
		if len(st.calls) != i+1 {
			t.Fatalf("Handles subject %s did not reach the store", s)
		}
	}
	if strings.Contains(logs.String(), "unlisted subject") {
		t.Fatalf("a Handles subject hit the default branch: %s", logs.String())
	}

	e := dueEvent(t, "evt-x", "acct-1")
	e.Subject = "xlearn.review.mistake_opened"
	if err := h.Handle(context.Background(), e); err != nil {
		t.Fatalf("an unlisted subject must ack, got %v", err)
	}
	if len(st.calls) != len(d.Handles) {
		t.Fatal("an unlisted subject reached the store")
	}
	if !strings.Contains(logs.String(), `"level":"ERROR"`) || !strings.Contains(logs.String(), "unlisted subject") {
		t.Fatalf("an unlisted subject must log ERROR: %s", logs.String())
	}
}
