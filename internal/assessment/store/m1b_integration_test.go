package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/assessment/store/gen"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// m1-03 (M1b), assessment's half: a session and its item row carry the caller's
// course, the trend and the mock roll-up are per course, scoring writes total only (the
// v1 /35 column stays NULL), and mock_completed is a v2 envelope carrying the session's
// course. A fixture course with a mock ("zz-preview"; the internal service leaves
// visibility to the gateway) beside the DSA course proves the course is carried.
func TestM1bMockCourseScoped(t *testing.T) {
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
	const fixture, dsa = "zz-preview", "dsa"
	base := time.Now().Add(-time.Hour)

	create := func(course, problem string, at time.Time) store.MockSession {
		t.Helper()
		m, err := st.CreateMock(ctx, acct, course, "set-1", problem, "med", at, at.Add(store.MockDuration))
		if err != nil {
			t.Fatalf("create in %s: %v", course, err)
		}
		if m.PathSlug != course {
			t.Fatalf("session path_slug = %q, want %q", m.PathSlug, course)
		}
		items, err := q.ListMockSessionItems(ctx, mustUUID(t, m.ID))
		if err != nil || len(items) != 1 || items[0].PathSlug != course {
			t.Fatalf("items = %+v (%v), want one in %s", items, err, course)
		}
		return m
	}
	scoreAll := func(m store.MockSession, v int) {
		t.Helper()
		scores := map[string]int{}
		for _, d := range store.Dimensions {
			scores[d] = v
		}
		if _, _, err := st.ScoreMock(ctx, acct, m.ID, scores, ""); err != nil {
			t.Fatalf("score: %v", err)
		}
	}

	dsaMock := create(dsa, "16", base)
	fixMock1 := create(fixture, "zz-preview:e-1", base.Add(time.Minute))
	fixMock2 := create(fixture, "", base.Add(2*time.Minute))
	scoreAll(dsaMock, 4)  // 28
	scoreAll(fixMock1, 2) // 14
	scoreAll(fixMock2, 3) // 21

	// Scoring writes total only: the v1 /35 column stays NULL on every new row.
	var legacy int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM assessment.mock_session WHERE account_id = $1 AND total_35 IS NOT NULL`, acct).Scan(&legacy); err != nil || legacy != 0 {
		t.Fatalf("rows with the v1 /35 column written = %d (%v), want 0", legacy, err)
	}

	for _, c := range []struct {
		course string
		totals []int
		best   int
	}{
		{dsa, []int{28}, 28},
		{fixture, []int{14, 21}, 21},
	} {
		trend, err := st.Trend(ctx, acct, c.course)
		if err != nil || len(trend) != len(c.totals) {
			t.Fatalf("%s trend = %+v (%v), want %v", c.course, trend, err, c.totals)
		}
		for i, want := range c.totals {
			if trend[i].Total != want || trend[i].MaxTotal != store.MaxTotal {
				t.Errorf("%s trend[%d] = %+v, want total %d /35", c.course, i, trend[i], want)
			}
		}
		stats, err := st.MockStats(ctx, acct, c.course)
		if err != nil || stats.Count != len(c.totals) || stats.Best != c.best {
			t.Errorf("%s mock stats = %+v (%v), want %d scored, best %d", c.course, stats, err, len(c.totals), c.best)
		}
	}
	if trend, err := st.Trend(ctx, acct, "zz-none"); err != nil || len(trend) != 0 {
		t.Errorf("a course without mocks: trend %+v (%v)", trend, err)
	}

	// Every mock_completed is a v2 envelope in its session's course.
	rows, err := pool.Query(ctx, `SELECT payload_json FROM assessment.outbox WHERE account_id = $1`, acct)
	if err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	defer rows.Close()
	byCourse := map[string]int{}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			t.Fatalf("scan: %v", err)
		}
		env, err := events.DecodeEnvelope(payload)
		if err != nil || env.Version != events.EnvelopeV2 || env.Subject != store.SubjectMockCompleted {
			t.Fatalf("envelope %+v (%v), want a v2 mock_completed", env, err)
		}
		byCourse[env.PathSlug]++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if byCourse[dsa] != 1 || byCourse[fixture] != 2 || len(byCourse) != 2 {
		t.Fatalf("mock_completed by course = %v, want dsa:1 %s:2", byCourse, fixture)
	}

	if _, err := st.CreateMock(ctx, acct, "", "set-1", "", "med", base, base.Add(store.MockDuration)); err == nil {
		t.Fatalf("a course-less mock was created")
	}
}
