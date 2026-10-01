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
	"sync"
	"testing"
	"testing/fstest"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// TestBFFPatchMeForwardsBody exercises PATCH /me → identity PATCH /accounts/{id} with a
// JWKS-verified, subject-scoped JWT, and checks the body is forwarded verbatim.
func TestBFFPatchMeForwardsBody(t *testing.T) {
	h := newBFFHarness(t)
	payload := `{"display_name":"Renamed","timezone":"Asia/Kolkata"}`
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPatch, h.gwServer.URL+"/xlearn/api/me", strings.NewReader(payload))
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	req.Header.Set("Content-Type", "application/json")
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("PATCH /me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200 (verifyErr=%v)", resp.StatusCode, h.verifyErr)
	}
	if h.verifyErr != nil {
		t.Fatalf("identity failed to verify the gateway JWT: %v", h.verifyErr)
	}
	if h.lastPatchBody != payload {
		t.Fatalf("forwarded body = %q, want %q", h.lastPatchBody, payload)
	}
}

// TestBFFPatchMeUnauthenticated: no session cookie → 401, identity never called.
func TestBFFPatchMeUnauthenticated(t *testing.T) {
	h := newBFFHarness(t)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPatch, h.gwServer.URL+"/xlearn/api/me", strings.NewReader(`{"display_name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("PATCH /me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", resp.StatusCode)
	}
	if h.lastPatchBody != "" {
		t.Fatalf("identity was called for an unauthenticated PATCH")
	}
}

// TestBFFCoachKeyEmptyStateWhenUnconfigured: with no coach service (the S10 default),
// GET /coach/key returns the "no key — coach off" empty state, not an error.
func TestBFFCoachKeyEmptyStateWhenUnconfigured(t *testing.T) {
	h := newBFFHarness(t) // no CoachBaseURL → g.coach == nil
	resp := h.get(t, "/xlearn/api/coach/key", &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	var out struct {
		Keys      []any `json:"keys"`
		Connected bool  `json:"connected"`
	}
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &out)
	if len(out.Keys) != 0 || out.Connected {
		t.Fatalf("empty state = %s, want {keys:[],connected:false}", body)
	}
}

// TestBFFCoachKeyRequiresAuth: no session cookie → 401.
func TestBFFCoachKeyRequiresAuth(t *testing.T) {
	h := newBFFHarness(t)
	resp := h.get(t, "/xlearn/api/coach/key", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", resp.StatusCode)
	}
}

// coachHarness wires a gateway to a fake coach service so the proxy branch (S11 shape)
// is exercised: it verifies the coach-scoped JWT against the gateway's JWKS.
//
// m1-07 extends it with fakes of every upstream the coach mode gate reads — curriculum,
// practice (GET /attempts/open, POST /attempts/{id}/assist), review (the due queue) and
// assessment (GET /mocks/live) — plus coach's GET /admission, /threads and /chat. The
// knobs below set the story; calls records every gate-relevant upstream call in order.
type coachHarness struct {
	gwServer  *httptest.Server
	verifyErr error
	keyStatus int
	keyBody   string

	mu    sync.Mutex
	calls []string // "practice POST /attempts/att-16/assist", "coach POST /chat", …

	// practice: the open attempts (filtered by ?problem_id= like the real service) and
	// the solved set behind the `problem` slot.
	attempts []map[string]any
	solved   map[string]bool
	// review: problem ids with a touch due now.
	due []string
	// assessment: the live mock (nil → {"live": null}).
	liveMock map[string]any
	// Status overrides (0 = 200 / 204 as appropriate).
	practiceStatus, assistStatus, reviewStatus, assessmentStatus int
	// coach /admission: status (0 = 204), Retry-After and body for a 429.
	admissionStatus     int
	admissionRetryAfter string
	// coach /chat: status (0 = 200 SSE), Retry-After, and what it saw.
	chatStatus     int
	chatRetryAfter string
	chatHeaders    http.Header
	chatBody       string
	chatQuery      string
}

// openAttemptFor builds a practice /attempts/open entry (coachAssistAt "" = null).
func openAttemptFor(id, problemID, coachAssistAt string) map[string]any {
	a := map[string]any{"attemptId": id, "problemId": problemID, "pathSlug": "dsa", "purpose": "course",
		"startedAt": "2026-10-01T08:00:00Z", "stageReached": "attempt", "coachAssistAt": nil}
	if coachAssistAt != "" {
		a["coachAssistAt"] = coachAssistAt
	}
	return a
}

func (h *coachHarness) record(call string) {
	h.mu.Lock()
	h.calls = append(h.calls, call)
	h.mu.Unlock()
}

// called returns the recorded calls (a copy) and resets the log.
func (h *coachHarness) called() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := append([]string(nil), h.calls...)
	h.calls = nil
	return out
}

func newCoachHarness(t *testing.T) *coachHarness {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	signer := auth.NewSigner(key, "xlearn-gateway", 0)
	h := &coachHarness{keyStatus: http.StatusOK, keyBody: `{"keys":[{"provider":"Anthropic","masked_key":"sk-ant-****4a2f","default_model":"claude-sonnet-5","enabled":true}],"connected":true}`,
		solved: map[string]bool{}}
	var gwJWKSURL string
	writeJSON := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	serve := func(mux *http.ServeMux) string {
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		return srv.URL
	}

	identityMux := http.NewServeMux()
	identityMux.HandleFunc("POST /sessions/validate", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"account_id": "acct-1"})
	})

	problem := func(id string) map[string]any {
		return map[string]any{"id": id, "path_slug": "dsa", "title": "Problem " + id, "pattern": "PATTERN-" + id}
	}
	curriculumMux := http.NewServeMux()
	curriculumMux.HandleFunc("GET /problems/{id}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, map[string]any{"problem": problem(r.PathValue("id")), "sections": []any{}})
	})
	curriculumMux.HandleFunc("GET /problems", func(w http.ResponseWriter, r *http.Request) {
		ps := []any{}
		for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
			if id != "" {
				ps = append(ps, problem(id))
			}
		}
		writeJSON(w, 0, map[string]any{"problems": ps})
	})

	practiceMux := http.NewServeMux()
	practiceMux.HandleFunc("GET /attempts/open", func(w http.ResponseWriter, r *http.Request) {
		h.record("practice GET /attempts/open?" + r.URL.RawQuery)
		if h.practiceStatus != 0 {
			writeJSON(w, h.practiceStatus, map[string]any{"error": map[string]any{"code": "internal"}})
			return
		}
		pid := r.URL.Query().Get("problem_id")
		list := []any{}
		for _, a := range h.attempts {
			if pid == "" || a["problemId"] == pid {
				list = append(list, a)
			}
		}
		out := map[string]any{"attempts": list}
		if pid != "" {
			status, first := "available", any(nil)
			if h.solved[pid] {
				status, first = "solved", "2026-09-20T00:00:00Z"
			}
			out["problem"] = map[string]any{"problemId": pid, "status": status, "firstSolvedAt": first}
		}
		writeJSON(w, 0, out)
	})
	practiceMux.HandleFunc("POST /attempts/{id}/assist", func(w http.ResponseWriter, r *http.Request) {
		h.record("practice POST /attempts/" + r.PathValue("id") + "/assist")
		if h.assistStatus != 0 {
			writeJSON(w, h.assistStatus, map[string]any{"error": map[string]any{"code": "internal"}})
			return
		}
		writeJSON(w, 0, map[string]any{"attemptId": r.PathValue("id"), "coachAssistAt": "2026-10-01T09:00:00Z"})
	})

	reviewMux := http.NewServeMux()
	reviewMux.HandleFunc("GET /revisions/due", func(w http.ResponseWriter, r *http.Request) {
		h.record("review GET /revisions/due")
		if h.reviewStatus != 0 {
			writeJSON(w, h.reviewStatus, map[string]any{"error": map[string]any{"code": "internal"}})
			return
		}
		items := []any{}
		for _, id := range h.due {
			items = append(items, map[string]any{"itemId": "rev-" + id, "problemId": id, "due": true})
		}
		writeJSON(w, 0, map[string]any{"dueCount": len(items), "items": items})
	})

	assessmentMux := http.NewServeMux()
	assessmentMux.HandleFunc("GET /mocks/live", func(w http.ResponseWriter, r *http.Request) {
		h.record("assessment GET /mocks/live")
		if h.assessmentStatus != 0 {
			writeJSON(w, h.assessmentStatus, map[string]any{"error": map[string]any{"code": "internal"}})
			return
		}
		writeJSON(w, 0, map[string]any{"live": h.liveMock})
	})

	coachMux := http.NewServeMux()
	coachMux.HandleFunc("GET /keys", func(w http.ResponseWriter, r *http.Request) {
		v := auth.NewJWKSVerifier(gwJWKSURL, "coach", "xlearn-gateway")
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		_, err := v.Verify(r.Context(), token)
		h.verifyErr = err
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(h.keyStatus)
		_, _ = w.Write([]byte(h.keyBody))
	})
	coachMux.HandleFunc("GET /admission", func(w http.ResponseWriter, r *http.Request) {
		h.record("coach GET /admission")
		if h.admissionStatus == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if h.admissionRetryAfter != "" {
			w.Header().Set("Retry-After", h.admissionRetryAfter)
		}
		writeJSON(w, h.admissionStatus, map[string]any{"error": map[string]any{"code": "coach_rate_limited", "message": "slow down"}})
	})
	coachMux.HandleFunc("GET /threads", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, map[string]any{"context": r.URL.Query().Get("context"), "messages": []any{}})
	})
	coachMux.HandleFunc("POST /chat", func(w http.ResponseWriter, r *http.Request) {
		h.record("coach POST /chat")
		b, _ := io.ReadAll(r.Body)
		h.mu.Lock()
		h.chatHeaders, h.chatBody, h.chatQuery = r.Header.Clone(), string(b), r.URL.RawQuery
		h.mu.Unlock()
		if h.chatStatus != 0 {
			if h.chatRetryAfter != "" {
				w.Header().Set("Retry-After", h.chatRetryAfter)
			}
			writeJSON(w, h.chatStatus, map[string]any{"error": map[string]any{"code": "coach_rate_limited", "message": "slow down"}})
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"delta\":\"hi\"}\n\nevent: done\ndata: {\"done\":true,\"truncated\":false}\n\n"))
	})

	dist := fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}
	gw := New(Options{
		BasePath:           "/xlearn",
		Version:            "test",
		Dist:               dist,
		Signer:             signer,
		IdentityBaseURL:    serve(identityMux),
		AudienceIdentity:   "identity",
		CurriculumBaseURL:  serve(curriculumMux),
		PracticeBaseURL:    serve(practiceMux),
		AudiencePractice:   "practice",
		ReviewBaseURL:      serve(reviewMux),
		AudienceReview:     "review",
		AssessmentBaseURL:  serve(assessmentMux),
		AudienceAssessment: "assessment",
		CoachBaseURL:       serve(coachMux),
		AudienceCoach:      "coach",
	})
	h.gwServer = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gwServer.Close)
	gwJWKSURL = h.gwServer.URL + "/.well-known/jwks.json"
	return h
}

// do issues an authenticated request to the gateway's /xlearn/api and returns the
// response (body read) — for the m1-07 gate tests.
func (h *coachHarness) do(t *testing.T, method, path, body string) (*http.Response, string) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, h.gwServer.URL+"/xlearn/api"+path, rdr)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp, string(out)
}

func (h *coachHarness) getKey(t *testing.T) *http.Response {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, h.gwServer.URL+"/xlearn/api/coach/key", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("GET /coach/key: %v", err)
	}
	return resp
}

func TestBFFCoachKeyProxiesMasked(t *testing.T) {
	h := newCoachHarness(t)
	resp := h.getKey(t)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200 (verifyErr=%v)", resp.StatusCode, h.verifyErr)
	}
	if h.verifyErr != nil {
		t.Fatalf("coach failed to verify the gateway JWT: %v", h.verifyErr)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"connected":true`) || strings.Contains(string(body), "sk-ant-real") {
		t.Fatalf("proxied masked key = %s", body)
	}
}

func TestBFFCoachKeyNotFoundNormalizesToEmpty(t *testing.T) {
	h := newCoachHarness(t)
	h.keyStatus = http.StatusNotFound
	h.keyBody = `{"error":{"code":"not_found"}}`
	resp := h.getKey(t)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200 (404 normalized to empty state)", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"connected":false`) {
		t.Fatalf("404 → empty state expected, got %s", body)
	}
}
