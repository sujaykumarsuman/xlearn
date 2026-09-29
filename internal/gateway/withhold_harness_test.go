package gateway

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// withholdHarness wires a real gateway to fakes of every upstream (identity, curriculum,
// practice, review, assessment, coach) with a fixed per-item story (m1-06):
//
//   - item 1: an OPEN counted attempt (attempting, attempt stage only, timer running);
//   - item 2: solved (hint and solution unlocked) and now a DUE touch;
//   - item 3: solved BLIND (unlockedStages=[attempt]) and not due (an upcoming touch);
//   - item 4: never attempted, not live.
//
// Curriculum returns a sentinel for every answer-bearing field of every item
// (leakMark(kind, id)), so a test can grep any response body for a leak.
type withholdHarness struct {
	gw     *httptest.Server
	mu     sync.Mutex
	coach  []string // bodies the fake coach received on POST /chat
	scored []string // item ids the fake review scored
	// Knobs a test may flip before issuing requests.
	practiceDown bool
	reviewDown   bool
}

// withholdItems are the harness's problem ids (all in DSA week 1).
var withholdItems = []string{"1", "2", "3", "4"}

// sentinel is the unique marker curriculum/review return for one answer-bearing field
// of one item. The trailing "." keeps "…-1." from matching "…-10.".
func leakMark(kind, id string) string { return fmt.Sprintf("%s-SENTINEL-%s.", kind, id) }

// sentinelKinds are the answer-bearing fields withhold() governs, by sentinel kind.
var sentinelKinds = []string{"PATTERN", "CONCEPT", "FACTS", "HINT", "SOLUTION", "MISTAKE-PATTERN"}

func harnessProblem(id string) map[string]any {
	return map[string]any{
		"id": id, "path_slug": "dsa", "week_n": 1, "title": "Problem " + id, "difficulty": "med",
		"pattern": leakMark("PATTERN", id), "concepts": []string{leakMark("CONCEPT", id)},
		"leetcode_url": "", "neetcode_url": "", "is_reinforcement": false, "role": "core", "links": []any{},
	}
}

func newWithholdHarness(t *testing.T) *withholdHarness {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	h := &withholdHarness{}
	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	serve := func(mux *http.ServeMux) string {
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		return srv.URL
	}

	identity := http.NewServeMux()
	identity.HandleFunc("POST /sessions/validate", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"account_id": "acct-1", "role": "learner", "status": "active"})
	})
	identity.HandleFunc("GET /accounts/{id}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"account":     map[string]any{"id": r.PathValue("id")},
			"onboarding":  map[string]any{"completed": true},
			"enrollments": []any{map[string]any{"path_slug": "dsa", "status": "active", "started_at": "2026-09-01T00:00:00Z"}},
		})
	})

	curriculum := http.NewServeMux()
	curriculum.HandleFunc("GET /paths", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"paths": []any{map[string]any{"slug": "dsa", "status": "active"}}})
	})
	curriculum.HandleFunc("GET /paths/{slug}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"path":   map[string]any{"slug": "dsa", "problem_total": len(withholdItems), "week_total": 1},
			"phases": []any{map[string]any{"order": 1, "name": "P1", "week_from": 1, "week_to": 1}},
			"weeks":  []any{map[string]any{"n": 1, "title": "Week one"}},
		})
	})
	list := func() []any {
		out := make([]any, 0, len(withholdItems))
		for _, id := range withholdItems {
			out = append(out, harnessProblem(id))
		}
		return out
	}
	curriculum.HandleFunc("GET /paths/{slug}/problems", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"problems": list()})
	})
	curriculum.HandleFunc("GET /paths/{slug}/weeks/{n}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"week":     map[string]any{"n": 1, "title": "Week one", "thesis": "t"},
			"path":     map[string]any{"slug": "dsa", "week_total": 1},
			"concepts": []any{map[string]any{"slug": "hashing", "title": "Hashing"}},
			"problems": list(),
		})
	})
	curriculum.HandleFunc("GET /paths/{slug}/concepts/{c}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"concept": map[string]any{"slug": r.PathValue("c"), "path_slug": "dsa", "title": "Hashing", "body_md": "read me"}})
	})
	curriculum.HandleFunc("GET /problems", func(w http.ResponseWriter, r *http.Request) {
		out := []any{}
		for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
			for _, known := range withholdItems {
				if id == known {
					out = append(out, harnessProblem(id))
				}
			}
		}
		writeJSON(w, map[string]any{"problems": out})
	})
	curriculum.HandleFunc("GET /problems/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		writeJSON(w, map[string]any{
			"problem": harnessProblem(id),
			"sections": []any{
				map[string]any{"stage": "attempt", "kind": "summary", "order": 1, "body_md": "statement " + id, "code": "", "language": ""},
				map[string]any{"stage": "hint", "kind": "key_observation", "order": 1, "body_md": leakMark("HINT", id), "code": "", "language": ""},
				map[string]any{"stage": "solution", "kind": "code", "order": 1, "body_md": "", "code": leakMark("SOLUTION", id), "language": "go"},
				map[string]any{"stage": "solution", "kind": "solution_facts", "order": 2, "body_md": "", "code": "", "language": "",
					"solution_facts": map[string]any{"complexity": map[string]any{"time": []string{leakMark("FACTS", id)}, "space": []string{"O(1)"}}}},
			},
		})
	})

	// practice: the full per-item state (GET /state/{id}) and the light bulk one.
	fullState := func(id string) map[string]any {
		st := map[string]any{"problemId": id, "status": "available", "currentTouch": 0, "lastOutcome": nil,
			"firstSolvedAt": nil, "revealedEarly": false, "timer": nil, "unlockedStages": []string{"attempt"}}
		switch id {
		case "1":
			st["status"], st["stageReached"] = "attempting", "attempt"
			st["timer"] = map[string]any{"kind": "attempt", "deadlineAt": "2099-01-01T00:00:00Z", "remainingSeconds": 900, "expired": false}
		case "2":
			st["status"], st["stageReached"], st["lastOutcome"] = "solved", "solution", "assisted"
			st["firstSolvedAt"], st["unlockedStages"], st["currentTouch"] = "2026-09-20T00:00:00Z", []string{"attempt", "hint", "solution"}, 1
		case "3":
			st["status"], st["stageReached"], st["lastOutcome"] = "solved", "attempt", "clean"
			st["firstSolvedAt"] = "2026-09-21T00:00:00Z"
		}
		return st
	}
	practice := http.NewServeMux()
	practice.HandleFunc("GET /state/{id}", func(w http.ResponseWriter, r *http.Request) {
		if h.practiceDown {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"state": fullState(r.PathValue("id"))})
	})
	practice.HandleFunc("GET /state", func(w http.ResponseWriter, r *http.Request) {
		if h.practiceDown {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		states := map[string]any{}
		for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
			if id == "4" || id == "" {
				continue // never attempted: no row
			}
			st := fullState(id)
			delete(st, "unlockedStages") // the bulk read is light (no stages, no timer)
			st["timer"] = nil
			states[id] = st
		}
		writeJSON(w, map[string]any{"states": states})
	})
	practice.HandleFunc("POST /problems/{id}/{op...}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"state": fullState(r.PathValue("id"))})
	})

	review := http.NewServeMux()
	review.HandleFunc("GET /revisions/due", func(w http.ResponseWriter, r *http.Request) {
		if h.reviewDown {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"dueCount": 1, "items": []any{
			map[string]any{"itemId": "rev-2", "problemId": "2", "touchLevel": 2, "dayLabel": "Day 3", "dueDate": "2026-09-28T00:00:00Z", "due": true, "mockMode": false, "status": "pending"},
			map[string]any{"itemId": "rev-3", "problemId": "3", "touchLevel": 1, "dayLabel": "Day 1", "dueDate": "2099-10-08T00:00:00Z", "due": false, "mockMode": false, "status": "pending"},
		}})
	})
	mistake := func(id string) map[string]any {
		return map[string]any{"id": "m-" + id, "problemId": id, "pattern": leakMark("MISTAKE-PATTERN", id), "mistake": "x", "rootCause": "y",
			"insight": "z", "category": "", "status": "open", "revisitCount": 0, "revisitDate": nil, "createdAt": "2026-09-22T00:00:00Z"}
	}
	review.HandleFunc("GET /mistakes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"mistakes": []any{mistake("1"), mistake("2"), mistake("3")}})
	})
	review.HandleFunc("GET /weak-area/current", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"weakArea": map[string]any{"category": "wrong_pattern", "count": 3}, "entries": []any{mistake("1"), mistake("2"), mistake("3")}})
	})
	review.HandleFunc("GET /reminders", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"reminders": []any{}})
	})
	review.HandleFunc("POST /revisions/{id}/score", func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		h.scored = append(h.scored, r.PathValue("id"))
		h.mu.Unlock()
		pid := strings.TrimPrefix(r.PathValue("id"), "rev-")
		writeJSON(w, map[string]any{"itemId": r.PathValue("id"), "problemId": pid, "touchLevel": 2, "autoPass": true, "status": "passed",
			"mockMode": false, "reset": false, "nextTouchLevel": 3, "nextDayLabel": "Day 7", "nextDueDate": "2099-10-01T00:00:00Z"})
	})

	assessment := http.NewServeMux()
	assessment.HandleFunc("GET /progress/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"streak": map[string]any{"current": 2, "longest": 3}, "solved": 2, "total": 4, "mock": map[string]any{}})
	})
	assessment.HandleFunc("GET /progress/mastery", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"problems": []any{map[string]any{"problemId": "2", "weight": 0.5}, map[string]any{"problemId": "3", "weight": 1}}})
	})

	coach := http.NewServeMux()
	coach.HandleFunc("POST /chat", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		h.mu.Lock()
		h.coach = append(h.coach, string(body))
		h.mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: ok\n\n"))
	})

	gw := New(Options{
		BasePath:           "/xlearn",
		Version:            "test",
		Dist:               fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}},
		Signer:             auth.NewSigner(key, "xlearn-gateway", 0),
		IdentityBaseURL:    serve(identity),
		AudienceIdentity:   "identity",
		CurriculumBaseURL:  serve(curriculum),
		PracticeBaseURL:    serve(practice),
		AudiencePractice:   "practice",
		ReviewBaseURL:      serve(review),
		AudienceReview:     "review",
		AssessmentBaseURL:  serve(assessment),
		AudienceAssessment: "assessment",
		CoachBaseURL:       serve(coach),
		AudienceCoach:      "coach",
	})
	h.gw = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gw.Close)
	return h
}

// do issues an authenticated request to the gateway and returns the status and body.
func (h *withholdHarness) do(t *testing.T, method, path, body string) (int, string) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, h.gw.URL+"/xlearn"+path, rdr)
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(out)
}

// goldenRoutes are the composed item surfaces whose JSON the m1-06 golden diff covers
// (Week, Today, revision due, mistakes, weak-area, problem course + arena).
var goldenRoutes = []string{
	"/api/paths/dsa/weeks/1",
	"/api/paths/dsa/problems",
	"/api/dashboard",
	"/api/paths/dsa/revision/due",
	"/api/paths/dsa/mistakes",
	"/api/paths/dsa/weak-area",
	"/api/problems/1", "/api/problems/2", "/api/problems/3", "/api/problems/4",
	"/api/problems/1?practice=1", "/api/problems/2?practice=1", "/api/problems/3?practice=1", "/api/problems/4?practice=1",
}

// TestWithholdGoldenCapture writes each golden route's body (indented) to
// $XLEARN_GOLDEN_OUT when set, for the before/after diff in the m1-06 PR. It is a no-op
// in CI.
func TestWithholdGoldenCapture(t *testing.T) {
	dir := os.Getenv("XLEARN_GOLDEN_OUT")
	if dir == "" {
		t.Skip("set XLEARN_GOLDEN_OUT to capture the golden JSON")
	}
	h := newWithholdHarness(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range goldenRoutes {
		status, body := h.do(t, http.MethodGet, p, "")
		var v any
		if err := json.Unmarshal([]byte(body), &v); err != nil {
			t.Fatalf("%s: %d %s", p, status, body)
		}
		pretty, _ := json.MarshalIndent(v, "", "  ")
		name := strings.NewReplacer("/api/", "", "/", "_", "?", "_", "=", "-").Replace(p) + ".json"
		if err := os.WriteFile(filepath.Join(dir, name), append(pretty, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
