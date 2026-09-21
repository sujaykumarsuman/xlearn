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
		start: func(_ context.Context, _, _ string) (store.State, error) {
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
		logOutcome: func(_ context.Context, _, _, v string) (store.State, error) {
			if v != "clean" {
				t.Fatalf("value = %q", v)
			}
			return store.State{ProblemID: "16", Status: "solved", LastOutcome: "clean", FirstSolvedAt: time.Now()}, nil
		},
	}
	h := newTestService(fs)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/problems/16/outcome", `{"outcome":"clean"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rr.Code, rr.Body.String())
	}
	st := decodeBody(t, rr)["state"].(map[string]any)
	if st["status"] != "solved" || st["lastOutcome"] != "clean" {
		t.Errorf("state = %v", st)
	}
}

func TestOutcomeInvalid(t *testing.T) {
	fs := &fakeStore{
		logOutcome: func(_ context.Context, _, _, _ string) (store.State, error) {
			return store.State{}, store.ErrInvalidOutcome
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
