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
type coachHarness struct {
	gwServer  *httptest.Server
	verifyErr error
	keyStatus int
	keyBody   string
}

func newCoachHarness(t *testing.T) *coachHarness {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	signer := auth.NewSigner(key, "xlearn-gateway", 0)
	h := &coachHarness{keyStatus: http.StatusOK, keyBody: `{"keys":[{"provider":"Anthropic","masked_key":"sk-ant-****4a2f","default_model":"claude-sonnet-5","enabled":true}],"connected":true}`}
	var gwJWKSURL string

	identityMux := http.NewServeMux()
	identityMux.HandleFunc("POST /sessions/validate", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"account_id": "acct-1"})
	})
	identity := httptest.NewServer(identityMux)
	t.Cleanup(identity.Close)

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
	coach := httptest.NewServer(coachMux)
	t.Cleanup(coach.Close)

	dist := fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}
	gw := New(Options{
		BasePath:         "/xlearn",
		Version:          "test",
		Dist:             dist,
		Signer:           signer,
		IdentityBaseURL:  identity.URL,
		AudienceIdentity: "identity",
		CoachBaseURL:     coach.URL,
		AudienceCoach:    "coach",
	})
	h.gwServer = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gwServer.Close)
	gwJWKSURL = h.gwServer.URL + "/.well-known/jwks.json"
	return h
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
