package gateway

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// aggHarness wires a real gateway to fake identity + assessment + curriculum + review to
// exercise the Progress + Dashboard BFF composition (the roll-ups, the plan ordering).
// The upstreams accept any bearer (the mint→JWKS-verify path is covered by mock_test).
type aggHarness struct {
	gwServer *httptest.Server
}

func newAggHarness(t *testing.T) *aggHarness {
	t.Helper()
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer := auth.NewSigner(key, "xlearn-gateway", 0)

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
	}))
	t.Cleanup(identity.Close)

	assessment := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /progress/summary": writeJSONFn(map[string]any{
			"solved": 3, "total": 151,
			"streak":     map[string]any{"current": 5, "longest": 9},
			"retention":  map[string]any{"pct": 80, "resets": 1, "ladders": 5},
			"mock":       map[string]any{"count": 2, "average": 22, "best": 24, "last": 24, "delta": 2},
			"outcomeMix": map[string]any{"total": 3, "clean": 2, "rough": 1, "assisted": 0, "miss": 0},
		}),
		"GET /progress/heatmap": writeJSONFn(map[string]any{"days": []any{
			map[string]any{"date": "2026-09-20", "solves": 1, "reviews": 2},
		}}),
		"GET /progress/mastery": writeJSONFn(map[string]any{"problems": []any{
			map[string]any{"problemId": "1", "weight": 1.0},  // week1 easy, Hashing
			map[string]any{"problemId": "2", "weight": 0.7},  // week1 med, Two pointers
			map[string]any{"problemId": "20", "weight": 1.0}, // week5 hard, Sliding window
		}}),
		"GET /mocks/trend": writeJSONFn(map[string]any{"points": []any{}, "targets": map[string]any{"w13": 24}}),
	}))
	t.Cleanup(assessment.Close)

	curriculum := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /paths/dsa": writeJSONFn(map[string]any{
			"path": map[string]any{"slug": "dsa", "problem_total": 151},
			"phases": []any{
				map[string]any{"order": 1, "name": "Fundamentals", "theme": "t", "week_from": 1, "week_to": 3},
				map[string]any{"order": 2, "name": "Core", "theme": "t", "week_from": 4, "week_to": 8},
			},
			"weeks": []any{
				map[string]any{"n": 1, "title": "Arrays"},
				map[string]any{"n": 5, "title": "Windows"},
			},
		}),
		"GET /paths/dsa/problems": writeJSONFn(map[string]any{"problems": []any{
			map[string]any{"id": "1", "week_n": 1, "title": "Two Sum", "difficulty": "easy", "pattern": "Hashing", "is_reinforcement": false},
			map[string]any{"id": "2", "week_n": 1, "title": "3Sum", "difficulty": "med", "pattern": "Two pointers", "is_reinforcement": false},
			map[string]any{"id": "9", "week_n": 1, "title": "Valid Anagram", "difficulty": "easy", "pattern": "Hashing", "is_reinforcement": false},
			map[string]any{"id": "20", "week_n": 5, "title": "Min Window", "difficulty": "hard", "pattern": "Sliding window", "is_reinforcement": false},
			map[string]any{"id": "99", "week_n": 1, "title": "Reinforce", "difficulty": "easy", "pattern": "Hashing", "is_reinforcement": true},
		}}),
		"GET /problems/{id}": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"problem": map[string]any{"id": r.PathValue("id"), "title": "P" + r.PathValue("id")}})
		},
		"GET /problems": func(w http.ResponseWriter, r *http.Request) {
			problems := []any{}
			for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
				if id != "" {
					problems = append(problems, map[string]any{"id": id, "title": "P" + id})
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"problems": problems})
		},
	}))
	t.Cleanup(curriculum.Close)

	review := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /revisions/due": writeJSONFn(map[string]any{"items": []any{
			map[string]any{"itemId": "r1", "problemId": "1", "touchLevel": 1, "dayLabel": "Day 1", "due": true, "status": "pending"},
		}, "dueCount": 1}),
		"GET /weak-area/current": writeJSONFn(map[string]any{"topCategory": "off_by_one", "topCount": 3, "entries": []any{}}),
		"GET /reminders":         writeJSONFn(map[string]any{"reminders": []any{map[string]any{"id": "n1", "kind": "revision_due"}}}),
	}))
	t.Cleanup(review.Close)

	dist := fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}
	gw := New(Options{
		BasePath: "/xlearn", Version: "test", Dist: dist, Signer: signer,
		IdentityBaseURL: identity.URL, AudienceIdentity: "identity",
		CurriculumBaseURL: curriculum.URL,
		ReviewBaseURL:     review.URL, AudienceReview: "review",
		AssessmentBaseURL: assessment.URL, AudienceAssessment: "assessment",
	})
	h := &aggHarness{gwServer: httptest.NewServer(gw.Handler())}
	t.Cleanup(h.gwServer.Close)
	return h
}

type handlerFn = http.HandlerFunc

func jsonMux(routes map[string]handlerFn) http.Handler {
	mux := http.NewServeMux()
	for pat, fn := range routes {
		mux.HandleFunc(pat, fn)
	}
	return mux
}

func writeJSONFn(v any) handlerFn {
	return func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(v) }
}

func (h *aggHarness) get(t *testing.T, path string, cookie *http.Cookie) map[string]any {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, h.gwServer.URL+path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d", path, resp.StatusCode)
	}
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out
}

func TestBFFProgressComposes(t *testing.T) {
	h := newAggHarness(t)
	body := h.get(t, "/xlearn/api/progress", &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})

	summary, _ := body["summary"].(map[string]any)
	if summary == nil || summary["solved"].(float64) != 3 || summary["total"].(float64) != 151 {
		t.Fatalf("summary = %v", body["summary"])
	}

	// Phase completion: Fundamentals (weeks 1-3) has 3 core problems, 2 solved (#1,#2).
	phases, _ := body["phases"].([]any)
	if len(phases) != 2 {
		t.Fatalf("phases = %d, want 2", len(phases))
	}
	f0, _ := phases[0].(map[string]any)
	if f0["name"] != "Fundamentals" || f0["total"].(float64) != 3 || f0["solved"].(float64) != 2 {
		t.Fatalf("Fundamentals completion = %v", f0)
	}

	// Pattern mastery: Hashing has 2 core (#1,#9), 1 solved clean → weight 1/2 = 50%.
	// Sliding window has 1 (#20), solved clean → 100%. Ordered strongest-first.
	patterns, _ := body["patterns"].([]any)
	byName := map[string]map[string]any{}
	for _, p := range patterns {
		m := p.(map[string]any)
		byName[m["name"].(string)] = m
	}
	if h := byName["Hashing"]; h == nil || h["total"].(float64) != 2 || h["solved"].(float64) != 1 || h["pct"].(float64) != 50 {
		t.Fatalf("Hashing mastery = %v", byName["Hashing"])
	}
	if sw := byName["Sliding window"]; sw == nil || sw["pct"].(float64) != 100 {
		t.Fatalf("Sliding window mastery = %v", byName["Sliding window"])
	}
	// Reinforcement problem #99 must not inflate Hashing's total.
	if byName["Hashing"]["total"].(float64) != 2 {
		t.Fatalf("reinforcement leaked into pattern total: %v", byName["Hashing"])
	}
	first := patterns[0].(map[string]any)
	if first["pct"].(float64) != 100 {
		t.Fatalf("patterns not ordered strongest-first: %v", first)
	}
}

func TestBFFDashboardComposesPlan(t *testing.T) {
	h := newAggHarness(t)
	body := h.get(t, "/xlearn/api/dashboard", &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})

	stats, _ := body["stats"].(map[string]any)
	solved, _ := stats["solved"].(map[string]any)
	if solved["count"].(float64) != 3 || solved["total"].(float64) != 151 {
		t.Fatalf("stats.solved = %v", solved)
	}
	streak, _ := stats["streak"].(map[string]any)
	if streak["current"].(float64) != 5 {
		t.Fatalf("stats.streak = %v", streak)
	}
	if stats["revisionsDue"].(float64) != 1 {
		t.Fatalf("revisionsDue = %v", stats["revisionsDue"])
	}

	// Plan: the due review leads (reviews before new work), then unsolved week-1 core
	// problems. Week 1 is current (2 of 3 core solved: #1,#2 → #9 remains).
	plan, _ := body["plan"].([]any)
	if len(plan) < 2 {
		t.Fatalf("plan too short: %v", plan)
	}
	first, _ := plan[0].(map[string]any)
	if first["kind"] != "review" {
		t.Fatalf("plan[0] kind = %v, want review (reviews before new work)", first["kind"])
	}
	sawProblem := false
	for _, p := range plan {
		m := p.(map[string]any)
		if m["kind"] == "problem" {
			sawProblem = true
			if m["problemId"] == "1" || m["problemId"] == "2" {
				t.Fatalf("plan suggested an already-solved problem: %v", m)
			}
		}
	}
	if !sawProblem {
		t.Fatalf("plan has no new-work problem: %v", plan)
	}

	// Week panel: current week 1, 2 of 3 core solved.
	week, _ := body["week"].(map[string]any)
	if week == nil || week["n"].(float64) != 1 || week["solved"].(float64) != 2 || week["total"].(float64) != 3 {
		t.Fatalf("week = %v", body["week"])
	}
}
