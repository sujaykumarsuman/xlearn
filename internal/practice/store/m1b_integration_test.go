package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

// m1-03 (M1b): every outbox row practice writes is a v2 envelope carrying the problem
// state's course (every practice subject is course-scoped), the course comes from the
// caller when the problem state row is created and a resume keeps the row's own, and
// occurred_at is still the write time. A fixture course ("zz-fixture") proves the course
// is carried, not defaulted.
func TestM1bPracticeV2Envelopes(t *testing.T) {
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
	const problem, fixture = "zz-fixture:e-1", "zz-fixture"

	before := time.Now().Add(-time.Second)
	if _, err := st.StartAttempt(ctx, acct, problem, fixture); err != nil {
		t.Fatalf("start: %v", err)
	}
	// A resume with another course keeps the row's own course.
	if _, err := st.StartAttempt(ctx, acct, problem, "dsa"); err != nil {
		t.Fatalf("resume: %v", err)
	}
	for i := 0; i < 2; i++ { // hint, then the solution early (solution_revealed_early)
		if _, err := st.Reveal(ctx, acct, problem); err != nil {
			t.Fatalf("reveal %d: %v", i, err)
		}
	}
	if _, err := st.LogOutcome(ctx, acct, problem, store.OutcomeRough); err != nil {
		t.Fatalf("log outcome: %v", err)
	}
	after := time.Now().Add(time.Second)

	var statePath, attemptPath string
	if err := pool.QueryRow(ctx, `
		SELECT s.path_slug, a.path_slug FROM practice.user_problem_state s
		JOIN practice.attempt a ON a.user_problem_state_id = s.id
		WHERE s.account_id = $1 AND s.problem_id = $2`, acct, problem).Scan(&statePath, &attemptPath); err != nil {
		t.Fatalf("read state: %v", err)
	}
	if statePath != fixture || attemptPath != fixture {
		t.Fatalf("path_slug: state %q attempt %q, want %q (a resume keeps the row's course)", statePath, attemptPath, fixture)
	}

	rows, err := pool.Query(ctx, `SELECT subject, payload_json FROM practice.outbox WHERE account_id = $1`, acct)
	if err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	defer rows.Close()
	subjects := map[string]int{}
	for rows.Next() {
		var subject string
		var payload []byte
		if err := rows.Scan(&subject, &payload); err != nil {
			t.Fatalf("scan: %v", err)
		}
		env, err := events.DecodeEnvelope(payload)
		if err != nil {
			t.Fatalf("%s: decode: %v", subject, err)
		}
		if env.Version != events.EnvelopeV2 || env.PathSlug != fixture || env.Subject != subject || env.AccountID != acct {
			t.Errorf("%s: version %d path_slug %q subject %q account %q; want v2 %q", subject, env.Version, env.PathSlug, env.Subject, env.AccountID, fixture)
		}
		at, err := time.Parse(time.RFC3339Nano, env.OccurredAt)
		if err != nil || at.Before(before) || at.After(after) {
			t.Errorf("%s: occurred_at %q (%v) is not the write time", subject, env.OccurredAt, err)
		}
		subjects[subject]++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	for _, want := range []string{store.SubjectSolutionRevealedEarly, store.SubjectProblemSolved, store.SubjectAttemptLogged} {
		if subjects[want] != 1 {
			t.Errorf("outbox: %s count = %d, want 1", want, subjects[want])
		}
	}

	// A course-less start is refused rather than written as '' or left to a default.
	if _, err := st.StartAttempt(ctx, newTestUUID(), "1", ""); err == nil {
		t.Fatalf("StartAttempt without a course succeeded")
	}
}
