package gateway

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// This is m1-06's route-enumeration test (task 2), next to the OpenAPI drift test and
// reading the same route table (apiRoutes). Static half: every route declares a withhold
// policy and every exemption is listed below with its reviewed reason — a new route that
// isn't listed fails CI. Behavioural half: every `applied` GET route is fetched against
// withholdHarness's sentinel fakes and must not leak an answer-bearing field of a live
// item (1: open attempt; 2: due touch), while a solved-blind, not-due item (3) keeps its
// pattern on lists and in its workspace (no over-withholding).

// reviewedWithholdPolicy is the reviewed policy of every route: "applied", or the exact
// exemption reason. Adding a route means adding it here, and an exemption needs a reason
// a reviewer accepts.
var reviewedWithholdPolicy = map[string]string{
	"GET /api/healthz":                      "ops endpoint; no item data",
	"GET /.well-known/jwks.json":            "ops endpoint; no item data",
	"GET /api/me":                           "account read or write; no item data",
	"PATCH /api/me":                         "account read or write; no item data",
	"POST /api/me/password":                 "account read or write; no item data",
	"DELETE /api/me/oauth/{provider}":       "account read or write; no item data",
	"POST /api/me/username":                 "account read or write; no item data",
	"GET /api/username/available":           "account read or write; no item data",
	"POST /api/onboarding/step":             "account read or write; no item data",
	"POST /api/auth/logout":                 "auth flow; no item data",
	"POST /api/auth/signup":                 "auth flow; no item data",
	"POST /api/auth/login":                  "auth flow; no item data",
	"POST /api/auth/{provider}/start":       "auth flow; no item data",
	"GET /api/auth/{provider}/callback":     "auth flow; no item data",
	"POST /api/auth/dev/login":              "auth flow; no item data",
	"GET /api/auth/dev/enabled":             "auth flow; no item data",
	"POST /api/paths/{slug}/start":          "enrollment write; returns the enrollment",
	"GET /api/paths":                        "course catalog or roadmap; no item fields",
	"GET /api/paths/{slug}":                 "course catalog or roadmap; no item fields",
	"GET /api/paths/{slug}/problems":        "applied",
	"GET /api/paths/{slug}/weeks/{n}":       "applied",
	"GET /api/paths/{slug}/concepts/{c}":    "concept reading; no item fields (its item chips come from the week view, which applies withhold)",
	"GET /api/concepts/{slug}":              "concept reading; no item fields (its item chips come from the week view, which applies withhold)",
	"GET /api/paths/{slug}/dashboard":       "applied",
	"GET /api/dashboard":                    "applied",
	"GET /api/paths/{slug}/progress":        "progress aggregate: counts and pattern-mastery totals, no per-item answer field",
	"GET /api/progress":                     "progress aggregate: counts and pattern-mastery totals, no per-item answer field",
	"GET /api/paths/{slug}/revision/due":    "applied",
	"GET /api/revision/due":                 "applied",
	"POST /api/revision/{itemId}/score":     "applied",
	"GET /api/paths/{slug}/mistakes":        "applied",
	"GET /api/mistakes":                     "applied",
	"GET /api/paths/{slug}/weak-area":       "applied",
	"GET /api/weak-area":                    "applied",
	"POST /api/paths/{slug}/mistakes":       "mistake write: echoes the learner's own entry",
	"POST /api/mistakes":                    "mistake write: echoes the learner's own entry",
	"PATCH /api/mistakes/{id}":              "mistake write: echoes the learner's own entry",
	"GET /api/problems/{id}":                "applied",
	"POST /api/problems/{id}/attempt/start": "practice write: returns practice state only, no content fields",
	"POST /api/problems/{id}/reveal":        "practice write: returns practice state only, no content fields",
	"POST /api/problems/{id}/outcome":       "practice write: returns practice state only, no content fields",
	"POST /api/paths/{slug}/mocks":          "mock session: the mock brief names its problem; t1 §10's surface list has no mock row (M6 redesigns mocks)",
	"POST /api/mocks":                       "mock session: the mock brief names its problem; t1 §10's surface list has no mock row (M6 redesigns mocks)",
	"GET /api/mocks/{id}":                   "mock session: the mock brief names its problem; t1 §10's surface list has no mock row (M6 redesigns mocks)",
	"POST /api/mocks/{id}/score":            "mock session: the mock brief names its problem; t1 §10's surface list has no mock row (M6 redesigns mocks)",
	"GET /api/paths/{slug}/mocks/trend":     "mock trend: rubric totals, no item data",
	"GET /api/mocks/trend":                  "mock trend: rubric totals, no item data",
	"GET /api/u/{username}":                 "public profile: aggregates only (publicShapeAllowlist, m1-05)",
	"GET /api/coach/key":                    "coach key config (masked); no item data",
	"PUT /api/coach/key":                    "coach key config (masked); no item data",
	"DELETE /api/coach/key":                 "coach key config (masked); no item data",
	"GET /api/coach/thread":                 "coach thread: the learner's own chat history; the problem context is withheld when the chat is composed",
	"POST /api/coach/chat":                  "applied",
	// m1-10 adds GET /api/coach/models: "model catalog; no item data" (exempt).
	"GET /api/coach/models": "model catalog; no item data",
}

// optionalReviewedRoutes may be absent from apiRoutes (a parallel sprint adds them).
var optionalReviewedRoutes = map[string]bool{"GET /api/coach/models": true}

func TestWithholdPolicyOnEveryRoute(t *testing.T) {
	seen := map[string]bool{}
	for _, rt := range (&Gateway{}).apiRoutes() {
		key := rt.Method + " " + rt.Pattern
		seen[key] = true
		want, listed := reviewedWithholdPolicy[key]
		switch {
		case rt.Withhold.Mode == 0:
			t.Errorf("%s declares no withhold policy (set Withhold: withholdApplies or exempt(reason))", key)
		case !listed:
			t.Errorf("%s is not in reviewedWithholdPolicy: add it with \"applied\" or its reviewed exemption reason", key)
		case rt.Withhold.Mode == withholdApplied && want != "applied":
			t.Errorf("%s applies withhold, but the reviewed policy is exempt (%q)", key, want)
		case rt.Withhold.Mode == withholdExempt && (want == "applied" || want != rt.Withhold.Reason):
			t.Errorf("%s is exempt with %q; reviewed: %q", key, rt.Withhold.Reason, want)
		case rt.Withhold.Mode == withholdExempt && strings.TrimSpace(rt.Withhold.Reason) == "":
			t.Errorf("%s is exempt without a reason", key)
		}
	}
	for key := range reviewedWithholdPolicy {
		if !seen[key] && !optionalReviewedRoutes[key] {
			t.Errorf("reviewedWithholdPolicy lists %s, which is not a route (stale entry)", key)
		}
	}
}

// sweepParams are the fixture values for every path parameter an applied GET route may
// have. An applied route with any other parameter fails the sweep until it gets one.
var sweepParams = map[string][]string{
	"slug": {"dsa"},
	"n":    {"1"},
	"id":   withholdItems,
}

var pathParam = regexp.MustCompile(`\{([a-zA-Z]+)\}`)

// expandRoute returns the concrete paths for a route pattern, one per fixture combination.
func expandRoute(t *testing.T, pattern string) []string {
	t.Helper()
	paths := []string{pattern}
	for _, m := range pathParam.FindAllStringSubmatch(pattern, -1) {
		vals, ok := sweepParams[m[1]]
		if !ok {
			t.Fatalf("applied route %s has path parameter {%s} with no sweep fixture: add one to sweepParams", pattern, m[1])
		}
		var next []string
		for _, p := range paths {
			for _, v := range vals {
				next = append(next, strings.Replace(p, m[0], v, 1))
			}
		}
		paths = next
	}
	return paths
}

// liveLeaks returns the sentinels of the live items (1, 2) found in body.
func liveLeaks(body string) []string {
	var out []string
	for _, id := range []string{"1", "2"} {
		for _, kind := range sentinelKinds {
			if s := leakMark(kind, id); strings.Contains(body, s) {
				out = append(out, s)
			}
		}
	}
	return out
}

func TestWithholdSweepAppliedRoutes(t *testing.T) {
	h := newWithholdHarness(t)
	swept := 0
	for _, rt := range (&Gateway{}).apiRoutes() {
		if rt.Withhold.Mode != withholdApplied || rt.Method != http.MethodGet {
			continue
		}
		workspace := strings.Contains(rt.Pattern, "{id}")
		for _, p := range expandRoute(t, rt.Pattern) {
			variants := []string{p}
			if workspace {
				variants = append(variants, p+"?practice=1") // the arena GET
			}
			for _, v := range variants {
				swept++
				status, body := h.do(t, http.MethodGet, v, "")
				if status != http.StatusOK {
					t.Errorf("GET %s: status %d (%s)", v, status, body)
					continue
				}
				if leaks := liveLeaks(body); len(leaks) > 0 {
					t.Errorf("GET %s leaks live-item fields %v", v, leaks)
				}
				// Not over-withholding: item 3 (solved blind, not due) keeps its pattern on
				// every list and in its own workspace and arena views.
				if (!workspace || strings.HasSuffix(strings.TrimSuffix(v, "?practice=1"), "/3")) &&
					!strings.Contains(body, leakMark("PATTERN", "3")) {
					t.Errorf("GET %s over-withholds: item 3's pattern (solved blind, not due) is missing", v)
				}
			}
		}
	}
	if swept < 18 { // 10 list routes (course-scoped + aliases) + 4 workspace + 4 arena views
		t.Fatalf("swept only %d requests; the applied GET routes shrank?", swept)
	}
}

// TestWithholdWorkspaceRules pins the workspace/arena rows of the rule table end to end.
func TestWithholdWorkspaceRules(t *testing.T) {
	h := newWithholdHarness(t)
	type view struct {
		Problem  map[string]any `json:"problem"`
		Sections []struct {
			Stage string `json:"stage"`
			Kind  string `json:"kind"`
		} `json:"sections"`
	}
	get := func(p string) view {
		status, body := h.do(t, http.MethodGet, p, "")
		if status != http.StatusOK {
			t.Fatalf("GET %s: %d %s", p, status, body)
		}
		var v view
		if err := json.Unmarshal([]byte(body), &v); err != nil {
			t.Fatalf("GET %s: %v", p, err)
		}
		return v
	}
	stages := func(v view) string {
		var s []string
		for _, sec := range v.Sections {
			s = append(s, sec.Stage+":"+sec.Kind)
		}
		sort.Strings(s)
		return strings.Join(s, ",")
	}
	cases := []struct {
		path        string
		wantPattern bool
		wantStages  string
	}{
		{"/api/problems/1", false, "attempt:summary"},            // open attempt, attempt stage: no chip
		{"/api/problems/2", false, "attempt:summary"},            // due touch: nothing but the statement
		{"/api/problems/3", true, "attempt:summary"},             // solved blind: v1's chip, no hint/solution
		{"/api/problems/4", false, "attempt:summary"},            // never solved: chip from the hint stage
		{"/api/problems/1?practice=1", false, "attempt:summary"}, // live arena: attempt stage only
		{"/api/problems/2?practice=1", false, "attempt:summary"}, // live arena
		{"/api/problems/3?practice=1", true, "attempt:summary,hint:key_observation,solution:code,solution:solution_facts"},
		{"/api/problems/4?practice=1", true, "attempt:summary,hint:key_observation,solution:code,solution:solution_facts"},
	}
	for _, c := range cases {
		v := get(c.path)
		_, hasPattern := v.Problem["pattern"]
		_, hasConcepts := v.Problem["concepts"]
		if hasPattern != c.wantPattern || hasConcepts != c.wantPattern {
			t.Errorf("%s: pattern=%v concepts=%v, want %v", c.path, hasPattern, hasConcepts, c.wantPattern)
		}
		if got := stages(v); got != c.wantStages {
			t.Errorf("%s: sections %q, want %q", c.path, got, c.wantStages)
		}
	}
}

func TestWithholdCoachContext(t *testing.T) {
	h := newWithholdHarness(t)
	chat := func(id string) string {
		h.mu.Lock()
		h.coach = nil
		h.mu.Unlock()
		// A spoofing client sends every withheld key itself; none may pass through.
		body := `{"context":"problem:` + id + `","message":"help","pattern":"` + leakMark("PATTERN", id) +
			`","concepts":["` + leakMark("CONCEPT", id) + `"],"solution_facts":{"x":"` + leakMark("FACTS", id) + `"}}`
		status, out := h.do(t, http.MethodPost, "/api/coach/chat", body)
		if status != http.StatusOK {
			t.Fatalf("chat %s: %d %s", id, status, out)
		}
		h.mu.Lock()
		defer h.mu.Unlock()
		if len(h.coach) != 1 {
			t.Fatalf("chat %s: coach received %d requests", id, len(h.coach))
		}
		return h.coach[0]
	}
	for _, id := range []string{"1", "2", "4"} { // live (attempt), live (due), never solved
		got := chat(id)
		for _, kind := range []string{"PATTERN", "CONCEPT", "FACTS"} {
			if strings.Contains(got, leakMark(kind, id)) {
				t.Errorf("coach context for problem %s carries %s: %s", id, leakMark(kind, id), got)
			}
		}
		if !strings.Contains(got, `"problemTitle":"Problem `+id+`"`) {
			t.Errorf("coach context for problem %s lost its title: %s", id, got)
		}
	}
	// Solved and not live: the authoritative pattern is kept (review mode, as v1).
	if got := chat("3"); !strings.Contains(got, `"pattern":"`+leakMark("PATTERN", "3")+`"`) {
		t.Errorf("coach context for solved problem 3 over-withholds: %s", got)
	}
}

func TestWithholdScoreRevealsPattern(t *testing.T) {
	h := newWithholdHarness(t)
	score := func(itemID string) map[string]any {
		status, body := h.do(t, http.MethodPost, "/api/revision/"+itemID+"/score", `{"namedPatternSecs":30,"solvedInTimer":true,"statedComplexity":true}`)
		if status != http.StatusOK {
			t.Fatalf("score %s: %d %s", itemID, status, body)
		}
		var res struct {
			AutoPass bool           `json:"autoPass"`
			Problem  map[string]any `json:"problem"`
		}
		if err := json.Unmarshal([]byte(body), &res); err != nil || !res.AutoPass || res.Problem == nil {
			t.Fatalf("score %s: bad result %s", itemID, body)
		}
		return res.Problem
	}
	// The touch concluded and item 3 isn't live: the scored result reveals the pattern.
	if p := score("rev-3"); p["pattern"] != leakMark("PATTERN", "3") {
		t.Errorf("scored item 3 should reveal its pattern, got %v", p)
	}
	// Still live another way (1: an open attempt; 2: the fake still lists a due touch).
	for _, it := range []string{"rev-1", "rev-2"} {
		if p := score(it); p["pattern"] != nil || p["concepts"] != nil {
			t.Errorf("scored %s is still live; pattern must stay withheld, got %v", it, p)
		}
	}
}

func TestWithholdFailsClosed(t *testing.T) {
	for _, down := range []string{"practice", "review"} {
		t.Run(down+" down", func(t *testing.T) {
			h := newWithholdHarness(t)
			h.practiceDown, h.reviewDown = down == "practice", down == "review"
			for _, p := range []string{"/api/paths/dsa/problems", "/api/paths/dsa/weeks/1", "/api/paths/dsa/mistakes", "/api/problems/3", "/api/problems/4?practice=1"} {
				status, body := h.do(t, http.MethodGet, p, "")
				if status != http.StatusOK {
					t.Fatalf("GET %s: %d %s", p, status, body)
				}
				for _, id := range withholdItems {
					for _, kind := range []string{"PATTERN", "CONCEPT", "HINT", "SOLUTION", "FACTS"} {
						if strings.Contains(body, leakMark(kind, id)) {
							t.Errorf("%s down: GET %s shows %s (must fail closed)", down, p, leakMark(kind, id))
						}
					}
				}
			}
		})
	}
}
