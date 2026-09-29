package review

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// m1-03 (M1b): the course-scoped reads and the journal create take the optional
// internal `?path=<slug>`: absent → course.DefaultSlug (a v1.6.0 gateway), a known slug
// → that course, an unknown one → 404 course_not_found before the store is touched.
// The routes by id (score, patch) and the account-wide reminders ignore it.
func TestPathParam(t *testing.T) {
	var gotPath string
	calls := 0
	record := func(p string) { calls++; gotPath = p }
	st := &fakeStore{
		dueQueue: func(_ context.Context, _, p string, _ int) ([]store.DueItem, error) {
			record(p)
			return nil, nil
		},
		listMistakes: func(_ context.Context, _, p, _ string) ([]store.Mistake, error) {
			record(p)
			return nil, nil
		},
		createMistake: func(_ context.Context, _, p string, in store.MistakeInput) (store.Mistake, error) {
			record(p)
			return store.Mistake{ID: "m", ProblemID: in.ProblemID, Status: store.MistakeOpen}, nil
		},
		weakAreaCurrent: func(_ context.Context, _, p string) (store.WeakArea, bool, error) {
			record(p)
			return store.WeakArea{}, false, nil
		},
		score: func(context.Context, string, string, store.ScoreInput) (store.ScoreResult, error) {
			calls++
			return store.ScoreResult{ItemID: "it", TouchLevel: 1, AutoPass: true}, nil
		},
		updateMistake: func(context.Context, string, string, store.MistakePatch) (store.Mistake, error) {
			calls++
			return store.Mistake{ID: "m", Status: store.MistakeOpen}, nil
		},
		listReminders: func(context.Context, string, int) ([]store.Reminder, error) {
			calls++
			return nil, nil
		},
	}
	h := NewService(st, fakeVerifier{subject: "acct-1"}, slog.New(slog.NewJSONHandler(io.Discard, nil))).
		WithCourses(coursetest.Registry(t)).Handler()

	scoped := []struct{ method, path, body string }{
		{http.MethodGet, "/revisions/due", ""},
		{http.MethodGet, "/mistakes", ""},
		{http.MethodPost, "/mistakes", `{"problemId":"16"}`},
		{http.MethodGet, "/weak-area/current", ""},
	}
	for _, r := range scoped {
		for _, c := range []struct{ query, want string }{
			{"", course.DefaultSlug},
			{"?path=" + course.DefaultSlug, course.DefaultSlug},
			{"?path=" + coursetest.FixtureActive, coursetest.FixtureActive},
		} {
			gotPath = ""
			target := r.path + c.query
			rec, _ := do(t, h, r.method, target, "tok", r.body)
			if rec.Code >= 300 || gotPath != c.want {
				t.Errorf("%s %s: status %d path %q, want 2xx %q", r.method, target, rec.Code, gotPath, c.want)
			}
		}
		// The status filter still composes with the course.
		if r.path == "/mistakes" && r.method == http.MethodGet {
			if rec, _ := do(t, h, r.method, "/mistakes?path="+coursetest.FixtureActive+"&status=open", "tok", ""); rec.Code != http.StatusOK || gotPath != coursetest.FixtureActive {
				t.Errorf("GET /mistakes?path=&status=: status %d path %q", rec.Code, gotPath)
			}
		}
	}

	calls = 0
	for _, r := range scoped {
		rec, body := do(t, h, r.method, r.path+"?path=no-such-course", "tok", r.body)
		errObj, _ := body["error"].(map[string]any)
		if rec.Code != http.StatusNotFound || errObj["code"] != "course_not_found" {
			t.Errorf("%s %s?path=no-such-course: status %d body %v, want 404 course_not_found", r.method, r.path, rec.Code, body)
		}
	}
	if calls != 0 {
		t.Errorf("the store was called %d times for an unknown course", calls)
	}

	// By id / account-wide: `path` is not read, so even an unknown one is ignored.
	for _, r := range []struct{ method, path, body string }{
		{http.MethodPost, "/revisions/it/score?path=no-such-course", `{"namedPatternSecs":1,"solvedInTimer":true,"statedComplexity":true}`},
		{http.MethodPatch, "/mistakes/m?path=no-such-course", `{"insight":"x"}`},
		{http.MethodGet, "/reminders?path=no-such-course", ""},
	} {
		if rec, _ := do(t, h, r.method, r.path, "tok", r.body); rec.Code != http.StatusOK {
			t.Errorf("%s %s: status %d, want 200 (not course-scoped)", r.method, r.path, rec.Code)
		}
	}
}
