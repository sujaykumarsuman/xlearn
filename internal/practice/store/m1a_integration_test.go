package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

// m1-02 (M1a expand): practice writes path_slug on the problem state, denormalises the
// attempt's account / course / problem, and sets outbox.account_id on every event.
func TestM1aPracticeDualWrite(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the practice store integration test")
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
	const problem = "m1a-7"

	if _, err := st.StartAttempt(ctx, acct, problem, "dsa"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := st.Reveal(ctx, acct, problem); err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if _, err := st.LogOutcome(ctx, acct, problem, store.OutcomeClean); err != nil {
		t.Fatalf("log outcome: %v", err)
	}

	var statePath string
	if err := pool.QueryRow(ctx, `SELECT path_slug FROM practice.user_problem_state WHERE account_id = $1 AND problem_id = $2`,
		acct, problem).Scan(&statePath); err != nil || statePath != "dsa" {
		t.Fatalf("user_problem_state.path_slug = %q (%v), want dsa", statePath, err)
	}

	var n, bad int
	if err := pool.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE a.account_id IS DISTINCT FROM s.account_id
		                           OR a.path_slug IS DISTINCT FROM s.path_slug
		                           OR a.problem_id IS DISTINCT FROM s.problem_id)
		FROM practice.attempt a JOIN practice.user_problem_state s ON s.id = a.user_problem_state_id
		WHERE s.account_id = $1`, acct).Scan(&n, &bad); err != nil {
		t.Fatalf("read attempts: %v", err)
	}
	if n == 0 || bad != 0 {
		t.Fatalf("%d attempts, %d without the denormalised account/course/problem", n, bad)
	}

	var events, withAccount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE account_id::text = $1::text)
		FROM practice.outbox WHERE payload_json ->> 'account_id' = $1::text`, acct).Scan(&events, &withAccount); err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	if events < 2 || withAccount != events {
		t.Fatalf("%d outbox rows, %d with account_id", events, withAccount)
	}
}
