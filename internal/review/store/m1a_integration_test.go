package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// m1-02 (M1a expand): review writes the event's course on every row it creates — the
// consumer paths carry the envelope's path_slug, the score path carries the scored
// touch's, the journal API and the weekly snapshot write the caller's course ('dsa'
// here; m1-03 made them course-scoped) — and sets
// outbox.account_id. The (account, path, week) unique index is valid. A non-DSA course
// ("sql") proves the course is carried, not defaulted.
func TestM1aReviewPathSlug(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the review store integration test")
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
	acct := newTestUUID()

	// paths returns the distinct path_slug values of an account's rows in table.
	paths := func(table string) []string {
		t.Helper()
		rows, err := pool.Query(ctx, `SELECT DISTINCT COALESCE(path_slug, '<null>') FROM review.`+table+` WHERE account_id = $1 ORDER BY 1`, acct)
		if err != nil {
			t.Fatalf("read %s: %v", table, err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				t.Fatalf("scan %s: %v", table, err)
			}
			out = append(out, s)
		}
		return out
	}
	eq := func(what string, got []string, want ...string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: path slugs %v, want %v", what, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("%s: path slugs %v, want %v", what, got, want)
			}
		}
	}

	// Consumer paths carry the event's course.
	if n, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, "sql", "q-1", "clean", true, time.Now()); err != nil || n != 5 {
		t.Fatalf("solved: n=%d err=%v", n, err)
	}
	if _, err := st.HandleSolutionRevealedEarly(ctx, newTestUUID(), acct, "dsa", "p-2", time.Now()); err != nil {
		t.Fatalf("revealed early: %v", err)
	}
	if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, "sql", "q-3", "miss", true, time.Now()); err != nil {
		t.Fatalf("miss: %v", err)
	}
	if _, err := st.HandleRevisionDue(ctx, newTestUUID(), acct, "sql", "revision_due", time.Now()); err != nil {
		t.Fatalf("due: %v", err)
	}
	eq("revision_item", paths("revision_item"), "dsa", "sql")
	eq("mistake_entry", paths("mistake_entry"), "sql")
	eq("reminder", paths("reminder"), "sql")

	// The score path keeps the scored touch's course: a failed re-solve re-anchors the
	// ladder and opens a mistake, both in "sql".
	var itemID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM review.revision_item WHERE account_id = $1 AND problem_id = 'q-1' AND touch_level = 1`, acct).Scan(&itemID); err != nil {
		t.Fatalf("touch: %v", err)
	}
	if _, err := st.Score(ctx, acct, itemID, store.ScoreInput{NamedPatternSecs: 600}); err != nil {
		t.Fatalf("score fail: %v", err)
	}
	var qPaths int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM review.revision_item WHERE account_id = $1 AND problem_id = 'q-1' AND path_slug = 'sql'`, acct).Scan(&qPaths); err != nil || qPaths != 5 {
		t.Fatalf("q-1 touches in sql = %d (%v), want 5", qPaths, err)
	}
	eq("mistake_entry after a failed re-solve", paths("mistake_entry"), "sql")

	// The journal API and the weekly snapshot write the course they are given.
	if _, err := st.CreateMistake(ctx, acct, "dsa", store.MistakeInput{ProblemID: "p-9"}); err != nil {
		t.Fatalf("create mistake: %v", err)
	}
	eq("mistake_entry after a manual entry", paths("mistake_entry"), "dsa", "sql")
	if err := st.SaveWeakAreaSnapshot(ctx, acct, "dsa", time.Now(), "", nil); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	eq("weak_area_snapshot", paths("weak_area_snapshot"), "dsa")

	// outbox.account_id on every row this account produced.
	var events, withAccount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE account_id::text = $1::text)
		FROM review.outbox WHERE payload_json ->> 'account_id' = $1::text`, acct).Scan(&events, &withAccount); err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	if events == 0 || withAccount != events {
		t.Fatalf("%d outbox rows, %d with account_id", events, withAccount)
	}

	// Both weak-area uniques exist and are valid (a failed CONCURRENTLY build leaves an
	// INVALID index that IF NOT EXISTS would skip).
	var valid bool
	if err := pool.QueryRow(ctx, `
		SELECT i.indisvalid AND i.indisunique FROM pg_index i
		JOIN pg_class c ON c.oid = i.indexrelid JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'review' AND c.relname = 'weak_area_snapshot_account_path_week_uq'`).Scan(&valid); err != nil || !valid {
		t.Fatalf("weak_area_snapshot_account_path_week_uq valid+unique = %v (%v)", valid, err)
	}
	var v1Unique int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM pg_constraint
		WHERE conrelid = 'review.weak_area_snapshot'::regclass AND contype = 'u'`).Scan(&v1Unique); err != nil || v1Unique != 1 {
		t.Fatalf("v1 UNIQUE (account_id, week_of) constraints = %d (%v), want 1 (it stays until M1c)", v1Unique, err)
	}
}
