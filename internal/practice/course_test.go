package practice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

// m1-03 (M1b): the three guided-flow writes take the item's course as the optional
// internal `?path=<slug>`. Absent → course.DefaultSlug (a v1.6.0 gateway); a known slug
// → that course; an unknown one → 404 course_not_found before the store is touched.
func TestPathParam(t *testing.T) {
	var gotPath string
	calls := 0
	fs := &fakeStore{
		start: func(_ context.Context, _, _, path string) (store.State, error) {
			calls++
			gotPath = path
			return store.State{ProblemID: "zz-1", Status: "attempting"}, nil
		},
		reveal: func(context.Context, string, string) (store.RevealResult, error) {
			calls++
			return store.RevealResult{Revealed: store.StageHint, State: store.State{ProblemID: "zz-1"}}, nil
		},
		logOutcome: func(context.Context, string, string, string) (store.State, error) {
			calls++
			return store.State{ProblemID: "zz-1", Status: "solved"}, nil
		},
	}
	h := NewService(fs, fakeVerifier{subject: testAccount}, testLogger()).
		WithCourses(coursetest.Registry(t)).Handler()

	for _, c := range []struct {
		query, wantPath string
	}{
		{"", course.DefaultSlug},
		{"?path=", course.DefaultSlug},
		{"?path=" + course.DefaultSlug, course.DefaultSlug},
		{"?path=" + coursetest.FixtureActive, coursetest.FixtureActive},
	} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, authedReq(http.MethodPost, "/problems/zz-1/attempt/start"+c.query, ""))
		if rr.Code != http.StatusOK || gotPath != c.wantPath {
			t.Errorf("start%s: status %d path %q, want 200 %q (%s)", c.query, rr.Code, gotPath, c.wantPath, rr.Body.String())
		}
		// reveal and outcome only validate the course (the events carry the row's).
		for _, target := range []string{"/problems/zz-1/reveal", "/problems/zz-1/outcome"} {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, authedReq(http.MethodPost, target+c.query, `{"outcome":"clean"}`))
			if rr.Code != http.StatusOK {
				t.Errorf("%s%s: status %d, want 200 (%s)", target, c.query, rr.Code, rr.Body.String())
			}
		}
	}

	// An unknown course is a 404 course_not_found on every write, and the store is
	// never called.
	calls = 0
	for _, target := range []string{"/problems/zz-1/attempt/start", "/problems/zz-1/reveal", "/problems/zz-1/outcome"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, authedReq(http.MethodPost, target+"?path=no-such-course", `{"outcome":"clean"}`))
		if rr.Code != http.StatusNotFound {
			t.Fatalf("%s?path=no-such-course: status %d, want 404 (%s)", target, rr.Code, rr.Body.String())
		}
		if code := decodeBody(t, rr)["error"].(map[string]any)["code"]; code != "course_not_found" {
			t.Errorf("%s: error code %v, want course_not_found", target, code)
		}
	}
	if calls != 0 {
		t.Errorf("the store was called %d times for an unknown course", calls)
	}

	// Production resolves against the embedded manifests only: a fixture course is
	// unknown there.
	prod := NewService(fs, fakeVerifier{subject: testAccount}, testLogger()).Handler()
	rr := httptest.NewRecorder()
	prod.ServeHTTP(rr, authedReq(http.MethodPost, "/problems/zz-1/attempt/start?path="+coursetest.FixtureActive, ""))
	if rr.Code != http.StatusNotFound {
		t.Errorf("embedded registry: fixture course status %d, want 404", rr.Code)
	}
}
