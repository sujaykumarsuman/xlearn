package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// assessment.event_dead_letter (mi-05): the sink writes one ids-only row per
// (durable, event_id); the same event dead-lettered by assessment's durable on both
// streams would be one row per durable name, and a repeat is a no-op. Gated on
// XLEARN_TEST_DATABASE_URL.
func TestDeadLetterIntegration(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the assessment dead-letter integration test")
	}
	ctx := context.Background()
	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	st := store.New(pool)

	var sink events.DeadLetterSink = st
	dl := events.DeadLetter{EventID: newTestUUID(), Subject: "xlearn.review.mistake_opened", Durable: "assessment",
		ErrClass: events.ErrClassOther, StreamSeq: 12, At: time.Now().UTC()}
	for range 2 {
		if err := sink.RecordDeadLetter(ctx, dl); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	rows, err := st.ListDeadLetters(ctx, 1000)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	n := 0
	for _, r := range rows {
		if r.EventID == dl.EventID {
			n++
			if r.Subject != dl.Subject || r.Durable != dl.Durable || r.ErrClass != dl.ErrClass || r.StreamSeq != 12 {
				t.Errorf("row = %+v, want %+v", r, dl)
			}
		}
	}
	if n != 1 {
		t.Fatalf("%d rows for the event, want 1 (ON CONFLICT DO NOTHING)", n)
	}
}
