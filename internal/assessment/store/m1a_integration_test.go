package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/assessment/store/gen"
)

// m1-02 (M1a expand): CreateMock writes the session and its ordinal-1 item in one tx
// (item_id NULL for a mixed set), the course and the rubric; ScoreMock dual-writes
// total / max_total / scored_by beside total_35; readers use COALESCE(total, total_35),
// so a v1.5.2-written row (total_35 only) and a v1.7.0-written row (total only) read the
// same. The status/total CHECK is the relaxed COALESCE one.
func TestM1aMockDualWrite(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the assessment store integration test")
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
	q := gen.New(pool)
	acct := newTestUUID()
	now := time.Now().UTC()

	pinned, err := st.CreateMock(ctx, acct, "set-1", "16", "med", now, now.Add(store.MockDuration))
	if err != nil {
		t.Fatalf("create pinned: %v", err)
	}
	mixed, err := st.CreateMock(ctx, acct, "set-2", "", "hard", now.Add(time.Second), now.Add(store.MockDuration))
	if err != nil {
		t.Fatalf("create mixed: %v", err)
	}
	if pinned.PathSlug != "dsa" || pinned.MaxTotal != 35 || pinned.Total35 != nil {
		t.Fatalf("pinned session = %+v", pinned)
	}
	for _, c := range []struct {
		id       string
		wantItem *string
	}{{pinned.ID, ptr("16")}, {mixed.ID, nil}} {
		items, err := q.ListMockSessionItems(ctx, mustUUID(t, c.id))
		if err != nil || len(items) != 1 {
			t.Fatalf("items for %s: %v (%d rows)", c.id, err, len(items))
		}
		it := items[0]
		gotItem := (*string)(nil)
		if it.ItemID.Valid {
			gotItem = &it.ItemID.String
		}
		if it.Ordinal != 1 || it.PathSlug != "dsa" || (gotItem == nil) != (c.wantItem == nil) ||
			(gotItem != nil && *gotItem != *c.wantItem) {
			t.Fatalf("item row = %+v, want ordinal 1, dsa, item %v", it, c.wantItem)
		}
	}
	var rubricID string
	var hasSnapshot bool
	if err := pool.QueryRow(ctx, `SELECT rubric_id, rubric_snapshot->>'id' = rubric_id FROM assessment.mock_session WHERE id = $1`,
		pinned.ID).Scan(&rubricID, &hasSnapshot); err != nil || rubricID != store.RubricID || !hasSnapshot {
		t.Fatalf("rubric = %q snapshot ok=%v (%v)", rubricID, hasSnapshot, err)
	}

	scores := map[string]int{}
	for _, d := range store.Dimensions {
		scores[d] = 4
	}
	scored, _, err := st.ScoreMock(ctx, acct, pinned.ID, scores, "ok")
	if err != nil {
		t.Fatalf("score: %v", err)
	}
	if scored.Total35 == nil || *scored.Total35 != 28 || scored.MaxTotal != 35 {
		t.Fatalf("scored = %+v", scored)
	}
	var total, total35, maxTotal int
	var scoredBy string
	if err := pool.QueryRow(ctx, `SELECT total, total_35, max_total, scored_by FROM assessment.mock_session WHERE id = $1`,
		pinned.ID).Scan(&total, &total35, &maxTotal, &scoredBy); err != nil {
		t.Fatalf("read scored: %v", err)
	}
	if total != 28 || total35 != 28 || maxTotal != 35 || scoredBy != store.ScoredBySelf {
		t.Fatalf("dual-write: total=%d total_35=%d max=%d by=%s", total, total35, maxTotal, scoredBy)
	}

	// A v1.5.2-written scored row (its own INSERT + MarkMockScored: total_35 only) and a
	// v1.7.0-shaped one (total only) both satisfy the relaxed CHECK and read alike.
	var v152, v170 string
	if err := pool.QueryRow(ctx, `
		INSERT INTO assessment.mock_session (account_id, set_id, problem_id, difficulty, started_at, deadline_at, status, total_35)
		VALUES ($1, 's-152', '3', 'easy', $2, $3, 'scored', 21) RETURNING id::text`,
		acct, now.Add(2*time.Second), now.Add(store.MockDuration)).Scan(&v152); err != nil {
		t.Fatalf("v1.5.2-shaped insert: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO assessment.mock_session (account_id, set_id, problem_id, started_at, deadline_at, status, total, max_total, scored_by)
		VALUES ($1, 's-170', '', $2, $3, 'scored', 30, 35, 'self') RETURNING id::text`,
		acct, now.Add(3*time.Second), now.Add(store.MockDuration)).Scan(&v170); err != nil {
		t.Fatalf("v1.7.0-shaped insert (no total_35, no difficulty): %v", err)
	}
	for id, want := range map[string]int{v152: 21, v170: 30} {
		m, _, err := st.GetMock(ctx, acct, id)
		if err != nil || m.Total35 == nil || *m.Total35 != want || m.MaxTotal != 35 {
			t.Fatalf("GetMock %s = %+v (%v), want total %d /35", id, m, err, want)
		}
	}
	trend, err := st.Trend(ctx, acct)
	if err != nil || len(trend) != 3 || trend[0].Total35 != 28 || trend[1].Total35 != 21 || trend[2].Total35 != 30 {
		t.Fatalf("trend = %+v (%v)", trend, err)
	}
	stats, err := st.MockStats(ctx, acct)
	if err != nil || stats.Count != 3 || stats.Best != 30 {
		t.Fatalf("mock stats = %+v (%v)", stats, err)
	}

	// The relaxed CHECK still couples status and total: a scored row without any total
	// and a live row with one are rejected.
	for _, bad := range []string{
		`INSERT INTO assessment.mock_session (account_id, set_id, started_at, deadline_at, status) VALUES ($1, 'x', now(), now(), 'scored')`,
		`INSERT INTO assessment.mock_session (account_id, set_id, started_at, deadline_at, status, total) VALUES ($1, 'x', now(), now(), 'live', 20)`,
	} {
		if _, err := pool.Exec(ctx, bad, acct); err == nil {
			t.Fatalf("CHECK accepted: %s", bad)
		}
	}
	var oldCheck, newCheck int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE conname = 'mock_session_check'),
		       count(*) FILTER (WHERE conname = 'mock_session_scored_total_check' AND convalidated)
		FROM pg_constraint WHERE conrelid = 'assessment.mock_session'::regclass`).Scan(&oldCheck, &newCheck); err != nil {
		t.Fatalf("constraints: %v", err)
	}
	if oldCheck != 0 || newCheck != 1 {
		t.Fatalf("mock_session_check=%d, mock_session_scored_total_check(validated)=%d; want 0 and 1", oldCheck, newCheck)
	}

	// outbox.account_id on the mock_completed row.
	var withAccount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM assessment.outbox WHERE account_id::text = $1::text AND payload_json ->> 'account_id' = $1::text`,
		acct).Scan(&withAccount); err != nil || withAccount != 1 {
		t.Fatalf("outbox rows with account_id = %d (%v), want 1", withAccount, err)
	}
}

func ptr(s string) *string { return &s }

func mustUUID(t *testing.T, s string) pgtype.UUID {
	t.Helper()
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		t.Fatalf("uuid %q: %v", s, err)
	}
	return u
}
