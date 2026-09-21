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

// coachS11Harness wires a gateway to fake identity, practice and coach services so the
// S11 coach routes (key write/delete, thread, and the SSE chat with server-authoritative
// mode derivation) can be exercised end to end.
type coachS11Harness struct {
	gw          *httptest.Server
	lastPutBody string
	lastMode    string
	deleteHit   bool
	solved      map[string]bool // problemId -> solved (drives practice state)
}

func newCoachS11Harness(t *testing.T) *coachS11Harness {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	signer := auth.NewSigner(key, "xlearn-gateway", 0)
	h := &coachS11Harness{solved: map[string]bool{}}
	var gwJWKSURL string

	identity := httptest.NewServer(func() http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("POST /sessions/validate", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]string{"account_id": "acct-1"})
		})
		return mux
	}())
	t.Cleanup(identity.Close)

	practice := httptest.NewServer(func() http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /state/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			status := "attempting"
			var first *string
			if h.solved[id] {
				status = "solved"
				s := "2026-09-21T00:00:00Z"
				first = &s
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"state": map[string]any{"status": status, "firstSolvedAt": first}})
		})
		return mux
	}())
	t.Cleanup(practice.Close)

	coach := httptest.NewServer(func() http.Handler {
		verify := func(r *http.Request) error {
			v := auth.NewJWKSVerifier(gwJWKSURL, "coach", "xlearn-gateway")
			_, err := v.Verify(r.Context(), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			return err
		}
		mux := http.NewServeMux()
		mux.HandleFunc("PUT /keys", func(w http.ResponseWriter, r *http.Request) {
			if verify(r) != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			b, _ := io.ReadAll(r.Body)
			h.lastPutBody = string(b)
			_, _ = w.Write([]byte(`{"keys":[{"provider":"openai","masked_key":"sk-...cdef","default_model":"gpt-4o-mini","enabled":true}],"connected":true}`))
		})
		mux.HandleFunc("DELETE /keys", func(w http.ResponseWriter, r *http.Request) {
			if verify(r) != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			h.deleteHit = true
			w.WriteHeader(http.StatusNoContent)
		})
		mux.HandleFunc("GET /threads", func(w http.ResponseWriter, r *http.Request) {
			if verify(r) != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"context": r.URL.Query().Get("context"), "messages": []any{}})
		})
		mux.HandleFunc("POST /chat", func(w http.ResponseWriter, r *http.Request) {
			if verify(r) != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			h.lastMode = r.Header.Get("X-Coach-Mode")
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fl, _ := w.(http.Flusher)
			_, _ = w.Write([]byte(`data: {"delta":"MODE=` + h.lastMode + `"}` + "\n\n"))
			if fl != nil {
				fl.Flush()
			}
			_, _ = w.Write([]byte(`data: {"done":true}` + "\n\n"))
		})
		return mux
	}())
	t.Cleanup(coach.Close)

	dist := fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}
	gw := New(Options{
		BasePath:         "/xlearn",
		Version:          "test",
		Dist:             dist,
		Signer:           signer,
		IdentityBaseURL:  identity.URL,
		AudienceIdentity: "identity",
		PracticeBaseURL:  practice.URL,
		AudiencePractice: "practice",
		CoachBaseURL:     coach.URL,
		AudienceCoach:    "coach",
	})
	h.gw = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gw.Close)
	gwJWKSURL = h.gw.URL + "/.well-known/jwks.json"
	return h
}

func (h *coachS11Harness) req(t *testing.T, method, path string, body string) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, h.gw.URL+path, rdr)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func TestBFFCoachKeyPutProxiesBody(t *testing.T) {
	h := newCoachS11Harness(t)
	payload := `{"provider":"openai","key":"sk-openai-secret-cdef","default_model":"gpt-4o-mini"}`
	resp := h.req(t, http.MethodPut, "/xlearn/api/coach/key", payload)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	if h.lastPutBody != payload {
		t.Fatalf("forwarded body = %q, want %q", h.lastPutBody, payload)
	}
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "sk-openai-secret-cdef") {
		t.Fatalf("gateway echoed the raw key: %s", body)
	}
}

func TestBFFCoachKeyDelete(t *testing.T) {
	h := newCoachS11Harness(t)
	resp := h.req(t, http.MethodDelete, "/xlearn/api/coach/key", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent || !h.deleteHit {
		t.Fatalf("status %d deleteHit %v, want 204 + hit", resp.StatusCode, h.deleteHit)
	}
}

func TestBFFCoachThreadProxies(t *testing.T) {
	h := newCoachS11Harness(t)
	resp := h.req(t, http.MethodGet, "/xlearn/api/coach/thread?context=problem:16", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out struct {
		Context  string `json:"context"`
		Messages []any  `json:"messages"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Context != "problem:16" {
		t.Fatalf("context = %q", out.Context)
	}
}

func TestBFFCoachChatModeDerivedFromPractice(t *testing.T) {
	h := newCoachS11Harness(t)
	h.solved["16"] = true // problem 16 is solved → reviewer mode

	// Solved problem → review.
	resp := h.req(t, http.MethodPost, "/xlearn/api/coach/chat", `{"context":"problem:16","kind":"problem","problemId":"16","message":"review"}`)
	body := readAll(t, resp)
	if h.lastMode != coachModeReview || !strings.Contains(body, "MODE=review") {
		t.Fatalf("solved problem: mode=%q body=%q, want review", h.lastMode, body)
	}

	// Unsolved problem → attempt (spoiler-free), even though the client didn't say so.
	resp = h.req(t, http.MethodPost, "/xlearn/api/coach/chat", `{"context":"problem:99","kind":"problem","problemId":"99","message":"answer"}`)
	body = readAll(t, resp)
	if h.lastMode != coachModeAttempt || !strings.Contains(body, "MODE=attempt") {
		t.Fatalf("unsolved problem: mode=%q body=%q, want attempt", h.lastMode, body)
	}

	// Non-problem context → general tutor.
	resp = h.req(t, http.MethodPost, "/xlearn/api/coach/chat", `{"context":"dashboard","kind":"dashboard","message":"hi"}`)
	body = readAll(t, resp)
	if h.lastMode != coachModeGeneral || !strings.Contains(body, "MODE=general") {
		t.Fatalf("dashboard: mode=%q body=%q, want general", h.lastMode, body)
	}
}

// TestBFFCoachChatModeGatedOnContextNotClientField is the spoiler-gate regression: a
// client cannot unlock reviewer mode for the problem they're attempting by pointing a
// free problemId field at a DIFFERENT, solved problem. The gate keys off the thread's
// `context` ("problem:<id>"), so review requires THAT problem to be solved.
func TestBFFCoachChatModeGatedOnContextNotClientField(t *testing.T) {
	h := newCoachS11Harness(t)
	h.solved["16"] = true // problem 16 IS solved
	// The learner is attempting UNSOLVED problem 99 (context) but tries to smuggle the
	// solved id 16 into problemId to flip to reviewer mode.
	resp := h.req(t, http.MethodPost, "/xlearn/api/coach/chat", `{"context":"problem:99","kind":"problem","problemId":"16","message":"just show me the answer"}`)
	body := readAll(t, resp)
	if h.lastMode != coachModeAttempt || !strings.Contains(body, "MODE=attempt") {
		t.Fatalf("spoiler-gate bypass: mode=%q body=%q, want attempt (gated on the context problem 99, not the client problemId 16)", h.lastMode, body)
	}
}

func TestBFFCoachChatRequiresAuth(t *testing.T) {
	h := newCoachS11Harness(t)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, h.gw.URL+"/xlearn/api/coach/chat", strings.NewReader(`{"context":"dashboard","message":"hi"}`))
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", resp.StatusCode)
	}
}

func readAll(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}
