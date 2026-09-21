package store_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
)

// Integration test against a real Postgres, gated on XLEARN_TEST_DATABASE_URL so CI
// (which has no database) skips it. Run locally, e.g.:
//
//	XLEARN_TEST_DATABASE_URL=postgres://xlearn_assessment:pw@localhost:5599/xlearndb?sslmode=disable \
//	  go test ./internal/assessment/store/ -run TestStore -count=1
//
// The DSN must be for the xlearn_assessment role (search_path assessment, owning only
// schema assessment) so the test also exercises the least-privilege model.
func TestStoreIntegration(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the assessment store integration test")
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

	t.Run("create -> live session with server window", func(t *testing.T) {
		acct := newTestUUID()
		start := time.Now().Truncate(time.Second)
		m, err := st.CreateMock(ctx, acct, "set-07", "16", "med", start, start.Add(store.MockDuration))
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if m.Status != store.StatusLive || m.Total35 != nil {
			t.Fatalf("new session not live/unscored: %+v", m)
		}
		if m.DeadlineAt.Sub(m.StartedAt) != store.MockDuration {
			t.Fatalf("window = %v, want 45m", m.DeadlineAt.Sub(m.StartedAt))
		}
		got, scores, err := st.GetMock(ctx, acct, m.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.ID != m.ID || got.SetID != "set-07" || got.ProblemID != "16" || got.Difficulty != "med" {
			t.Fatalf("round-trip mismatch: %+v", got)
		}
		if len(scores) != 0 {
			t.Fatalf("live session has %d scores, want 0", len(scores))
		}
	})

	t.Run("score computes /35, latches scored, emits one mock_completed", func(t *testing.T) {
		acct := newTestUUID()
		start := time.Now()
		m, err := st.CreateMock(ctx, acct, "set-07", "16", "med", start, start.Add(store.MockDuration))
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		rubric := map[string]int{
			"communication": 4, "problem_understanding": 4, "brute_force": 3,
			"optimisation": 3, "code_quality": 4, "edge_cases": 3, "complexity": 3,
		}
		scored, scores, err := st.ScoreMock(ctx, acct, m.ID, rubric, "named the pattern late")
		if err != nil {
			t.Fatalf("score: %v", err)
		}
		if scored.Status != store.StatusScored || scored.Total35 == nil || *scored.Total35 != 24 {
			t.Fatalf("scored total wrong: %+v", scored)
		}
		if len(scores) != 7 {
			t.Fatalf("returned %d scores, want 7", len(scores))
		}
		if c := countRubric(ctx, t, pool, m.ID); c != 7 {
			t.Fatalf("rubric rows = %d, want 7", c)
		}
		if c := countOutbox(ctx, t, pool, store.SubjectMockCompleted, acct); c != 1 {
			t.Fatalf("mock_completed events = %d, want 1", c)
		}
		// Outbox payload carries mock_id, total_35, rubric (events.md).
		assertMockCompletedPayload(ctx, t, pool, acct, m.ID, 24)

		// GetMock now returns the scored session + rubric.
		got, gotScores, err := st.GetMock(ctx, acct, m.ID)
		if err != nil {
			t.Fatalf("get scored: %v", err)
		}
		if got.Status != store.StatusScored || len(gotScores) != 7 || got.Notes != "named the pattern late" {
			t.Fatalf("scored round-trip wrong: %+v scores=%d", got, len(gotScores))
		}
	})

	t.Run("re-score is idempotent (no new rubric/outbox rows)", func(t *testing.T) {
		acct := newTestUUID()
		start := time.Now()
		m, _ := st.CreateMock(ctx, acct, "set-07", "16", "med", start, start.Add(store.MockDuration))
		rubric := allFives()
		if _, _, err := st.ScoreMock(ctx, acct, m.ID, rubric, "first"); err != nil {
			t.Fatalf("first score: %v", err)
		}
		// Re-submit with DIFFERENT scores: the stored result must win (idempotent).
		lower := map[string]int{
			"communication": 1, "problem_understanding": 1, "brute_force": 1,
			"optimisation": 1, "code_quality": 1, "edge_cases": 1, "complexity": 1,
		}
		again, _, err := st.ScoreMock(ctx, acct, m.ID, lower, "second")
		if err != nil {
			t.Fatalf("re-score: %v", err)
		}
		if again.Total35 == nil || *again.Total35 != 35 || again.Notes != "first" {
			t.Fatalf("re-score should return the stored 35/'first', got %+v", again)
		}
		if c := countRubric(ctx, t, pool, m.ID); c != 7 {
			t.Fatalf("rubric rows after re-score = %d, want 7", c)
		}
		if c := countOutbox(ctx, t, pool, store.SubjectMockCompleted, acct); c != 1 {
			t.Fatalf("mock_completed after re-score = %d, want 1", c)
		}
	})

	t.Run("invalid rubric and cross-account are rejected", func(t *testing.T) {
		acct := newTestUUID()
		start := time.Now()
		m, _ := st.CreateMock(ctx, acct, "set-07", "16", "med", start, start.Add(store.MockDuration))

		if _, _, err := st.ScoreMock(ctx, acct, m.ID, map[string]int{"communication": 4}, ""); err != store.ErrInvalidRubric {
			t.Fatalf("partial rubric err = %v, want ErrInvalidRubric", err)
		}
		// A different account cannot score or read this session.
		other := newTestUUID()
		if _, _, err := st.ScoreMock(ctx, other, m.ID, allFives(), ""); err != store.ErrNotFound {
			t.Fatalf("cross-account score err = %v, want ErrNotFound", err)
		}
		if _, _, err := st.GetMock(ctx, other, m.ID); err != store.ErrNotFound {
			t.Fatalf("cross-account get err = %v, want ErrNotFound", err)
		}
		if _, _, err := st.GetMock(ctx, acct, newTestUUID()); err != store.ErrNotFound {
			t.Fatalf("missing get err = %v, want ErrNotFound", err)
		}
	})

	t.Run("trend returns scored mocks oldest-first", func(t *testing.T) {
		acct := newTestUUID()
		base := time.Now().Add(-30 * 24 * time.Hour)
		// Three sessions started 14 / 7 / 0 days ago; score the first two only.
		older, _ := st.CreateMock(ctx, acct, "set-05", "16", "med", base, base.Add(store.MockDuration))
		mid, _ := st.CreateMock(ctx, acct, "set-06", "17", "med", base.AddDate(0, 0, 7), base.AddDate(0, 0, 7).Add(store.MockDuration))
		_, _ = st.CreateMock(ctx, acct, "set-07", "18", "med", base.AddDate(0, 0, 14), base.AddDate(0, 0, 14).Add(store.MockDuration))
		mustScore(ctx, t, st, acct, older.ID, 20)
		mustScore(ctx, t, st, acct, mid.ID, 24)

		points, err := st.Trend(ctx, acct)
		if err != nil {
			t.Fatalf("trend: %v", err)
		}
		if len(points) != 2 {
			t.Fatalf("trend points = %d, want 2 (only scored)", len(points))
		}
		if points[0].Total35 != 20 || points[1].Total35 != 24 {
			t.Fatalf("trend order/totals wrong: %+v", points)
		}
		if !points[0].StartedAt.Before(points[1].StartedAt) {
			t.Fatalf("trend not oldest-first")
		}
	})

	t.Run("projection consume dedupes on event_id", func(t *testing.T) {
		acct := newTestUUID()
		eid := newTestUUID()
		fresh, err := st.RecordProjectionEvent(ctx, eid, "xlearn.practice.problem_solved", acct, []byte(`{"problem_id":"16"}`))
		if err != nil {
			t.Fatalf("record: %v", err)
		}
		if !fresh {
			t.Fatal("first delivery should be fresh")
		}
		fresh2, err := st.RecordProjectionEvent(ctx, eid, "xlearn.practice.problem_solved", acct, []byte(`{"problem_id":"16"}`))
		if err != nil {
			t.Fatalf("record dup: %v", err)
		}
		if fresh2 {
			t.Fatal("re-delivery of the same event_id must be a no-op (not fresh)")
		}
	})

	t.Run("concurrent scoring stays idempotent (one event, one settle)", func(t *testing.T) {
		acct := newTestUUID()
		start := time.Now()
		m, _ := st.CreateMock(ctx, acct, "set-07", "16", "med", start, start.Add(store.MockDuration))

		const n = 6
		var wg sync.WaitGroup
		errs := make([]error, n)
		totals := make([]int, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				scored, _, err := st.ScoreMock(ctx, acct, m.ID, allFives(), "concurrent")
				errs[i] = err
				if err == nil && scored.Total35 != nil {
					totals[i] = *scored.Total35
				}
			}(i)
		}
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Fatalf("concurrent score %d: %v", i, err)
			}
			if totals[i] != 35 {
				t.Fatalf("concurrent score %d total = %d, want 35", i, totals[i])
			}
		}
		if c := countRubric(ctx, t, pool, m.ID); c != 7 {
			t.Fatalf("rubric rows after concurrent = %d, want 7 (no duplicates)", c)
		}
		if c := countOutbox(ctx, t, pool, store.SubjectMockCompleted, acct); c != 1 {
			t.Fatalf("mock_completed after concurrent = %d, want exactly 1", c)
		}
	})
}

// --- helpers ---

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func allFives() map[string]int {
	m := make(map[string]int, len(store.Dimensions))
	for _, d := range store.Dimensions {
		m[d] = 5
	}
	return m
}

func mustScore(ctx context.Context, t *testing.T, st store.Store, acct, mockID string, total int) {
	t.Helper()
	// Build a rubric summing to `total` (spread across the seven dims, each in [1,5]).
	m := make(map[string]int, len(store.Dimensions))
	remaining := total
	for i, d := range store.Dimensions {
		left := len(store.Dimensions) - i
		v := remaining - (left - 1) // leave >=1 for each remaining dim
		if v > 5 {
			v = 5
		}
		if v < 1 {
			v = 1
		}
		m[d] = v
		remaining -= v
	}
	if got := store.TotalScore(m); got != total {
		t.Fatalf("rubric builder made %d, want %d", got, total)
	}
	if _, _, err := st.ScoreMock(ctx, acct, mockID, m, ""); err != nil {
		t.Fatalf("score to %d: %v", total, err)
	}
}

func countRubric(ctx context.Context, t *testing.T, pool *pgxpool.Pool, mockID string) int {
	t.Helper()
	var c int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM assessment.rubric_score WHERE mock_session_id = $1`, mockID).Scan(&c); err != nil {
		t.Fatalf("count rubric: %v", err)
	}
	return c
}

func countOutbox(ctx context.Context, t *testing.T, pool *pgxpool.Pool, subject, accountID string) int {
	t.Helper()
	var c int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM assessment.outbox WHERE subject = $1 AND payload_json->>'account_id' = $2`,
		subject, accountID).Scan(&c); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	return c
}

func assertMockCompletedPayload(ctx context.Context, t *testing.T, pool *pgxpool.Pool, accountID, mockID string, wantTotal int) {
	t.Helper()
	var raw []byte
	if err := pool.QueryRow(ctx,
		`SELECT payload_json FROM assessment.outbox WHERE subject = $1 AND payload_json->>'account_id' = $2`,
		store.SubjectMockCompleted, accountID).Scan(&raw); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	var env struct {
		Subject   string `json:"subject"`
		AccountID string `json:"account_id"`
		Data      struct {
			MockID  string         `json:"mock_id"`
			Total35 int            `json:"total_35"`
			Rubric  map[string]int `json:"rubric"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if env.Data.MockID != mockID || env.Data.Total35 != wantTotal {
		t.Fatalf("payload mock_id/total = %q/%d, want %q/%d", env.Data.MockID, env.Data.Total35, mockID, wantTotal)
	}
	if len(env.Data.Rubric) != 7 {
		t.Fatalf("payload rubric has %d dims, want 7", len(env.Data.Rubric))
	}
}

func newTestUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf[:])
}
