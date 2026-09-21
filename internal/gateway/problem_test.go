package gateway

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// problemHarness wires a real gateway to fake identity + curriculum + practice
// services. The fake practice verifies the gateway-minted PRACTICE-scoped JWT against
// the gateway's real JWKS (ADR-0006), so these tests exercise the mint→forward→verify
// path for the practice audience, plus the Problem-workspace aggregation (section
// gating) and the attempt/reveal/outcome proxies.
type problemHarness struct {
	gwServer            *httptest.Server
	practiceAuthErr     error  // last verify error the fake practice saw
	lastPracticePath    string // last path (incl. query) the fake practice received
	lastPracticeMethod  string
	practiceStateStatus int // status the fake practice GET /state/{id} returns (default 200)
}

func newProblemHarness(t *testing.T) *problemHarness {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	signer := auth.NewSigner(key, "xlearn-gateway", 0)

	h := &problemHarness{practiceStateStatus: http.StatusOK}
	var gwJWKSURL string

	// Fake identity: session sess-1 → acct-1.
	identityMux := http.NewServeMux()
	identityMux.HandleFunc("POST /sessions/validate", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SessionID string `json:"session_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.SessionID != "sess-1" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"account_id": "acct-1"})
	})
	identity := httptest.NewServer(identityMux)
	t.Cleanup(identity.Close)

	// Fake curriculum: problem 16 with sections across all three stages, and week 2
	// content with problem 16. Curriculum takes no user JWT.
	curriculumMux := http.NewServeMux()
	curriculumMux.HandleFunc("GET /problems/{id}", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"problem": map[string]any{"id": r.PathValue("id"), "title": "3Sum", "difficulty": "med", "pattern": "Two Pointers"},
			"sections": []any{
				map[string]any{"stage": "attempt", "kind": "summary", "order": 1, "body_md": "statement", "code": ""},
				map[string]any{"stage": "hint", "kind": "key_observation", "order": 1, "body_md": "hint text", "code": ""},
				map[string]any{"stage": "solution", "kind": "code", "order": 1, "body_md": "", "code": "func threeSum(){}"},
			},
		})
	})
	curriculumMux.HandleFunc("GET /paths/{slug}/weeks/{n}", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"week":     map[string]any{"n": 2, "title": "Two Pointers"},
			"problems": []any{map[string]any{"id": "16", "difficulty": "med", "is_reinforcement": false}},
		})
	})
	curriculum := httptest.NewServer(curriculumMux)
	t.Cleanup(curriculum.Close)

	// Fake practice: verifies the practice-aud JWT, records the path/method, and
	// returns canned guided-flow responses.
	verify := func(r *http.Request) bool {
		v := auth.NewJWKSVerifier(gwJWKSURL, "practice", "xlearn-gateway")
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		claims, err := v.Verify(r.Context(), token)
		h.practiceAuthErr = err
		return err == nil && claims.Subject == "acct-1"
	}
	practiceMux := http.NewServeMux()
	practiceMux.HandleFunc("GET /state/{problemId}", func(w http.ResponseWriter, r *http.Request) {
		h.lastPracticePath, h.lastPracticeMethod = r.URL.Path, r.Method
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if h.practiceStateStatus != http.StatusOK {
			w.WriteHeader(h.practiceStateStatus)
			_, _ = w.Write([]byte(`{"error":{"code":"boom"}}`))
			return
		}
		// Attempting, only the attempt stage unlocked, timer running.
		_ = json.NewEncoder(w).Encode(map[string]any{"state": map[string]any{
			"problemId": r.PathValue("problemId"), "status": "attempting", "stageReached": "attempt",
			"unlockedStages": []string{"attempt"}, "currentTouch": 0, "lastOutcome": nil,
			"firstSolvedAt": nil, "revealedEarly": false,
			"timer": map[string]any{"kind": "attempt", "deadlineAt": "2099-01-01T00:00:00Z", "remainingSeconds": 900, "expired": false},
		}})
	})
	practiceMux.HandleFunc("GET /state", func(w http.ResponseWriter, r *http.Request) {
		h.lastPracticePath, h.lastPracticeMethod = r.URL.Path+"?"+r.URL.RawQuery, r.Method
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"states": map[string]any{
			"16": map[string]any{"problemId": "16", "status": "solved", "lastOutcome": "clean", "currentTouch": 0},
		}})
	})
	practiceMux.HandleFunc("POST /problems/{id}/attempt/start", func(w http.ResponseWriter, r *http.Request) {
		h.lastPracticePath, h.lastPracticeMethod = r.URL.Path, r.Method
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"state": map[string]any{"problemId": r.PathValue("id"), "status": "attempting"}})
	})
	practiceMux.HandleFunc("POST /problems/{id}/reveal", func(w http.ResponseWriter, r *http.Request) {
		h.lastPracticePath, h.lastPracticeMethod = r.URL.Path, r.Method
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"revealed": "solution",
			"penalty":  map[string]any{"owedAttempt": true, "dueInDays": 3, "message": "you owe #16 another attempt in 3 days"},
			"state":    map[string]any{"problemId": r.PathValue("id"), "status": "attempting", "revealedEarly": true},
		})
	})
	practiceMux.HandleFunc("POST /problems/{id}/outcome", func(w http.ResponseWriter, r *http.Request) {
		h.lastPracticePath, h.lastPracticeMethod = r.URL.Path, r.Method
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"state": map[string]any{"problemId": r.PathValue("id"), "status": "solved", "lastOutcome": "clean"}})
	})
	practice := httptest.NewServer(practiceMux)
	t.Cleanup(practice.Close)

	dist := fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>xLearn</title>")}}
	gw := New(Options{
		BasePath:          "/xlearn",
		Version:           "test",
		Dist:              dist,
		Signer:            signer,
		IdentityBaseURL:   identity.URL,
		AudienceIdentity:  "identity",
		CurriculumBaseURL: curriculum.URL,
		PracticeBaseURL:   practice.URL,
		AudiencePractice:  "practice",
	})
	h.gwServer = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gwServer.Close)
	gwJWKSURL = h.gwServer.URL + "/.well-known/jwks.json"
	return h
}

func (h *problemHarness) do(t *testing.T, method, path, body string) *http.Response {
	t.Helper()
	var r *http.Request
	if body == "" {
		r, _ = http.NewRequestWithContext(context.Background(), method, h.gwServer.URL+path, nil)
	} else {
		r, _ = http.NewRequestWithContext(context.Background(), method, h.gwServer.URL+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	resp, err := noRedirect().Do(r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func TestBFFProblemAggGatesSections(t *testing.T) {
	h := newProblemHarness(t)
	resp := h.do(t, http.MethodGet, "/xlearn/api/problems/16", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200 (practiceAuthErr=%v)", resp.StatusCode, h.practiceAuthErr)
	}
	if h.practiceAuthErr != nil {
		t.Fatalf("practice failed to verify the gateway JWT: %v", h.practiceAuthErr)
	}
	var out struct {
		Problem  map[string]any `json:"problem"`
		Sections []struct {
			Stage string `json:"stage"`
		} `json:"sections"`
		State map[string]any `json:"state"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("bad json: %v (%s)", err, body)
	}
	// R-PF1: only the unlocked (attempt) stage is delivered; hint/solution are dropped.
	if len(out.Sections) != 1 || out.Sections[0].Stage != "attempt" {
		t.Fatalf("sections not gated to attempt: %+v", out.Sections)
	}
	if out.State["status"] != "attempting" {
		t.Fatalf("state not embedded: %v", out.State)
	}
	if out.Problem["id"] != "16" {
		t.Fatalf("problem not preserved: %v", out.Problem)
	}
}

func TestBFFProblemAggPracticeDownDefaults(t *testing.T) {
	h := newProblemHarness(t)
	h.practiceStateStatus = http.StatusInternalServerError // practice unreachable/erroring
	resp := h.do(t, http.MethodGet, "/xlearn/api/problems/16", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200 (should degrade gracefully)", resp.StatusCode)
	}
	var out struct {
		Sections []struct {
			Stage string `json:"stage"`
		} `json:"sections"`
		State map[string]any `json:"state"`
	}
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &out)
	// Default state → available, statement-only.
	if len(out.Sections) != 1 || out.Sections[0].Stage != "attempt" {
		t.Fatalf("expected statement-only default, got %+v", out.Sections)
	}
	if out.State["status"] != "available" {
		t.Fatalf("expected default available state, got %v", out.State)
	}
}

func TestBFFAttemptStartProxies(t *testing.T) {
	h := newProblemHarness(t)
	resp := h.do(t, http.MethodPost, "/xlearn/api/problems/16/attempt/start", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200 (practiceAuthErr=%v)", resp.StatusCode, h.practiceAuthErr)
	}
	if h.lastPracticeMethod != http.MethodPost || h.lastPracticePath != "/problems/16/attempt/start" {
		t.Fatalf("practice not called correctly: %s %s", h.lastPracticeMethod, h.lastPracticePath)
	}
	if h.practiceAuthErr != nil {
		t.Fatalf("practice verify error: %v", h.practiceAuthErr)
	}
}

func TestBFFRevealProxiesPenalty(t *testing.T) {
	h := newProblemHarness(t)
	resp := h.do(t, http.MethodPost, "/xlearn/api/problems/16/reveal", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	var out struct {
		Revealed string `json:"revealed"`
		Penalty  *struct {
			OwedAttempt bool `json:"owedAttempt"`
		} `json:"penalty"`
	}
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &out)
	if out.Revealed != "solution" || out.Penalty == nil || !out.Penalty.OwedAttempt {
		t.Fatalf("penalty ack not passed through: %s", body)
	}
}

func TestBFFOutcomeProxies(t *testing.T) {
	h := newProblemHarness(t)
	resp := h.do(t, http.MethodPost, "/xlearn/api/problems/16/outcome", `{"outcome":"clean"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	var out struct {
		State struct {
			Status string `json:"status"`
		} `json:"state"`
	}
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &out)
	if out.State.Status != "solved" {
		t.Fatalf("outcome not proxied: %s", body)
	}
}

func TestBFFProblemWriteRequiresSession(t *testing.T) {
	h := newProblemHarness(t)
	req, _ := http.NewRequest(http.MethodPost, h.gwServer.URL+"/xlearn/api/problems/16/attempt/start", nil)
	resp, err := noRedirect().Do(req) // no session cookie
	if err != nil {
		t.Fatalf("req: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", resp.StatusCode)
	}
	if h.lastPracticePath != "" {
		t.Fatalf("practice must not be called without a session; got %q", h.lastPracticePath)
	}
}

func TestBFFWeekPopulatedFromPractice(t *testing.T) {
	h := newProblemHarness(t)
	resp := h.do(t, http.MethodGet, "/xlearn/api/paths/dsa/weeks/2", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	var out struct {
		UserState struct {
			Week struct {
				Solved    int  `json:"solved"`
				CoreTotal int  `json:"coreTotal"`
				Populated bool `json:"populated"`
			} `json:"week"`
			Problems map[string]struct {
				Status      string  `json:"status"`
				LastOutcome *string `json:"lastOutcome"`
			} `json:"problems"`
		} `json:"userState"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("bad json: %v (%s)", err, body)
	}
	if !out.UserState.Week.Populated {
		t.Fatalf("week should be populated once practice sourced state: %s", body)
	}
	if out.UserState.Week.Solved != 1 || out.UserState.Week.CoreTotal != 1 {
		t.Fatalf("rollup should reflect the solved problem: %+v", out.UserState.Week)
	}
	ps, ok := out.UserState.Problems["16"]
	if !ok || ps.Status != "solved" || ps.LastOutcome == nil || *ps.LastOutcome != "clean" {
		t.Fatalf("problem 16 should be solved/clean from practice: %+v", ps)
	}
	// The gateway asks practice for exactly the week's problem ids.
	if !strings.Contains(h.lastPracticePath, "ids=16") {
		t.Fatalf("practice /state should be scoped to the week's ids; got %q", h.lastPracticePath)
	}
}
