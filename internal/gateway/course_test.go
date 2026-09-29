package gateway

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// courseHarness wires a real gateway (with the test-only course registry: the embedded
// courses plus the zz-* fixtures, one per status) to recording fakes of every upstream,
// so m1-03's course resolution can be exercised route by route: resolution + 404s, the
// DSA aliases' byte parity, the course reaching every downstream call, the catalog
// filter, enrollment validation, item routes by global id, slugged cache names and the
// coach context parser. The fakes accept any bearer (mint→JWKS is covered elsewhere) and
// answer deterministically from the request (path + query), so two calls that reach the
// same upstream URLs get byte-identical bodies.
type courseHarness struct {
	t        *testing.T
	gwServer *httptest.Server

	mu       sync.Mutex
	calls    []string // "<svc> <METHOD> <path>?<query>" for every upstream call
	enrolled []string // the courses acct-1 has started
	started  []string // slugs identity's POST /paths/{slug}/start saw
	coachCtx []string // contexts coach's /threads and /chat saw
	coachQ   []string // raw queries coach's /chat saw
}

func (h *courseHarness) record(svc string, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	u := r.URL.Path
	if r.URL.RawQuery != "" {
		u += "?" + r.URL.RawQuery
	}
	h.calls = append(h.calls, svc+" "+r.Method+" "+u)
}

func (h *courseHarness) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls, h.started, h.coachCtx, h.coachQ = nil, nil, nil, nil
}

func (h *courseHarness) called(sub string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, c := range h.calls {
		if strings.Contains(c, sub) {
			return true
		}
	}
	return false
}

func (h *courseHarness) countCalls(sub string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, c := range h.calls {
		if strings.Contains(c, sub) {
			n++
		}
	}
	return n
}

// itemCourseFor is the fakes' item → course map: zz-NNN is zz-fixture's, zzp-NNN
// zz-preview's, everything else DSA's.
func itemCourseFor(id string) string {
	switch {
	case strings.HasPrefix(id, "zzp-"):
		return coursetest.FixturePreview
	case strings.HasPrefix(id, "zz-"):
		return coursetest.FixtureActive
	default:
		return course.DefaultSlug
	}
}

func newCourseHarness(t *testing.T, cacheTTL time.Duration) *courseHarness {
	t.Helper()
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer := auth.NewSigner(key, "xlearn-gateway", 0)
	h := &courseHarness{t: t, enrolled: []string{course.DefaultSlug, coursetest.FixtureActive}}

	echo := func(svc string) handlerFn {
		return func(w http.ResponseWriter, r *http.Request) {
			h.record(svc, r)
			b, _ := io.ReadAll(r.Body)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"svc": svc, "path": r.URL.Path, "course": r.URL.Query().Get("path"), "body": string(b),
			})
		}
	}

	identity := httptest.NewServer(jsonMux(map[string]handlerFn{
		"POST /sessions/validate": func(w http.ResponseWriter, r *http.Request) {
			var b struct {
				SessionID string `json:"session_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b.SessionID != "sess-1" {
				w.WriteHeader(401)
				_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"account_id": "acct-1"})
		},
		"GET /accounts/{id}": func(w http.ResponseWriter, r *http.Request) {
			h.mu.Lock()
			var ens []any
			for _, s := range h.enrolled {
				ens = append(ens, map[string]any{"path_slug": s, "status": "active", "started_at": "2026-09-20T00:00:00Z"})
			}
			h.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"account": map[string]any{"id": r.PathValue("id")}, "enrollments": ens})
		},
		"POST /paths/{slug}/start": func(w http.ResponseWriter, r *http.Request) {
			h.mu.Lock()
			h.started = append(h.started, r.PathValue("slug"))
			h.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"enrollment": map[string]any{"path_slug": r.PathValue("slug")}})
		},
	}))
	t.Cleanup(identity.Close)

	curriculum := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /paths": func(w http.ResponseWriter, r *http.Request) {
			h.record("curriculum", r)
			var ps []any
			for _, s := range []string{course.DefaultSlug, coursetest.FixtureActive, coursetest.FixtureComingSoon, coursetest.FixturePreview, coursetest.FixtureRetired, "ghost"} {
				ps = append(ps, map[string]any{"slug": s, "title": "T " + s, "course": map[string]any{"slug": s}})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"paths": ps})
		},
		"GET /paths/{slug}": func(w http.ResponseWriter, r *http.Request) {
			h.record("curriculum", r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path":   map[string]any{"slug": r.PathValue("slug"), "problem_total": 2},
				"phases": []any{map[string]any{"order": 1, "name": "P1", "week_from": 1, "week_to": 2}},
				"weeks":  []any{map[string]any{"n": 1, "title": "W1"}},
			})
		},
		"GET /paths/{slug}/problems": func(w http.ResponseWriter, r *http.Request) {
			h.record("curriculum", r)
			id := "16"
			if r.PathValue("slug") == coursetest.FixtureActive {
				id = "zz-001"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"problems": []any{
				map[string]any{"id": id, "path_slug": r.PathValue("slug"), "week_n": 1, "title": "P " + id, "difficulty": "med", "pattern": "Two pointers"},
			}})
		},
		"GET /paths/{slug}/weeks/{n}": func(w http.ResponseWriter, r *http.Request) {
			h.record("curriculum", r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"week": map[string]any{"n": 1, "title": "W1"}, "path": map[string]any{"slug": r.PathValue("slug")},
				"problems": []any{},
			})
		},
		"GET /paths/{slug}/concepts/{c}": func(w http.ResponseWriter, r *http.Request) {
			h.record("curriculum", r)
			_ = json.NewEncoder(w).Encode(map[string]any{"concept": map[string]any{"slug": r.PathValue("c"), "path_slug": r.PathValue("slug")}})
		},
		"GET /problems/{id}": func(w http.ResponseWriter, r *http.Request) {
			h.record("curriculum", r)
			id := r.PathValue("id")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"problem":  map[string]any{"id": id, "path_slug": itemCourseFor(id), "week_n": 1, "title": "P " + id, "pattern": "Two pointers"},
				"sections": []any{map[string]any{"stage": "attempt", "kind": "summary", "order": 1, "body_md": "s"}},
			})
		},
		"GET /problems": func(w http.ResponseWriter, r *http.Request) {
			h.record("curriculum", r)
			var ps []any
			for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
				if id != "" {
					ps = append(ps, map[string]any{"id": id, "path_slug": itemCourseFor(id), "title": "P " + id})
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"problems": ps})
		},
	}))
	t.Cleanup(curriculum.Close)

	review := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /revisions/due": func(w http.ResponseWriter, r *http.Request) {
			h.record("review", r)
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{
				map[string]any{"itemId": "r-" + r.URL.Query().Get("path"), "problemId": "16", "touchLevel": 1, "dayLabel": "Day 1", "due": true},
			}, "dueCount": 1})
		},
		"GET /mistakes": func(w http.ResponseWriter, r *http.Request) {
			h.record("review", r)
			_ = json.NewEncoder(w).Encode(map[string]any{"mistakes": []any{
				map[string]any{"id": "m-" + r.URL.Query().Get("path"), "problemId": "16", "status": r.URL.Query().Get("status")},
			}, "openCount": 1})
		},
		"POST /mistakes":         echo("review"),
		"GET /weak-area/current": echo("review"),
		"GET /reminders": func(w http.ResponseWriter, r *http.Request) {
			h.record("review", r)
			_, _ = w.Write([]byte(`{"reminders":[]}`))
		},
	}))
	t.Cleanup(review.Close)

	assessment := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /progress/summary": func(w http.ResponseWriter, r *http.Request) {
			h.record("assessment", r)
			_ = json.NewEncoder(w).Encode(map[string]any{"solved": 1, "total": 151, "streak": map[string]any{"current": 2}, "mock": map[string]any{"best": 20}})
		},
		"GET /progress/heatmap": echo("assessment"),
		"GET /progress/mastery": func(w http.ResponseWriter, r *http.Request) {
			h.record("assessment", r)
			_ = json.NewEncoder(w).Encode(map[string]any{"problems": []any{}})
		},
		"GET /mocks/trend": echo("assessment"),
		"POST /mocks":      echo("assessment"),
	}))
	t.Cleanup(assessment.Close)

	practice := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /state": func(w http.ResponseWriter, r *http.Request) {
			h.record("practice", r)
			_, _ = w.Write([]byte(`{"states":{}}`))
		},
		"GET /state/{id}": func(w http.ResponseWriter, r *http.Request) {
			h.record("practice", r)
			_ = json.NewEncoder(w).Encode(map[string]any{"state": map[string]any{"problemId": r.PathValue("id"), "status": "available", "unlockedStages": []string{"attempt"}}})
		},
		"POST /problems/{id}/attempt/start": echo("practice"),
		"POST /problems/{id}/reveal":        echo("practice"),
		"POST /problems/{id}/outcome":       echo("practice"),
	}))
	t.Cleanup(practice.Close)

	coach := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /threads": func(w http.ResponseWriter, r *http.Request) {
			h.record("coach", r)
			h.mu.Lock()
			h.coachCtx = append(h.coachCtx, r.URL.Query().Get("context"))
			h.mu.Unlock()
			_, _ = w.Write([]byte(`{"messages":[]}`))
		},
		"POST /chat": func(w http.ResponseWriter, r *http.Request) {
			h.record("coach", r)
			var b struct {
				Context string `json:"context"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			h.mu.Lock()
			h.coachCtx = append(h.coachCtx, b.Context)
			h.coachQ = append(h.coachQ, r.URL.RawQuery)
			h.mu.Unlock()
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: done\ndata: {}\n\n"))
		},
	}))
	t.Cleanup(coach.Close)

	gw := New(Options{
		BasePath: "/xlearn", Version: "test", Signer: signer,
		Dist:            fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}},
		IdentityBaseURL: identity.URL, AudienceIdentity: "identity",
		CurriculumBaseURL: curriculum.URL,
		PracticeBaseURL:   practice.URL, AudiencePractice: "practice",
		ReviewBaseURL: review.URL, AudienceReview: "review",
		AssessmentBaseURL: assessment.URL, AudienceAssessment: "assessment",
		CoachBaseURL: coach.URL, AudienceCoach: "coach",
		Courses:     coursetest.Registry(t),
		AggCacheTTL: cacheTTL,
	})
	h.gwServer = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gwServer.Close)
	return h
}

// do issues an authenticated request and returns the status and body.
func (h *courseHarness) do(method, path, body string) (int, []byte) {
	h.t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, h.gwServer.URL+"/xlearn/api/v1"+path, rdr)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	if method != http.MethodGet {
		// Like the SPA: every write is JSON, with or without a body (security.go).
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func errorCode(b []byte) string {
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(b, &env)
	return env.Error.Code
}

// courseScopedRoutes are every /api/paths/{slug}/… route (the suffix after the slug).
var courseScopedRoutes = []struct{ method, suffix, body string }{
	{"GET", "", ""},
	{"GET", "/problems", ""},
	{"GET", "/weeks/1", ""},
	{"GET", "/concepts/two-pointers", ""},
	{"GET", "/dashboard", ""},
	{"GET", "/progress", ""},
	{"GET", "/revision/due", ""},
	{"GET", "/mistakes", ""},
	{"POST", "/mistakes", `{"problemId":"16","category":"misread"}`},
	{"GET", "/weak-area", ""},
	{"POST", "/mocks", `{"setId":"s1","difficulty":"med"}`},
	{"GET", "/mocks/trend", ""},
}

// Every course-scoped route serves an active course (DSA and a second course) and
// returns the SAME 404 course_not_found for an unknown, retired, preview (outside the
// cohort) or coming_soon course — without calling any upstream.
func TestCourseScopedRoutesResolve(t *testing.T) {
	h := newCourseHarness(t, 0)
	for _, rt := range courseScopedRoutes {
		for _, slug := range []string{course.DefaultSlug, coursetest.FixtureActive} {
			status, body := h.do(rt.method, "/paths/"+slug+rt.suffix, rt.body)
			if status/100 != 2 {
				t.Errorf("%s /paths/%s%s: status %d (%s), want 2xx", rt.method, slug, rt.suffix, status, body)
			}
		}
		var first []byte
		for _, slug := range []string{"nope", coursetest.FixtureRetired, coursetest.FixturePreview, coursetest.FixtureComingSoon} {
			h.reset()
			status, body := h.do(rt.method, "/paths/"+slug+rt.suffix, rt.body)
			if status != http.StatusNotFound || errorCode(body) != "course_not_found" {
				t.Errorf("%s /paths/%s%s: %d %s, want 404 course_not_found", rt.method, slug, rt.suffix, status, body)
			}
			if first == nil {
				first = body
			} else if !bytes.Equal(first, body) {
				t.Errorf("%s /paths/%s%s: body %s differs from the unknown course's %s (no hint a hidden course exists)", rt.method, slug, rt.suffix, body, first)
			}
			for _, svc := range []string{"curriculum", "review", "assessment", "practice"} {
				if h.called(svc + " ") {
					t.Errorf("%s /paths/%s%s: %s was called for a hidden course: %v", rt.method, slug, rt.suffix, svc, h.calls)
				}
			}
		}
	}
}

// Every DSA alias returns a body byte-identical to its /api/paths/<DefaultSlug>/… twin
// against the same upstream state.
func TestAliasByteParity(t *testing.T) {
	h := newCourseHarness(t, 0)
	d := "/paths/" + course.DefaultSlug
	pairs := []struct{ method, alias, scoped, body string }{
		{"GET", "/dashboard", d + "/dashboard", ""},
		{"GET", "/progress", d + "/progress", ""},
		{"GET", "/revision/due", d + "/revision/due", ""},
		{"GET", "/mistakes", d + "/mistakes", ""},
		{"GET", "/mistakes?status=open", d + "/mistakes?status=open", ""},
		{"POST", "/mistakes", d + "/mistakes", `{"problemId":"16","category":"misread"}`},
		{"GET", "/weak-area", d + "/weak-area", ""},
		{"POST", "/mocks", d + "/mocks", `{"setId":"s1","difficulty":"med"}`},
		{"GET", "/mocks/trend", d + "/mocks/trend", ""},
		{"GET", "/concepts/two-pointers", d + "/concepts/two-pointers", ""},
	}
	for _, p := range pairs {
		as, ab := h.do(p.method, p.alias, p.body)
		ss, sb := h.do(p.method, p.scoped, p.body)
		if as/100 != 2 || as != ss {
			t.Errorf("%s %s: alias status %d, scoped %d", p.method, p.alias, as, ss)
		}
		if !bytes.Equal(ab, sb) {
			t.Errorf("%s %s: alias body differs from %s:\nalias:  %s\nscoped: %s", p.method, p.alias, p.scoped, ab, sb)
		}
	}
}

// A course-scoped route carries its course into every downstream call (?path= for
// practice, review and assessment; the course path for curriculum) and never falls back
// to DSA.
func TestCourseReachesEveryDownstreamCall(t *testing.T) {
	h := newCourseHarness(t, 0)
	fx := coursetest.FixtureActive
	for _, rt := range courseScopedRoutes {
		h.reset()
		if status, body := h.do(rt.method, "/paths/"+fx+rt.suffix, rt.body); status/100 != 2 {
			t.Fatalf("%s %s: %d %s", rt.method, rt.suffix, status, body)
		}
		h.mu.Lock()
		calls := slices.Clone(h.calls)
		h.mu.Unlock()
		for _, c := range calls {
			svc, rest, _ := strings.Cut(c, " ")
			switch svc {
			case "review", "assessment":
				if strings.Contains(rest, "/reminders") {
					continue // account-wide by design
				}
				if !strings.Contains(rest, "path="+fx) {
					t.Errorf("%s %s: %s call without the course: %s", rt.method, rt.suffix, svc, c)
				}
			case "curriculum":
				if strings.Contains(rest, "/paths/") && !strings.Contains(rest, "/paths/"+fx) {
					t.Errorf("%s %s: curriculum call for another course: %s", rt.method, rt.suffix, c)
				}
			}
			if strings.Contains(c, "/paths/"+course.DefaultSlug) || strings.Contains(c, "path="+course.DefaultSlug) {
				t.Errorf("%s %s: a DSA call leaked into another course's route: %s", rt.method, rt.suffix, c)
			}
		}
	}
}

// The catalog lists active and coming_soon courses and drops preview (outside the
// cohort), retired and unknown slugs, keeping curriculum's order and bytes per entry.
func TestCatalogFiltersHiddenCourses(t *testing.T) {
	h := newCourseHarness(t, 0)
	status, body := h.do("GET", "/paths", "")
	if status != 200 {
		t.Fatalf("GET /paths: %d %s", status, body)
	}
	var out struct {
		Paths []struct {
			Slug   string         `json:"slug"`
			Course map[string]any `json:"course"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range out.Paths {
		got = append(got, p.Slug)
		if p.Course == nil {
			t.Errorf("%s: the course block was dropped", p.Slug)
		}
	}
	want := []string{course.DefaultSlug, coursetest.FixtureActive, coursetest.FixtureComingSoon}
	if !slices.Equal(got, want) {
		t.Fatalf("catalog = %v, want %v", got, want)
	}
}

// Enrollment accepts only an active course: unknown/preview → 404 course_not_found,
// coming_soon/retired → 409 course_not_available, and identity isn't called for either.
func TestStartPathValidatesCourse(t *testing.T) {
	h := newCourseHarness(t, 0)
	cases := []struct {
		slug   string
		status int
		code   string
	}{
		{course.DefaultSlug, 200, ""},
		{coursetest.FixtureActive, 200, ""},
		{coursetest.FixtureComingSoon, 409, "course_not_available"},
		{coursetest.FixtureRetired, 409, "course_not_available"},
		{coursetest.FixturePreview, 404, "course_not_found"},
		{"nope", 404, "course_not_found"},
	}
	for _, c := range cases {
		h.reset()
		status, body := h.do("POST", "/paths/"+c.slug+"/start", "")
		if status != c.status || errorCode(body) != c.code {
			t.Errorf("start %s: %d %s, want %d %q", c.slug, status, body, c.status, c.code)
		}
		h.mu.Lock()
		started := slices.Clone(h.started)
		h.mu.Unlock()
		if (c.status == 200) != (len(started) == 1) {
			t.Errorf("start %s: identity saw %v", c.slug, started)
		}
	}
}

// Items keep global ids: the course comes from the item's path_slug. A visible course's
// item works (its writes carry the course and gate on ITS enrollment); a hidden course's
// item is course_not_found.
func TestItemRoutesResolveTheItemsCourse(t *testing.T) {
	h := newCourseHarness(t, 0)
	if status, body := h.do("GET", "/problems/zz-001", ""); status != 200 {
		t.Fatalf("GET zz-001: %d %s", status, body)
	}
	if status, body := h.do("GET", "/problems/zzp-001", ""); status != 404 || errorCode(body) != "course_not_found" {
		t.Fatalf("GET a preview item: %d %s, want 404 course_not_found", status, body)
	}
	for _, w := range []string{"/attempt/start", "/reveal", "/outcome"} {
		h.reset()
		status, body := h.do("POST", "/problems/zz-001"+w, `{"outcome":"clean"}`)
		if status != 200 || !h.called("practice POST /problems/zz-001"+w+"?path="+coursetest.FixtureActive) {
			t.Errorf("POST zz-001%s: %d %s, calls %v", w, status, body, h.calls)
		}
		h.reset()
		status, body = h.do("POST", "/problems/16"+w, `{"outcome":"clean"}`)
		if status != 200 || !h.called("practice POST /problems/16"+w+"?path="+course.DefaultSlug) {
			t.Errorf("POST 16%s: %d %s, calls %v", w, status, body, h.calls)
		}
		if status, body := h.do("POST", "/problems/zzp-001"+w, `{}`); status != 404 || errorCode(body) != "course_not_found" {
			t.Errorf("POST a preview item%s: %d %s", w, status, body)
		}
	}
	// Enrollment is per course: without zz-fixture, its items are gated; DSA's aren't.
	h.mu.Lock()
	h.enrolled = []string{course.DefaultSlug}
	h.mu.Unlock()
	status, body := h.do("POST", "/problems/zz-001/attempt/start", "")
	if status != http.StatusForbidden || errorCode(body) != "not_enrolled" || !strings.Contains(string(body), coursetest.FixtureActive) {
		t.Errorf("not enrolled in zz-fixture: %d %s", status, body)
	}
	if status, body := h.do("POST", "/problems/16/attempt/start", ""); status != 200 {
		t.Errorf("enrolled in DSA: %d %s", status, body)
	}
}

// The aggregation cache is per course: DSA's and a second course's Today are separate
// entries (the cache name carries the slug).
func TestCacheNamesCarryTheCourse(t *testing.T) {
	h := newCourseHarness(t, time.Minute)
	fx := coursetest.FixtureActive
	for i := 0; i < 2; i++ {
		for _, slug := range []string{course.DefaultSlug, fx} {
			if status, body := h.do("GET", "/paths/"+slug+"/dashboard", ""); status != 200 {
				t.Fatalf("dashboard %s: %d %s", slug, status, body)
			}
		}
	}
	// One compose per course; the second round is served from each course's own entry.
	if n := h.countCalls("curriculum GET /paths/" + course.DefaultSlug + "/problems"); n != 1 {
		t.Errorf("DSA dashboard composed %d times, want 1", n)
	}
	if n := h.countCalls("curriculum GET /paths/" + fx + "/problems"); n != 1 {
		t.Errorf("%s dashboard composed %d times, want 1 (it must not reuse DSA's entry)", fx, n)
	}
	// The alias shares DSA's entry.
	if status, _ := h.do("GET", "/dashboard", ""); status != 200 {
		t.Fatal("alias dashboard")
	}
	if n := h.countCalls("curriculum GET /paths/" + course.DefaultSlug + "/problems"); n != 1 {
		t.Errorf("the DSA alias recomposed (%d) instead of sharing the DSA entry", n)
	}
}

// The coach context is normalized with the shared dual parser before it reaches coach:
// v1 forms map to the DSA course, new forms pass through, and a problem context carries
// the item's course as ?path=.
func TestCoachContextNormalized(t *testing.T) {
	h := newCourseHarness(t, 0)
	fx := coursetest.FixtureActive
	threads := map[string]string{
		"week:3":                     course.DefaultSlug + ":week:3",
		"concept:two-pointers":       course.DefaultSlug + ":concept:two-pointers",
		"dashboard":                  course.DefaultSlug + ":dashboard",
		course.DefaultSlug + ":mock": course.DefaultSlug + ":mock",
		fx + ":week:3":               fx + ":week:3",
		"problem:16":                 "problem:16",
		"catalog":                    "catalog",
		"settings":                   "settings",
	}
	for in, want := range threads {
		h.reset()
		if status, body := h.do("GET", "/coach/thread?context="+in, ""); status != 200 {
			t.Fatalf("thread %s: %d %s", in, status, body)
		}
		if len(h.coachCtx) != 1 || h.coachCtx[0] != want {
			t.Errorf("thread %s: coach saw %v, want %s", in, h.coachCtx, want)
		}
	}
	chats := []struct{ ctx, want, query string }{
		{"week:3", course.DefaultSlug + ":week:3", ""},
		{fx + ":week:3", fx + ":week:3", ""},
		{"general", "general", ""},
		{"problem:zz-001", "problem:zz-001", "path=" + fx},
		{"problem:16", "problem:16", "path=" + course.DefaultSlug},
	}
	for _, c := range chats {
		h.reset()
		status, body := h.do("POST", "/coach/chat", `{"context":"`+c.ctx+`","message":"hi"}`)
		if status != 200 {
			t.Fatalf("chat %s: %d %s", c.ctx, status, body)
		}
		if len(h.coachCtx) != 1 || h.coachCtx[0] != c.want || h.coachQ[0] != c.query {
			t.Errorf("chat %s: coach saw context %v query %v, want %s %q", c.ctx, h.coachCtx, h.coachQ, c.want, c.query)
		}
	}
}
