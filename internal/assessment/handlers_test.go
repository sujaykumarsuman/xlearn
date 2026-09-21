package assessment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
)

const testAccount = "11111111-1111-1111-1111-111111111111"

// newTestService builds a Service over a fake store, accepting any bearer token as
// testAccount.
func newTestService(fs *fakeStore) *Service {
	return NewService(fs, fakeVerifier{subject: testAccount}, testLogger())
}

func do(t *testing.T, h http.Handler, method, path, body string, auth bool) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	if auth {
		r.Header.Set("Authorization", "Bearer test")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func liveSession() store.MockSession {
	now := time.Now()
	return store.MockSession{
		ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", AccountID: testAccount,
		SetID: "set-07", ProblemID: "16", Difficulty: "med", Date: now,
		Status: store.StatusLive, StartedAt: now, DeadlineAt: now.Add(store.MockDuration),
	}
}

func fullRubric() map[string]int {
	m := map[string]int{}
	for _, d := range store.Dimensions {
		m[d] = 3
	}
	m["communication"] = 4
	return m
}

func TestStartMock(t *testing.T) {
	var gotSet, gotProblem, gotDiff string
	fs := &fakeStore{createMock: func(_ context.Context, acct, setID, problemID, difficulty string, started, deadline time.Time) (store.MockSession, error) {
		gotSet, gotProblem, gotDiff = setID, problemID, difficulty
		if acct != testAccount {
			t.Fatalf("account = %q", acct)
		}
		if deadline.Sub(started) != store.MockDuration {
			t.Fatalf("window = %v, want 45m", deadline.Sub(started))
		}
		m := liveSession()
		m.SetID, m.ProblemID, m.Difficulty = setID, problemID, difficulty
		m.StartedAt, m.DeadlineAt = started, deadline
		return m, nil
	}}
	h := newTestService(fs).Handler()

	w := do(t, h, http.MethodPost, "/mocks", `{"setId":"set-07","problemId":"16","difficulty":"med"}`, true)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	if gotSet != "set-07" || gotProblem != "16" || gotDiff != "med" {
		t.Fatalf("store got set=%q problem=%q diff=%q", gotSet, gotProblem, gotDiff)
	}
	var view mockViewJSON
	mustJSON(t, w, &view)
	if view.Status != store.StatusLive {
		t.Fatalf("status = %q, want live", view.Status)
	}
	if view.Rail.PhaseIndex != 0 || view.Rail.RemainingSeconds > 2700 || len(view.Rail.Phases) != 6 {
		t.Fatalf("bad rail: %+v", view.Rail)
	}
	if len(view.Dimensions) != 0 {
		t.Fatalf("live session should have no dimensions, got %d", len(view.Dimensions))
	}
	if view.Targets.W13 != 24 || view.Targets.W15 != 28 || view.Targets.Pre != 30 {
		t.Fatalf("targets = %+v", view.Targets)
	}
}

func TestStartMockValidation(t *testing.T) {
	fs := &fakeStore{createMock: func(context.Context, string, string, string, string, time.Time, time.Time) (store.MockSession, error) {
		t.Fatal("store must not be called on invalid setup")
		return store.MockSession{}, nil
	}}
	h := newTestService(fs).Handler()

	cases := []struct {
		name, body string
		want       int
	}{
		{"missing setId", `{"difficulty":"med"}`, http.StatusUnprocessableEntity},
		{"bad difficulty", `{"setId":"s","difficulty":"impossible"}`, http.StatusUnprocessableEntity},
		{"unknown field", `{"setId":"s","bogus":1}`, http.StatusBadRequest},
		{"not json", `nope`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := do(t, h, http.MethodPost, "/mocks", tc.body, true)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d (%s)", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

func TestStartMockDefaultsDifficulty(t *testing.T) {
	fs := &fakeStore{createMock: func(_ context.Context, _, _, _, difficulty string, s, d time.Time) (store.MockSession, error) {
		if difficulty != "med" {
			t.Fatalf("difficulty = %q, want defaulted med", difficulty)
		}
		m := liveSession()
		return m, nil
	}}
	h := newTestService(fs).Handler()
	w := do(t, h, http.MethodPost, "/mocks", `{"setId":"set-07"}`, true)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d (%s)", w.Code, w.Body.String())
	}
}

func TestMockRequiresAuth(t *testing.T) {
	fs := &fakeStore{}
	svc := NewService(fs, fakeVerifier{err: context.Canceled}, testLogger())
	h := svc.Handler()
	for _, p := range []struct{ method, path string }{
		{http.MethodPost, "/mocks"},
		{http.MethodGet, "/mocks/abc"},
		{http.MethodGet, "/mocks/trend"},
		{http.MethodPost, "/mocks/abc/score"},
	} {
		w := do(t, h, p.method, p.path, `{}`, false) // no bearer
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status = %d, want 401", p.method, p.path, w.Code)
		}
	}
}

func TestGetMockNotFound(t *testing.T) {
	fs := &fakeStore{getMock: func(context.Context, string, string) (store.MockSession, []store.RubricScore, error) {
		return store.MockSession{}, nil, store.ErrNotFound
	}}
	h := newTestService(fs).Handler()
	w := do(t, h, http.MethodGet, "/mocks/deadbeef", "", true)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestGetScoredMockReturnsRubric(t *testing.T) {
	total := 24
	fs := &fakeStore{getMock: func(_ context.Context, _, id string) (store.MockSession, []store.RubricScore, error) {
		m := liveSession()
		m.ID = id
		m.Status = store.StatusScored
		m.Total35 = &total
		scores := []store.RubricScore{
			{Dimension: "communication", Score: 4},
			{Dimension: "complexity", Score: 3},
		}
		return m, scores, nil
	}}
	h := newTestService(fs).Handler()
	w := do(t, h, http.MethodGet, "/mocks/aaaa", "", true)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", w.Code, w.Body.String())
	}
	var view mockViewJSON
	mustJSON(t, w, &view)
	if view.Status != store.StatusScored || view.Total35 == nil || *view.Total35 != 24 {
		t.Fatalf("bad scored view: %+v", view)
	}
	// Dimensions are emitted for all seven in canonical order (even those absent from
	// the sparse store return default to 0), so the client renders a full radar.
	if len(view.Dimensions) != 7 {
		t.Fatalf("dimensions = %d, want 7", len(view.Dimensions))
	}
	if view.Dimensions[0].Key != "communication" || view.Dimensions[0].Name != "Communication" || view.Dimensions[0].Score != 4 {
		t.Fatalf("first dim = %+v", view.Dimensions[0])
	}
}

func TestScoreMock(t *testing.T) {
	var gotScores map[string]int
	var gotNotes string
	total := 24
	fs := &fakeStore{scoreMock: func(_ context.Context, acct, id string, scores map[string]int, notes string) (store.MockSession, []store.RubricScore, error) {
		gotScores, gotNotes = scores, notes
		if acct != testAccount {
			t.Fatalf("account = %q", acct)
		}
		m := liveSession()
		m.ID = id
		m.Status = store.StatusScored
		m.Total35 = &total
		out := make([]store.RubricScore, 0, len(scores))
		for _, d := range store.Dimensions {
			out = append(out, store.RubricScore{Dimension: d, Score: scores[d]})
		}
		return m, out, nil
	}}
	h := newTestService(fs).Handler()

	body, _ := json.Marshal(map[string]any{"scores": fullRubric(), "notes": "  lost time on the two-pointer insight  "})
	w := do(t, h, http.MethodPost, "/mocks/aaaa/score", string(body), true)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", w.Code, w.Body.String())
	}
	if len(gotScores) != 7 {
		t.Fatalf("store got %d scores, want 7", len(gotScores))
	}
	if gotNotes != "lost time on the two-pointer insight" {
		t.Fatalf("notes = %q (should be trimmed)", gotNotes)
	}
	var view mockViewJSON
	mustJSON(t, w, &view)
	if *view.Total35 != 24 || view.Status != store.StatusScored {
		t.Fatalf("bad view: %+v", view)
	}
}

func TestScoreMockRejectsBadRubric(t *testing.T) {
	fs := &fakeStore{scoreMock: func(context.Context, string, string, map[string]int, string) (store.MockSession, []store.RubricScore, error) {
		t.Fatal("store must not be called on an invalid rubric")
		return store.MockSession{}, nil, nil
	}}
	h := newTestService(fs).Handler()

	cases := map[string]string{
		"missing dim":  `{"scores":{"communication":4}}`,
		"out of range": scoresBody(t, "communication", 6),
		"empty":        `{"scores":{}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			w := do(t, h, http.MethodPost, "/mocks/aaaa/score", body, true)
			if w.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422 (%s)", w.Code, w.Body.String())
			}
		})
	}
}

func TestTrend(t *testing.T) {
	fs := &fakeStore{trend: func(_ context.Context, acct string) ([]store.TrendPoint, error) {
		if acct != testAccount {
			t.Fatalf("account = %q", acct)
		}
		now := time.Now()
		return []store.TrendPoint{
			{MockID: "m1", SetID: "set-05", Date: now.AddDate(0, 0, -14), StartedAt: now.AddDate(0, 0, -14), Total35: 20},
			{MockID: "m2", SetID: "set-07", Date: now, StartedAt: now, Total35: 24},
		}, nil
	}}
	h := newTestService(fs).Handler()
	// Must route to the trend handler, not GET /mocks/{id}.
	w := do(t, h, http.MethodGet, "/mocks/trend", "", true)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", w.Code, w.Body.String())
	}
	var resp struct {
		Points  []trendPointJSON `json:"points"`
		Targets Targets          `json:"targets"`
	}
	mustJSON(t, w, &resp)
	if len(resp.Points) != 2 || resp.Points[0].Total35 != 20 || resp.Points[1].Total35 != 24 {
		t.Fatalf("points = %+v", resp.Points)
	}
	if resp.Targets.W13 != 24 || resp.Targets.W15 != 28 || resp.Targets.Pre != 30 {
		t.Fatalf("targets = %+v", resp.Targets)
	}
}

func TestHealthAndReady(t *testing.T) {
	fs := &fakeStore{}
	h := newTestService(fs).Handler()
	for _, p := range []string{"/healthz", "/readyz"} {
		w := do(t, h, http.MethodGet, p, "", false)
		if w.Code != http.StatusOK {
			t.Fatalf("%s = %d, want 200", p, w.Code)
		}
	}
}

// --- helpers ---

func mustJSON(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("decode response: %v (%s)", err, w.Body.String())
	}
}

func scoresBody(t *testing.T, overrideKey string, overrideVal int) string {
	t.Helper()
	m := fullRubric()
	m[overrideKey] = overrideVal
	b, _ := json.Marshal(map[string]any{"scores": m})
	return string(b)
}
