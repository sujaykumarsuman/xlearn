package store_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// m1-03 (M1b), review's half: every outbox row review writes is a v2 envelope carrying
// the course of the row it reports (revision_scheduled / revision_due: the touch's;
// mistake_opened / mistake_closed: the entry's), the course-scoped reads filter on
// path_slug, the journal create writes the caller's course, and the one-open-entry rule
// stays per item. A fixture course ("zz-fixture") beside the DSA course proves the
// course is carried, not defaulted.
func TestM1bReviewCourseScoped(t *testing.T) {
	ctx, pool, st := m1bStore(t)
	const fixture, dsa = "zz-fixture", "dsa"
	acct := newTestUUID()
	sixtyDaysAgo := time.Now().AddDate(0, 0, -60)

	// zz-fixture: an overdue ladder (all five touches due) on e-1, a below-clean miss on
	// e-1 (opens a mistake), an early reveal on e-2; DSA: a fresh ladder on 16.
	if n, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, fixture, "e-1", "clean", true, sixtyDaysAgo); err != nil || n != 5 {
		t.Fatalf("schedule e-1: n=%d err=%v", n, err)
	}
	if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, fixture, "e-1", "miss", false, time.Now()); err != nil {
		t.Fatalf("miss e-1: %v", err)
	}
	if _, err := st.HandleSolutionRevealedEarly(ctx, newTestUUID(), acct, fixture, "e-2", time.Now()); err != nil {
		t.Fatalf("reveal e-2: %v", err)
	}
	if _, err := st.HandleProblemSolved(ctx, newTestUUID(), acct, dsa, "16", "clean", true, time.Now()); err != nil {
		t.Fatalf("schedule 16: %v", err)
	}

	// The due queue is per course.
	fixtureDue, err := st.DueQueue(ctx, acct, fixture, 100)
	if err != nil {
		t.Fatalf("due (fixture): %v", err)
	}
	dsaDue, err := st.DueQueue(ctx, acct, dsa, 100)
	if err != nil {
		t.Fatalf("due (dsa): %v", err)
	}
	if len(fixtureDue) != 6 || len(dsaDue) != 5 { // e-1 ×5 + e-2's owed Day-3 touch; 16 ×5
		t.Fatalf("due queue: %d fixture, %d dsa items; want 6 and 5", len(fixtureDue), len(dsaDue))
	}
	for _, it := range fixtureDue {
		if it.ProblemID == "16" {
			t.Fatalf("the fixture course's queue holds a DSA touch: %+v", it)
		}
	}

	// Two clean revisits of e-1 close its mistake (mistake_closed); a failed third
	// re-opens it (mistake_opened) and resets the ladder (revision_scheduled).
	byLevel := map[int]string{}
	for _, it := range fixtureDue {
		if it.ProblemID == "e-1" {
			byLevel[it.TouchLevel] = it.ItemID
		}
	}
	pass := store.ScoreInput{NamedPatternSecs: 10, SolvedInTimer: true, StatedComplexity: true}
	for _, level := range []int{1, 2} {
		if _, err := st.Score(ctx, acct, byLevel[level], pass); err != nil {
			t.Fatalf("score level %d: %v", level, err)
		}
	}
	// The sweep surfaces the still-overdue fixture touches (revision_due) before the
	// failed re-solve re-anchors the ladder.
	if _, err := st.Sweep(ctx, 500); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if _, err := st.Score(ctx, acct, byLevel[3], store.ScoreInput{NamedPatternSecs: 600}); err != nil {
		t.Fatalf("score level 3 (fail): %v", err)
	}

	// Every outbox row this account produced is v2 with the right course.
	subjects := map[string]map[string]int{} // path_slug → subject → count
	rows, err := pool.Query(ctx, `SELECT subject, payload_json FROM review.outbox WHERE account_id = $1`, acct)
	if err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	defer rows.Close()
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
		if env.Version != events.EnvelopeV2 || env.Subject != subject || env.AccountID != acct {
			t.Errorf("%s: version %d subject %q account %q; want a v2 envelope", subject, env.Version, env.Subject, env.AccountID)
		}
		if subjects[env.PathSlug] == nil {
			subjects[env.PathSlug] = map[string]int{}
		}
		subjects[env.PathSlug][subject]++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	for _, s := range []string{store.SubjectRevisionScheduled, store.SubjectRevisionDue, store.SubjectMistakeOpened, store.SubjectMistakeClosed} {
		if subjects[fixture][s] == 0 {
			t.Errorf("no %s in course %s (outbox by course: %v)", s, fixture, subjects)
		}
	}
	if subjects[dsa][store.SubjectRevisionScheduled] != 5 || len(subjects) != 2 {
		t.Errorf("outbox by course = %v; want only %s and %s, with 5 DSA revision_scheduled", subjects, fixture, dsa)
	}

	// The journal is per course, and a manual entry lands in the caller's course.
	if ms, err := st.ListMistakes(ctx, acct, dsa, ""); err != nil || len(ms) != 0 {
		t.Fatalf("DSA journal = %v (%v), want empty", ms, err)
	}
	if ms, err := st.ListMistakes(ctx, acct, fixture, store.MistakeOpen); err != nil || len(ms) != 1 || ms[0].PathSlug != fixture {
		t.Fatalf("fixture open journal = %+v (%v), want e-1's re-opened entry", ms, err)
	}
	created, err := st.CreateMistake(ctx, acct, dsa, store.MistakeInput{ProblemID: "18", Category: "off_by_one"})
	if err != nil || created.PathSlug != dsa {
		t.Fatalf("create in dsa = %+v (%v)", created, err)
	}
	// One open entry per item, whatever the course.
	if _, err := st.CreateMistake(ctx, acct, fixture, store.MistakeInput{ProblemID: "e-1"}); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("second open entry for e-1: err = %v, want ErrConflict", err)
	}
	if _, err := st.CreateMistake(ctx, acct, "", store.MistakeInput{ProblemID: "19"}); err == nil {
		t.Fatalf("a course-less create succeeded")
	}
}

// m1-03: the weekly weak-area is per (account, course). The upsert targets the
// per-course unique (m1-02's weak_area_snapshot_account_path_week_uq); while the v1
// UNIQUE (account_id, week_of) still exists (m1-08 drops it), a second course's snapshot
// in the same week is REFUSED — never merged into the first course's row, which the v1
// conflict target would have done. Once that unique is gone (simulated here), two
// courses in the same week get two snapshots and each reads its own.
func TestM1bWeakAreaPerCourse(t *testing.T) {
	ctx, pool, st := m1bStore(t)
	const fixture, dsa = "zz-fixture", "dsa"
	acct := newTestUUID()
	weekOf := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)

	// One classified open mistake per course this week.
	for _, c := range []struct{ course, problem, category string }{
		{dsa, "wa-16", "off_by_one"},
		{fixture, "wa-e1", "communication"},
	} {
		if _, err := st.CreateMistake(ctx, acct, c.course, store.MistakeInput{ProblemID: c.problem, Category: c.category}); err != nil {
			t.Fatalf("create %s: %v", c.course, err)
		}
	}
	scopes, err := st.MistakeScopes(ctx)
	if err != nil {
		t.Fatalf("scopes: %v", err)
	}
	var mine []string
	for _, sc := range scopes {
		if sc.AccountID == acct {
			mine = append(mine, sc.PathSlug)
		}
	}
	if len(mine) != 2 || mine[0] != dsa || mine[1] != fixture {
		t.Fatalf("scopes for the account = %v, want [%s %s]", mine, dsa, fixture)
	}
	start, end := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	counts := map[string]map[string]int{}
	for _, c := range []string{dsa, fixture} {
		if counts[c], err = st.CountOpenMistakesByCategory(ctx, acct, c, start, end); err != nil {
			t.Fatalf("count %s: %v", c, err)
		}
	}
	if counts[dsa]["off_by_one"] != 1 || len(counts[dsa]) != 1 || counts[fixture]["communication"] != 1 || len(counts[fixture]) != 1 {
		t.Fatalf("counts = %v, want one category per course", counts)
	}

	if err := st.SaveWeakAreaSnapshot(ctx, acct, dsa, weekOf, "off_by_one", counts[dsa]); err != nil {
		t.Fatalf("save dsa: %v", err)
	}
	// The v1 per-account unique still stands: the second course's same-week snapshot
	// fails, and the DSA snapshot is untouched.
	if err := st.SaveWeakAreaSnapshot(ctx, acct, fixture, weekOf, "communication", counts[fixture]); err == nil {
		t.Fatalf("a second course's same-week snapshot was accepted while the v1 unique exists")
	}
	if wa, found, err := st.WeakAreaCurrent(ctx, acct, dsa); err != nil || !found || wa.TopCategory != "off_by_one" {
		t.Fatalf("dsa weak area after the refused save = %+v found=%v (%v)", wa, found, err)
	}

	dropV1WeakAreaUnique(ctx, t, pool, acct)

	if err := st.SaveWeakAreaSnapshot(ctx, acct, fixture, weekOf, "communication", counts[fixture]); err != nil {
		t.Fatalf("save fixture: %v", err)
	}
	// Idempotent per (course, week): a re-run upserts.
	if err := st.SaveWeakAreaSnapshot(ctx, acct, fixture, weekOf, "communication", counts[fixture]); err != nil {
		t.Fatalf("save fixture (rerun): %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM review.weak_area_snapshot WHERE account_id = $1 AND week_of = $2`, acct, weekOf).Scan(&n); err != nil || n != 2 {
		t.Fatalf("snapshots this week = %d (%v), want 2 (one per course)", n, err)
	}
	for course, want := range map[string]string{dsa: "off_by_one", fixture: "communication"} {
		wa, found, err := st.WeakAreaCurrent(ctx, acct, course)
		if err != nil || !found || wa.TopCategory != want || wa.TopCount != 1 || len(wa.Entries) != 1 || wa.Entries[0].PathSlug != course {
			t.Errorf("%s weak area = %+v found=%v (%v), want %s with its own entry", course, wa, found, err, want)
		}
	}
	if _, found, err := st.WeakAreaCurrent(ctx, acct, "zz-none"); err != nil || found {
		t.Errorf("a course without a snapshot: found=%v (%v)", found, err)
	}
}

// m1bStore migrates the review test database and opens a store over it.
func m1bStore(t *testing.T) (context.Context, *pgxpool.Pool, *store.PgStore) {
	t.Helper()
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
	t.Cleanup(pool.Close)
	return ctx, pool, store.New(pool)
}

// dropV1WeakAreaUnique simulates m1-08's contract step for this test: it drops the v1
// UNIQUE (account_id, week_of) and, on cleanup, removes the account's non-DSA snapshots
// and restores the constraint under its own name, so the shared test database keeps
// the v1.7.0 schema for every other test.
func dropV1WeakAreaUnique(ctx context.Context, t *testing.T, pool *pgxpool.Pool, acct string) {
	t.Helper()
	var name string
	if err := pool.QueryRow(ctx, `
		SELECT conname FROM pg_constraint
		WHERE conrelid = 'review.weak_area_snapshot'::regclass AND contype = 'u'`).Scan(&name); err != nil {
		t.Fatalf("find the v1 weak-area unique: %v", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE review.weak_area_snapshot DROP CONSTRAINT `+name); err != nil {
		t.Fatalf("drop %s: %v", name, err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM review.weak_area_snapshot WHERE account_id = $1 AND path_slug <> 'dsa'`, acct); err != nil {
			t.Errorf("cleanup: delete fixture snapshots: %v", err)
		}
		if _, err := pool.Exec(ctx, `ALTER TABLE review.weak_area_snapshot ADD CONSTRAINT `+name+` UNIQUE (account_id, week_of)`); err != nil {
			t.Errorf("cleanup: restore %s: %v", name, err)
		}
	})
}
