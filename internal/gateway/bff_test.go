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

// bffHarness wires a real gateway (real RSA signer + JWKS) to a fake identity that
// verifies the gateway-minted JWT against the gateway's real JWKS — the end-to-end
// mint→forward→JWKS-verify path (ADR-0006, acceptance #3).
type bffHarness struct {
	gwServer           *httptest.Server
	verifyErr          error // last error the fake identity's verifier returned
	lastRevoke         string
	lastCurriculumPath string // last path the fake curriculum service received
	lastPatchBody      string // last PATCH /accounts/{id} body the fake identity received
}

func newBFFHarness(t *testing.T) *bffHarness {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	signer := auth.NewSigner(key, "xlearn-gateway", 0)

	h := &bffHarness{}
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
	identityMux.HandleFunc("POST /sessions/revoke", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SessionID string `json:"session_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		h.lastRevoke = body.SessionID
		w.WriteHeader(http.StatusNoContent)
	})
	identityMux.HandleFunc("GET /accounts/{id}", func(w http.ResponseWriter, r *http.Request) {
		v := auth.NewJWKSVerifier(gwJWKSURL, "identity", "xlearn-gateway")
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		claims, err := v.Verify(r.Context(), token)
		h.verifyErr = err
		if err != nil || claims.Subject != r.PathValue("id") {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"account":    map[string]any{"id": claims.Subject, "display_name": "Ada"},
			"onboarding": map[string]any{"path_chosen": nil, "budget_set": false, "completed": false},
		})
	})
	identityMux.HandleFunc("PATCH /accounts/{id}", func(w http.ResponseWriter, r *http.Request) {
		v := auth.NewJWKSVerifier(gwJWKSURL, "identity", "xlearn-gateway")
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		claims, err := v.Verify(r.Context(), token)
		h.verifyErr = err
		if err != nil || claims.Subject != r.PathValue("id") {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		h.lastPatchBody = string(body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"account":    map[string]any{"id": claims.Subject, "display_name": "Renamed", "timezone": "Asia/Kolkata", "study_budget": map[string]any{}, "reminders": map[string]any{}},
			"onboarding": map[string]any{"path_chosen": "dsa", "budget_set": true, "completed": false},
		})
	})
	identityMux.HandleFunc("POST /onboarding/step", func(w http.ResponseWriter, r *http.Request) {
		v := auth.NewJWKSVerifier(gwJWKSURL, "identity", "xlearn-gateway")
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if _, err := v.Verify(r.Context(), token); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"onboarding": map[string]any{"path_chosen": "dsa"}})
	})
	identityMux.HandleFunc("POST /auth/{provider}/start", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "xl_oauthtx", Value: "tx", Path: "/xlearn"})
		w.Header().Set("Location", "https://github.com/login/oauth/authorize?x=1")
		w.WriteHeader(http.StatusFound)
	})
	identity := httptest.NewServer(identityMux)
	t.Cleanup(identity.Close)

	// Fake curriculum service: records the last path it was asked for and echoes a
	// tiny content payload. Curriculum takes no user JWT, so no verification here.
	curriculumMux := http.NewServeMux()
	curriculumMux.HandleFunc("GET /paths", func(w http.ResponseWriter, r *http.Request) {
		h.lastCurriculumPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{"paths": []any{map[string]any{"slug": "dsa"}}})
	})
	curriculumMux.HandleFunc("GET /paths/{slug}", func(w http.ResponseWriter, r *http.Request) {
		h.lastCurriculumPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{"path": map[string]any{"slug": r.PathValue("slug")}})
	})
	// Week content for the BFF aggregation test: a core Medium + a reinforcement Easy.
	// Week 99 is unknown → curriculum's own 404 envelope, which the gateway propagates.
	curriculumMux.HandleFunc("GET /paths/{slug}/weeks/{n}", func(w http.ResponseWriter, r *http.Request) {
		h.lastCurriculumPath = r.URL.Path
		if r.PathValue("n") == "99" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"resource not found"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"week":     map[string]any{"n": 2, "title": "Two Pointers", "thesis": "t2"},
			"phase":    map[string]any{"order": 1, "name": "Fundamentals"},
			"path":     map[string]any{"slug": r.PathValue("slug"), "week_total": 16},
			"concepts": []any{map[string]any{"slug": "two-pointers", "title": "Two Pointers"}},
			"problems": []any{
				map[string]any{"id": "16", "difficulty": "med", "is_reinforcement": false},
				map[string]any{"id": "3", "difficulty": "easy", "is_reinforcement": true},
			},
		})
	})
	curriculum := httptest.NewServer(curriculumMux)
	t.Cleanup(curriculum.Close)

	dist := fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>xLearn</title>")}}
	gw := New(Options{
		BasePath:          "/xlearn",
		Version:           "test",
		Dist:              dist,
		Signer:            signer,
		IdentityBaseURL:   identity.URL,
		AudienceIdentity:  "identity",
		CurriculumBaseURL: curriculum.URL,
	})
	h.gwServer = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gwServer.Close)
	gwJWKSURL = h.gwServer.URL + "/.well-known/jwks.json"
	return h
}

func (h *bffHarness) get(t *testing.T, path string, cookie *http.Cookie) *http.Response {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, h.gwServer.URL+path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func noRedirect() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func TestBFFMeUnauthenticated(t *testing.T) {
	h := newBFFHarness(t)
	resp := h.get(t, "/xlearn/api/me", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status %d, want 401", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"unauthenticated"`) {
		t.Fatalf("want unauthenticated envelope, got %s", body)
	}
}

func TestBFFMeEndToEndJWKS(t *testing.T) {
	h := newBFFHarness(t)
	resp := h.get(t, "/xlearn/api/me", &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200 (verifyErr=%v)", resp.StatusCode, h.verifyErr)
	}
	if h.verifyErr != nil {
		t.Fatalf("identity failed to verify the gateway JWT via JWKS: %v", h.verifyErr)
	}
	var me struct {
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
	}
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &me)
	if me.Account.ID != "acct-1" {
		t.Fatalf("/me account = %q, body=%s", me.Account.ID, body)
	}
}

func TestBFFMeInvalidSession(t *testing.T) {
	h := newBFFHarness(t)
	resp := h.get(t, "/xlearn/api/me", &http.Cookie{Name: auth.SessionCookieName, Value: "bogus"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid session: status %d, want 401", resp.StatusCode)
	}
}

func TestBFFJWKSServed(t *testing.T) {
	h := newBFFHarness(t)
	for _, path := range []string{"/.well-known/jwks.json", "/xlearn/.well-known/jwks.json"} {
		resp := h.get(t, path, nil)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: status %d", path, resp.StatusCode)
		}
		if !strings.Contains(string(body), `"keys"`) || !strings.Contains(string(body), `"RS256"`) {
			t.Fatalf("%s: not a JWK set: %s", path, body)
		}
	}
}

func TestBFFLogout(t *testing.T) {
	h := newBFFHarness(t)
	req, _ := http.NewRequest(http.MethodPost, h.gwServer.URL+"/xlearn/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status %d, want 204", resp.StatusCode)
	}
	if h.lastRevoke != "sess-1" {
		t.Fatalf("identity revoke not called with session; got %q", h.lastRevoke)
	}
	if !clearsSession(resp.Cookies()) {
		t.Fatalf("logout did not clear the session cookie")
	}
}

func TestBFFAuthStartProxy(t *testing.T) {
	h := newBFFHarness(t)
	req, _ := http.NewRequest(http.MethodPost, h.gwServer.URL+"/xlearn/api/auth/github/start", nil)
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("start status %d, want 302", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Location"), "github.com/login/oauth/authorize") {
		t.Fatalf("start did not proxy provider redirect: %q", resp.Header.Get("Location"))
	}
	if !hasCookieNamed(resp.Cookies(), "xl_oauthtx") {
		t.Fatalf("start did not pass through the tx cookie")
	}
}

func TestBFFCurriculumRequiresSession(t *testing.T) {
	h := newBFFHarness(t)
	resp := h.get(t, "/xlearn/api/paths", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status %d, want 401", resp.StatusCode)
	}
	if h.lastCurriculumPath != "" {
		t.Fatalf("curriculum should not be called without a session; got %q", h.lastCurriculumPath)
	}
}

func TestBFFCurriculumProxiesPaths(t *testing.T) {
	h := newBFFHarness(t)
	resp := h.get(t, "/xlearn/api/paths", &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	if h.lastCurriculumPath != "/paths" {
		t.Fatalf("curriculum path = %q, want /paths", h.lastCurriculumPath)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"dsa"`) {
		t.Fatalf("expected proxied content, got %s", body)
	}
}

func TestBFFCurriculumProxiesPathBySlug(t *testing.T) {
	h := newBFFHarness(t)
	resp := h.get(t, "/xlearn/api/paths/dsa", &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	if h.lastCurriculumPath != "/paths/dsa" {
		t.Fatalf("curriculum path = %q, want /paths/dsa", h.lastCurriculumPath)
	}
}

func TestBFFWeekRejectsNonInteger(t *testing.T) {
	h := newBFFHarness(t)
	resp := h.get(t, "/xlearn/api/paths/dsa/weeks/abc", &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("non-integer week: status %d, want 400", resp.StatusCode)
	}
	if h.lastCurriculumPath != "" {
		t.Fatalf("curriculum should not be called for a bad week; got %q", h.lastCurriculumPath)
	}
}

// TestBFFWeekAggregatesUserState is the S04 acceptance: the week route stitches the
// curriculum content to a placeholder userState with the FROZEN shape (per-problem
// touches[5] + week rollup, populated:false) that S05/S06 will fill (ADR-0013).
func TestBFFWeekAggregatesUserState(t *testing.T) {
	h := newBFFHarness(t)
	resp := h.get(t, "/xlearn/api/paths/dsa/weeks/2", &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	if h.lastCurriculumPath != "/paths/dsa/weeks/2" {
		t.Fatalf("curriculum path = %q, want /paths/dsa/weeks/2", h.lastCurriculumPath)
	}

	var out struct {
		Week      map[string]any `json:"week"`
		Concepts  []any          `json:"concepts"`
		Problems  []any          `json:"problems"`
		UserState struct {
			Week struct {
				Solved       int `json:"solved"`
				CoreTotal    int `json:"coreTotal"`
				ByDifficulty struct {
					Easy int `json:"easy"`
					Med  int `json:"med"`
					Hard int `json:"hard"`
				} `json:"byDifficulty"`
				Populated bool `json:"populated"`
			} `json:"week"`
			Problems map[string]struct {
				Status       string  `json:"status"`
				LastOutcome  *string `json:"lastOutcome"`
				CurrentTouch int     `json:"currentTouch"`
				Touches      []struct {
					Level   int     `json:"level"`
					DueDate *string `json:"dueDate"`
					Result  string  `json:"result"`
				} `json:"touches"`
			} `json:"problems"`
		} `json:"userState"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("response not JSON: %v (%s)", err, body)
	}

	// Curriculum content is preserved verbatim.
	if out.Week["n"].(float64) != 2 || len(out.Concepts) != 1 || len(out.Problems) != 2 {
		t.Fatalf("curriculum content not preserved: %s", body)
	}

	// Week rollup: 1 core Medium (the Easy is reinforcement → not counted), 0 solved,
	// populated:false so S05/S06 can flip it.
	wk := out.UserState.Week
	if wk.Solved != 0 || wk.CoreTotal != 1 || wk.Populated {
		t.Fatalf("week rollup wrong: %+v", wk)
	}
	if wk.ByDifficulty.Med != 1 || wk.ByDifficulty.Easy != 0 || wk.ByDifficulty.Hard != 0 {
		t.Fatalf("byDifficulty should count only the core Medium: %+v", wk.ByDifficulty)
	}

	// Every problem (core AND reinforcement) gets an honest available state with five
	// empty touches (levels 1..5, no due date, no result).
	if len(out.UserState.Problems) != 2 {
		t.Fatalf("expected per-problem state for both problems, got %d", len(out.UserState.Problems))
	}
	ps, ok := out.UserState.Problems["16"]
	if !ok {
		t.Fatalf("missing state for problem 16: %s", body)
	}
	if ps.Status != "available" || ps.LastOutcome != nil || ps.CurrentTouch != 0 {
		t.Fatalf("problem 16 state should be honest/empty: %+v", ps)
	}
	if len(ps.Touches) != 5 {
		t.Fatalf("expected 5 touches, got %d", len(ps.Touches))
	}
	for i, tch := range ps.Touches {
		if tch.Level != i+1 || tch.DueDate != nil || tch.Result != "none" {
			t.Fatalf("touch %d not empty: %+v", i, tch)
		}
	}
}

func TestBFFWeekPropagatesNotFound(t *testing.T) {
	h := newBFFHarness(t)
	resp := h.get(t, "/xlearn/api/paths/dsa/weeks/99", &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown week: status %d, want 404", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"not_found"`) {
		t.Fatalf("want curriculum not_found envelope, got %s", body)
	}
}

func clearsSession(cs []*http.Cookie) bool {
	for _, c := range cs {
		if c.Name == auth.SessionCookieName && c.MaxAge < 0 {
			return true
		}
	}
	return false
}

func hasCookieNamed(cs []*http.Cookie, name string) bool {
	for _, c := range cs {
		if c.Name == name {
			return true
		}
	}
	return false
}
