package assessment

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// m1-03 (M1b): POST /mocks, the trend and the progress reads take the optional internal
// `?path=<slug>`: absent → course.DefaultSlug (a v1.6.0 gateway), a known slug → that
// course, an unknown one → 404 course_not_found before the store is touched. A course
// with no mock block refuses POST /mocks with 404 not_found "course has no mock". Only
// the mock aggregate of /progress/summary is per course; the projection reads stay
// account-grain until M2b. A session by id ignores `path`.
func TestPathParam(t *testing.T) {
	var gotPath string
	calls := 0
	record := func(p string) { calls++; gotPath = p }
	fs := &fakeStore{
		createMock: func(_ context.Context, _, p, _, _, _ string, s, d time.Time) (store.MockSession, error) {
			record(p)
			m := liveSession()
			m.PathSlug, m.StartedAt, m.DeadlineAt = p, s, d
			return m, nil
		},
		getMock: func(context.Context, string, string) (store.MockSession, []store.RubricScore, error) {
			calls++
			return liveSession(), nil, nil
		},
		trend: func(_ context.Context, _, p string) ([]store.TrendPoint, error) {
			record(p)
			return nil, nil
		},
		mockStats: func(_ context.Context, _, p string) (store.MockStats, error) {
			record(p)
			return store.MockStats{}, nil
		},
		solvedCount: func(context.Context, string) (int, error) { calls++; return 0, nil },
		retention:   func(context.Context, string) (int, int, error) { return 0, 0, nil },
		outcomeMix:  func(context.Context, string) (map[string]int, error) { return nil, nil },
		heatmap: func(context.Context, string, time.Time) ([]store.HeatmapDay, error) {
			calls++
			return nil, nil
		},
		mastery: func(context.Context, string) ([]store.ProblemMastery, error) {
			calls++
			return nil, nil
		},
	}
	h := newTestService(fs).WithCourses(coursetest.Registry(t)).Handler()

	// POST /mocks: the default course, and a fixture course with a mock (zz-preview:
	// the internal service leaves visibility to the gateway).
	for _, c := range []struct{ query, want string }{
		{"", course.DefaultSlug},
		{"?path=" + course.DefaultSlug, course.DefaultSlug},
		{"?path=" + coursetest.FixturePreview, coursetest.FixturePreview},
	} {
		gotPath = ""
		w := do(t, h, http.MethodPost, "/mocks"+c.query, `{"setId":"set-07"}`, true)
		if w.Code != http.StatusCreated || gotPath != c.want {
			t.Errorf("POST /mocks%s: status %d path %q, want 201 %q (%s)", c.query, w.Code, gotPath, c.want, w.Body.String())
		}
	}

	// The trend and the summary's mock aggregate are per course.
	for _, target := range []string{"/mocks/trend", "/progress/summary"} {
		for _, c := range []struct{ query, want string }{
			{"", course.DefaultSlug},
			{"?path=" + coursetest.FixtureActive, coursetest.FixtureActive},
		} {
			gotPath = ""
			if w := do(t, h, http.MethodGet, target+c.query, "", true); w.Code != http.StatusOK || gotPath != c.want {
				t.Errorf("GET %s%s: status %d path %q, want 200 %q", target, c.query, w.Code, gotPath, c.want)
			}
		}
	}
	// heatmap and mastery accept a known course (account-grain reads).
	for _, target := range []string{"/progress/heatmap", "/progress/mastery"} {
		if w := do(t, h, http.MethodGet, target+"?path="+coursetest.FixtureActive, "", true); w.Code != http.StatusOK {
			t.Errorf("GET %s?path=%s: status %d, want 200", target, coursetest.FixtureActive, w.Code)
		}
	}

	// A course without a mock block: 404 not_found "course has no mock".
	w := do(t, h, http.MethodPost, "/mocks?path="+coursetest.FixtureActive, `{"setId":"set-07"}`, true)
	if w.Code != http.StatusNotFound || errorOf(t, w.Body.Bytes()) != [2]string{"not_found", "course has no mock"} {
		t.Errorf("POST /mocks in a course without a mock: status %d body %s", w.Code, w.Body.String())
	}

	// An unknown course: 404 course_not_found everywhere, the store untouched.
	calls = 0
	for _, r := range []struct{ method, path, body string }{
		{http.MethodPost, "/mocks", `{"setId":"set-07"}`},
		{http.MethodGet, "/mocks/trend", ""},
		{http.MethodGet, "/progress/summary", ""},
		{http.MethodGet, "/progress/heatmap", ""},
		{http.MethodGet, "/progress/mastery", ""},
	} {
		w := do(t, h, r.method, r.path+"?path=no-such-course", r.body, true)
		if w.Code != http.StatusNotFound || errorOf(t, w.Body.Bytes())[0] != "course_not_found" {
			t.Errorf("%s %s?path=no-such-course: status %d body %s, want 404 course_not_found", r.method, r.path, w.Code, w.Body.String())
		}
	}
	if calls != 0 {
		t.Errorf("the store was called %d times for an unknown course", calls)
	}

	// A session by id is not course-scoped: `path` is not read.
	if w := do(t, h, http.MethodGet, "/mocks/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa?path=no-such-course", "", true); w.Code != http.StatusOK {
		t.Errorf("GET /mocks/{id}?path=no-such-course: status %d, want 200", w.Code)
	}
}

// errorOf returns an error envelope's code and message.
func errorOf(t *testing.T, body []byte) [2]string {
	t.Helper()
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode error envelope: %v (%s)", err, body)
	}
	return [2]string{env.Error.Code, env.Error.Message}
}
