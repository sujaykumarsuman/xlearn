package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
)

// m1-07 (M1b): LiveMock — the account's live session while its window is open; none
// once it is scored, once its deadline has passed, or for another account.
func TestM1bLiveMock(t *testing.T) {
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

	acct := newTestUUID()
	if _, ok, err := st.LiveMock(ctx, acct); err != nil || ok {
		t.Fatalf("no session yet: ok=%v err=%v", ok, err)
	}

	// An expired live session (its window closed) does not lock the coach.
	past := time.Now().Add(-2 * store.MockDuration)
	if _, err := st.CreateMock(ctx, acct, "dsa", "set-1", "16", "med", past, past.Add(store.MockDuration)); err != nil {
		t.Fatalf("create expired: %v", err)
	}
	if _, ok, err := st.LiveMock(ctx, acct); err != nil || ok {
		t.Fatalf("expired session: ok=%v err=%v, want none", ok, err)
	}

	now := time.Now().Truncate(time.Second)
	m, err := st.CreateMock(ctx, acct, "dsa", "set-2", "17", "med", now, now.Add(store.MockDuration))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, ok, err := st.LiveMock(ctx, acct)
	if err != nil || !ok || got.ID != m.ID || got.PathSlug != "dsa" || !got.DeadlineAt.Equal(m.DeadlineAt) {
		t.Fatalf("live = %+v ok=%v err=%v, want %s", got, ok, err, m.ID)
	}
	if _, ok, _ := st.LiveMock(ctx, newTestUUID()); ok {
		t.Fatalf("another account sees the live session")
	}

	scores := map[string]int{}
	for _, d := range store.Dimensions {
		scores[d] = 3
	}
	if _, _, err := st.ScoreMock(ctx, acct, m.ID, scores, ""); err != nil {
		t.Fatalf("score: %v", err)
	}
	if _, ok, err := st.LiveMock(ctx, acct); err != nil || ok {
		t.Fatalf("scored session: ok=%v err=%v, want none", ok, err)
	}
}
