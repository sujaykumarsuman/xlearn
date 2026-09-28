package notifications

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// The envelope fixtures live beside the decoder (internal/platform/events).
func envelopeFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "platform", "events", "testdata", "envelope", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func handleDueFixture(t *testing.T, name string) ([]reminderCall, error) {
	t.Helper()
	st := &fakeReminderStore{wrote: true}
	h := testWorker(st, stubResolver{tz: "UTC"})
	err := h.Handle(context.Background(), events.Event{ID: "nats-id", Subject: SubjectRevisionDue, Data: envelopeFixture(t, name)})
	return st.calls, err
}

// m1-02 (consumers before producers): the notifications worker accepts a v2
// revision_due with the same reminder as its v1 twin — course "dsa" — and dead-letters
// a v2 revision_due without path_slug.
func TestWorkerV1V2Twins(t *testing.T) {
	v1, err := handleDueFixture(t, "revision_due.v1.json")
	if err != nil || len(v1) != 1 {
		t.Fatalf("v1: calls=%+v err=%v", v1, err)
	}
	v2, err := handleDueFixture(t, "revision_due.v2.json")
	if err != nil || len(v2) != 1 {
		t.Fatalf("v2: calls=%+v err=%v", v2, err)
	}
	if v1[0] != v2[0] {
		t.Fatalf("v1 %+v\nv2 %+v", v1[0], v2[0])
	}
	if v1[0].pathSlug != "dsa" || v1[0].kind != ReminderKind {
		t.Fatalf("call = %+v", v1[0])
	}

	calls, err := handleDueFixture(t, "revision_due.v2-nopath.json")
	if !errors.Is(err, events.ErrInvalidEnvelope) || len(calls) != 0 {
		t.Fatalf("v2 without path_slug: calls=%+v err=%v, want ErrInvalidEnvelope (dead-letter)", calls, err)
	}
}
