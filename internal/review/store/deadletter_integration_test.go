package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// review.event_dead_letter (mi-05): the sink writes one ids-only row per
// (durable, event_id), a repeat is a no-op, a zero stream_seq is stored as NULL, and
// the on-demand read returns newest first. Gated on XLEARN_TEST_DATABASE_URL.
func TestDeadLetterIntegration(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the review dead-letter integration test")
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

	var sink events.DeadLetterSink = st // the store is the consumers' sink
	older := events.DeadLetter{EventID: newTestUUID(), Subject: "xlearn.practice.problem_solved", Durable: "review",
		ErrClass: events.ErrClassDB, StreamSeq: 7, At: time.Now().Add(-time.Minute).UTC()}
	newer := events.DeadLetter{EventID: newTestUUID(), Subject: "xlearn.review.revision_due", Durable: "notifications",
		ErrClass: events.ErrClassTimeout, At: time.Now().UTC()}
	for _, dl := range []events.DeadLetter{older, newer, older} { // older twice: idempotent
		if err := sink.RecordDeadLetter(ctx, dl); err != nil {
			t.Fatalf("record %s: %v", dl.EventID, err)
		}
	}

	rows, err := st.ListDeadLetters(ctx, 1000)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	seen := map[string]int{}
	iNewer, iOlder := -1, -1
	for i, r := range rows {
		seen[r.EventID]++
		switch r.EventID {
		case newer.EventID:
			iNewer = i
			if r.StreamSeq != 0 || r.ErrClass != events.ErrClassTimeout || r.Durable != "notifications" {
				t.Errorf("newer row = %+v (a zero stream_seq must round-trip as NULL/0)", r)
			}
		case older.EventID:
			iOlder = i
			if r.StreamSeq != 7 || r.Subject != older.Subject || r.ErrClass != events.ErrClassDB {
				t.Errorf("older row = %+v", r)
			}
		}
	}
	if seen[older.EventID] != 1 || seen[newer.EventID] != 1 {
		t.Fatalf("row counts %v, want one row each (ON CONFLICT DO NOTHING)", seen)
	}
	if iNewer < 0 || iOlder < 0 || iNewer > iOlder {
		t.Fatalf("ListDeadLetters must be newest first: newer at %d, older at %d", iNewer, iOlder)
	}
}
