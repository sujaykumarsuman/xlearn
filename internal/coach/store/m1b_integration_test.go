package store_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// preM1bVersion is coach's last goose version before m1-03's data migration (00005).
const preM1bVersion = 4

// m1-03 (M1b) data migration 00005: v1 course-scoped page contexts take the `dsa:` prefix
// and the course-scoped threads and their messages get path_slug = the DSA course;
// problem:<id>, the account-wide contexts and anything the parser passes through keep
// their key and a NULL path_slug; a v1 row whose `dsa:` twin exists is skipped (UNIQUE
// (account_id, page_context)); and a second run changes nothing. The seed goes in below
// 00005 (MigrateDownTo; 00005's Down is a no-op), so goose itself applies the migration
// to it, twice.
func TestM1bPageContextMigration(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the coach store integration test")
	}
	ctx := context.Background()
	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := store.MigrateDownTo(ctx, dsn, preM1bVersion); err != nil {
		t.Fatalf("migrate down: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	// seed writes a thread the way v1.6.0 does (no path_slug) with one message, and
	// returns the thread id.
	seed := func(acct, pageContext string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO coach.coach_thread (account_id, page_context) VALUES ($1, $2) RETURNING id::text`,
			acct, pageContext).Scan(&id); err != nil {
			t.Fatalf("seed thread %q: %v", pageContext, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO coach.coach_message (thread_id, role, content) VALUES ($1, 'user', $2)`,
			id, "on "+pageContext); err != nil {
			t.Fatalf("seed message %q: %v", pageContext, err)
		}
		return id
	}
	type row struct {
		id       string
		path     string // "" = NULL
		msgPaths []string
	}
	// read returns the (account, key) thread with its path_slug and its messages'.
	read := func(acct, key string) (row, bool) {
		t.Helper()
		var r row
		var path *string
		err := pool.QueryRow(ctx,
			`SELECT id::text, path_slug FROM coach.coach_thread WHERE account_id = $1 AND page_context = $2`,
			acct, key).Scan(&r.id, &path)
		if err != nil {
			return row{}, false
		}
		if path != nil {
			r.path = *path
		}
		rows, err := pool.Query(ctx, `SELECT coalesce(path_slug, '') FROM coach.coach_message WHERE thread_id = $1 ORDER BY seq`, r.id)
		if err != nil {
			t.Fatalf("read messages: %v", err)
		}
		defer rows.Close()
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err != nil {
				t.Fatalf("scan message: %v", err)
			}
			r.msgPaths = append(r.msgPaths, p)
		}
		return r, true
	}
	// snapshot is every coach thread and message, for the re-run check.
	snapshot := func() string {
		t.Helper()
		var s string
		if err := pool.QueryRow(ctx, `
			SELECT coalesce((SELECT string_agg(id::text || '|' || account_id::text || '|' || page_context || '|' || coalesce(path_slug, '-'), ',' ORDER BY id)
			                 FROM coach.coach_thread), '')
			    || '#' ||
			       coalesce((SELECT string_agg(id::text || '|' || thread_id::text || '|' || coalesce(path_slug, '-'), ',' ORDER BY id)
			                 FROM coach.coach_message), '')`).Scan(&s); err != nil {
			t.Fatalf("snapshot: %v", err)
		}
		return s
	}

	acct, twin := newTestUUID(), newTestUUID()
	v1CourseScoped := []string{"concept:two-pointers", "week:3", "roadmap", "dashboard", "revision", "mistakes", "mock", "progress"}
	kept := []string{"problem:16", "catalog", "settings", "general", "concept:", "concept:a:b", "no-such-course:week:3"}
	seeded := map[string]string{}
	for _, c := range append(append(append([]string{}, v1CourseScoped...), kept...), "dsa:week:4") {
		seeded[c] = seed(acct, c)
	}
	// The collision: the v1 row and its dsa: twin for one account.
	twinLegacy := seed(twin, "dashboard")
	twinNew := seed(twin, "dsa:dashboard")

	// Run 1: goose applies 00005.
	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate (run 1): %v", err)
	}
	for _, c := range v1CourseScoped {
		r, ok := read(acct, "dsa:"+c)
		if !ok || r.id != seeded[c] {
			t.Fatalf("%q: not rewritten in place to dsa:%s (got %+v, ok %v)", c, c, r, ok)
		}
		if r.path != "dsa" || len(r.msgPaths) != 1 || r.msgPaths[0] != "dsa" {
			t.Fatalf("%q: path_slug %q, messages %v; want dsa", c, r.path, r.msgPaths)
		}
		if _, ok := read(acct, c); ok {
			t.Fatalf("%q: the v1 key survived the rewrite", c)
		}
	}
	if r, ok := read(acct, "dsa:week:4"); !ok || r.path != "dsa" || r.msgPaths[0] != "dsa" {
		t.Fatalf("already-prefixed dsa:week:4: %+v ok %v, want path_slug dsa", r, ok)
	}
	for _, c := range kept {
		r, ok := read(acct, c)
		if !ok || r.id != seeded[c] || r.path != "" || r.msgPaths[0] != "" {
			t.Fatalf("%q: %+v ok %v; want the key kept and path_slug NULL", c, r, ok)
		}
	}
	// The collision: the twin is untouched but labelled; the v1 row keeps its key (the
	// rewrite would break UNIQUE) and its message, labelled as the DSA course's.
	if r, ok := read(twin, "dsa:dashboard"); !ok || r.id != twinNew || r.path != "dsa" || r.msgPaths[0] != "dsa" {
		t.Fatalf("twin: %+v ok %v", r, ok)
	}
	if r, ok := read(twin, "dashboard"); !ok || r.id != twinLegacy || r.path != "dsa" || len(r.msgPaths) != 1 || r.msgPaths[0] != "dsa" {
		t.Fatalf("skipped v1 row: %+v ok %v; want kept under its key, labelled dsa", r, ok)
	}

	// Run 2: 00005 again, through goose, is a no-op.
	before := snapshot()
	if err := store.MigrateDownTo(ctx, dsn, preM1bVersion); err != nil {
		t.Fatalf("migrate down (run 2): %v", err)
	}
	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate (run 2): %v", err)
	}
	if after := snapshot(); after != before {
		t.Fatalf("the second run changed rows:\nbefore %s\nafter  %s", before, after)
	}

	// The migrated history is what v1.7.0 reads under the normalized key.
	st := store.New(pool)
	hist, err := st.ThreadHistory(ctx, acct, course.NormalizeCoachContext("week:3").Key)
	if err != nil || len(hist) != 1 || hist[0].Content != "on week:3" {
		t.Fatalf("history under the normalized key: %+v err %v", hist, err)
	}
}

// m1-03: the thread and each message carry the thread's course. An existing path_slug
// is kept, a NULL one (a v1 problem thread) is filled on the next chat, an account-wide
// thread stays NULL, and two courses' week 3 are two threads.
func TestM1bThreadPathSlug(t *testing.T) {
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
	defer pool.Close()
	st := store.New(pool)
	acct := newTestUUID()

	// pathOf returns the thread's path_slug and each of its messages' ("" = NULL).
	pathOf := func(threadID string) (string, []string) {
		t.Helper()
		var p string
		if err := pool.QueryRow(ctx, `SELECT coalesce(path_slug, '') FROM coach.coach_thread WHERE id = $1`, threadID).Scan(&p); err != nil {
			t.Fatalf("thread path: %v", err)
		}
		rows, err := pool.Query(ctx, `SELECT coalesce(path_slug, '') FROM coach.coach_message WHERE thread_id = $1 ORDER BY seq`, threadID)
		if err != nil {
			t.Fatalf("message paths: %v", err)
		}
		defer rows.Close()
		var ms []string
		for rows.Next() {
			var m string
			if err := rows.Scan(&m); err != nil {
				t.Fatalf("scan: %v", err)
			}
			ms = append(ms, m)
		}
		return p, ms
	}
	ensure := func(key, pathSlug string) string {
		t.Helper()
		id, err := st.EnsureThread(ctx, acct, key, pathSlug)
		if err != nil {
			t.Fatalf("ensure %q: %v", key, err)
		}
		if err := st.AppendMessage(ctx, id, store.RoleUser, "hi"); err != nil {
			t.Fatalf("append on %q: %v", key, err)
		}
		return id
	}

	// Two courses' week 3: two threads, each with its course on thread and message.
	dsaWeek := ensure("dsa:week:3", course.DefaultSlug)
	fxWeek := ensure(coursetest.FixtureActive+":week:3", coursetest.FixtureActive)
	if dsaWeek == fxWeek {
		t.Fatal("two courses' week 3 share a thread")
	}
	if p, ms := pathOf(dsaWeek); p != course.DefaultSlug || len(ms) != 1 || ms[0] != course.DefaultSlug {
		t.Fatalf("DSA week 3: %q %v", p, ms)
	}
	if p, ms := pathOf(fxWeek); p != coursetest.FixtureActive || len(ms) != 1 || ms[0] != coursetest.FixtureActive {
		t.Fatalf("fixture week 3: %q %v", p, ms)
	}
	// An existing path_slug is kept (same thread, same course).
	if again := ensure("dsa:week:3", coursetest.FixtureActive); again != dsaWeek {
		t.Fatal("re-ensure made a new thread")
	}
	if p, ms := pathOf(dsaWeek); p != course.DefaultSlug || ms[1] != course.DefaultSlug {
		t.Fatalf("re-ensure relabelled the thread: %q %v", p, ms)
	}

	// A v1 problem thread (NULL path_slug) is filled on its next chat.
	var legacy string
	if err := pool.QueryRow(ctx, `INSERT INTO coach.coach_thread (account_id, page_context) VALUES ($1, 'problem:16') RETURNING id::text`, acct).Scan(&legacy); err != nil {
		t.Fatalf("seed v1 problem thread: %v", err)
	}
	if id := ensure("problem:16", course.DefaultSlug); id != legacy {
		t.Fatal("the v1 problem thread was not reused")
	}
	if p, ms := pathOf(legacy); p != course.DefaultSlug || ms[0] != course.DefaultSlug {
		t.Fatalf("v1 problem thread: %q %v; want filled", p, ms)
	}

	// Account-wide: NULL on thread and message.
	if p, ms := pathOf(ensure("settings", "")); p != "" || ms[0] != "" {
		t.Fatalf("account-wide: %q %v; want NULL", p, ms)
	}

	// A message on a missing thread is ErrNotFound (no row to take the course from).
	if err := st.AppendMessage(ctx, newTestUUID(), store.RoleUser, "x"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("append on a missing thread: %v, want ErrNotFound", err)
	}
}
