package curriculum

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

// Database-backed tests of the curriculum seed (sprint m1-09). They need a real
// Postgres and are gated on XLEARN_TEST_DATABASE_URL, like the store integration test.
// The DSN must be the xlearn_curriculum role (owning only schema curriculum), e.g.:
//
//	XLEARN_TEST_DATABASE_URL=postgres://xlearn_curriculum:pw@localhost:5432/xlearn_store?sslmode=disable \
//	  go test -count=1 -p 1 ./internal/curriculum/...
//
// Every test starts from a fresh schema (every table in schema curriculum dropped, then
// the goose migrations), so run the curriculum packages with -p 1 against one database:
// the store integration test shares the schema.

// update rewrites the testdata fixtures instead of comparing against them.
var update = flag.Bool("update", false, "rewrite internal/curriculum/testdata fixtures")

// snapshotFile is the v1 seeded-row snapshot, committed before the converter (m1-09
// task 1a) so it provably predates the new loader.
var snapshotFile = filepath.Join("testdata", "v1-seed-snapshot.json")

// testPool returns a pool on XLEARN_TEST_DATABASE_URL, skipping the test without one.
func testPool(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the curriculum seed tests against Postgres")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool, dsn
}

// freshSchema drops every table in schema curriculum (goose's version table included)
// and applies the migrations, so the test sees a schema exactly as a new install would.
func freshSchema(t *testing.T, pool *pgxpool.Pool, dsn string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
DO $$
DECLARE r record;
BEGIN
    FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'curriculum' LOOP
        EXECUTE format('DROP TABLE IF EXISTS curriculum.%I CASCADE', r.tablename);
    END LOOP;
END $$`); err != nil {
		t.Fatalf("drop curriculum tables: %v", err)
	}
	if err := store.Migrate(ctx, dsn, discardLogger()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

// rowSnapshot is every seeded row of the curriculum tables, uuids stripped, in natural-key
// order. week_concept is recorded as (path, week n, concept slug) triples. Only the v1
// columns are captured, so the same snapshot compares a v1 seed with a v2 one.
type rowSnapshot struct {
	Path           []map[string]any `json:"path"`
	Phase          []map[string]any `json:"phase"`
	Week           []map[string]any `json:"week"`
	Concept        []map[string]any `json:"concept"`
	WeekConcept    []map[string]any `json:"week_concept"`
	Problem        []map[string]any `json:"problem"`
	ProblemSection []map[string]any `json:"problem_section"`
}

// v1SnapshotQueries select the v1 columns of each table in natural-key order.
var v1SnapshotQueries = []struct {
	table string
	sql   string
}{
	{"path", `SELECT slug, title, status, summary, problem_total, week_total, sort_order
		FROM curriculum.path ORDER BY slug COLLATE "C"`},
	{"phase", `SELECT path_slug, "order", name, theme, week_from, week_to
		FROM curriculum.phase ORDER BY path_slug COLLATE "C", "order"`},
	{"week", `SELECT path_slug, n, title, thesis
		FROM curriculum.week ORDER BY path_slug COLLATE "C", n`},
	{"concept", `SELECT path_slug, slug, title, body_md, when_to_use_md, code_template
		FROM curriculum.concept ORDER BY path_slug COLLATE "C", slug COLLATE "C"`},
	{"week_concept", `SELECT w.path_slug, w.n AS week_n, c.slug AS concept_slug
		FROM curriculum.week_concept wc
		JOIN curriculum.week w    ON w.id = wc.week_id
		JOIN curriculum.concept c ON c.id = wc.concept_id
		ORDER BY w.path_slug COLLATE "C", w.n, c.slug COLLATE "C"`},
	{"problem", `SELECT id, path_slug, week_n, title, difficulty, pattern,
			leetcode_url, neetcode_url, is_reinforcement, sort_order
		FROM curriculum.problem ORDER BY path_slug COLLATE "C", id COLLATE "C"`},
	{"problem_section", `SELECT problem_id, stage, "order", kind, body_md, code
		FROM curriculum.problem_section ORDER BY problem_id COLLATE "C", stage COLLATE "C", "order"`},
}

// queryRows runs sql and returns its rows as column -> value maps.
func queryRows(t *testing.T, pool *pgxpool.Pool, sql string) []map[string]any {
	t.Helper()
	rows, err := pool.Query(context.Background(), sql)
	if err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		t.Fatalf("collect %q: %v", sql, err)
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out
}

// snapshotRows captures the v1 columns of every curriculum table.
func snapshotRows(t *testing.T, pool *pgxpool.Pool) rowSnapshot {
	t.Helper()
	var s rowSnapshot
	dst := map[string]*[]map[string]any{
		"path": &s.Path, "phase": &s.Phase, "week": &s.Week, "concept": &s.Concept,
		"week_concept": &s.WeekConcept, "problem": &s.Problem, "problem_section": &s.ProblemSection,
	}
	for _, q := range v1SnapshotQueries {
		*dst[q.table] = queryRows(t, pool, q.sql)
	}
	return s
}

// encodeSnapshot renders a snapshot the way the fixture stores it: indented, sorted keys,
// no HTML escaping (so the Markdown bodies stay readable in review).
func encodeSnapshot(t *testing.T, v any) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		t.Fatalf("encode snapshot: %v", err)
	}
	return buf.Bytes()
}

// normalize round-trips a snapshot through JSON so values compare by their JSON form
// (int32 vs float64 and the like).
func normalize(t *testing.T, b []byte) map[string][]map[string]any {
	t.Helper()
	var out map[string][]map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	return out
}

// diffSnapshots lists row-level differences between two encoded snapshots.
func diffSnapshots(t *testing.T, want, got []byte) []string {
	t.Helper()
	w, g := normalize(t, want), normalize(t, got)
	var diffs []string
	for _, q := range v1SnapshotQueries {
		wr, gr := w[q.table], g[q.table]
		if len(wr) != len(gr) {
			diffs = append(diffs, fmt.Sprintf("%s: %d rows, want %d", q.table, len(gr), len(wr)))
		}
		for i := 0; i < len(wr) && i < len(gr); i++ {
			if !reflect.DeepEqual(wr[i], gr[i]) {
				diffs = append(diffs, fmt.Sprintf("%s[%d]:\n  want %v\n  got  %v", q.table, i, wr[i], gr[i]))
			}
		}
	}
	return diffs
}

// assertSnapshot compares the seeded rows with the committed fixture (or rewrites the
// fixture under -update).
func assertSnapshot(t *testing.T, pool *pgxpool.Pool, file string) {
	t.Helper()
	got := encodeSnapshot(t, snapshotRows(t, pool))
	if *update {
		if err := os.WriteFile(file, got, 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
		t.Logf("wrote %s", file)
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s (run with -update to create it): %v", file, err)
	}
	if bytes.Equal(want, got) {
		return
	}
	diffs := diffSnapshots(t, want, got)
	if len(diffs) == 0 {
		t.Fatalf("%s: same rows, different encoding; re-run with -update after review", file)
	}
	if len(diffs) > 20 {
		diffs = append(diffs[:20], fmt.Sprintf("… and %d more", len(diffs)-20))
	}
	for _, d := range diffs {
		t.Error(d)
	}
	t.Fatalf("%s: the seeded rows differ from the fixture (%d differences)", file, len(diffs))
}
