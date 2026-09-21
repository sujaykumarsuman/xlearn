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

// mockHarness wires a real gateway to fake identity + curriculum + assessment. The
// fake assessment verifies the gateway-minted ASSESSMENT-scoped JWT against the
// gateway's real JWKS (ADR-0006), so these tests exercise the mint→forward→verify path
// for the assessment audience plus the single-mock curriculum enrichment.
type mockHarness struct {
	gwServer        *httptest.Server
	assessAuthErr   error
	lastScorePath   string
	lastScoreBody   string
	lastStartBody   string
	lastUpstreamGet string
}

func newMockHarness(t *testing.T) *mockHarness {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	signer := auth.NewSigner(key, "xlearn-gateway", 0)

	h := &mockHarness{}
	var gwJWKSURL string

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

	titles := map[string]string{"16": "3Sum"}
	curriculumMux := http.NewServeMux()
	curriculumMux.HandleFunc("GET /problems/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		title, ok := titles[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"not_found"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"problem": map[string]any{"id": id, "title": title, "difficulty": "med", "pattern": "Two Pointers", "week_n": 2},
		})
	})
	curriculum := httptest.NewServer(curriculumMux)
	t.Cleanup(curriculum.Close)

	verify := func(r *http.Request) bool {
		v := auth.NewJWKSVerifier(gwJWKSURL, "assessment", "xlearn-gateway")
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		claims, err := v.Verify(r.Context(), token)
		h.assessAuthErr = err
		return err == nil && claims.Subject == "acct-1"
	}
	// A bare mock view (problemId only; no problem object) — the gateway enriches it.
	mockView := func(status string) map[string]any {
		return map[string]any{
			"id": "mk-1", "status": status, "setId": "set-07", "problemId": "16",
			"difficulty": "med", "date": "2026-09-21",
			"startedAt": "2026-09-21T10:00:00Z", "deadlineAt": "2026-09-21T10:45:00Z",
			"total35": nil, "notes": "",
			"rail":       map[string]any{"phaseIndex": 0, "remainingSeconds": 2700, "phases": []any{}},
			"dimensions": []any{},
			"targets":    map[string]any{"w13": 24, "w15": 28, "pre": 30},
		}
	}
	assessMux := http.NewServeMux()
	assessMux.HandleFunc("POST /mocks", func(w http.ResponseWriter, r *http.Request) {
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		b, _ := io.ReadAll(r.Body)
		h.lastStartBody = string(b)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(mockView("live"))
	})
	assessMux.HandleFunc("GET /mocks/trend", func(w http.ResponseWriter, r *http.Request) {
		h.lastUpstreamGet = r.URL.Path
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"points":  []any{map[string]any{"mockId": "mk-0", "total35": 20, "date": "2026-09-07"}},
			"targets": map[string]any{"w13": 24, "w15": 28, "pre": 30},
		})
	})
	assessMux.HandleFunc("GET /mocks/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.lastUpstreamGet = r.URL.Path
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(mockView("live"))
	})
	assessMux.HandleFunc("POST /mocks/{id}/score", func(w http.ResponseWriter, r *http.Request) {
		h.lastScorePath = r.URL.Path
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		b, _ := io.ReadAll(r.Body)
		h.lastScoreBody = string(b)
		v := mockView("scored")
		v["total35"] = 24
		_ = json.NewEncoder(w).Encode(v)
	})
	assessment := httptest.NewServer(assessMux)
	t.Cleanup(assessment.Close)

	dist := fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>xLearn</title>")}}
	gw := New(Options{
		BasePath:           "/xlearn",
		Version:            "test",
		Dist:               dist,
		Signer:             signer,
		IdentityBaseURL:    identity.URL,
		AudienceIdentity:   "identity",
		CurriculumBaseURL:  curriculum.URL,
		AssessmentBaseURL:  assessment.URL,
		AudienceAssessment: "assessment",
	})
	h.gwServer = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gwServer.Close)
	gwJWKSURL = h.gwServer.URL + "/.well-known/jwks.json"
	return h
}

func (h *mockHarness) do(t *testing.T, method, path, body string, cookie bool) *http.Response {
	t.Helper()
	var r *http.Request
	if body == "" {
		r, _ = http.NewRequestWithContext(context.Background(), method, h.gwServer.URL+path, nil)
	} else {
		r, _ = http.NewRequestWithContext(context.Background(), method, h.gwServer.URL+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if cookie {
		r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	}
	resp, err := noRedirect().Do(r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func TestBFFStartMockEnriches(t *testing.T) {
	h := newMockHarness(t)
	resp := h.do(t, http.MethodPost, "/xlearn/api/mocks", `{"setId":"set-07","problemId":"16","difficulty":"med"}`, true)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	body := decode(t, resp)
	if h.assessAuthErr != nil {
		t.Fatalf("assessment rejected the minted assessment-aud JWT: %v", h.assessAuthErr)
	}
	if !strings.Contains(h.lastStartBody, `"setId":"set-07"`) {
		t.Fatalf("start body not forwarded: %q", h.lastStartBody)
	}
	prob, _ := body["problem"].(map[string]any)
	if prob == nil || prob["title"] != "3Sum" {
		t.Fatalf("mock view not enriched with curriculum metadata: %v", body["problem"])
	}
	if body["status"] != "live" {
		t.Fatalf("status = %v, want live", body["status"])
	}
}

func TestBFFGetMockEnriches(t *testing.T) {
	h := newMockHarness(t)
	resp := h.do(t, http.MethodGet, "/xlearn/api/mocks/mk-1", "", true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if h.lastUpstreamGet != "/mocks/mk-1" {
		t.Fatalf("proxied path = %q, want /mocks/mk-1", h.lastUpstreamGet)
	}
	body := decode(t, resp)
	prob, _ := body["problem"].(map[string]any)
	if prob == nil || prob["title"] != "3Sum" {
		t.Fatalf("get mock not enriched: %v", body["problem"])
	}
}

func TestBFFScoreMockProxies(t *testing.T) {
	h := newMockHarness(t)
	resp := h.do(t, http.MethodPost, "/xlearn/api/mocks/mk-1/score",
		`{"scores":{"communication":4,"problem_understanding":4,"brute_force":3,"optimisation":3,"code_quality":4,"edge_cases":3,"complexity":3}}`, true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if h.lastScorePath != "/mocks/mk-1/score" {
		t.Fatalf("proxied path = %q, want /mocks/mk-1/score", h.lastScorePath)
	}
	if !strings.Contains(h.lastScoreBody, `"communication":4`) {
		t.Fatalf("score body not forwarded: %q", h.lastScoreBody)
	}
	body := decode(t, resp)
	if tot, _ := body["total35"].(float64); tot != 24 {
		t.Fatalf("total35 = %v, want 24", body["total35"])
	}
}

func TestBFFMockTrendRoutesAndProxies(t *testing.T) {
	h := newMockHarness(t)
	// "trend" must route to the trend handler, NOT GET /mocks/{id}.
	resp := h.do(t, http.MethodGet, "/xlearn/api/mocks/trend", "", true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if h.lastUpstreamGet != "/mocks/trend" {
		t.Fatalf("proxied path = %q, want /mocks/trend", h.lastUpstreamGet)
	}
	body := decode(t, resp)
	pts, _ := body["points"].([]any)
	if len(pts) != 1 {
		t.Fatalf("points = %d, want 1", len(pts))
	}
	if _, hasProblem := body["problem"]; hasProblem {
		t.Fatalf("trend must not be problem-enriched")
	}
}

func TestBFFMockRequiresSession(t *testing.T) {
	h := newMockHarness(t)
	for _, p := range []struct{ method, path, body string }{
		{http.MethodPost, "/xlearn/api/mocks", `{"setId":"s"}`},
		{http.MethodGet, "/xlearn/api/mocks/mk-1", ""},
		{http.MethodGet, "/xlearn/api/mocks/trend", ""},
		{http.MethodPost, "/xlearn/api/mocks/mk-1/score", `{}`},
	} {
		resp := h.do(t, p.method, p.path, p.body, false)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s: status = %d, want 401", p.method, p.path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}
