package store_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// Integration test against a real Postgres, gated on XLEARN_TEST_DATABASE_URL so CI
// (which has no database) skips it. Run locally, e.g.:
//
//	XLEARN_TEST_DATABASE_URL=postgres://xlearn_review:pw@localhost:5433/xlearndb?sslmode=disable \
//	  go test ./internal/review/store/ -run TestStore -count=1
//
// The DSN must be for the xlearn_review role (search_path review, owning only schema
// review) so the test also exercises the least-privilege model.
func TestStoreIntegration(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the review store integration test")
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

	t.Run("first clean solve schedules five touches + emits five events", func(t *testing.T) {
		acct := newTestUUID()
		const problem = "16"
		solvedAt := time.Now()

		n, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, problem, "clean", true, solvedAt)
		if err != nil {
			t.Fatalf("handle problem_solved: %v", err)
		}
		if n != 5 {
			t.Fatalf("scheduled %d touches, want 5", n)
		}
		if c := countTouches(ctx, t, pool, acct); c != 5 {
			t.Fatalf("revision_item rows = %d, want 5", c)
		}
		if c := countOutbox(ctx, t, st, acct, store.SubjectRevisionScheduled); c != 5 {
			t.Fatalf("revision_scheduled events = %d, want 5", c)
		}

		// Re-delivery of the SAME event id: deduped by the inbox → no new touches/events.
		// (Use the same event id via a second call — model a redelivery.)
		evtID := newTestUUID()
		if _, err := st.HandleProblemSolved(ctx, evtID, acct, "17", "clean", true, solvedAt); err != nil {
			t.Fatalf("schedule 17: %v", err)
		}
		n2, err := st.HandleProblemSolved(ctx, evtID, acct, "17", "clean", true, solvedAt)
		if err != nil {
			t.Fatalf("redeliver 17: %v", err)
		}
		if n2 != 0 {
			t.Fatalf("redelivery scheduled %d touches, want 0 (deduped)", n2)
		}
		if c := countTouches(ctx, t, pool, acct); c != 10 {
			t.Fatalf("after 16 + 17: revision_item rows = %d, want 10", c)
		}
	})

	t.Run("below-clean first solve schedules nothing", func(t *testing.T) {
		acct := newTestUUID()
		n, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, "16", "miss", true, time.Now())
		if err != nil {
			t.Fatalf("handle problem_solved (miss): %v", err)
		}
		if n != 0 {
			t.Fatalf("below-clean scheduled %d touches, want 0", n)
		}
		if c := countTouches(ctx, t, pool, acct); c != 0 {
			t.Fatalf("below-clean revision_item rows = %d, want 0", c)
		}
	})

	t.Run("solution_revealed_early schedules the owed 3-day touch", func(t *testing.T) {
		acct := newTestUUID()
		n, err := st.HandleSolutionRevealedEarly(ctx, newTestUUID(), acct, "42", time.Now())
		if err != nil {
			t.Fatalf("handle solution_revealed_early: %v", err)
		}
		if n != 1 {
			t.Fatalf("owed attempt scheduled %d, want 1", n)
		}
		items, err := st.DueQueue(ctx, acct, 10)
		if err != nil {
			t.Fatalf("due queue: %v", err)
		}
		if len(items) != 1 || items[0].TouchLevel != store.OwedAttemptTouchLevel {
			t.Fatalf("owed attempt = %+v, want one Day-3 (level 2) touch", items)
		}
	})

	t.Run("auto-score: pass advances, miss resets to Day 1", func(t *testing.T) {
		acct := newTestUUID()
		const problem = "88"
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, problem, "clean", true, time.Now()); err != nil {
			t.Fatalf("schedule: %v", err)
		}
		day1 := touchByLevel(ctx, t, st, acct, 1)

		// Pass the Day-1 touch (pattern < 2 min, in timer, complexity stated).
		res, err := st.Score(ctx, acct, day1.ItemID, store.ScoreInput{NamedPatternSecs: 45, SolvedInTimer: true, StatedComplexity: true})
		if err != nil {
			t.Fatalf("score pass: %v", err)
		}
		if !res.AutoPass || res.Reset || res.NextTouchLevel != 2 {
			t.Fatalf("pass result = %+v, want autoPass + advance to level 2", res)
		}
		if s := touchStatus(ctx, t, pool, day1.ItemID); s != store.StatusPassed {
			t.Fatalf("passed touch status = %q, want passed", s)
		}

		// Fail the Day-3 touch → the whole ladder resets to Day 1 (all pending).
		day3 := touchByLevel(ctx, t, st, acct, 2)
		fres, err := st.Score(ctx, acct, day3.ItemID, store.ScoreInput{NamedPatternSecs: 200, SolvedInTimer: true, StatedComplexity: true})
		if err != nil {
			t.Fatalf("score fail: %v", err)
		}
		if fres.AutoPass || !fres.Reset || fres.NextTouchLevel != 1 {
			t.Fatalf("fail result = %+v, want !autoPass + reset to Day 1", fres)
		}
		// After the reset, the Day-1 touch is pending again (re-anchored), not passed.
		if s := touchStatus(ctx, t, pool, day1.ItemID); s != store.StatusPending {
			t.Fatalf("after reset, Day-1 status = %q, want pending", s)
		}

		// Two touch_result rows recorded (one pass, one fail).
		if c := countTouchResults(ctx, t, pool, acct); c != 2 {
			t.Fatalf("touch_result rows = %d, want 2", c)
		}
	})

	t.Run("re-scoring a passed touch is idempotent (no duplicate result/event)", func(t *testing.T) {
		acct := newTestUUID()
		const problem = "321"
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, problem, "clean", true, time.Now()); err != nil {
			t.Fatalf("schedule: %v", err)
		}
		day1 := touchByLevel(ctx, t, st, acct, 1)
		in := store.ScoreInput{NamedPatternSecs: 30, SolvedInTimer: true, StatedComplexity: true}

		res1, err := st.Score(ctx, acct, day1.ItemID, in)
		if err != nil || !res1.AutoPass {
			t.Fatalf("first score = %+v, err %v", res1, err)
		}
		// Re-score the now-passed touch (a double-submit / replay): idempotent no-op.
		res2, err := st.Score(ctx, acct, day1.ItemID, in)
		if err != nil {
			t.Fatalf("re-score: %v", err)
		}
		if !res2.AutoPass || res2.NextTouchLevel != 2 {
			t.Fatalf("re-score result = %+v, want settled pass advancing to level 2", res2)
		}
		if c := countTouchResults(ctx, t, pool, acct); c != 1 {
			t.Fatalf("touch_result rows = %d after re-score, want 1 (no duplicate)", c)
		}
	})

	t.Run("sweep skips a touch passed before it was surfaced", func(t *testing.T) {
		acct := newTestUUID()
		const problem = "654"
		// Overdue ladder (solved 60 days ago) → all five touches are due.
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, problem, "clean", true, time.Now().AddDate(0, 0, -60)); err != nil {
			t.Fatalf("schedule overdue: %v", err)
		}
		// The learner passes the Day-1 touch BEFORE the sweep runs (surfaced_at NULL).
		day1 := touchByLevel(ctx, t, st, acct, 1)
		if _, err := st.Score(ctx, acct, day1.ItemID, store.ScoreInput{NamedPatternSecs: 20, SolvedInTimer: true, StatedComplexity: true}); err != nil {
			t.Fatalf("score day1: %v", err)
		}
		if _, err := st.Sweep(ctx, 100); err != nil {
			t.Fatalf("sweep: %v", err)
		}
		// Only the four still-pending touches (Day 3/7/21/45) are surfaced — the passed
		// Day-1 touch gets no spurious revision_due.
		if c := countOutbox(ctx, t, st, acct, store.SubjectRevisionDue); c != 4 {
			t.Fatalf("revision_due events = %d, want 4 (passed Day-1 not surfaced)", c)
		}
	})

	t.Run("scoring another account's item is not found", func(t *testing.T) {
		acct := newTestUUID()
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, "5", "clean", true, time.Now()); err != nil {
			t.Fatalf("schedule: %v", err)
		}
		item := touchByLevel(ctx, t, st, acct, 1)
		other := newTestUUID()
		if _, err := st.Score(ctx, other, item.ItemID, store.ScoreInput{NamedPatternSecs: 1, SolvedInTimer: true, StatedComplexity: true}); err == nil {
			t.Fatalf("expected ErrNotFound scoring another account's item")
		}
	})

	t.Run("sweep surfaces overdue touches idempotently + emits revision_due", func(t *testing.T) {
		acct := newTestUUID()
		const problem = "77"
		// Anchor the solve 60 days ago so every touch (Day 1..45) is overdue.
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, problem, "clean", true, time.Now().AddDate(0, 0, -60)); err != nil {
			t.Fatalf("schedule overdue: %v", err)
		}
		n, err := st.Sweep(ctx, 100)
		if err != nil {
			t.Fatalf("sweep: %v", err)
		}
		if n < 5 {
			t.Fatalf("sweep surfaced %d, want >= 5 (this account's overdue touches)", n)
		}
		// Idempotent: a second sweep surfaces none of the already-surfaced touches.
		n2, err := st.Sweep(ctx, 100)
		if err != nil {
			t.Fatalf("sweep rerun: %v", err)
		}
		if c := countOutbox(ctx, t, st, acct, store.SubjectRevisionDue); c != 5 {
			t.Fatalf("revision_due events for account = %d, want 5 (emitted once)", c)
		}
		_ = n2 // a concurrent run may surface other test accounts' rows; only ours is asserted
	})

	t.Run("due queue prioritises soonest-due first", func(t *testing.T) {
		acct := newTestUUID()
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, "100", "clean", true, time.Now()); err != nil {
			t.Fatalf("schedule: %v", err)
		}
		items, err := st.DueQueue(ctx, acct, 10)
		if err != nil {
			t.Fatalf("due queue: %v", err)
		}
		if len(items) != 5 {
			t.Fatalf("due queue len = %d, want 5", len(items))
		}
		for i := 1; i < len(items); i++ {
			if items[i].DueDate.Before(items[i-1].DueDate) {
				t.Fatalf("due queue not sorted by due_date: %v", items)
			}
		}
	})

	t.Run("outbox relay list + mark sent", func(t *testing.T) {
		acct := newTestUUID()
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, "200", "clean", true, time.Now()); err != nil {
			t.Fatalf("schedule: %v", err)
		}
		rows, err := st.ListUnsentOutbox(ctx, 1000)
		if err != nil {
			t.Fatalf("list outbox: %v", err)
		}
		var mine []store.OutboxRow
		for _, r := range rows {
			if bytes.Contains(r.Payload, []byte(acct)) {
				mine = append(mine, r)
			}
		}
		if len(mine) == 0 {
			t.Fatalf("expected unsent outbox rows for the account")
		}
		if err := st.MarkOutboxSent(ctx, mine[0].EventID); err != nil {
			t.Fatalf("mark sent: %v", err)
		}
		after, err := st.ListUnsentOutbox(ctx, 1000)
		if err != nil {
			t.Fatalf("list outbox after: %v", err)
		}
		for _, r := range after {
			if r.EventID == mine[0].EventID {
				t.Fatalf("event %s still unsent after mark", mine[0].EventID)
			}
		}
	})
}

// --- helpers ---

func countTouches(ctx context.Context, t *testing.T, pool *pgxpool.Pool, accountID string) int {
	t.Helper()
	var c int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM review.revision_item WHERE account_id = $1", accountID).Scan(&c); err != nil {
		t.Fatalf("count touches: %v", err)
	}
	return c
}

func countTouchResults(ctx context.Context, t *testing.T, pool *pgxpool.Pool, accountID string) int {
	t.Helper()
	var c int
	const q = `SELECT count(*) FROM review.touch_result tr
	           JOIN review.revision_item ri ON ri.id = tr.revision_item_id
	           WHERE ri.account_id = $1`
	if err := pool.QueryRow(ctx, q, accountID).Scan(&c); err != nil {
		t.Fatalf("count touch results: %v", err)
	}
	return c
}

func touchStatus(ctx context.Context, t *testing.T, pool *pgxpool.Pool, itemID string) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(ctx, "SELECT status FROM review.revision_item WHERE id = $1", itemID).Scan(&s); err != nil {
		t.Fatalf("touch status: %v", err)
	}
	return s
}

func countOutbox(ctx context.Context, t *testing.T, st *store.PgStore, accountID, subject string) int {
	t.Helper()
	rows, err := st.ListUnsentOutbox(ctx, 10000)
	if err != nil {
		t.Fatalf("list outbox: %v", err)
	}
	c := 0
	for _, r := range rows {
		if r.Subject == subject && bytes.Contains(r.Payload, []byte(accountID)) {
			c++
		}
	}
	return c
}

func touchByLevel(ctx context.Context, t *testing.T, st *store.PgStore, accountID string, level int) store.DueItem {
	t.Helper()
	items, err := st.DueQueue(ctx, accountID, 100)
	if err != nil {
		t.Fatalf("due queue: %v", err)
	}
	for _, it := range items {
		if it.TouchLevel == level {
			return it
		}
	}
	t.Fatalf("no touch at level %d for account %s", level, accountID)
	return store.DueItem{}
}

// TestMistakeJournalIntegration exercises the S07 mistake-journal state machine, the
// weak-area store pieces, and the reminder dedupe against a real Postgres (gated on
// XLEARN_TEST_DATABASE_URL, like TestStoreIntegration).
func TestMistakeJournalIntegration(t *testing.T) {
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

	pass := store.ScoreInput{NamedPatternSecs: 30, SolvedInTimer: true, StatedComplexity: true}
	fail := store.ScoreInput{NamedPatternSecs: 200, SolvedInTimer: true, StatedComplexity: true}

	t.Run("below-clean solve opens a mistake (deduped, uncategorised) + emits mistake_opened", func(t *testing.T) {
		acct := newTestUUID()
		const problem = "bc-1"
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, problem, "miss", true, time.Now()); err != nil {
			t.Fatalf("problem_solved miss: %v", err)
		}
		open, err := st.ListMistakes(ctx, acct, "open")
		if err != nil {
			t.Fatalf("list mistakes: %v", err)
		}
		if len(open) != 1 || open[0].Status != store.MistakeOpen || open[0].RevisitCount != 0 || open[0].Category != "" {
			t.Fatalf("open mistakes = %+v, want one uncategorised open entry with count 0", open)
		}
		if c := countOutbox(ctx, t, st, acct, store.SubjectMistakeOpened); c != 1 {
			t.Fatalf("mistake_opened events = %d, want 1", c)
		}
		// A second below-clean solve of the same problem must not open a duplicate.
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, problem, "rough", true, time.Now()); err != nil {
			t.Fatalf("second below-clean: %v", err)
		}
		open, _ = st.ListMistakes(ctx, acct, "open")
		if len(open) != 1 {
			t.Fatalf("after repeat miss: open entries = %d, want 1 (one-open-entry invariant)", len(open))
		}
	})

	t.Run("failed re-solve opens; 2 clean revisits close; a later fail re-opens", func(t *testing.T) {
		acct := newTestUUID()
		const problem = "sm-1"
		// Clean first solve → 5 touches, no mistake yet.
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, problem, "clean", true, time.Now()); err != nil {
			t.Fatalf("clean solve: %v", err)
		}
		if ms, _ := st.ListMistakes(ctx, acct, ""); len(ms) != 0 {
			t.Fatalf("clean solve opened %d mistakes, want 0", len(ms))
		}

		// Fail the Day-1 re-solve → opens a mistake (open, count 0) + resets to Day 1.
		day1 := touchByLevel(ctx, t, st, acct, 1)
		if _, err := st.Score(ctx, acct, day1.ItemID, fail); err != nil {
			t.Fatalf("fail day1: %v", err)
		}
		if m := onlyOpen(ctx, t, st, acct); m.RevisitCount != 0 {
			t.Fatalf("after fail: count = %d, want 0", m.RevisitCount)
		}
		if c := countOutbox(ctx, t, st, acct, store.SubjectMistakeOpened); c != 1 {
			t.Fatalf("mistake_opened = %d, want 1", c)
		}

		// Clean revisit 1 (Day 1 passes) → count 1.
		if _, err := st.Score(ctx, acct, touchByLevel(ctx, t, st, acct, 1).ItemID, pass); err != nil {
			t.Fatalf("pass day1: %v", err)
		}
		if m := onlyOpen(ctx, t, st, acct); m.RevisitCount != 1 {
			t.Fatalf("after 1 clean revisit: count = %d, want 1", m.RevisitCount)
		}

		// Clean revisit 2 (Day 3 passes) → count 2 → closed + mistake_closed.
		if _, err := st.Score(ctx, acct, touchByLevel(ctx, t, st, acct, 2).ItemID, pass); err != nil {
			t.Fatalf("pass day3: %v", err)
		}
		if open, _ := st.ListMistakes(ctx, acct, "open"); len(open) != 0 {
			t.Fatalf("after 2 clean revisits: open entries = %d, want 0 (closed)", len(open))
		}
		if closed, _ := st.ListMistakes(ctx, acct, "closed"); len(closed) != 1 || closed[0].RevisitCount != 2 {
			t.Fatalf("closed entries = %+v, want one with count 2", closed)
		}
		if c := countOutbox(ctx, t, st, acct, store.SubjectMistakeClosed); c != 1 {
			t.Fatalf("mistake_closed = %d, want 1", c)
		}

		// A later fail (Day 7) re-opens the closed entry (count 0) + a 2nd mistake_opened.
		if _, err := st.Score(ctx, acct, touchByLevel(ctx, t, st, acct, 3).ItemID, fail); err != nil {
			t.Fatalf("fail day7: %v", err)
		}
		if m := onlyOpen(ctx, t, st, acct); m.RevisitCount != 0 {
			t.Fatalf("after re-open: count = %d, want 0", m.RevisitCount)
		}
		if closed, _ := st.ListMistakes(ctx, acct, "closed"); len(closed) != 0 {
			t.Fatalf("after re-open: closed entries = %d, want 0", len(closed))
		}
		if c := countOutbox(ctx, t, st, acct, store.SubjectMistakeOpened); c != 2 {
			t.Fatalf("mistake_opened = %d, want 2 (open + re-open)", c)
		}
	})

	t.Run("a fail mid-streak resets the clean-revisit count (fails never count)", func(t *testing.T) {
		acct := newTestUUID()
		const problem = "sm-2"
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, problem, "clean", true, time.Now()); err != nil {
			t.Fatalf("clean solve: %v", err)
		}
		// Fail once to open the entry, then one clean revisit (count 1), then a fail.
		if _, err := st.Score(ctx, acct, touchByLevel(ctx, t, st, acct, 1).ItemID, fail); err != nil {
			t.Fatalf("fail: %v", err)
		}
		if _, err := st.Score(ctx, acct, touchByLevel(ctx, t, st, acct, 1).ItemID, pass); err != nil {
			t.Fatalf("pass: %v", err)
		}
		if m := onlyOpen(ctx, t, st, acct); m.RevisitCount != 1 {
			t.Fatalf("count = %d, want 1", m.RevisitCount)
		}
		if _, err := st.Score(ctx, acct, touchByLevel(ctx, t, st, acct, 2).ItemID, fail); err != nil {
			t.Fatalf("fail again: %v", err)
		}
		if m := onlyOpen(ctx, t, st, acct); m.RevisitCount != 0 {
			t.Fatalf("after mid-streak fail: count = %d, want 0 (reset, still open)", m.RevisitCount)
		}
	})

	t.Run("clean re-solve of a never-missed problem touches no journal", func(t *testing.T) {
		acct := newTestUUID()
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, "nm-1", "clean", true, time.Now()); err != nil {
			t.Fatalf("clean solve: %v", err)
		}
		if _, err := st.Score(ctx, acct, touchByLevel(ctx, t, st, acct, 1).ItemID, pass); err != nil {
			t.Fatalf("pass: %v", err)
		}
		if ms, _ := st.ListMistakes(ctx, acct, ""); len(ms) != 0 {
			t.Fatalf("journal has %d entries, want 0 (no mistake for a clean problem)", len(ms))
		}
	})

	t.Run("weak-area: count in range, classify, snapshot upsert idempotent, current banner", func(t *testing.T) {
		acct := newTestUUID()
		if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, "wa-1", "miss", true, time.Now()); err != nil {
			t.Fatalf("open mistake: %v", err)
		}
		open := onlyOpen(ctx, t, st, acct)
		if _, err := st.UpdateMistake(ctx, acct, open.ID, store.MistakePatch{Category: strptr("off_by_one")}); err != nil {
			t.Fatalf("classify: %v", err)
		}
		start := time.Now().Add(-24 * time.Hour)
		end := time.Now().Add(24 * time.Hour)
		counts, err := st.CountOpenMistakesByCategory(ctx, acct, start, end)
		if err != nil {
			t.Fatalf("count by category: %v", err)
		}
		if counts["off_by_one"] != 1 {
			t.Fatalf("counts = %v, want off_by_one:1", counts)
		}
		weekOf := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
		if err := st.SaveWeakAreaSnapshot(ctx, acct, weekOf, "off_by_one", counts); err != nil {
			t.Fatalf("save snapshot: %v", err)
		}
		// Idempotent per week_of: a second save upserts, doesn't duplicate.
		if err := st.SaveWeakAreaSnapshot(ctx, acct, weekOf, "off_by_one", counts); err != nil {
			t.Fatalf("save snapshot (rerun): %v", err)
		}
		wa, found, err := st.WeakAreaCurrent(ctx, acct)
		if err != nil || !found {
			t.Fatalf("weak area current: found=%v err=%v", found, err)
		}
		if wa.TopCategory != "off_by_one" || wa.TopCount != 1 || len(wa.Entries) != 1 {
			t.Fatalf("weak area = %+v, want off_by_one/1/1 entry", wa)
		}
	})

	t.Run("concurrent below-clean open + failed re-solve re-open don't collide (advisory lock)", func(t *testing.T) {
		// The race the journal advisory lock closes: a below-clean problem_solved opens a
		// NEW entry while a failed re-solve re-opens the problem's CLOSED entry — without
		// serialisation both would leave two open rows and trip the one-open partial
		// unique index, rolling back a legitimate failed re-solve. Run several rounds.
		for i := 0; i < 12; i++ {
			acct := newTestUUID()
			problem := fmt.Sprintf("race-%d", i)
			if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, problem, "clean", true, time.Now()); err != nil {
				t.Fatalf("iter %d: clean solve: %v", i, err)
			}
			if _, err := st.CreateMistake(ctx, acct, store.MistakeInput{ProblemID: problem, Category: "off_by_one", Status: "closed"}); err != nil {
				t.Fatalf("iter %d: seed closed entry: %v", i, err)
			}
			day1 := touchByLevel(ctx, t, st, acct, 1)

			var wg sync.WaitGroup
			var errOpen, errScore error
			wg.Add(2)
			go func() {
				defer wg.Done()
				_, errOpen = st.HandleProblemSolved(ctx, newTestUUID(), acct, problem, "miss", false, time.Now())
			}()
			go func() {
				defer wg.Done()
				_, errScore = st.Score(ctx, acct, day1.ItemID, fail)
			}()
			wg.Wait()
			if errOpen != nil || errScore != nil {
				t.Fatalf("iter %d: concurrent ops errored (no serialisation?): open=%v score=%v", i, errOpen, errScore)
			}
			open, err := st.ListMistakes(ctx, acct, "open")
			if err != nil {
				t.Fatalf("iter %d: list open: %v", i, err)
			}
			n := 0
			for _, m := range open {
				if m.ProblemID == problem {
					n++
				}
			}
			if n != 1 {
				t.Fatalf("iter %d: open entries for %s = %d, want exactly 1 (one-open invariant)", i, problem, n)
			}
		}
	})

	t.Run("reminder: revision_due writes once (deduped) and lists as due", func(t *testing.T) {
		acct := newTestUUID()
		evt := newTestUUID()
		wrote, err := st.HandleRevisionDue(ctx, evt, acct, "revision_due", time.Now().Add(-time.Minute))
		if err != nil || !wrote {
			t.Fatalf("first revision_due: wrote=%v err=%v", wrote, err)
		}
		wrote2, err := st.HandleRevisionDue(ctx, evt, acct, "revision_due", time.Now().Add(-time.Minute))
		if err != nil {
			t.Fatalf("dup revision_due: %v", err)
		}
		if wrote2 {
			t.Fatalf("duplicate revision_due wrote a second reminder")
		}
		rems, err := st.ListDueReminders(ctx, acct, 10)
		if err != nil {
			t.Fatalf("list reminders: %v", err)
		}
		if len(rems) != 1 || rems[0].Kind != "revision_due" {
			t.Fatalf("reminders = %+v, want one revision_due", rems)
		}
	})
}

// onlyOpen returns the account's single open mistake, failing if there isn't exactly one.
func onlyOpen(ctx context.Context, t *testing.T, st *store.PgStore, accountID string) store.Mistake {
	t.Helper()
	open, err := st.ListMistakes(ctx, accountID, "open")
	if err != nil {
		t.Fatalf("list open mistakes: %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("open mistakes = %d, want exactly 1", len(open))
	}
	return open[0]
}

func strptr(s string) *string { return &s }
