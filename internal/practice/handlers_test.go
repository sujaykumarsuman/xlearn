package practice

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

func testLogger() *slog.Logger { return slog.New(slog.NewJSONHandler(io.Discard, nil)) }

const testAccount = "11111111-1111-4111-8111-111111111111"

// newTestService wires the service with a fake store + a verifier that accepts any
// token as testAccount.
func newTestService(fs *fakeStore) http.Handler {
	svc := NewService(fs, fakeVerifier{subject: testAccount}, testLogger())
	return svc.Handler()
}

func authedReq(method, target string, body string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	r.Header.Set("Authorization", "Bearer good-token")
	return r
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode body: %v (%s)", err, rr.Body.String())
	}
	return m
}

func TestGetStateRequiresJWT(t *testing.T) {
	fs := &fakeStore{}
	h := newTestService(fs)
	rr := httptest.NewRecorder()
	// No Authorization header → 401.
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/state/16", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestGetState(t *testing.T) {
	deadline := time.Now().Add(15 * time.Minute)
	fs := &fakeStore{
		getState: func(_ context.Context, acct, pid string) (store.State, error) {
			if acct != testAccount || pid != "16" {
				t.Fatalf("unexpected args: %s %s", acct, pid)
			}
			return store.State{
				ProblemID:      "16",
				Status:         "attempting",
				StageReached:   store.StageAttempt,
				UnlockedStages: []string{store.StageAttempt},
				Timer:          &store.Timer{Kind: store.TimerAttempt, DeadlineAt: deadline},
			}, nil
		},
	}
	h := newTestService(fs)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/state/16", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	st, _ := body["state"].(map[string]any)
	if st["status"] != "attempting" {
		t.Errorf("status = %v", st["status"])
	}
	tm, ok := st["timer"].(map[string]any)
	if !ok {
		t.Fatalf("timer missing: %v", st["timer"])
	}
	if rem, _ := tm["remainingSeconds"].(float64); rem < 800 || rem > 900 {
		t.Errorf("remainingSeconds = %v, want ~900", tm["remainingSeconds"])
	}
	if unlocked, _ := st["unlockedStages"].([]any); len(unlocked) != 1 {
		t.Errorf("unlockedStages = %v", st["unlockedStages"])
	}
}

func TestStartAttempt(t *testing.T) {
	fs := &fakeStore{
		start: func(_ context.Context, _, _, path string) (store.State, error) {
			// No ?path= (a v1.6.0 gateway): the default course.
			if path != "dsa" {
				t.Errorf("start path = %q, want dsa", path)
			}
			return store.State{ProblemID: "16", Status: "attempting", StageReached: store.StageAttempt,
				UnlockedStages: []string{store.StageAttempt}, Timer: &store.Timer{Kind: store.TimerAttempt, DeadlineAt: time.Now().Add(15 * time.Minute)}}, nil
		},
	}
	h := newTestService(fs)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/problems/16/attempt/start", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rr.Code, rr.Body.String())
	}
	st := decodeBody(t, rr)["state"].(map[string]any)
	if st["status"] != "attempting" {
		t.Errorf("status = %v", st["status"])
	}
}

func TestRevealWithPenalty(t *testing.T) {
	fs := &fakeStore{
		reveal: func(_ context.Context, _, _ string) (store.RevealResult, error) {
			return store.RevealResult{
				Revealed: store.StageSolution,
				Penalty:  &store.Penalty{OwedAttempt: true, DueInDays: store.EarlyRevealPenaltyDays},
				State: store.State{ProblemID: "16", Status: "attempting", StageReached: store.StageSolution,
					UnlockedStages: []string{store.StageAttempt, store.StageHint, store.StageSolution}, RevealedEarly: true},
			}, nil
		},
	}
	h := newTestService(fs)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/problems/16/reveal", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["revealed"] != "solution" {
		t.Errorf("revealed = %v", body["revealed"])
	}
	pen, ok := body["penalty"].(map[string]any)
	if !ok {
		t.Fatalf("penalty missing: %v", body["penalty"])
	}
	if pen["owedAttempt"] != true {
		t.Errorf("owedAttempt = %v", pen["owedAttempt"])
	}
	msg, _ := pen["message"].(string)
	if !strings.Contains(msg, "#16") || !strings.Contains(msg, "3 days") {
		t.Errorf("penalty message = %q", msg)
	}
}

func TestRevealNothingLeft(t *testing.T) {
	fs := &fakeStore{
		reveal: func(_ context.Context, _, _ string) (store.RevealResult, error) {
			return store.RevealResult{}, store.ErrNothingToReveal
		},
	}
	h := newTestService(fs)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/problems/16/reveal", ""))
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (%s)", rr.Code, rr.Body.String())
	}
}

func TestOutcome(t *testing.T) {
	fs := &fakeStore{
		logOutcome: func(_ context.Context, _, _, v string) (store.OutcomeResult, error) {
			if v != "clean" {
				t.Fatalf("value = %q", v)
			}
			return store.OutcomeResult{State: store.State{ProblemID: "16", Status: "solved", LastOutcome: "clean", FirstSolvedAt: time.Now()}}, nil
		},
	}
	h := newTestService(fs)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/problems/16/outcome", `{"outcome":"clean"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	st := body["state"].(map[string]any)
	if st["status"] != "solved" || st["lastOutcome"] != "clean" {
		t.Errorf("state = %v", st)
	}
	if _, ok := body["cappedBy"]; ok {
		t.Errorf("an uncapped outcome carries cappedBy: %v", body)
	}
	if v, ok := st["coachAssistAt"]; !ok || v != nil {
		t.Errorf("state.coachAssistAt = %v (present %v), want null", v, ok)
	}
}

// m1-07 (D27): a clamped outcome reports cappedBy "coach" and the state's coachAssistAt.
func TestOutcomeCappedByCoach(t *testing.T) {
	assisted := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	fs := &fakeStore{
		logOutcome: func(_ context.Context, _, _, _ string) (store.OutcomeResult, error) {
			return store.OutcomeResult{
				State:    store.State{ProblemID: "16", Status: "solved", LastOutcome: "assisted", CoachAssistAt: assisted},
				CappedBy: store.CappedByCoach,
			}, nil
		},
	}
	rr := httptest.NewRecorder()
	newTestService(fs).ServeHTTP(rr, authedReq(http.MethodPost, "/problems/16/outcome", `{"outcome":"clean"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	st := body["state"].(map[string]any)
	if body["cappedBy"] != "coach" || st["lastOutcome"] != "assisted" || st["coachAssistAt"] != "2026-10-01T09:00:00Z" {
		t.Errorf("capped outcome = %v", body)
	}
}

// m1-07: POST /attempts/{id}/assist — 200 with the recorded time, 404 unknown / not
// the account's, 409 attempt_closed for a concluded attempt.
func TestAssist(t *testing.T) {
	const attempt = "0b6b8a52-5c6f-4f50-9a3a-2a1b7c0d9e11"
	at := time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC)
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantErr  string
	}{
		{"recorded", nil, http.StatusOK, ""},
		{"unknown or another account's", store.ErrNotFound, http.StatusNotFound, "not_found"},
		{"concluded", store.ErrAttemptClosed, http.StatusConflict, "attempt_closed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var gotAccount, gotID string
			fs := &fakeStore{assist: func(_ context.Context, a, id string) (time.Time, error) {
				gotAccount, gotID = a, id
				if c.err != nil {
					return time.Time{}, c.err
				}
				return at, nil
			}}
			rr := httptest.NewRecorder()
			newTestService(fs).ServeHTTP(rr, authedReq(http.MethodPost, "/attempts/"+attempt+"/assist", ""))
			if rr.Code != c.wantCode {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, c.wantCode, rr.Body.String())
			}
			if gotAccount != testAccount || gotID != attempt {
				t.Errorf("store got account %q attempt %q", gotAccount, gotID)
			}
			body := decodeBody(t, rr)
			if c.wantErr != "" {
				if code := body["error"].(map[string]any)["code"]; code != c.wantErr {
					t.Errorf("error code = %v, want %s", code, c.wantErr)
				}
				return
			}
			if body["attemptId"] != attempt || body["coachAssistAt"] != "2026-10-01T09:30:00Z" {
				t.Errorf("body = %v", body)
			}
		})
	}
}

// m1-07: GET /attempts/open — the account's open attempts (purpose "course"), and with
// ?problem_id= the item's solve state.
func TestOpenAttempts(t *testing.T) {
	started := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	var gotProblem string
	fs := &fakeStore{
		listOpen: func(_ context.Context, _, p string) ([]store.OpenAttempt, error) {
			gotProblem = p
			return []store.OpenAttempt{{ID: "att-1", ProblemID: "16", PathSlug: "dsa", Purpose: store.PurposeCourse,
				StartedAt: started, StageReached: "hint", CoachAssistAt: started.Add(time.Minute)}}, nil
		},
		getState: func(_ context.Context, _, p string) (store.State, error) {
			return store.State{ProblemID: p, Status: "attempting"}, nil
		},
	}
	h := newTestService(fs)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/attempts/open?problem_id=16", ""))
	if rr.Code != http.StatusOK || gotProblem != "16" {
		t.Fatalf("status = %d problem %q (%s)", rr.Code, gotProblem, rr.Body.String())
	}
	body := decodeBody(t, rr)
	atts := body["attempts"].([]any)
	a := atts[0].(map[string]any)
	if len(atts) != 1 || a["attemptId"] != "att-1" || a["problemId"] != "16" || a["pathSlug"] != "dsa" ||
		a["purpose"] != "course" || a["startedAt"] != "2026-10-01T08:00:00Z" || a["stageReached"] != "hint" ||
		a["coachAssistAt"] != "2026-10-01T08:01:00Z" {
		t.Errorf("attempts = %v", atts)
	}
	p := body["problem"].(map[string]any)
	if p["problemId"] != "16" || p["status"] != "attempting" || p["firstSolvedAt"] != nil {
		t.Errorf("problem = %v", p)
	}

	// Account-wide: no problem slot.
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/attempts/open", ""))
	if rr.Code != http.StatusOK || gotProblem != "" {
		t.Fatalf("account-wide: status %d problem %q", rr.Code, gotProblem)
	}
	if _, ok := decodeBody(t, rr)["problem"]; ok {
		t.Errorf("account-wide read carries a problem slot")
	}
}

func TestOutcomeInvalid(t *testing.T) {
	fs := &fakeStore{
		logOutcome: func(_ context.Context, _, _, _ string) (store.OutcomeResult, error) {
			return store.OutcomeResult{}, store.ErrInvalidOutcome
		},
	}
	h := newTestService(fs)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/problems/16/outcome", `{"outcome":"great"}`))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (%s)", rr.Code, rr.Body.String())
	}
}

func TestOutcomeBadBody(t *testing.T) {
	fs := &fakeStore{}
	h := newTestService(fs)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/problems/16/outcome", `{"nope":1}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (%s)", rr.Code, rr.Body.String())
	}
}

func TestListStates(t *testing.T) {
	fs := &fakeStore{
		listStates: func(_ context.Context, _ string, ids []string) (map[string]store.State, error) {
			if len(ids) != 2 {
				t.Fatalf("ids = %v", ids)
			}
			return map[string]store.State{
				"16": {ProblemID: "16", Status: "solved", LastOutcome: "clean"},
			}, nil
		},
	}
	h := newTestService(fs)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/state?week=2&ids=16,17", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rr.Code, rr.Body.String())
	}
	states := decodeBody(t, rr)["states"].(map[string]any)
	got := states["16"].(map[string]any)
	if got["status"] != "solved" {
		t.Errorf("16 status = %v", got["status"])
	}
}
