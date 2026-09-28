package curriculum

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgxpool"

	seeddata "github.com/sujaykumarsuman/xlearn/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

// Re-seed cases (sprint m1-09 task 6) against a real Postgres: idempotency, one-item
// edits, retire/withdraw, delete-missing, the re-parenting and slug guards, a v1.5.2
// writer in between (the R-b case), and the upgrade from a v1-seeded schema.

// fullSnapshotQueries capture every column (v1 and v2) except uuids; retired_at is
// reduced to "is set" so the snapshot has no clock in it.
var fullSnapshotQueries = []struct {
	table string
	sql   string
}{
	{"path", `SELECT slug, title, status, summary, problem_total, week_total, sort_order, id_prefix
		FROM curriculum.path ORDER BY slug COLLATE "C"`},
	{"phase", `SELECT path_slug, "order", name, theme, week_from, week_to
		FROM curriculum.phase ORDER BY path_slug COLLATE "C", "order"`},
	{"week", `SELECT path_slug, n, title, thesis FROM curriculum.week ORDER BY path_slug COLLATE "C", n`},
	{"concept", `SELECT path_slug, slug, title, body_md, when_to_use_md, code_template, templates
		FROM curriculum.concept ORDER BY path_slug COLLATE "C", slug COLLATE "C"`},
	{"week_concept", `SELECT w.path_slug, w.n AS week_n, c.slug AS concept_slug
		FROM curriculum.week_concept wc
		JOIN curriculum.week w    ON w.id = wc.week_id
		JOIN curriculum.concept c ON c.id = wc.concept_id
		ORDER BY w.path_slug COLLATE "C", w.n, c.slug COLLATE "C"`},
	{"problem", `SELECT id, path_slug, week_n, title, difficulty, pattern, leetcode_url, neetcode_url,
			is_reinforcement, sort_order, role, status, retired_at IS NOT NULL AS retired, links, content_hash,
			contract_hash, grading_summary
		FROM curriculum.problem ORDER BY path_slug COLLATE "C", id COLLATE "C"`},
	{"problem_section", `SELECT problem_id, stage, "order", language, kind, body_md, code
		FROM curriculum.problem_section
		ORDER BY problem_id COLLATE "C", stage COLLATE "C", "order", language COLLATE "C"`},
}

type fullSnapshot map[string][]map[string]any

func fullRows(t *testing.T, pool *pgxpool.Pool) fullSnapshot {
	t.Helper()
	out := fullSnapshot{}
	for _, q := range fullSnapshotQueries {
		// Round-trip through JSON so jsonb and integer types compare by value.
		var rows []map[string]any
		if err := json.Unmarshal(encodeSnapshot(t, queryRows(t, pool, q.sql)), &rows); err != nil {
			t.Fatal(err)
		}
		out[q.table] = rows
	}
	return out
}

// diffFull returns "<table>: <row>" for every row present on one side only.
func diffFull(a, b fullSnapshot) []string {
	var out []string
	for _, q := range fullSnapshotQueries {
		inA := map[string]int{}
		for _, r := range a[q.table] {
			inA[rowKey(r)]++
		}
		for _, r := range b[q.table] {
			k := rowKey(r)
			if inA[k] > 0 {
				inA[k]--
				continue
			}
			out = append(out, q.table+" +"+k)
		}
		for k, n := range inA {
			for ; n > 0; n-- {
				out = append(out, q.table+" -"+k)
			}
		}
	}
	return out
}

func rowKey(r map[string]any) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// seedWith runs the production Seed over fsys and returns its log output.
func seedWith(t *testing.T, pool *pgxpool.Pool, fsys fs.FS) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	err := Seed(context.Background(), store.New(pool), fsys, log)
	return buf.String(), err
}

func mustSeed(t *testing.T, pool *pgxpool.Pool, fsys fs.FS) string {
	t.Helper()
	out, err := seedWith(t, pool, fsys)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return out
}

// freshSeeded is a fresh schema seeded with the embedded curriculum.
func freshSeeded(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, dsn := testPool(t)
	freshSchema(t, pool, dsn)
	mustSeed(t, pool, seeddata.FS)
	return pool
}

// TestSeedV2Columns is the snapshot test's second table: the new columns a fresh seed
// writes (role, status, links, content_hash, language, templates, id_prefix), and the
// dual-written v1 columns agreeing with them.
func TestSeedV2Columns(t *testing.T) {
	pool := freshSeeded(t)
	ctx := context.Background()

	for _, c := range []struct{ what, sql string }{
		{"role mirrors is_reinforcement", `SELECT count(*) FROM curriculum.problem
			WHERE role <> CASE WHEN is_reinforcement THEN 'reinforcement' ELSE 'core' END`},
		{"every item is live with no retired_at", `SELECT count(*) FROM curriculum.problem
			WHERE status <> 'live' OR retired_at IS NOT NULL`},
		{"links mirror the two URL columns", `SELECT count(*) FROM curriculum.problem WHERE links <>
			(CASE WHEN leetcode_url <> '' THEN jsonb_build_array(jsonb_build_object('kind','leetcode','url',leetcode_url)) ELSE '[]'::jsonb END)
			|| (CASE WHEN neetcode_url <> '' THEN jsonb_build_array(jsonb_build_object('kind','neetcode','url',neetcode_url)) ELSE '[]'::jsonb END)`},
		{"content_hash is set", `SELECT count(*) FROM curriculum.problem WHERE content_hash NOT LIKE 'sha256:%'`},
		// 00004 (m3-01): every v1 item is on the self path.
		{"self-path items have no contract_hash", `SELECT count(*) FROM curriculum.problem WHERE contract_hash <> ''`},
		{"self-path items summarize as self", `SELECT count(*) FROM curriculum.problem WHERE grading_summary <> '{"mode": "self"}'::jsonb`},
		{"code sections are go, prose none", `SELECT count(*) FROM curriculum.problem_section
			WHERE language <> CASE WHEN kind = 'code' THEN 'go' ELSE '' END`},
		{"templates mirror code_template", `SELECT count(*) FROM curriculum.concept
			WHERE templates <> CASE WHEN code_template <> '' THEN jsonb_build_object('go', code_template) ELSE '{}'::jsonb END`},
		{"no NULL in a dual-written column", `SELECT
			(SELECT count(*) FROM curriculum.problem WHERE is_reinforcement IS NULL OR leetcode_url IS NULL OR neetcode_url IS NULL)
			+ (SELECT count(*) FROM curriculum.concept WHERE code_template IS NULL)`},
	} {
		var n int
		if err := pool.QueryRow(ctx, c.sql).Scan(&n); err != nil {
			t.Fatalf("%s: %v", c.what, err)
		}
		if n != 0 {
			t.Errorf("%s: %d violating rows", c.what, n)
		}
	}

	// content_hash equals canon.ContentHash of each resolved item; id_prefix is the
	// manifest's.
	content, err := LoadContent(seeddata.FS)
	if err != nil {
		t.Fatal(err)
	}
	for _, cc := range content.Courses {
		var prefix *string
		if err := pool.QueryRow(ctx, `SELECT id_prefix FROM curriculum.path WHERE slug = $1`, cc.Slug).Scan(&prefix); err != nil {
			t.Fatalf("path %s: %v", cc.Slug, err)
		}
		if prefix == nil || *prefix != content.Manifests[cc.Slug].IDPrefix {
			t.Errorf("path %s: id_prefix %v, want %q", cc.Slug, prefix, content.Manifests[cc.Slug].IDPrefix)
		}
		for i := range cc.Items {
			assertStoredHashes(t, pool, &cc.Items[i])
		}
	}
}

// assertStoredHashes checks one item's content_hash, contract_hash and grading_summary
// against canon and SummarizeGrading over the loaded item.
func assertStoredHashes(t *testing.T, pool *pgxpool.Pool, ri *course.ResolvedItem) {
	t.Helper()
	wantContent, err := canon.ContentHash(ri)
	if err != nil {
		t.Fatal(err)
	}
	wantContract, err := canon.ContractHash(&ri.Item)
	if err != nil {
		t.Fatal(err)
	}
	wantSummary, err := json.Marshal(SummarizeGrading(&ri.Item))
	if err != nil {
		t.Fatal(err)
	}
	var content, contract, summary string
	if err := pool.QueryRow(context.Background(),
		`SELECT content_hash, contract_hash, grading_summary::text FROM curriculum.problem WHERE id = $1`, ri.Item.ID).
		Scan(&content, &contract, &summary); err != nil {
		t.Fatal(err)
	}
	if content != wantContent {
		t.Errorf("item %s: content_hash %s, want %s", ri.Item.ID, content, wantContent)
	}
	if contract != wantContract {
		t.Errorf("item %s: contract_hash %q, want %q", ri.Item.ID, contract, wantContract)
	}
	var got, want any
	_ = json.Unmarshal([]byte(summary), &got)
	_ = json.Unmarshal(wantSummary, &want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("item %s: grading_summary %s, want %s", ri.Item.ID, summary, wantSummary)
	}
}

// m3-01 task 7: an item that gains a graded part is seeded with canon's contract_hash
// and an auto summary; a content-only edit then moves content_hash only; re-seeding
// changes nothing; a contract edit moves both.
func TestReseedContractHash(t *testing.T) {
	pool := freshSeeded(t)
	ctx := context.Background()
	code := codeFixture(t)

	m := contentFS(t)
	editJSON(t, m, "courses/dsa/items/1/item.json", func(it *course.Item) {
		it.Parts, it.Grader, it.SolutionFacts = code.Parts, code.Grader, code.SolutionFacts
	})
	mustSeed(t, pool, m)
	c1, err := LoadContent(m)
	if err != nil {
		t.Fatal(err)
	}
	item1 := func(c *Content) *course.ResolvedItem {
		for _, cc := range c.Courses {
			for i := range cc.Items {
				if cc.Items[i].Item.ID == "1" {
					return &cc.Items[i]
				}
			}
		}
		t.Fatal("item 1 missing")
		return nil
	}
	assertStoredHashes(t, pool, item1(c1))
	read := func() (content, contract, mode string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT content_hash, contract_hash, grading_summary->>'mode' FROM curriculum.problem WHERE id = '1'`).
			Scan(&content, &contract, &mode); err != nil {
			t.Fatal(err)
		}
		return
	}
	content1, contract1, mode := read()
	if !strings.HasPrefix(contract1, "sha256:") || mode != ModeAuto {
		t.Fatalf("item 1 with a graded code part: contract %q, mode %q", contract1, mode)
	}

	// Re-seed: idempotent.
	before := fullRows(t, pool)
	mustSeed(t, pool, m)
	if d := diffFull(before, fullRows(t, pool)); len(d) > 0 {
		t.Fatalf("re-seeding changed rows:\n%s", strings.Join(d, "\n"))
	}

	// A content-only edit (the time limit) moves content_hash only.
	editJSON(t, m, "courses/dsa/items/1/item.json", func(it *course.Item) { it.Parts[0].Config.Limits.TimeMS = 3000 })
	mustSeed(t, pool, m)
	content2, contract2, _ := read()
	if content2 == content1 || contract2 != contract1 {
		t.Fatalf("limits edit: content %v moved, contract %v moved (want true, false)", content2 != content1, contract2 != contract1)
	}

	// A contract edit (the harness version) moves both.
	editJSON(t, m, "courses/dsa/items/1/item.json", func(it *course.Item) { it.Parts[0].Config.Harness = "func-json@2" })
	mustSeed(t, pool, m)
	content3, contract3, _ := read()
	if content3 == content2 || contract3 == contract2 {
		t.Fatal("a harness bump must move both hashes")
	}
	c3, err := LoadContent(m)
	if err != nil {
		t.Fatal(err)
	}
	assertStoredHashes(t, pool, item1(c3))
}

// codeFixture is m1-01's valid-code.json fixture item.
func codeFixture(t *testing.T) *course.Item {
	t.Helper()
	b, err := os.ReadFile("../course/testdata/items/valid-code.json")
	if err != nil {
		t.Fatal(err)
	}
	it, err := course.DecodeItem(b)
	if err != nil {
		t.Fatal(err)
	}
	return it
}

func TestReseedIsIdempotent(t *testing.T) {
	pool := freshSeeded(t)
	before := fullRows(t, pool)
	mustSeed(t, pool, seeddata.FS)
	if d := diffFull(before, fullRows(t, pool)); len(d) > 0 {
		t.Fatalf("re-seeding changed rows:\n%s", strings.Join(d, "\n"))
	}
	assertSnapshot(t, pool, snapshotFile)
}

func TestReseedOneItemChange(t *testing.T) {
	pool := freshSeeded(t)
	before := fullRows(t, pool)

	m := contentFS(t)
	editJSON(t, m, "courses/dsa/items/2/item.json", func(it *course.Item) { it.Title = "Valid Anagram (edited)" })
	m["courses/dsa/items/2/sections/attempt/01-summary.md"] = &fstest.MapFile{Data: []byte("An edited summary.")}
	mustSeed(t, pool, m)

	for _, d := range diffFull(before, fullRows(t, pool)) {
		if !strings.Contains(d, `"id":"2"`) && !strings.Contains(d, `"problem_id":"2"`) {
			t.Errorf("an edit of item 2 changed an unrelated row: %s", d)
		}
	}
	var title, hash, body string
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `SELECT title, content_hash FROM curriculum.problem WHERE id = '2'`).Scan(&title, &hash); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT body_md FROM curriculum.problem_section WHERE problem_id = '2'`).Scan(&body); err != nil {
		t.Fatal(err)
	}
	if title != "Valid Anagram (edited)" || body != "An edited summary." {
		t.Fatalf("item 2 not updated: title %q, body %q", title, body)
	}
	for _, r := range before["problem"] {
		if r["id"] == "2" && r["content_hash"] == hash {
			t.Error("item 2's content_hash did not move with the edit")
		}
	}
}

func TestReseedRetireAndWithdraw(t *testing.T) {
	pool := freshSeeded(t)
	ctx := context.Background()
	st := store.New(pool)

	// Remove item 2 from the seed entirely (its directory and its lock row): the seed
	// retires it defensively and logs a WARN.
	m := contentFS(t)
	removeTree(m, "courses/dsa/items/2")
	editJSON(t, m, "ids.lock.json", func(l *course.IDsLock) { delete(l.Items, "2") })
	// Retire item 5 and withdraw item 7 explicitly.
	for id, status := range map[string]string{"5": "retired", "7": "withdrawn"} {
		editJSON(t, m, "courses/dsa/items/"+id+"/item.json", func(it *course.Item) { it.Status = status })
		editJSON(t, m, "ids.lock.json", func(l *course.IDsLock) { l.Items[id] = course.LockEntry{Course: "dsa", Status: status} })
	}
	logs := mustSeed(t, pool, m)
	if !strings.Contains(logs, "level=WARN") || !strings.Contains(logs, "id=2") {
		t.Errorf("no WARN for the defensively retired item 2 in:\n%s", logs)
	}

	for id, status := range map[string]string{"2": "retired", "5": "retired", "7": "withdrawn"} {
		var got string
		var retiredAt bool
		if err := pool.QueryRow(ctx, `SELECT status, retired_at IS NOT NULL FROM curriculum.problem WHERE id = $1`, id).Scan(&got, &retiredAt); err != nil {
			t.Fatal(err)
		}
		if got != status || !retiredAt {
			t.Errorf("item %s: status %q retired_at set %v, want %q and set", id, got, retiredAt, status)
		}
	}

	// Gone from the index and every count (ListWeeksWithCounts included)...
	gone := map[string]bool{"2": true, "5": true, "7": true}
	byPath, err := st.ListProblemsByPath(ctx, "dsa")
	if err != nil {
		t.Fatal(err)
	}
	byWeek, err := st.ListProblemsByWeek(ctx, "dsa", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range append(byPath, byWeek...) {
		if gone[p.ID] {
			t.Errorf("item %s is still in the index", p.ID)
		}
	}
	if n, err := st.CountProblems(ctx, "dsa"); err != nil || n != 11 {
		t.Errorf("CountProblems(dsa) = %d, %v; want 11", n, err)
	}
	weeks, err := st.ListWeeks(ctx, "dsa")
	if err != nil {
		t.Fatal(err)
	}
	// Week 1 held 1, 2, 3 (easy), 5, 7, 9 (med): 2 easy + 1 med remain live.
	if w := weeks[0]; w.N != 1 || w.Easy != 2 || w.Med != 1 || w.Total != 3 {
		t.Errorf("week 1 counts %+v, want easy 2, med 1, total 3", w)
	}
	// ...but still resolvable by id.
	for id := range gone {
		if _, err := st.GetProblem(ctx, id); err != nil {
			t.Errorf("GetProblem(%s): %v", id, err)
		}
	}
	bulk, err := st.GetProblemsByIDs(ctx, []string{"2", "5", "7"})
	if err != nil || len(bulk) != 3 {
		t.Errorf("GetProblemsByIDs(2,5,7) = %d rows, %v; want 3", len(bulk), err)
	}
	// A retired item keeps its sections; a withdrawn one serves none (title kept).
	if secs, err := st.ListSections(ctx, "5"); err != nil || len(secs) == 0 {
		t.Errorf("retired item 5 sections = %d, %v; want them kept", len(secs), err)
	}
	if secs, err := st.ListSections(ctx, "7"); err != nil || len(secs) != 0 {
		t.Errorf("withdrawn item 7 sections = %d, %v; want none", len(secs), err)
	}

	// Restoring the content brings every item back live.
	mustSeed(t, pool, seeddata.FS)
	var notLive int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM curriculum.problem WHERE status <> 'live' OR retired_at IS NOT NULL`).Scan(&notLive); err != nil {
		t.Fatal(err)
	}
	if notLive != 0 {
		t.Errorf("%d items are not live after restoring the content", notLive)
	}
	assertSnapshot(t, pool, snapshotFile)
}

func TestReseedDeletesMissingPerCourse(t *testing.T) {
	pool := freshSeeded(t)
	ctx := context.Background()

	m := contentFS(t)
	editJSON(t, m, "courses/dsa/concepts.json", func(cs *[]course.Concept) {
		kept := (*cs)[:0]
		for _, c := range *cs {
			if c.Slug != "prefix-sums" {
				kept = append(kept, c)
			}
		}
		*cs = kept
	})
	for _, f := range []string{"prefix-sums.md", "prefix-sums.when.md", "_code/prefix-sums.go.snip"} {
		delete(m, "courses/dsa/concepts/"+f)
	}
	editJSON(t, m, "courses/dsa/weeks.json", func(ws *[]course.Week) { *ws = (*ws)[:len(*ws)-1] })
	editJSON(t, m, "courses/dsa/phases.json", func(ps *[]course.Phase) { *ps = (*ps)[:len(*ps)-1] })
	mustSeed(t, pool, m)

	for what, sql := range map[string]string{
		"concept prefix-sums": `SELECT count(*) FROM curriculum.concept WHERE slug = 'prefix-sums'`,
		"its week link":       `SELECT count(*) FROM curriculum.week_concept wc JOIN curriculum.concept c ON c.id = wc.concept_id WHERE c.slug = 'prefix-sums'`,
		"week 16":             `SELECT count(*) FROM curriculum.week WHERE path_slug = 'dsa' AND n = 16`,
		"phase 4":             `SELECT count(*) FROM curriculum.phase WHERE path_slug = 'dsa' AND "order" = 4`,
	} {
		var n int
		if err := pool.QueryRow(ctx, sql).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s survived the seed that dropped it (%d rows)", what, n)
		}
	}
	mustSeed(t, pool, seeddata.FS)
	assertSnapshot(t, pool, snapshotFile)
}

func TestReseedAborts(t *testing.T) {
	pool := freshSeeded(t)
	before := fullRows(t, pool)
	ctx := context.Background()

	t.Run("item moved to another course (loader)", func(t *testing.T) {
		m := contentFS(t)
		moveTree(m, "courses/dsa/items/2", "courses/sql/items/2")
		if _, err := seedWith(t, pool, m); err == nil {
			t.Fatal("seed accepted an item moved to another course")
		}
	})
	t.Run("item moved to another course (id guard, :execrows)", func(t *testing.T) {
		content, err := LoadContent(seeddata.FS)
		if err != nil {
			t.Fatal(err)
		}
		seed, err := content.SeedContent()
		if err != nil {
			t.Fatal(err)
		}
		for i := range seed.Problems {
			if seed.Problems[i].ID == "2" {
				seed.Problems[i].PathSlug = "sql"
			}
		}
		_, err = store.New(pool).SeedAll(ctx, seed)
		if !errors.Is(err, store.ErrReparent) {
			t.Fatalf("SeedAll = %v, want ErrReparent", err)
		}
	})
	t.Run("reserved course slug", func(t *testing.T) {
		m := contentFS(t)
		addCourse(t, m, "api", "ap")
		if _, err := seedWith(t, pool, m); err == nil || !strings.Contains(err.Error(), "reserved") {
			t.Fatalf("seed = %v, want a reserved-slug error", err)
		}
	})
	t.Run("malformed course slug", func(t *testing.T) {
		m := contentFS(t)
		addCourse(t, m, "Bad_Slug", "bs")
		if _, err := seedWith(t, pool, m); err == nil {
			t.Fatal("seed accepted a malformed course slug")
		}
	})

	// Every abort rolled back: nothing changed.
	if d := diffFull(before, fullRows(t, pool)); len(d) > 0 {
		t.Fatalf("an aborted seed changed rows:\n%s", strings.Join(d, "\n"))
	}
}

// --- the v1.5.2 writer (R-b) and the v1 upgrade path ---------------------------------

// v1.5.2's writer SQL, verbatim from internal/curriculum/store/queries/*.sql at v1.5.2
// (the R-b target while the rollback floor is "none"). It targets the v1 uniques and
// never touches the v2 columns.
const (
	v1UpsertPath = `INSERT INTO curriculum.path (slug, title, status, summary, problem_total, week_total, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (slug) DO UPDATE SET
    title = EXCLUDED.title, status = EXCLUDED.status, summary = EXCLUDED.summary,
    problem_total = EXCLUDED.problem_total, week_total = EXCLUDED.week_total, sort_order = EXCLUDED.sort_order`
	v1UpsertPhase = `INSERT INTO curriculum.phase (path_slug, "order", name, theme, week_from, week_to)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (path_slug, "order") DO UPDATE SET
    name = EXCLUDED.name, theme = EXCLUDED.theme, week_from = EXCLUDED.week_from, week_to = EXCLUDED.week_to`
	v1UpsertWeek = `INSERT INTO curriculum.week (path_slug, n, title, thesis)
VALUES ($1, $2, $3, $4)
ON CONFLICT (path_slug, n) DO UPDATE SET title = EXCLUDED.title, thesis = EXCLUDED.thesis`
	v1UpsertConcept = `INSERT INTO curriculum.concept (path_slug, slug, title, body_md, when_to_use_md, code_template)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (slug) DO UPDATE SET
    path_slug = EXCLUDED.path_slug, title = EXCLUDED.title, body_md = EXCLUDED.body_md,
    when_to_use_md = EXCLUDED.when_to_use_md, code_template = EXCLUDED.code_template`
	v1LinkWeekConcept = `INSERT INTO curriculum.week_concept (week_id, concept_id)
SELECT w.id, c.id FROM curriculum.week w, curriculum.concept c
WHERE w.path_slug = $1 AND w.n = $2 AND c.slug = $3
ON CONFLICT (week_id, concept_id) DO NOTHING`
	v1UpsertProblem = `INSERT INTO curriculum.problem (
    id, path_slug, week_n, title, difficulty, pattern,
    leetcode_url, neetcode_url, is_reinforcement, sort_order
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE SET
    path_slug        = EXCLUDED.path_slug,
    week_n           = EXCLUDED.week_n,
    title            = EXCLUDED.title,
    difficulty       = EXCLUDED.difficulty,
    pattern          = EXCLUDED.pattern,
    leetcode_url     = EXCLUDED.leetcode_url,
    neetcode_url     = EXCLUDED.neetcode_url,
    is_reinforcement = EXCLUDED.is_reinforcement,
    sort_order       = EXCLUDED.sort_order`
	v1UpsertSection = `INSERT INTO curriculum.problem_section (problem_id, stage, kind, "order", body_md, code)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (problem_id, stage, "order") DO UPDATE SET
    kind    = EXCLUDED.kind,
    body_md = EXCLUDED.body_md,
    code    = EXCLUDED.code`
)

// v1Rows are the rows a v1.5.2 seed writes: the committed v1 snapshot, with items 3 and
// 16's Example-1 bodies as v1.5.2 has them (read from the v1.5.2 tag with git when the
// clone has it, otherwise a stand-in body: either way a body the new seed must replace).
func v1Rows(t *testing.T) map[string][]map[string]any {
	t.Helper()
	b, err := os.ReadFile(snapshotFile)
	if err != nil {
		t.Fatal(err)
	}
	rows := normalize(t, b)
	old := v152Examples(t)
	for _, r := range rows["problem_section"] {
		if r["kind"] == "example" {
			if body, ok := old[r["problem_id"].(string)]; ok {
				r["body_md"] = body
			}
		}
	}
	return rows
}

// v152Examples returns items 3 and 16's Example-1 bodies at v1.5.2.
func v152Examples(t *testing.T) map[string]string {
	t.Helper()
	stand := map[string]string{"3": "v1.5.2 example body for item 3", "16": "v1.5.2 example body for item 16"}
	out, err := exec.Command("git", "show", "v1.5.2:curriculum/dsa/problems.json").Output()
	if err != nil {
		t.Logf("v1.5.2 not in this clone (%v): using stand-in Example-1 bodies", err)
		return stand
	}
	var problems []struct {
		ID       string `json:"id"`
		Sections []struct {
			Kind   string `json:"kind"`
			BodyMD string `json:"body_md"`
		} `json:"sections"`
	}
	if err := json.Unmarshal(out, &problems); err != nil {
		t.Fatalf("decode v1.5.2 problems.json: %v", err)
	}
	got := map[string]string{}
	for _, p := range problems {
		for _, s := range p.Sections {
			if s.Kind == "example" && (p.ID == "3" || p.ID == "16") {
				got[p.ID] = s.BodyMD
			}
		}
	}
	if len(got) != 2 {
		t.Fatalf("v1.5.2 problems.json: found examples for %v, want items 3 and 16", got)
	}
	return got
}

func num(v any) int { return int(v.(float64)) }

// writeV1 applies rows with v1.5.2's writer SQL. With problemsOnly it writes just the
// problems and sections (what a v1.5.2 pod's seed rewrites over an existing schema).
func writeV1(t *testing.T, pool *pgxpool.Pool, rows map[string][]map[string]any, problemsOnly bool) {
	t.Helper()
	ctx := context.Background()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("v1 writer: %v\n%s", err, sql)
		}
	}
	if !problemsOnly {
		for _, r := range rows["path"] {
			exec(v1UpsertPath, r["slug"], r["title"], r["status"], r["summary"], num(r["problem_total"]), num(r["week_total"]), num(r["sort_order"]))
		}
		for _, r := range rows["phase"] {
			exec(v1UpsertPhase, r["path_slug"], num(r["order"]), r["name"], r["theme"], num(r["week_from"]), num(r["week_to"]))
		}
		for _, r := range rows["week"] {
			exec(v1UpsertWeek, r["path_slug"], num(r["n"]), r["title"], r["thesis"])
		}
		for _, r := range rows["concept"] {
			exec(v1UpsertConcept, r["path_slug"], r["slug"], r["title"], r["body_md"], r["when_to_use_md"], r["code_template"])
		}
		for _, r := range rows["week_concept"] {
			exec(v1LinkWeekConcept, r["path_slug"], num(r["week_n"]), r["concept_slug"])
		}
	}
	for _, r := range rows["problem"] {
		exec(v1UpsertProblem, r["id"], r["path_slug"], num(r["week_n"]), r["title"], r["difficulty"], r["pattern"],
			r["leetcode_url"], r["neetcode_url"], r["is_reinforcement"], num(r["sort_order"]))
	}
	for _, r := range rows["problem_section"] {
		exec(v1UpsertSection, r["problem_id"], r["stage"], r["kind"], num(r["order"]), r["body_md"], r["code"])
	}
}

// A v1.5.2 pod (R-b) rewrites section bodies (restoring the copied Example-1s) without
// touching content_hash; rolling forward must restore the new bodies, which is why the
// seed never skips on content_hash.
func TestReseedAfterV152Writer(t *testing.T) {
	pool := freshSeeded(t)
	ctx := context.Background()
	hashes := map[string]string{}
	for _, r := range fullRows(t, pool)["problem"] {
		hashes[r["id"].(string)] = r["content_hash"].(string)
	}

	rows := v1Rows(t)
	writeV1(t, pool, rows, true)
	old := v152Examples(t)
	for id, body := range old {
		var got, hash string
		if err := pool.QueryRow(ctx, `SELECT s.body_md, p.content_hash FROM curriculum.problem_section s
			JOIN curriculum.problem p ON p.id = s.problem_id WHERE s.problem_id = $1 AND s.kind = 'example'`, id).Scan(&got, &hash); err != nil {
			t.Fatal(err)
		}
		if got != body || hash != hashes[id] {
			t.Fatalf("item %s after the v1.5.2 writer: body %q hash %s; want v1.5.2's body and the unchanged hash", id, got, hash)
		}
	}

	mustSeed(t, pool, seeddata.FS)
	assertSnapshot(t, pool, snapshotFile) // the new Example-1s are back
}

// A database seeded by v1 (schema 00001, v1.5.2's rows), then migrated (00002 backfills,
// 00003 indexes) and seeded by this release, ends with exactly the rows of a fresh install.
func TestUpgradeFromV1(t *testing.T) {
	pool, dsn := testPool(t)
	freshSchema(t, pool, dsn)
	mustSeed(t, pool, seeddata.FS)
	want := fullRows(t, pool)

	// Rebuild as v1 left it: schema at 00001, rows written by v1.5.2's SQL.
	if _, err := pool.Exec(context.Background(), `
DO $$
DECLARE r record;
BEGIN
    FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'curriculum' LOOP
        EXECUTE format('DROP TABLE IF EXISTS curriculum.%I CASCADE', r.tablename);
    END LOOP;
END $$`); err != nil {
		t.Fatal(err)
	}
	if err := store.MigrateTo(context.Background(), dsn, 1, discardLogger()); err != nil {
		t.Fatalf("migrate to 00001: %v", err)
	}
	writeV1(t, pool, v1Rows(t), false)

	// Upgrade: 00002 + 00003, then check the backfills before any v2 seed runs.
	if err := store.Migrate(context.Background(), dsn, discardLogger()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	backfilled := fullRows(t, pool)
	for _, r := range backfilled["problem"] {
		if r["role"] != "core" || r["status"] != "live" || r["retired"] != false || r["content_hash"] != "" ||
			r["contract_hash"] != "" || fmt.Sprint(r["grading_summary"]) != "map[]" {
			t.Errorf("backfilled problem %v", r)
		}
	}
	wantLinks := map[string]any{}
	for _, r := range want["problem"] {
		wantLinks[r["id"].(string)] = r["links"]
	}
	for _, r := range backfilled["problem"] {
		if !reflect.DeepEqual(r["links"], wantLinks[r["id"].(string)]) {
			t.Errorf("problem %s: backfilled links %v, want %v", r["id"], r["links"], wantLinks[r["id"].(string)])
		}
	}
	for _, r := range backfilled["problem_section"] {
		if lang := map[bool]string{true: "go", false: ""}[r["kind"] == "code"]; r["language"] != lang {
			t.Errorf("section %v: backfilled language %q, want %q", r, r["language"], lang)
		}
	}
	for _, r := range backfilled["concept"] {
		if tm, _ := r["templates"].(map[string]any); fmt.Sprint(tm["go"]) != fmt.Sprint(r["code_template"]) {
			t.Errorf("concept %s: backfilled templates %v", r["slug"], r["templates"])
		}
	}

	mustSeed(t, pool, seeddata.FS)
	if d := diffFull(want, fullRows(t, pool)); len(d) > 0 {
		t.Fatalf("upgraded database differs from a fresh install:\n%s", strings.Join(d, "\n"))
	}
	assertSnapshot(t, pool, snapshotFile)
}
