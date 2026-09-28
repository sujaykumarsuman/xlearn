//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	assessmentstore "github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	reviewstore "github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// TestDeadLetterRowsE2E drives the mi-05 dead-letter hook over a real JetStream into
// the real review and assessment stores: a handler that fails every delivery is
// retried to MaxDeliver, then leaves exactly one ids-only event_dead_letter row and is
// terminated (ADR-0035 §1.2). WithMaxDeliver(2) stands in for the production 100.
func TestDeadLetterRowsE2E(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the dead-letter e2e")
	}
	natsURL := startEmbeddedNATS(t) // its own broker: never touches the core loop's streams
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mustExec(t, dsn, "CREATE SCHEMA IF NOT EXISTS review", "CREATE SCHEMA IF NOT EXISTS assessment")
	if err := reviewstore.Migrate(ctx, dsn, log); err != nil {
		t.Fatalf("migrate review: %v", err)
	}
	if err := assessmentstore.Migrate(ctx, dsn, log); err != nil {
		t.Fatalf("migrate assessment: %v", err)
	}
	rStore := reviewstore.New(mustPool(t, ctx, dsn))
	aStore := assessmentstore.New(mustPool(t, ctx, dsn))

	cases := []struct {
		name, owner, stream, subject, consumerSvc, durable, filter string
		sink                                                       events.DeadLetterSink
		list                                                       func(context.Context, int32) ([]events.DeadLetter, error)
	}{
		{"review", "practice", events.StreamPractice, "xlearn.practice.problem_solved", "review", "review", "xlearn.practice.*", rStore, rStore.ListDeadLetters},
		{"assessment", "review", events.StreamReview, "xlearn.review.revision_due", "assessment", "assessment", "xlearn.review.*", aStore, aStore.ListDeadLetters},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eventID := newUUID(t)
			pub := mustPublisher(t, ctx, natsURL, c.owner, c.stream, log)
			data, _ := json.Marshal(map[string]any{"event_id": eventID, "subject": c.subject, "account_id": newUUID(t)})
			if err := pub.Publish(ctx, events.Event{ID: eventID, Subject: c.subject, Data: data}); err != nil {
				t.Fatalf("publish: %v", err)
			}

			cons := mustConsumer(t, ctx, natsURL, c.consumerSvc, c.stream, log)
			poison := handlerFunc(func(context.Context, events.Event) error { return errors.New("poison") })
			sub, err := cons.Subscribe(ctx, c.durable, c.filter, poison, events.WithDeadLetter(c.sink), events.WithMaxDeliver(2))
			startSub(t, sub, err)

			var got events.DeadLetter
			eventually(t, "the last failing delivery leaves an event_dead_letter row", func() bool {
				rows, lerr := c.list(ctx, 50)
				if lerr != nil {
					t.Fatalf("list dead letters: %v", lerr)
				}
				for _, r := range rows {
					if r.EventID == eventID {
						got = r
						return true
					}
				}
				return false
			})
			if got.Subject != c.subject || got.Durable != c.durable || got.ErrClass != events.ErrClassOther ||
				got.StreamSeq == 0 || got.At.IsZero() {
				t.Fatalf("dead-letter row = %+v", got)
			}
			// Idempotent: recording the same (durable, event_id) again stays one row.
			if err := c.sink.RecordDeadLetter(ctx, got); err != nil {
				t.Fatalf("re-record: %v", err)
			}
			rows, _ := c.list(ctx, 50)
			n := 0
			for _, r := range rows {
				if r.EventID == eventID {
					n++
				}
			}
			if n != 1 {
				t.Fatalf("%d rows for %s, want 1 (ON CONFLICT DO NOTHING)", n, eventID)
			}
		})
	}
}

type handlerFunc func(context.Context, events.Event) error

func (f handlerFunc) Handle(ctx context.Context, e events.Event) error { return f(ctx, e) }
