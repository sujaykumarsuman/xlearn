package store_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// Sprint m1-07 against a real Postgres (gated on XLEARN_TEST_DATABASE_URL, like the other
// store integration tests): migration 00007's prompt_v / attempt_id on both rows of a
// turn, the recent-history window, and migration 00008's daily cap — whose atomic
// conditional upsert is the one piece of L18 the in-memory fake cannot prove.

func newM1bEnv(t *testing.T) (context.Context, *pgxpool.Pool, *store.PgStore) {
	t.Helper()
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the coach store integration test")
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

// Both rows of a turn carry prompt_v, attempt_id and path_slug; "" and a non-UUID attempt
// store NULL.
func TestM1bMessageMetaColumns(t *testing.T) {
	ctx, pool, st := newM1bEnv(t)
	acct := newTestUUID()
	attempt := newTestUUID()

	tid, err := st.EnsureThread(ctx, acct, "problem:16", course.DefaultSlug)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	meta := store.MessageMeta{PromptV: "coach-prompt@2", AttemptID: attempt}
	if err := st.AppendMessage(ctx, tid, store.RoleUser, "q1", meta); err != nil {
		t.Fatalf("append user: %v", err)
	}
	if err := st.AppendAssistantMessage(ctx, tid, "a1", store.MessageUsage{Provider: "openai", Model: "m"}, meta); err != nil {
		t.Fatalf("append assistant: %v", err)
	}
	// A turn with no attempt, then one whose attempt header was garbage.
	if err := st.AppendMessage(ctx, tid, store.RoleUser, "q2", store.MessageMeta{PromptV: "coach-prompt@2"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := st.AppendAssistantMessage(ctx, tid, "a2", store.MessageUsage{}, store.MessageMeta{PromptV: "coach-prompt@2", AttemptID: "not-a-uuid"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	// And one with no meta at all (a v1-shaped row).
	if err := st.AppendMessage(ctx, tid, store.RoleUser, "q3", store.MessageMeta{}); err != nil {
		t.Fatalf("append: %v", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT role, coalesce(prompt_v, '<null>'), coalesce(attempt_id::text, '<null>'), coalesce(path_slug, '<null>')
		FROM coach.coach_message WHERE thread_id = $1 ORDER BY seq`, tid)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	type row struct{ role, promptV, attempt, path string }
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.role, &r.promptV, &r.attempt, &r.path); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, r)
	}
	want := []row{
		{"user", "coach-prompt@2", attempt, "dsa"},
		{"assistant", "coach-prompt@2", attempt, "dsa"},
		{"user", "coach-prompt@2", "<null>", "dsa"},
		{"assistant", "coach-prompt@2", "<null>", "dsa"},
		{"user", "<null>", "<null>", "dsa"},
	}
	if len(got) != len(want) {
		t.Fatalf("rows = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// RecentMessages returns the last n oldest-first; ThreadHistory still returns them all.
func TestM1bRecentMessages(t *testing.T) {
	ctx, _, st := newM1bEnv(t)
	acct := newTestUUID()
	tid, err := st.EnsureThread(ctx, acct, "dsa:week:3", course.DefaultSlug)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	for i := 0; i < 25; i++ {
		if err := st.AppendMessage(ctx, tid, store.RoleUser, "m"+string(rune('a'+i)), store.MessageMeta{}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	recent, err := st.RecentMessages(ctx, tid, 20)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent) != 20 || recent[0].Content != "mf" || recent[19].Content != "my" {
		t.Fatalf("recent = %d, first %q last %q; want 20 from mf to my", len(recent), first(recent), last(recent))
	}
	all, err := st.ThreadHistory(ctx, acct, "dsa:week:3")
	if err != nil || len(all) != 25 || all[0].Content != "ma" {
		t.Fatalf("history = %d (%v), want all 25", len(all), err)
	}
	if few, err := st.RecentMessages(ctx, tid, 100); err != nil || len(few) != 25 {
		t.Fatalf("recent(100) = %d (%v), want 25", len(few), err)
	}
	if none, err := st.RecentMessages(ctx, newTestUUID(), 20); err != nil || len(none) != 0 {
		t.Fatalf("recent on an unknown thread = %d (%v), want empty", len(none), err)
	}
}

func first(ms []store.Message) string {
	if len(ms) == 0 {
		return ""
	}
	return ms[0].Content
}

func last(ms []store.Message) string {
	if len(ms) == 0 {
		return ""
	}
	return ms[len(ms)-1].Content
}

// The daily cap's upsert, sequentially: a fresh day inserts 1, a claim at the cap writes
// nothing and reports !ok, a refund gives one back (never below zero), and each UTC day —
// computed from the instant, whatever its zone — is its own row.
func TestM1bDailyQuota(t *testing.T) {
	ctx, _, st := newM1bEnv(t)
	acct := newTestUUID()
	// 23:30 UTC on Oct 1 is 05:00 on Oct 2 in IST: still Oct 1's row.
	day := time.Date(2026, time.October, 2, 5, 0, 0, 0, time.FixedZone("IST", 5*3600+1800))
	const limit = 3

	if n, err := st.DailyMessages(ctx, acct, day); err != nil || n != 0 {
		t.Fatalf("fresh day = %d (%v), want 0", n, err)
	}
	for want := 1; want <= limit; want++ {
		n, ok, err := st.TakeDailyMessage(ctx, acct, day, limit)
		if err != nil || !ok || n != want {
			t.Fatalf("take %d = %d %v %v", want, n, ok, err)
		}
	}
	if _, ok, err := st.TakeDailyMessage(ctx, acct, day, limit); err != nil || ok {
		t.Fatalf("take at the cap: ok=%v err=%v, want refused", ok, err)
	}
	if n, _ := st.DailyMessages(ctx, acct, day); n != limit {
		t.Fatalf("a refused take changed n: %d", n)
	}
	utc := time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC)
	if n, _ := st.DailyMessages(ctx, acct, utc); n != limit {
		t.Fatalf("the IST instant was not filed under its UTC day: %d", n)
	}

	if err := st.RefundDailyMessage(ctx, acct, day); err != nil {
		t.Fatalf("refund: %v", err)
	}
	if n, ok, err := st.TakeDailyMessage(ctx, acct, day, limit); err != nil || !ok || n != limit {
		t.Fatalf("take after a refund = %d %v %v", n, ok, err)
	}

	// The next UTC day is a fresh row; refunding a day with no row is a no-op.
	next := utc.Add(24 * time.Hour)
	if n, ok, err := st.TakeDailyMessage(ctx, acct, next, limit); err != nil || !ok || n != 1 {
		t.Fatalf("next day's first take = %d %v %v", n, ok, err)
	}
	other := next.Add(24 * time.Hour)
	if err := st.RefundDailyMessage(ctx, acct, other); err != nil {
		t.Fatalf("refund on an empty day: %v", err)
	}
	if n, _ := st.DailyMessages(ctx, acct, other); n != 0 {
		t.Fatalf("refund on an empty day made n=%d", n)
	}
}

// TestM1bDailyQuotaRace: concurrent claims cannot overshoot the cap. 50 goroutines start
// from n = 290 under the real cap of 300: exactly 10 are admitted and n ends at 300.
func TestM1bDailyQuotaRace(t *testing.T) {
	ctx, pool, st := newM1bEnv(t)
	acct := newTestUUID()
	day := time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC)
	const limit, start, workers = 300, 290, 50

	if _, err := pool.Exec(ctx, `INSERT INTO coach.message_quota_day (account_id, day, n) VALUES ($1, $2, $3)`,
		acct, "2026-10-01", start); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var admitted, refused atomic.Int64
	var wg sync.WaitGroup
	gate := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			_, ok, err := st.TakeDailyMessage(ctx, acct, day, limit)
			switch {
			case err != nil:
				t.Errorf("take: %v", err)
			case ok:
				admitted.Add(1)
			default:
				refused.Add(1)
			}
		}()
	}
	close(gate)
	wg.Wait()

	if admitted.Load() != limit-start || refused.Load() != workers-(limit-start) {
		t.Fatalf("admitted %d, refused %d; want %d and %d", admitted.Load(), refused.Load(), limit-start, workers-(limit-start))
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT n FROM coach.message_quota_day WHERE account_id = $1 AND day = $2`, acct, "2026-10-01").Scan(&n); err != nil {
		t.Fatalf("read n: %v", err)
	}
	if n != limit {
		t.Fatalf("final n = %d, want %d", n, limit)
	}
}
