package store_test

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

// Integration test against a real Postgres, gated on XLEARN_TEST_DATABASE_URL so CI
// (which has no database) skips it. Run locally, e.g.:
//
//	XLEARN_TEST_DATABASE_URL=postgres://xlearn_curriculum:pw@localhost:5433/xlearndb?sslmode=disable \
//	  go test ./internal/curriculum/store/ -run TestStore -count=1
//
// The DSN must be for the xlearn_curriculum role (search_path curriculum, owning
// only schema curriculum) so the test also exercises the least-privilege model.
func TestStoreIntegration(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the curriculum store integration test")
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

	assertIndexesValid(ctx, t, pool)

	if err := st.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	// Start from a clean schema so the exact-count assertions are deterministic
	// regardless of any prior seed in this test database.
	if _, err := pool.Exec(ctx,
		`TRUNCATE curriculum.path, curriculum.phase, curriculum.week, curriculum.concept,
		 curriculum.week_concept, curriculum.problem, curriculum.problem_section RESTART IDENTITY CASCADE`,
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	content := sampleContent()

	// Seed twice: the second run must not change any row counts (idempotency).
	if _, err := st.SeedAll(ctx, content); err != nil {
		t.Fatalf("SeedAll (first): %v", err)
	}
	before := counts(ctx, t, pool)
	if _, err := st.SeedAll(ctx, content); err != nil {
		t.Fatalf("SeedAll (rerun): %v", err)
	}
	after := counts(ctx, t, pool)
	for tbl, n := range before {
		if after[tbl] != n {
			t.Fatalf("table %s row count changed on reseed: %d -> %d (not idempotent)", tbl, n, after[tbl])
		}
	}

	// Reads.
	paths, err := st.ListPaths(ctx)
	if err != nil {
		t.Fatalf("ListPaths: %v", err)
	}
	if len(paths) == 0 || paths[0].Slug != "dsa" {
		t.Fatalf("ListPaths: expected dsa first, got %+v", paths)
	}

	if _, err := st.GetPath(ctx, "missing"); err != store.ErrNotFound {
		t.Fatalf("GetPath(missing): want ErrNotFound, got %v", err)
	}

	weeks, err := st.ListWeeks(ctx, "dsa")
	if err != nil {
		t.Fatalf("ListWeeks: %v", err)
	}
	if len(weeks) != 2 {
		t.Fatalf("ListWeeks: want 2, got %d", len(weeks))
	}
	// Week 1 has one easy + one med seeded; the difficulty mix must reflect that.
	var w1 store.WeekSummary
	for _, w := range weeks {
		if w.N == 1 {
			w1 = w
		}
	}
	if w1.Easy != 1 || w1.Med != 1 || w1.Total != 2 {
		t.Fatalf("week 1 mix wrong: %+v", w1)
	}

	prob, err := st.GetProblem(ctx, "16")
	if err != nil {
		t.Fatalf("GetProblem(16): %v", err)
	}
	if prob.Title != "3Sum" || prob.Difficulty != "med" {
		t.Fatalf("unexpected problem: %+v", prob)
	}
	// 00004 (m3-01): contract_hash and grading_summary round-trip; a seed row without a
	// summary stores {}.
	if prob.ContractHash != sampleContract || !jsonEqual(t, prob.GradingSummary, sampleSummary) {
		t.Fatalf("GetProblem(16) contract fields: %q %s", prob.ContractHash, prob.GradingSummary)
	}
	bulk, err := st.GetProblemsByIDs(ctx, []string{"1", "16"})
	if err != nil || len(bulk) != 2 {
		t.Fatalf("GetProblemsByIDs: %v %+v", err, bulk)
	}
	for _, p := range bulk {
		switch p.ID {
		case "1":
			if p.ContractHash != "" || string(p.GradingSummary) != "{}" {
				t.Fatalf("problem 1 contract fields: %q %s", p.ContractHash, p.GradingSummary)
			}
		case "16":
			if p.ContractHash != sampleContract || !jsonEqual(t, p.GradingSummary, sampleSummary) {
				t.Fatalf("problem 16 bulk contract fields: %q %s", p.ContractHash, p.GradingSummary)
			}
		}
	}

	sections, err := st.ListSections(ctx, "16")
	if err != nil {
		t.Fatalf("ListSections: %v", err)
	}
	if len(sections) != 2 || sections[0].Stage != "attempt" {
		t.Fatalf("unexpected sections: %+v", sections)
	}
	// m1-03: sections carry their language ("" for prose).
	if sections[0].Language != "" || sections[1].Language != "go" {
		t.Fatalf("section languages %q, %q; want \"\" and go", sections[0].Language, sections[1].Language)
	}

	// m1-03: problem reads carry role and links (never nil), in every read.
	index, err := st.ListProblemsByPath(ctx, "dsa")
	if err != nil {
		t.Fatalf("ListProblemsByPath: %v", err)
	}
	week1, err := st.ListProblemsByWeek(ctx, "dsa", 1)
	if err != nil {
		t.Fatalf("ListProblemsByWeek: %v", err)
	}
	wantLinks := map[string][]store.Link{
		"1":  {{Kind: "leetcode", URL: "https://leetcode.com/problems/contains-duplicate/"}},
		"9":  {},
		"16": {},
	}
	wantRole := map[string]string{"1": "core", "9": "reinforcement", "16": "core"}
	for _, ps := range [][]store.Problem{index, week1, bulk} {
		for _, p := range ps {
			if p.Role != wantRole[p.ID] || p.Links == nil || !reflect.DeepEqual(p.Links, wantLinks[p.ID]) {
				t.Errorf("problem %s: role %q links %#v; want %q %#v", p.ID, p.Role, p.Links, wantRole[p.ID], wantLinks[p.ID])
			}
		}
	}
	if len(index) != 3 || len(week1) != 2 {
		t.Fatalf("index %d, week 1 %d problems; want 3, 2", len(index), len(week1))
	}

	concepts, err := st.ListConceptsByWeek(ctx, "dsa", 2)
	if err != nil {
		t.Fatalf("ListConceptsByWeek: %v", err)
	}
	if len(concepts) != 1 || concepts[0].Slug != "two-pointers" {
		t.Fatalf("unexpected week concepts: %+v", concepts)
	}

	c, err := st.GetConcept(ctx, "dsa", "two-pointers")
	if err != nil {
		t.Fatalf("GetConcept: %v", err)
	}
	if !reflect.DeepEqual(c.Templates, map[string]string{"go": "c"}) || c.PathSlug != "dsa" {
		t.Fatalf("concept templates: %+v", c)
	}
	// m1-03: a concept resolves only under its own course, (path_slug, slug).
	zc, err := st.GetConcept(ctx, "zz-fixture", "zz-loops")
	if err != nil || zc.PathSlug != "zz-fixture" || zc.Templates == nil || len(zc.Templates) != 0 {
		t.Fatalf("GetConcept(zz-fixture, zz-loops) = %+v, %v; want it with no templates", zc, err)
	}
	for _, k := range [][2]string{{"dsa", "zz-loops"}, {"zz-fixture", "two-pointers"}, {"dsa", "missing"}} {
		if _, err := st.GetConcept(ctx, k[0], k[1]); err != store.ErrNotFound {
			t.Fatalf("GetConcept(%s, %s): want ErrNotFound, got %v", k[0], k[1], err)
		}
	}

	// m1-03: the seed writes none of the columns M1c drops, so each holds NULL or its
	// 00001 default; problem 1's link, problem 9's role and the concept's template would
	// show if it did.
	var written int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM curriculum.problem
			WHERE COALESCE(is_reinforcement, false) OR COALESCE(leetcode_url, '') <> '' OR COALESCE(neetcode_url, '') <> '')
		+ (SELECT count(*) FROM curriculum.concept WHERE COALESCE(code_template, '') <> '')`).Scan(&written); err != nil {
		t.Fatal(err)
	}
	if written != 0 {
		t.Fatalf("the seed wrote %d M1c-drop values", written)
	}

	n, err := st.CountProblems(ctx, "dsa")
	if err != nil {
		t.Fatalf("CountProblems: %v", err)
	}
	if n != 3 {
		t.Fatalf("CountProblems(dsa) = %d, want 3", n)
	}
}

// Synthetic contract fields for problem 16 (not a real canon hash).
const (
	sampleContract = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	sampleSummary  = `{"mode": "auto", "parts": [{"id": "solution", "type": "code", "grading": "auto", "cadence": "iterate"}], "grader_kinds": ["code"], "languages": ["go"]}`
)

func jsonEqual(t *testing.T, a json.RawMessage, b string) bool {
	t.Helper()
	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		t.Fatalf("decode %s: %v", a, err)
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(va, vb)
}

func sampleContent() store.SeedContent {
	return store.SeedContent{
		Courses: []string{"dsa", "zz-fixture"},
		Paths: []store.SeedPath{
			{Slug: "dsa", Title: "DSA", Status: "active", Summary: "s", ProblemTotal: 151, WeekTotal: 2, SortOrder: 0, IDPrefix: "dsa"},
			{Slug: "zz-fixture", Title: "Fixture", Status: "active", Summary: "z", SortOrder: 90, IDPrefix: "zz"},
		},
		Phases: []store.SeedPhase{
			{PathSlug: "dsa", Order: 1, Name: "Fundamentals", Theme: "arrays", WeekFrom: 1, WeekTo: 2},
		},
		Weeks: []store.SeedWeek{
			{PathSlug: "dsa", N: 1, Title: "Arrays", Thesis: "t1"},
			{PathSlug: "dsa", N: 2, Title: "Two Pointers", Thesis: "t2"},
		},
		Concepts: []store.SeedConcept{
			{PathSlug: "dsa", Slug: "two-pointers", Title: "Two Pointers", BodyMD: "b", WhenToUseMD: "w", Templates: map[string]string{"go": "c"}, Weeks: []int{2}},
			// A second course's concept (m1-03's (path_slug, slug) lookup).
			{PathSlug: "zz-fixture", Slug: "zz-loops", Title: "Loops", BodyMD: "zb", WhenToUseMD: "zw"},
		},
		Problems: []store.SeedProblem{
			{ID: "1", PathSlug: "dsa", WeekN: 1, Title: "Contains Duplicate", Difficulty: "easy", Pattern: "Hashing", SortOrder: 1,
				Role: "core", Status: "live",
				Links:    []store.SeedLink{{Kind: "leetcode", URL: "https://leetcode.com/problems/contains-duplicate/"}},
				Sections: []store.SeedSection{{Stage: "attempt", Kind: "summary", Order: 1, BodyMD: "x"}}},
			{ID: "9", PathSlug: "dsa", WeekN: 1, Title: "Subarray Sum", Difficulty: "med", Pattern: "Prefix Sum", SortOrder: 2,
				Role: "reinforcement", Status: "live"},
			{ID: "16", PathSlug: "dsa", WeekN: 2, Title: "3Sum", Difficulty: "med", Pattern: "Two Pointers", SortOrder: 1,
				Role: "core", Status: "live", ContractHash: sampleContract, GradingSummary: json.RawMessage(sampleSummary),
				Sections: []store.SeedSection{
					{Stage: "attempt", Kind: "summary", Order: 1, BodyMD: "a"},
					{Stage: "solution", Kind: "code", Order: 1, Language: "go", Code: "func threeSum() {}"},
				}},
		},
	}
}

func counts(ctx context.Context, t *testing.T, pool *pgxpool.Pool) map[string]int {
	t.Helper()
	tables := []string{"path", "phase", "week", "concept", "week_concept", "problem", "problem_section"}
	out := make(map[string]int, len(tables))
	for _, tbl := range tables {
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM curriculum."+tbl).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", tbl, err)
		}
		out[tbl] = n
	}
	return out
}

// assertIndexesValid fails on any INVALID index in schema curriculum and checks that the
// M1a expand uniques and the v1 uniques exist: 00003 builds the new ones CONCURRENTLY,
// and a failed CONCURRENTLY build leaves an INVALID index that a re-run's IF NOT EXISTS
// would silently skip.
func assertIndexesValid(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	rows, err := pool.Query(ctx, `
		SELECT c.relname, i.indisvalid, i.indisunique
		FROM pg_index i
		JOIN pg_class c     ON c.oid = i.indexrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'curriculum'`)
	if err != nil {
		t.Fatalf("list indexes: %v", err)
	}
	defer rows.Close()
	unique := map[string]bool{}
	for rows.Next() {
		var name string
		var valid, isUnique bool
		if err := rows.Scan(&name, &valid, &isUnique); err != nil {
			t.Fatalf("scan index: %v", err)
		}
		if !valid {
			t.Errorf("index curriculum.%s is INVALID (a failed CREATE INDEX CONCURRENTLY): drop it and re-migrate", name)
		}
		unique[name] = isUnique
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("list indexes: %v", err)
	}
	for _, name := range []string{
		"path_id_prefix_key", "concept_path_slug_slug_key", "problem_section_problem_id_stage_order_language_key",
		// The v1 uniques stay until M1c (v1.5.2's upserts target them).
		"concept_slug_key", "problem_section_problem_id_stage_order_key",
	} {
		if !unique[name] {
			t.Errorf("unique index curriculum.%s is missing", name)
		}
	}
}
