package store_test

import (
	"context"
	"os"
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
	if err := st.SeedAll(ctx, content); err != nil {
		t.Fatalf("SeedAll (first): %v", err)
	}
	before := counts(ctx, t, pool)
	if err := st.SeedAll(ctx, content); err != nil {
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

	sections, err := st.ListSections(ctx, "16")
	if err != nil {
		t.Fatalf("ListSections: %v", err)
	}
	if len(sections) != 2 || sections[0].Stage != "attempt" {
		t.Fatalf("unexpected sections: %+v", sections)
	}

	concepts, err := st.ListConceptsByWeek(ctx, "dsa", 2)
	if err != nil {
		t.Fatalf("ListConceptsByWeek: %v", err)
	}
	if len(concepts) != 1 || concepts[0].Slug != "two-pointers" {
		t.Fatalf("unexpected week concepts: %+v", concepts)
	}

	c, err := st.GetConcept(ctx, "two-pointers")
	if err != nil {
		t.Fatalf("GetConcept: %v", err)
	}
	if c.CodeTemplate == "" {
		t.Fatalf("concept missing code template: %+v", c)
	}

	n, err := st.CountProblems(ctx, "dsa")
	if err != nil {
		t.Fatalf("CountProblems: %v", err)
	}
	if n != 3 {
		t.Fatalf("CountProblems(dsa) = %d, want 3", n)
	}
}

func sampleContent() store.SeedContent {
	return store.SeedContent{
		Paths: []store.SeedPath{
			{Slug: "dsa", Title: "DSA", Status: "active", Summary: "s", ProblemTotal: 151, WeekTotal: 2, SortOrder: 0},
		},
		Phases: []store.SeedPhase{
			{PathSlug: "dsa", Order: 1, Name: "Fundamentals", Theme: "arrays", WeekFrom: 1, WeekTo: 2},
		},
		Weeks: []store.SeedWeek{
			{PathSlug: "dsa", N: 1, Title: "Arrays", Thesis: "t1"},
			{PathSlug: "dsa", N: 2, Title: "Two Pointers", Thesis: "t2"},
		},
		Concepts: []store.SeedConcept{
			{PathSlug: "dsa", Slug: "two-pointers", Title: "Two Pointers", BodyMD: "b", WhenToUseMD: "w", CodeTemplate: "c", Weeks: []int{2}},
		},
		Problems: []store.SeedProblem{
			{ID: "1", PathSlug: "dsa", WeekN: 1, Title: "Contains Duplicate", Difficulty: "easy", Pattern: "Hashing", SortOrder: 1,
				Sections: []store.SeedSection{{Stage: "attempt", Kind: "summary", Order: 1, BodyMD: "x"}}},
			{ID: "9", PathSlug: "dsa", WeekN: 1, Title: "Subarray Sum", Difficulty: "med", Pattern: "Prefix Sum", SortOrder: 2},
			{ID: "16", PathSlug: "dsa", WeekN: 2, Title: "3Sum", Difficulty: "med", Pattern: "Two Pointers", SortOrder: 1,
				Sections: []store.SeedSection{
					{Stage: "attempt", Kind: "summary", Order: 1, BodyMD: "a"},
					{Stage: "solution", Kind: "code", Order: 1, Code: "func threeSum() {}"},
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
