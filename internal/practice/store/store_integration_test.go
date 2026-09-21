package store_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

// Integration test against a real Postgres, gated on XLEARN_TEST_DATABASE_URL so CI
// (which has no database) skips it. Run locally, e.g.:
//
//	XLEARN_TEST_DATABASE_URL=postgres://xlearn_practice:pw@localhost:5433/xlearndb?sslmode=disable \
//	  go test ./internal/practice/store/ -run TestStore -count=1
//
// The DSN must be for the xlearn_practice role (search_path practice, owning only
// schema practice) so the test also exercises the least-privilege model.
func TestStoreIntegration(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the practice store integration test")
	}
	ctx := context.Background()

	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Idempotent re-run (advisory lock + goose versioning).
	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate (rerun): %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	st := store.New(pool)

	if err := st.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	t.Run("full gated flow with early reveal + below-clean outcome", func(t *testing.T) {
		acct := newTestUUID()
		const problem = "16"

		// Fresh problem: available, statement-only, no timer.
		fresh, err := st.GetState(ctx, acct, problem)
		if err != nil {
			t.Fatalf("get fresh state: %v", err)
		}
		if fresh.Status != "available" || fresh.Timer != nil {
			t.Fatalf("fresh state = %+v", fresh)
		}
		if !equalStrs(fresh.UnlockedStages, []string{store.StageAttempt}) {
			t.Fatalf("fresh unlocked = %v, want [attempt]", fresh.UnlockedStages)
		}

		// Start the attempt → attempting, 15-min attempt timer running.
		s1, err := st.StartAttempt(ctx, acct, problem)
		if err != nil {
			t.Fatalf("start attempt: %v", err)
		}
		if s1.Status != "attempting" || s1.StageReached != store.StageAttempt {
			t.Fatalf("after start = %+v", s1)
		}
		if s1.Timer == nil || s1.Timer.Kind != store.TimerAttempt || s1.Timer.Expired {
			t.Fatalf("after start timer = %+v", s1.Timer)
		}

		// Restart is idempotent (resume): the same attempt/timer, not reset.
		s1b, err := st.StartAttempt(ctx, acct, problem)
		if err != nil {
			t.Fatalf("resume attempt: %v", err)
		}
		if s1b.Timer == nil || !s1b.Timer.DeadlineAt.Equal(s1.Timer.DeadlineAt) {
			t.Fatalf("resume changed the deadline: %+v vs %+v", s1b.Timer, s1.Timer)
		}

		// Reveal the hint → hint unlocked, 10-min hint timer.
		r1, err := st.Reveal(ctx, acct, problem)
		if err != nil {
			t.Fatalf("reveal hint: %v", err)
		}
		if r1.Revealed != store.StageHint || r1.Penalty != nil {
			t.Fatalf("reveal hint = %+v", r1)
		}
		if !equalStrs(r1.State.UnlockedStages, []string{store.StageAttempt, store.StageHint}) {
			t.Fatalf("after hint unlocked = %v", r1.State.UnlockedStages)
		}
		if r1.State.Timer == nil || r1.State.Timer.Kind != store.TimerHint {
			t.Fatalf("after hint timer = %+v", r1.State.Timer)
		}

		// Reveal the solution BEFORE the attempt timer elapses → penalty + early flag.
		r2, err := st.Reveal(ctx, acct, problem)
		if err != nil {
			t.Fatalf("reveal solution: %v", err)
		}
		if r2.Revealed != store.StageSolution {
			t.Fatalf("reveal solution = %+v", r2)
		}
		if r2.Penalty == nil || !r2.Penalty.OwedAttempt || r2.Penalty.DueInDays != store.EarlyRevealPenaltyDays {
			t.Fatalf("expected early-reveal penalty, got %+v", r2.Penalty)
		}
		if !r2.State.RevealedEarly {
			t.Fatalf("expected revealed-early flag set")
		}
		if !equalStrs(r2.State.UnlockedStages, []string{store.StageAttempt, store.StageHint, store.StageSolution}) {
			t.Fatalf("after solution unlocked = %v", r2.State.UnlockedStages)
		}
		if r2.State.Timer != nil {
			t.Fatalf("solution stage should have no active timer, got %+v", r2.State.Timer)
		}

		// Nothing left to reveal.
		if _, err := st.Reveal(ctx, acct, problem); err == nil {
			t.Fatalf("expected ErrNothingToReveal on a third reveal")
		}

		// Log a below-clean outcome → solved, last outcome recorded.
		s2, err := st.LogOutcome(ctx, acct, problem, store.OutcomeMiss)
		if err != nil {
			t.Fatalf("log outcome: %v", err)
		}
		if s2.Status != "solved" || s2.LastOutcome != store.OutcomeMiss || s2.FirstSolvedAt.IsZero() {
			t.Fatalf("after outcome = %+v", s2)
		}

		// Re-logging is rejected (no open attempt).
		if _, err := st.LogOutcome(ctx, acct, problem, store.OutcomeClean); err == nil {
			t.Fatalf("expected ErrAlreadySolved on a second outcome")
		}

		// Three events in the outbox: solution_revealed_early, problem_solved,
		// attempt_logged — all referencing this account, none sent yet.
		rows, err := st.ListUnsentOutbox(ctx, 100)
		if err != nil {
			t.Fatalf("list outbox: %v", err)
		}
		subjects := map[string]int{}
		for _, r := range rows {
			if bytes.Contains(r.Payload, []byte(acct)) {
				subjects[r.Subject]++
			}
		}
		for _, want := range []string{store.SubjectSolutionRevealedEarly, store.SubjectProblemSolved, store.SubjectAttemptLogged} {
			if subjects[want] != 1 {
				t.Errorf("outbox: %s count = %d, want 1", want, subjects[want])
			}
		}

		// Marking one sent removes it from the unsent set.
		before := len(rows)
		if err := st.MarkOutboxSent(ctx, rows[0].EventID); err != nil {
			t.Fatalf("mark sent: %v", err)
		}
		after, err := st.ListUnsentOutbox(ctx, 100)
		if err != nil {
			t.Fatalf("list outbox after: %v", err)
		}
		if len(after) != before-1 {
			t.Errorf("unsent after mark = %d, want %d", len(after), before-1)
		}
	})

	t.Run("clean solve blind, no reveal, first_solve", func(t *testing.T) {
		acct := newTestUUID()
		const problem = "1"
		if _, err := st.StartAttempt(ctx, acct, problem); err != nil {
			t.Fatalf("start: %v", err)
		}
		s, err := st.LogOutcome(ctx, acct, problem, store.OutcomeClean)
		if err != nil {
			t.Fatalf("outcome: %v", err)
		}
		if s.Status != "solved" || s.LastOutcome != store.OutcomeClean {
			t.Fatalf("solved-blind = %+v", s)
		}
		// Solved blind never unlocked hint/solution.
		if !equalStrs(s.UnlockedStages, []string{store.StageAttempt}) {
			t.Errorf("solved-blind unlocked = %v, want [attempt]", s.UnlockedStages)
		}
	})

	t.Run("list states for a week's problems", func(t *testing.T) {
		acct := newTestUUID()
		if _, err := st.StartAttempt(ctx, acct, "16"); err != nil {
			t.Fatalf("start: %v", err)
		}
		states, err := st.ListStates(ctx, acct, []string{"16", "17", "18"})
		if err != nil {
			t.Fatalf("list states: %v", err)
		}
		if _, ok := states["16"]; !ok {
			t.Errorf("expected a state for 16")
		}
		if _, ok := states["17"]; ok {
			t.Errorf("did not expect a state for the untouched 17")
		}
	})

	t.Run("reveal/outcome require an attempt", func(t *testing.T) {
		acct := newTestUUID()
		if _, err := st.Reveal(ctx, acct, "99"); err == nil {
			t.Errorf("expected ErrNoAttempt revealing before start")
		}
		if _, err := st.LogOutcome(ctx, acct, "99", store.OutcomeClean); err == nil {
			t.Errorf("expected ErrNoAttempt logging before start")
		}
	})
}

func equalStrs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
