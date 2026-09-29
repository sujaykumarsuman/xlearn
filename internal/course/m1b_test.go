package course_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// The compiled-in registry holds exactly the courses curriculum ships, DSA active.
func TestEmbeddedRegistry(t *testing.T) {
	r, err := course.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	all := coursetest.All(t)
	if got, want := len(r.Slugs()), len(all); got != want {
		t.Fatalf("embedded registry has %d courses, curriculum.FS has %d", got, want)
	}
	m, ok := course.Lookup(course.DefaultSlug)
	if !ok || m.Status != course.StatusActive {
		t.Fatalf("Lookup(DefaultSlug) = %v, %v; want the active DSA manifest", m, ok)
	}
	if course.DSASlug != course.DefaultSlug {
		t.Fatal("DSASlug and DefaultSlug name the same course")
	}
	if _, ok := course.Lookup("nope"); ok {
		t.Fatal("unknown slug resolved")
	}
	for _, s := range []string{coursetest.FixtureActive, coursetest.FixturePreview, coursetest.FixtureComingSoon, coursetest.FixtureRetired} {
		if _, ok := course.Lookup(s); ok {
			t.Fatalf("fixture %s is embedded; fixtures must exist only in coursetest.Registry", s)
		}
	}
	var nilReg *course.Registry
	if _, ok := nilReg.Lookup(course.DefaultSlug); ok {
		t.Fatal("a nil registry resolved a slug")
	}
}

// The test registry adds one fixture per status beside the embedded courses.
func TestFixtureRegistry(t *testing.T) {
	r := coursetest.Registry(t)
	want := map[string]string{
		course.DefaultSlug:           course.StatusActive,
		coursetest.FixtureActive:     course.StatusActive,
		coursetest.FixturePreview:    course.StatusPreview,
		coursetest.FixtureComingSoon: course.StatusComingSoon,
		coursetest.FixtureRetired:    course.StatusRetired,
	}
	for slug, status := range want {
		m, ok := r.Lookup(slug)
		if !ok || m.Status != status {
			t.Errorf("Lookup(%s) = %v, %v; want status %s", slug, m, ok, status)
		}
	}
	if err := course.CheckCourseSlug(coursetest.FixtureActive); err != nil {
		t.Fatal(err)
	}
}

// The learner view carries presentation data only, and DSA's matches its manifest.
func TestLearnerViewDSA(t *testing.T) {
	m := coursetest.DSA(t)
	v := m.LearnerView()
	if v.Slug != course.DefaultSlug || v.Title != m.Title || v.Status != course.StatusActive {
		t.Fatalf("view header = %+v", v)
	}
	if v.ShortCode != "DSA" {
		t.Fatalf("short code = %q, want DSA (v1's selector badge)", v.ShortCode)
	}
	if v.Nav == nil || v.Nav.ItemNoun != "problem" || len(v.Nav.Groups) != 2 {
		t.Fatalf("nav = %+v", v.Nav)
	}
	var labels []string
	for _, g := range v.Nav.Groups {
		for _, it := range g.Items {
			labels = append(labels, it.Label)
		}
	}
	if got, want := strings.Join(labels, "|"), "Today|Roadmap|Problems|Progress|Revision|Mistakes|Mock interview"; got != want {
		t.Fatalf("nav labels = %s, want %s (v1's navForPath)", got, want)
	}
	if v.Nav.Groups[1].Cap != "Practice loop" {
		t.Fatalf("cap = %q", v.Nav.Groups[1].Cap)
	}
	if v.Stages == nil || v.Stages.Attempt.DurationS != 900 || v.Stages.Hint.DurationS != 600 {
		t.Fatalf("stages = %+v", v.Stages)
	}
	if v.EstMinutes["course_attempt"] != 45 || len(v.MockRail) != 6 || v.MockRail[0].Label != "Clarify" {
		t.Fatalf("est/mock rail = %v %+v", v.EstMinutes, v.MockRail)
	}
	if v.PrimaryLanguage != "go" {
		t.Fatalf("primary language = %q", v.PrimaryLanguage)
	}
	// m1-06: the revision bands' display fields (AB03's format badges): DSA L1–3 is a
	// 20:00 re-solve, L4–5 the same under mock conditions; minutes from the plan.
	want := []course.ViewBand{
		{Levels: []int{1, 2, 3}, Format: "resolve", Label: "Re-solve", TimerS: 1200, EstMinutes: 20, MockMode: false},
		{Levels: []int{4, 5}, Format: "resolve", Label: "Re-solve", TimerS: 1200, EstMinutes: 20, MockMode: true},
	}
	if v.Revision == nil || !reflect.DeepEqual(v.Revision.Bands, want) {
		t.Fatalf("revision bands = %+v, want %+v", v.Revision, want)
	}

	// Nothing answer-bearing or server-only reaches the JSON (the bands carry no criteria).
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"persona", "criteria", "prefill", "rubric", "pools", "evidence", "params", "threshold", "prompt", "targets"} {
		if strings.Contains(string(b), banned) {
			t.Errorf("learner view leaks %q: %s", banned, b)
		}
	}
}

// A coming_soon course's view is its header only; a second course renders its own nav.
func TestLearnerViewFixtures(t *testing.T) {
	r := coursetest.Registry(t)
	soon, _ := r.Lookup(coursetest.FixtureComingSoon)
	v := soon.LearnerView()
	if v.Nav != nil || v.Stages != nil || v.MockRail != nil || v.EstMinutes != nil || v.Revision != nil {
		t.Fatalf("coming_soon view = %+v, want header only", v)
	}
	fx, _ := r.Lookup(coursetest.FixtureActive)
	fv := fx.LearnerView()
	if fv.Nav.ItemNoun != "exercise" || fv.Nav.Groups[0].Items[2].Label != "Exercises" || fv.MockRail != nil {
		t.Fatalf("fixture view = %+v", fv)
	}
	if fv.ShortCode != "ZZ" {
		t.Fatalf("fixture short code = %q", fv.ShortCode)
	}
}

// The dual coach-context parser: every v1 form maps to its DefaultSlug key, new forms and
// account-wide contexts are unchanged, and normalizing is idempotent.
func TestNormalizeCoachContext(t *testing.T) {
	r := coursetest.Registry(t)
	d := course.DefaultSlug
	cases := []struct {
		in, key, path, kind string
	}{
		// v1 course-scoped forms (open v1.6.0 tabs) → the default course.
		{"concept:two-pointers", d + ":concept:two-pointers", d, "concept"},
		{"week:3", d + ":week:3", d, "week"},
		{"roadmap", d + ":roadmap", d, "roadmap"},
		{"dashboard", d + ":dashboard", d, "dashboard"},
		{"revision", d + ":revision", d, "revision"},
		{"mistakes", d + ":mistakes", d, "mistakes"},
		{"mock", d + ":mock", d, "mock"},
		{"progress", d + ":progress", d, "progress"},
		// New forms pass through with their course.
		{d + ":concept:two-pointers", d + ":concept:two-pointers", d, "concept"},
		{d + ":week:3", d + ":week:3", d, "week"},
		{d + ":dashboard", d + ":dashboard", d, "dashboard"},
		{coursetest.FixtureActive + ":week:3", coursetest.FixtureActive + ":week:3", coursetest.FixtureActive, "week"},
		{coursetest.FixtureActive + ":mistakes", coursetest.FixtureActive + ":mistakes", coursetest.FixtureActive, "mistakes"},
		// Items (global ids) and account-wide contexts are untouched and carry no course.
		{"problem:16", "problem:16", "", "problem"},
		{"problem:gc-003", "problem:gc-003", "", "problem"},
		{"catalog", "catalog", "", "catalog"},
		{"settings", "settings", "", "settings"},
		{"general", "general", "", "general"},
		// Unrecognised contexts pass through unchanged.
		{"nope", "nope", "", ""},
		{"unknown-course:week:3", "unknown-course:week:3", "", ""},
		{d + ":bogus", d + ":bogus", "", ""},
		{"week:", "week:", "", ""},
		{"problem:", "problem:", "", ""},
		{"", "", "", ""},
	}
	for _, c := range cases {
		got := r.NormalizeCoachContext(c.in)
		if got.Key != c.key || got.PathSlug != c.path || got.Kind != c.kind {
			t.Errorf("NormalizeCoachContext(%q) = %+v, want key=%q path=%q kind=%q", c.in, got, c.key, c.path, c.kind)
		}
		if again := r.NormalizeCoachContext(got.Key); again.Key != got.Key || again.PathSlug != got.PathSlug {
			t.Errorf("not idempotent: %q → %q → %q", c.in, got.Key, again.Key)
		}
	}
	// Two courses' week 3 get different keys (separate threads).
	a := r.NormalizeCoachContext(d + ":week:3")
	b := r.NormalizeCoachContext(coursetest.FixtureActive + ":week:3")
	if a.Key == b.Key {
		t.Fatalf("dsa and %s week 3 share key %q", coursetest.FixtureActive, a.Key)
	}
	// The package-level parser uses the embedded registry.
	if got := course.NormalizeCoachContext("week:3"); got.Key != d+":week:3" {
		t.Fatalf("package parser: %+v", got)
	}
}
