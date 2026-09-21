package review

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

	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

func testService(st store.Store) *Service {
	return NewService(st, fakeVerifier{subject: "acct-1"}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func do(t *testing.T, h http.Handler, method, path, token, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	var decoded map[string]any
	if b, _ := io.ReadAll(rec.Body); len(b) > 0 {
		_ = json.Unmarshal(b, &decoded)
	}
	return rec, decoded
}

func TestDueQueueRequiresJWT(t *testing.T) {
	svc := testService(&fakeStore{})
	rec, body := do(t, svc.Handler(), http.MethodGet, "/revisions/due", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if errObj, _ := body["error"].(map[string]any); errObj["code"] != "unauthenticated" {
		t.Fatalf("body = %v, want unauthenticated envelope", body)
	}
}

func TestDueQueue(t *testing.T) {
	now := time.Now()
	st := &fakeStore{
		dueQueue: func(_ context.Context, accountID string, _ int) ([]store.DueItem, error) {
			if accountID != "acct-1" {
				t.Fatalf("account = %q, want acct-1 (token subject)", accountID)
			}
			return []store.DueItem{
				{ItemID: "it-1", ProblemID: "3", TouchLevel: 1, DueDate: now.Add(-time.Hour), Due: true, Status: "pending"},
				{ItemID: "it-2", ProblemID: "2", TouchLevel: 4, DueDate: now.Add(48 * time.Hour), Due: false, MockMode: true, Status: "pending"},
			}, nil
		},
	}
	rec, body := do(t, testService(st).Handler(), http.MethodGet, "/revisions/due", "tok", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if dc, _ := body["dueCount"].(float64); dc != 1 {
		t.Fatalf("dueCount = %v, want 1", body["dueCount"])
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	first, _ := items[0].(map[string]any)
	if first["dayLabel"] != "Day 1" {
		t.Errorf("first dayLabel = %v, want Day 1", first["dayLabel"])
	}
	second, _ := items[1].(map[string]any)
	if second["dayLabel"] != "Day 21" || second["mockMode"] != true {
		t.Errorf("second = %v, want Day 21 + mockMode", second)
	}
}

func TestScorePassAdvances(t *testing.T) {
	st := &fakeStore{
		score: func(_ context.Context, accountID, itemID string, in store.ScoreInput) (store.ScoreResult, error) {
			if accountID != "acct-1" || itemID != "it-9" {
				t.Fatalf("score(%q,%q)", accountID, itemID)
			}
			if in.NamedPatternSecs != 45 || !in.SolvedInTimer || !in.StatedComplexity {
				t.Fatalf("score input = %+v", in)
			}
			return store.ScoreResult{ItemID: "it-9", ProblemID: "3", TouchLevel: 1, AutoPass: true, NextTouchLevel: 2, NextDueDate: time.Now().Add(72 * time.Hour)}, nil
		},
	}
	rec, body := do(t, testService(st).Handler(), http.MethodPost, "/revisions/it-9/score", "tok",
		`{"namedPatternSecs":45,"solvedInTimer":true,"statedComplexity":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body["autoPass"] != true || body["status"] != "passed" {
		t.Errorf("body = %v, want passed", body)
	}
	if body["nextDayLabel"] != "Day 3" {
		t.Errorf("nextDayLabel = %v, want Day 3", body["nextDayLabel"])
	}
	if body["nextDueDate"] == nil {
		t.Errorf("nextDueDate should be present on a pass")
	}
}

func TestScoreFailResets(t *testing.T) {
	st := &fakeStore{
		score: func(_ context.Context, _, _ string, _ store.ScoreInput) (store.ScoreResult, error) {
			return store.ScoreResult{ItemID: "it-9", ProblemID: "3", TouchLevel: 3, AutoPass: false, Reset: true, NextTouchLevel: 1, NextDueDate: time.Now().Add(24 * time.Hour)}, nil
		},
	}
	rec, body := do(t, testService(st).Handler(), http.MethodPost, "/revisions/it-9/score", "tok",
		`{"namedPatternSecs":200,"solvedInTimer":true,"statedComplexity":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body["autoPass"] != false || body["status"] != "failed" || body["reset"] != true {
		t.Errorf("body = %v, want failed + reset", body)
	}
	if body["nextDayLabel"] != "Day 1" {
		t.Errorf("nextDayLabel = %v, want Day 1", body["nextDayLabel"])
	}
}

func TestScoreRejectsNegativeSecs(t *testing.T) {
	st := &fakeStore{
		score: func(context.Context, string, string, store.ScoreInput) (store.ScoreResult, error) {
			t.Fatalf("store.Score should not be called on invalid input")
			return store.ScoreResult{}, nil
		},
	}
	rec, _ := do(t, testService(st).Handler(), http.MethodPost, "/revisions/it-9/score", "tok",
		`{"namedPatternSecs":-1,"solvedInTimer":true,"statedComplexity":true}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestScoreNotFound(t *testing.T) {
	st := &fakeStore{
		score: func(context.Context, string, string, store.ScoreInput) (store.ScoreResult, error) {
			return store.ScoreResult{}, store.ErrNotFound
		},
	}
	rec, body := do(t, testService(st).Handler(), http.MethodPost, "/revisions/missing/score", "tok",
		`{"namedPatternSecs":10,"solvedInTimer":true,"statedComplexity":true}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if errObj, _ := body["error"].(map[string]any); errObj["code"] != "not_found" {
		t.Errorf("body = %v, want not_found", body)
	}
}
