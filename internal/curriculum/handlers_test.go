package curriculum

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

// testService serves st with the test registry: every embedded course plus the
// fixtures (zz-fixture, zz-preview, zz-soon, zz-retired).
func testService(t *testing.T, st store.Store) *Service {
	t.Helper()
	return NewService(st, coursetest.Registry(t), slog.New(slog.NewTextHandler(io.Discard, nil)))
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
		{Slug: "dsa", Title: "Data Structures & Algorithms", Status: "active", Summary: "s", ProblemTotal: 151, WeekTotal: 16},
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
		{ID: "16", PathSlug: "dsa", WeekN: 2, Title: "3Sum", Difficulty: "med", Pattern: "Two Pointers", Role: "core",
			Links: []store.Link{{Kind: "leetcode", URL: "lc"}, {Kind: "neetcode", URL: "nc"}}},
	}
	f.problem["16"] = store.Problem{ID: "16", PathSlug: "dsa", WeekN: 2, Title: "3Sum", Difficulty: "med", Pattern: "Two Pointers", Role: "core"}
	f.sections["16"] = []store.Section{
		{Stage: "attempt", Kind: "summary", Order: 1, BodyMD: "a"},
		{Stage: "solution", Kind: "code", Order: 1, Code: "func threeSum() {}", Language: "go"},
	}
	f.addConcept(store.Concept{Slug: "two-pointers", PathSlug: "dsa", Title: "Two Pointers", BodyMD: "b", WhenToUseMD: "w",
		Templates: map[string]string{"go": "c"}})
	return f
}

func TestListPaths(t *testing.T) {
	h := testService(t, seededFake()).Handler()
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
	h := testService(t, seededFake()).Handler()
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
	h := testService(t, seededFake()).Handler()
	rec, body := do(t, h, http.MethodGet, "/paths/nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if body["error"].(map[string]any)["code"] != "not_found" {
		t.Fatalf("expected not_found envelope, got %v", body)
	}
}

func TestGetWeek(t *testing.T) {
	h := testService(t, seededFake()).Handler()
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
	h := testService(t, seededFake()).Handler()
	rec, _ := do(t, h, http.MethodGet, "/paths/dsa/weeks/abc")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestGetWeekNotFound(t *testing.T) {
	h := testService(t, seededFake()).Handler()
	rec, _ := do(t, h, http.MethodGet, "/paths/dsa/weeks/99")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetProblem(t *testing.T) {
	h := testService(t, seededFake()).Handler()
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

// GET /problems/{id} adds contract_hash_prefix (12 hex, "" on the self path) and
// grading_summary (m3-01); list and bulk routes stay unchanged.
func TestGetProblemContractFields(t *testing.T) {
	f := seededFake()
	full := "sha256:ab12cd34ef56" + strings.Repeat("0", 52)
	p := f.problem["16"]
	p.ContractHash = full
	p.GradingSummary = json.RawMessage(`{"mode":"auto","parts":[{"id":"solution","type":"code","grading":"auto","cadence":"iterate"}],"grader_kinds":["code"],"languages":["go"]}`)
	f.problem["16"] = p
	f.problem["1"] = store.Problem{ID: "1", PathSlug: "dsa", WeekN: 1, Title: "Contains Duplicate", GradingSummary: json.RawMessage(`{"mode":"self"}`)}
	h := testService(t, f).Handler()

	_, body := do(t, h, http.MethodGet, "/problems/16")
	pj := body["problem"].(map[string]any)
	if pj["contract_hash_prefix"] != "ab12cd34ef56" {
		t.Fatalf("contract_hash_prefix = %v, want ab12cd34ef56", pj["contract_hash_prefix"])
	}
	if gs := pj["grading_summary"].(map[string]any); gs["mode"] != "auto" || len(gs["parts"].([]any)) != 1 {
		t.Fatalf("grading_summary = %v", pj["grading_summary"])
	}
	if _, ok := pj["contract_hash"]; ok {
		t.Fatal("the full contract hash must not be served (humans see a prefix)")
	}

	_, body = do(t, h, http.MethodGet, "/problems/1")
	pj = body["problem"].(map[string]any)
	if pj["contract_hash_prefix"] != "" || pj["grading_summary"].(map[string]any)["mode"] != "self" {
		t.Fatalf("self-path problem = %v", pj)
	}

	// The bulk and list routes keep their shape.
	for _, path := range []string{"/problems?ids=16,1", "/paths/dsa/problems", "/paths/dsa/weeks/2"} {
		rec, body := do(t, h, http.MethodGet, path)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", path, rec.Code)
		}
		for _, raw := range body["problems"].([]any) {
			for _, k := range []string{"contract_hash_prefix", "grading_summary", "contract_hash"} {
				if _, ok := raw.(map[string]any)[k]; ok {
					t.Errorf("%s serves %s", path, k)
				}
			}
		}
	}
}

// An item's solution_facts are served only inside a solution-stage block of GET
// /problems/{id} (m1-06, t1 §4) — never top-level, never on the list, bulk or week routes
// — so the gateway's stage filter and withhold() gate them. The loader doesn't persist the
// field yet (m2-01's spec column will), so this is a fixture test.
func TestGetProblemSolutionFactsOnlyInSolutionStage(t *testing.T) {
	f := seededFake()
	p := f.problem["16"]
	p.SolutionFacts = json.RawMessage(`{"complexity":{"time":["O(n^2)"],"space":["O(1)"]}}`)
	f.problem["16"] = p
	h := testService(t, f).Handler()

	_, body := do(t, h, http.MethodGet, "/problems/16")
	if _, ok := body["problem"].(map[string]any)["solution_facts"]; ok {
		t.Fatal("solution_facts must not be served top-level on the problem")
	}
	var facts []map[string]any
	maxSolutionOrder := 0.0
	for _, raw := range body["sections"].([]any) {
		sec := raw.(map[string]any)
		if _, ok := sec["solution_facts"]; ok {
			facts = append(facts, sec)
		} else if sec["stage"] == "solution" && sec["order"].(float64) > maxSolutionOrder {
			maxSolutionOrder = sec["order"].(float64)
		}
	}
	if len(facts) != 1 || facts[0]["stage"] != "solution" || facts[0]["kind"] != "solution_facts" {
		t.Fatalf("want exactly one solution-stage facts block, got %v", facts)
	}
	if facts[0]["order"].(float64) <= maxSolutionOrder {
		t.Errorf("the facts block (order %v) should follow the solution sections (max %v)", facts[0]["order"], maxSolutionOrder)
	}
	if got := facts[0]["solution_facts"].(map[string]any)["complexity"].(map[string]any)["time"].([]any)[0]; got != "O(n^2)" {
		t.Errorf("facts payload = %v", facts[0]["solution_facts"])
	}

	for _, path := range []string{"/problems?ids=16", "/paths/dsa/problems", "/paths/dsa/weeks/2"} {
		rec, raw := doRaw(t, h, path)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", path, rec.Code)
		}
		if strings.Contains(raw, "solution_facts") || strings.Contains(raw, "O(n^2)") {
			t.Errorf("%s serves solution facts: %s", path, raw)
		}
	}

	// An item without facts has no facts block (v1's sections unchanged).
	_, body = do(t, testService(t, seededFake()).Handler(), http.MethodGet, "/problems/16")
	for _, raw := range body["sections"].([]any) {
		if raw.(map[string]any)["kind"] == "solution_facts" {
			t.Fatal("a facts block without facts")
		}
	}
}

// doRaw issues a GET and returns the recorder and the raw body.
func doRaw(t *testing.T, h http.Handler, path string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec, rec.Body.String()
}

func TestGetProblemNotFound(t *testing.T) {
	h := testService(t, seededFake()).Handler()
	rec, _ := do(t, h, http.MethodGet, "/problems/9999")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetConcept(t *testing.T) {
	h := testService(t, seededFake()).Handler()
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
	h := testService(t, seededFake()).Handler()
	rec, _ := do(t, h, http.MethodGet, "/concepts/nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestReadyz(t *testing.T) {
	h := testService(t, seededFake()).Handler()
	rec, _ := do(t, h, http.MethodGet, "/readyz")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
