package curriculum

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

func testService(st store.Store) *Service {
	return NewService(st, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func do(t *testing.T, h http.Handler, method, path string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s %s: response not JSON: %v (%s)", method, path, err, rec.Body.String())
		}
	}
	return rec, body
}

func seededFake() *fakeStore {
	f := newFakeStore()
	f.paths = []store.Path{
		{Slug: "dsa", Title: "DSA Interview Mastery", Status: "active", Summary: "s", ProblemTotal: 151, WeekTotal: 16},
		{Slug: "system-design", Title: "System Design Interviews", Status: "coming_soon", Summary: "sd", ProblemTotal: 40, WeekTotal: 12},
	}
	f.phases["dsa"] = []store.Phase{
		{Order: 1, Name: "Fundamentals", Theme: "arrays", WeekFrom: 1, WeekTo: 3},
	}
	f.weeks["dsa"] = []store.WeekSummary{
		{N: 1, Title: "Arrays", Thesis: "t1", Easy: 3, Med: 3, Hard: 0, Total: 6},
		{N: 2, Title: "Two Pointers", Thesis: "t2", Easy: 0, Med: 2, Hard: 1, Total: 3},
	}
	f.week[wkKey{"dsa", 2}] = store.Week{N: 2, Title: "Two Pointers", Thesis: "t2"}
	f.concepts[wkKey{"dsa", 2}] = []store.ConceptRef{{Slug: "two-pointers", Title: "Two Pointers"}}
	f.problems[wkKey{"dsa", 2}] = []store.Problem{
		{ID: "16", PathSlug: "dsa", WeekN: 2, Title: "3Sum", Difficulty: "med", Pattern: "Two Pointers", LeetcodeURL: "lc", NeetcodeURL: "nc"},
	}
	f.problem["16"] = store.Problem{ID: "16", PathSlug: "dsa", WeekN: 2, Title: "3Sum", Difficulty: "med", Pattern: "Two Pointers"}
	f.sections["16"] = []store.Section{
		{Stage: "attempt", Kind: "summary", Order: 1, BodyMD: "a"},
		{Stage: "solution", Kind: "code", Order: 1, Code: "func threeSum() {}"},
	}
	f.concept["two-pointers"] = store.Concept{Slug: "two-pointers", PathSlug: "dsa", Title: "Two Pointers", BodyMD: "b", WhenToUseMD: "w", CodeTemplate: "c"}
	return f
}

func TestListPaths(t *testing.T) {
	h := testService(seededFake()).Handler()
	rec, body := do(t, h, http.MethodGet, "/paths")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	paths, ok := body["paths"].([]any)
	if !ok || len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %v", body["paths"])
	}
	first := paths[0].(map[string]any)
	if first["slug"] != "dsa" || first["status"] != "active" {
		t.Fatalf("unexpected first path: %v", first)
	}
	// problem_total must be a real number, not a string.
	if first["problem_total"].(float64) != 151 {
		t.Fatalf("problem_total = %v, want 151", first["problem_total"])
	}
}

func TestGetPath(t *testing.T) {
	h := testService(seededFake()).Handler()
	rec, body := do(t, h, http.MethodGet, "/paths/dsa")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body["path"].(map[string]any)["slug"] != "dsa" {
		t.Fatalf("unexpected path: %v", body["path"])
	}
	if len(body["phases"].([]any)) != 1 {
		t.Fatalf("expected 1 phase")
	}
	weeks := body["weeks"].([]any)
	if len(weeks) != 2 {
		t.Fatalf("expected 2 weeks, got %d", len(weeks))
	}
	w2 := weeks[1].(map[string]any)
	if w2["hard"].(float64) != 1 || w2["total"].(float64) != 3 {
		t.Fatalf("week 2 difficulty mix wrong: %v", w2)
	}
}

func TestGetPathNotFound(t *testing.T) {
	h := testService(seededFake()).Handler()
	rec, body := do(t, h, http.MethodGet, "/paths/nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if body["error"].(map[string]any)["code"] != "not_found" {
		t.Fatalf("expected not_found envelope, got %v", body)
	}
}

func TestGetWeek(t *testing.T) {
	h := testService(seededFake()).Handler()
	rec, body := do(t, h, http.MethodGet, "/paths/dsa/weeks/2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body["week"].(map[string]any)["n"].(float64) != 2 {
		t.Fatalf("unexpected week: %v", body["week"])
	}
	if len(body["concepts"].([]any)) != 1 || len(body["problems"].([]any)) != 1 {
		t.Fatalf("expected 1 concept + 1 problem, got %v / %v", body["concepts"], body["problems"])
	}
	// The week carries its phase (order/name for the eyebrow) and slim path totals.
	phase, ok := body["phase"].(map[string]any)
	if !ok || phase["order"].(float64) != 1 || phase["name"] != "Fundamentals" {
		t.Fatalf("expected week 2 to resolve phase 1 Fundamentals, got %v", body["phase"])
	}
	path, ok := body["path"].(map[string]any)
	if !ok || path["slug"] != "dsa" || path["week_total"].(float64) != 16 {
		t.Fatalf("expected path context (dsa, week_total 16), got %v", body["path"])
	}
}

func TestGetWeekBadNumber(t *testing.T) {
	h := testService(seededFake()).Handler()
	rec, _ := do(t, h, http.MethodGet, "/paths/dsa/weeks/abc")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestGetWeekNotFound(t *testing.T) {
	h := testService(seededFake()).Handler()
	rec, _ := do(t, h, http.MethodGet, "/paths/dsa/weeks/99")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetProblem(t *testing.T) {
	h := testService(seededFake()).Handler()
	rec, body := do(t, h, http.MethodGet, "/problems/16")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body["problem"].(map[string]any)["title"] != "3Sum" {
		t.Fatalf("unexpected problem: %v", body["problem"])
	}
	sections := body["sections"].([]any)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	// Each section carries its stage (the shape S05 filters on).
	if sections[0].(map[string]any)["stage"] != "attempt" {
		t.Fatalf("first section stage = %v, want attempt", sections[0])
	}
}

func TestGetProblemNotFound(t *testing.T) {
	h := testService(seededFake()).Handler()
	rec, _ := do(t, h, http.MethodGet, "/problems/9999")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetConcept(t *testing.T) {
	h := testService(seededFake()).Handler()
	rec, body := do(t, h, http.MethodGet, "/concepts/two-pointers")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	c := body["concept"].(map[string]any)
	if c["slug"] != "two-pointers" || c["code_template"] != "c" {
		t.Fatalf("unexpected concept: %v", c)
	}
}

func TestGetConceptNotFound(t *testing.T) {
	h := testService(seededFake()).Handler()
	rec, _ := do(t, h, http.MethodGet, "/concepts/nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestReadyz(t *testing.T) {
	h := testService(seededFake()).Handler()
	rec, _ := do(t, h, http.MethodGet, "/readyz")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
