package curriculum

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

// Sprint m1-03 task 2: the course-scoped concept route and its DSA alias, the `course`
// block on /paths*, and the v1 JSON fields derived from the v2 columns (additive only).

// raw serves GET path and returns the status and the raw body.
func raw(t *testing.T, h http.Handler, path string) (int, []byte) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code, rec.Body.Bytes()
}

// field returns the raw JSON of key in the object b.
func field(t *testing.T, b []byte, key string) json.RawMessage {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatalf("not a JSON object: %v (%s)", err, b)
	}
	v, ok := obj[key]
	if !ok {
		t.Fatalf("no %q in %s", key, b)
	}
	return v
}

// elems returns the raw elements of the JSON array b.
func elems(t *testing.T, b []byte) []json.RawMessage {
	t.Helper()
	var out []json.RawMessage
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("not a JSON array: %v (%s)", err, b)
	}
	return out
}

// assertV1Prefix checks that the served object got begins with v1's fields, byte for
// byte and in v1's order (v1 is a value of the v1 shape), followed only by new fields.
func assertV1Prefix(t *testing.T, what string, got json.RawMessage, v1 any) {
	t.Helper()
	b, err := json.Marshal(v1)
	if err != nil {
		t.Fatal(err)
	}
	prefix := b[:len(b)-1] // drop the closing brace
	if !bytes.HasPrefix(got, prefix) || (len(got) > len(prefix) && got[len(prefix)] != ',' && got[len(prefix)] != '}') {
		t.Errorf("%s: served %s\n  want the v1 fields first: %s", what, got, b)
	}
}

// The v1 response shapes (the JSON v1.6.0 serves), for the byte-level checks.
type (
	v1Path struct {
		Slug         string `json:"slug"`
		Title        string `json:"title"`
		Status       string `json:"status"`
		Summary      string `json:"summary"`
		ProblemTotal int    `json:"problem_total"`
		WeekTotal    int    `json:"week_total"`
	}
	v1Problem struct {
		ID              string `json:"id"`
		PathSlug        string `json:"path_slug"`
		WeekN           int    `json:"week_n"`
		Title           string `json:"title"`
		Difficulty      string `json:"difficulty"`
		Pattern         string `json:"pattern"`
		LeetcodeURL     string `json:"leetcode_url"`
		NeetcodeURL     string `json:"neetcode_url"`
		IsReinforcement bool   `json:"is_reinforcement"`
	}
	v1Section struct {
		Stage  string `json:"stage"`
		Kind   string `json:"kind"`
		Order  int    `json:"order"`
		BodyMD string `json:"body_md"`
		Code   string `json:"code"`
	}
	v1Concept struct {
		Slug         string `json:"slug"`
		PathSlug     string `json:"path_slug"`
		Title        string `json:"title"`
		BodyMD       string `json:"body_md"`
		WhenToUseMD  string `json:"when_to_use_md"`
		CodeTemplate string `json:"code_template"`
	}
)

func TestGetConceptCourseScoped(t *testing.T) {
	f := seededFake()
	f.addConcept(store.Concept{Slug: "zz-loops", PathSlug: coursetest.FixtureActive, Title: "Loops", BodyMD: "zb",
		WhenToUseMD: "zw", Templates: map[string]string{"go": "zz go", "py": "zz py"}})
	h := testService(t, f).Handler()

	code, b := raw(t, h, "/paths/"+course.DefaultSlug+"/concepts/two-pointers")
	if code != http.StatusOK {
		t.Fatalf("DSA concept: status %d (%s)", code, b)
	}
	c := field(t, b, "concept")
	assertV1Prefix(t, "DSA concept", c, v1Concept{
		Slug: "two-pointers", PathSlug: course.DefaultSlug, Title: "Two Pointers", BodyMD: "b", WhenToUseMD: "w", CodeTemplate: "c",
	})
	if tm := field(t, c, "templates"); string(tm) != `{"go":"c"}` {
		t.Errorf("DSA concept templates = %s", tm)
	}

	// A concept resolves only under its own course: the fixture's under zz-fixture, with
	// code_template from zz-fixture's primary language.
	code, b = raw(t, h, "/paths/"+coursetest.FixtureActive+"/concepts/zz-loops")
	if code != http.StatusOK {
		t.Fatalf("zz-fixture concept: status %d (%s)", code, b)
	}
	c = field(t, b, "concept")
	assertV1Prefix(t, "zz-fixture concept", c, v1Concept{
		Slug: "zz-loops", PathSlug: coursetest.FixtureActive, Title: "Loops", BodyMD: "zb", WhenToUseMD: "zw", CodeTemplate: "zz go",
	})
	if tm := field(t, c, "templates"); string(tm) != `{"go":"zz go","py":"zz py"}` {
		t.Errorf("zz-fixture concept templates = %s", tm)
	}

	for _, path := range []string{
		"/paths/" + course.DefaultSlug + "/concepts/nope",               // unknown concept
		"/paths/" + course.DefaultSlug + "/concepts/zz-loops",           // another course's concept
		"/paths/" + coursetest.FixtureActive + "/concepts/two-pointers", // DSA's concept under zz-fixture
		"/concepts/zz-loops",                          // the DSA alias only reaches DSA
		"/paths/no-such-course/concepts/two-pointers", // unknown course
		"/concepts/nope",                              // unknown concept via the alias
	} {
		rec, body := do(t, h, http.MethodGet, path)
		if rec.Code != http.StatusNotFound || body["error"].(map[string]any)["code"] != "not_found" {
			t.Errorf("%s: status %d body %v, want 404 not_found", path, rec.Code, body)
		}
	}
}

// GET /concepts/{slug} is the DSA alias: byte-identical to the course-scoped route.
func TestConceptAliasByteIdentical(t *testing.T) {
	h := testService(t, seededFake()).Handler()
	for _, slug := range []string{"two-pointers", "nope"} {
		ac, ab := raw(t, h, "/concepts/"+slug)
		sc, sb := raw(t, h, "/paths/"+course.DefaultSlug+"/concepts/"+slug)
		if ac != sc || !bytes.Equal(ab, sb) {
			t.Errorf("%s: alias %d %s\n  scoped %d %s", slug, ac, ab, sc, sb)
		}
	}
}

// code_template is the course's primary language's template: "" when the course has no
// manifest (or names no language); templates is always an object.
func TestConceptTemplateLanguage(t *testing.T) {
	f := newFakeStore()
	f.addConcept(store.Concept{Slug: "x", PathSlug: "no-manifest", Title: "X", Templates: map[string]string{"go": "g"}})
	f.addConcept(store.Concept{Slug: "empty", PathSlug: course.DefaultSlug, Title: "Empty"})
	h := testService(t, f).Handler()

	_, b := raw(t, h, "/paths/no-manifest/concepts/x")
	c := field(t, b, "concept")
	if ct := field(t, c, "code_template"); string(ct) != `""` {
		t.Errorf("no manifest: code_template = %s, want \"\"", ct)
	}
	if tm := field(t, c, "templates"); string(tm) != `{"go":"g"}` {
		t.Errorf("no manifest: templates = %s", tm)
	}
	_, b = raw(t, h, "/concepts/empty")
	c = field(t, b, "concept")
	if ct, tm := field(t, c, "code_template"), field(t, c, "templates"); string(ct) != `""` || string(tm) != `{}` {
		t.Errorf("no templates: code_template = %s, templates = %s; want \"\" and {}", ct, tm)
	}
}

// catalogFake holds one path per case of the catalog filter.
func catalogFake() *fakeStore {
	f := newFakeStore()
	f.paths = []store.Path{
		{Slug: course.DefaultSlug, Title: "Data Structures & Algorithms", Status: "active", Summary: "s", ProblemTotal: 151, WeekTotal: 16},
		{Slug: coursetest.FixtureComingSoon, Title: "Soon", Status: "coming_soon", Summary: "z", ProblemTotal: 5, WeekTotal: 2, SortOrder: 10},
		// The manifest decides: this row says coming_soon, zz-retired's manifest retired.
		{Slug: coursetest.FixtureRetired, Title: "Gone", Status: "coming_soon", Summary: "r", SortOrder: 20},
		// preview is listed: the gateway, which knows the viewer, filters it.
		{Slug: coursetest.FixturePreview, Title: "Preview", Status: "coming_soon", Summary: "p", SortOrder: 30},
		// No manifest: listed without a course block; the row's status decides.
		{Slug: "no-manifest", Title: "Row only", Status: "coming_soon", Summary: "n", SortOrder: 40},
		{Slug: "no-manifest-retired", Title: "Row retired", Status: course.StatusRetired, Summary: "nr", SortOrder: 50},
	}
	return f
}

func TestListPathsCourseView(t *testing.T) {
	h := testService(t, catalogFake()).Handler()
	code, b := raw(t, h, "/paths")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	paths := elems(t, field(t, b, "paths"))
	var slugs []string
	byslug := map[string]json.RawMessage{}
	for _, p := range paths {
		var s string
		_ = json.Unmarshal(field(t, p, "slug"), &s)
		slugs = append(slugs, s)
		byslug[s] = p
	}
	want := []string{course.DefaultSlug, coursetest.FixtureComingSoon, coursetest.FixturePreview, "no-manifest"}
	if !reflect.DeepEqual(slugs, want) {
		t.Fatalf("GET /paths slugs = %v, want %v (retired out, order kept)", slugs, want)
	}

	// DSA: the v1 fields unchanged, then its course block.
	dsa := byslug[course.DefaultSlug]
	assertV1Prefix(t, "DSA path", dsa, v1Path{
		Slug: course.DefaultSlug, Title: "Data Structures & Algorithms", Status: "active", Summary: "s", ProblemTotal: 151, WeekTotal: 16,
	})
	var v course.View
	if err := json.Unmarshal(field(t, dsa, "course"), &v); err != nil {
		t.Fatal(err)
	}
	if v.Slug != course.DefaultSlug || v.Status != course.StatusActive || v.ShortCode != "DSA" || v.Nav == nil || v.Nav.ItemNoun != "problem" {
		t.Errorf("DSA course view = %+v", v)
	}
	if !reflect.DeepEqual(v, coursetest.DSA(t).LearnerView()) {
		t.Errorf("DSA course block is not the manifest's LearnerView()")
	}

	var soon course.View
	if err := json.Unmarshal(field(t, byslug[coursetest.FixtureComingSoon], "course"), &soon); err != nil {
		t.Fatal(err)
	}
	var rowStatus string
	_ = json.Unmarshal(field(t, byslug[coursetest.FixtureComingSoon], "status"), &rowStatus)
	if rowStatus != course.StatusComingSoon || soon.Status != course.StatusComingSoon {
		t.Errorf("zz-soon: status %q, course.status %q; want coming_soon", rowStatus, soon.Status)
	}
	if strings.Contains(string(byslug["no-manifest"]), `"course"`) {
		t.Errorf("a path without a manifest carries a course block: %s", byslug["no-manifest"])
	}
}

// GET /paths/{slug} carries the course block inside "path" for any status (the gateway
// gates visibility); phases and weeks are unchanged.
func TestGetPathCourseView(t *testing.T) {
	f := catalogFake()
	f.phases[course.DefaultSlug] = []store.Phase{{Order: 1, Name: "Fundamentals", Theme: "arrays", WeekFrom: 1, WeekTo: 3}}
	f.weeks[course.DefaultSlug] = []store.WeekSummary{{N: 1, Title: "Arrays", Thesis: "t1", Easy: 3, Total: 3}}
	h := testService(t, f).Handler()

	code, b := raw(t, h, "/paths/"+course.DefaultSlug)
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	p := field(t, b, "path")
	assertV1Prefix(t, "DSA roadmap path", p, v1Path{
		Slug: course.DefaultSlug, Title: "Data Structures & Algorithms", Status: "active", Summary: "s", ProblemTotal: 151, WeekTotal: 16,
	})
	var v course.View
	if err := json.Unmarshal(field(t, p, "course"), &v); err != nil || v.ShortCode != "DSA" {
		t.Errorf("DSA roadmap course block = %+v, %v", v, err)
	}
	if ph := field(t, b, "phases"); string(ph) != `[{"order":1,"name":"Fundamentals","theme":"arrays","week_from":1,"week_to":3}]` {
		t.Errorf("phases = %s", ph)
	}
	if wk := field(t, b, "weeks"); string(wk) != `[{"n":1,"title":"Arrays","thesis":"t1","easy":3,"med":0,"hard":0,"total":3}]` {
		t.Errorf("weeks = %s", wk)
	}

	code, b = raw(t, h, "/paths/"+coursetest.FixtureRetired)
	if code != http.StatusOK {
		t.Fatalf("retired path: status %d", code)
	}
	if err := json.Unmarshal(field(t, field(t, b, "path"), "course"), &v); err != nil || v.Status != course.StatusRetired {
		t.Errorf("retired path course block = %+v, %v", v, err)
	}
	_, b = raw(t, h, "/paths/no-manifest")
	if strings.Contains(string(field(t, b, "path")), `"course"`) {
		t.Errorf("a path without a manifest carries a course block: %s", b)
	}
}

// Every problem route serves the v1 fields first, byte for byte, derived from role and
// links, then the additive role and links.
func TestProblemJSONV1Fields(t *testing.T) {
	f := seededFake()
	reinf := store.Problem{ID: "9", PathSlug: "dsa", WeekN: 1, Title: "Subarray Sum", Difficulty: "med", Pattern: "Prefix Sum",
		Role: "reinforcement", Links: []store.Link{
			{Kind: "neetcode", URL: "nc9"}, {Kind: "other", URL: "o9"}, {Kind: "leetcode", URL: "lc9"}, {Kind: "leetcode", URL: "lc9-second"},
		}}
	plain := store.Problem{ID: "1", PathSlug: "dsa", WeekN: 1, Title: "Contains Duplicate", Difficulty: "easy", Pattern: "Hashing", Role: "core"}
	f.problem["9"], f.problem["1"] = reinf, plain
	f.week[wkKey{"dsa", 1}] = store.Week{N: 1, Title: "Arrays", Thesis: "t1"}
	f.problems[wkKey{"dsa", 1}] = []store.Problem{plain, reinf}
	h := testService(t, f).Handler()

	want := map[string]struct {
		v1    v1Problem
		role  string
		links string
	}{
		"9": {v1Problem{ID: "9", PathSlug: "dsa", WeekN: 1, Title: "Subarray Sum", Difficulty: "med", Pattern: "Prefix Sum",
			LeetcodeURL: "lc9", NeetcodeURL: "nc9", IsReinforcement: true}, "reinforcement",
			`[{"kind":"neetcode","url":"nc9"},{"kind":"other","url":"o9"},{"kind":"leetcode","url":"lc9"},{"kind":"leetcode","url":"lc9-second"}]`},
		"1": {v1Problem{ID: "1", PathSlug: "dsa", WeekN: 1, Title: "Contains Duplicate", Difficulty: "easy", Pattern: "Hashing"}, "core", `[]`},
	}
	check := func(route string, p json.RawMessage) {
		t.Helper()
		var id string
		_ = json.Unmarshal(field(t, p, "id"), &id)
		w, ok := want[id]
		if !ok {
			return
		}
		assertV1Prefix(t, route+" problem "+id, p, w.v1)
		if r := field(t, p, "role"); string(r) != `"`+w.role+`"` {
			t.Errorf("%s problem %s: role %s, want %q", route, id, r, w.role)
		}
		if l := field(t, p, "links"); string(l) != w.links {
			t.Errorf("%s problem %s: links %s, want %s", route, id, l, w.links)
		}
	}
	for _, id := range []string{"9", "1"} {
		_, b := raw(t, h, "/problems/"+id)
		check("/problems/"+id, field(t, b, "problem"))
	}
	for _, route := range []string{"/problems?ids=1,9", "/paths/dsa/problems", "/paths/dsa/weeks/1"} {
		code, b := raw(t, h, route)
		if code != http.StatusOK {
			t.Fatalf("%s: status %d", route, code)
		}
		ps := elems(t, field(t, b, "problems"))
		if len(ps) < 2 {
			t.Fatalf("%s: %d problems", route, len(ps))
		}
		for _, p := range ps {
			check(route, p)
		}
	}

	// Sections keep the v1 fields, then language.
	_, b := raw(t, h, "/problems/16")
	secs := elems(t, field(t, b, "sections"))
	assertV1Prefix(t, "prose section", secs[0], v1Section{Stage: "attempt", Kind: "summary", Order: 1, BodyMD: "a"})
	assertV1Prefix(t, "code section", secs[1], v1Section{Stage: "solution", Kind: "code", Order: 1, Code: "func threeSum() {}"})
	if l0, l1 := field(t, secs[0], "language"), field(t, secs[1], "language"); string(l0) != `""` || string(l1) != `"go"` {
		t.Errorf("section languages = %s, %s; want \"\" and \"go\"", l0, l1)
	}
}
