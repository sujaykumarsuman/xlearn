package identity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

func testConfig() Config {
	return Config{
		Port:     "8081",
		LogLevel: "error",
		Auth: AuthConfig{
			PublicBaseURL: "http://localhost:8080/xlearn",
			GitHub:        OAuthClient{ClientID: "gh-id", ClientSecret: "gh-secret"},
			CookieSecure:  false,
			SessionTTL:    time.Hour,
		},
	}
}

func newTestService(st *fakeStore, v auth.Verifier) *Service {
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return NewService(testConfig(), st, v, log)
}

// fakeOAuth stands in for a provider's token + userinfo endpoints.
func fakeOAuth(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("code") == "" || r.Form.Get("code_verifier") == "" {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok-abc","token_type":"bearer"}`))
	})
	mux.HandleFunc("/gh-user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":4242,"login":"ada","name":"Ada Lovelace","email":"ada@example.com"}`))
	})
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

// wireFakeProviders points the service's providers at the fake OAuth server.
func wireFakeProviders(svc *Service, base string) {
	svc.providers["github"] = &oauthProvider{
		name: "github", authURL: base + "/authorize", tokenURL: base + "/token",
		userInfoURL: base + "/gh-user", scopes: []string{"read:user"},
		client: svc.cfg.Auth.GitHub,
	}
}

func TestHandleStart(t *testing.T) {
	svc := newTestService(newFakeStore(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/github/start", nil)
	req.SetPathValue("provider", "github")
	svc.handleStart(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status %d, want 302", rec.Code)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("bad Location: %v", err)
	}
	q := loc.Query()
	if q.Get("client_id") != "gh-id" || q.Get("state") == "" || q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		t.Fatalf("authorize URL missing params: %s", loc.String())
	}
	if !hasCookie(rec.Result().Cookies(), oauthTxCookieName) {
		t.Fatalf("expected oauth tx cookie")
	}
}

func TestHandleStartUnknownAndUnconfigured(t *testing.T) {
	svc := newTestService(newFakeStore(), nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/gitlab/start", nil)
	req.SetPathValue("provider", "gitlab")
	svc.handleStart(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown provider: status %d, want 404", rec.Code)
	}

	svc.cfg.Auth.GitHub = OAuthClient{} // unconfigured
	svc.providers = newProviders(svc.cfg.Auth)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/github/start", nil)
	req.SetPathValue("provider", "github")
	svc.handleStart(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured provider: status %d, want 503", rec.Code)
	}
}

func TestOAuthRoundTrip(t *testing.T) {
	for _, provider := range []string{"github"} {
		t.Run(provider, func(t *testing.T) {
			st := newFakeStore()
			svc := newTestService(st, nil)
			oauth := fakeOAuth(t)
			wireFakeProviders(svc, oauth.URL)

			// start → capture tx cookie + state
			startRec := httptest.NewRecorder()
			startReq := httptest.NewRequest(http.MethodPost, "/auth/"+provider+"/start", nil)
			startReq.SetPathValue("provider", provider)
			svc.handleStart(startRec, startReq)
			loc, _ := url.Parse(startRec.Header().Get("Location"))
			state := loc.Query().Get("state")
			txCookie := findCookie(startRec.Result().Cookies(), oauthTxCookieName)
			if txCookie == nil || state == "" {
				t.Fatalf("start did not produce tx cookie/state")
			}

			// callback with matching state + a code, carrying the tx cookie
			cbRec := httptest.NewRecorder()
			cbReq := httptest.NewRequest(http.MethodGet, "/auth/"+provider+"/callback?code=abc&state="+state, nil)
			cbReq.SetPathValue("provider", provider)
			cbReq.AddCookie(txCookie)
			svc.handleCallback(cbRec, cbReq)

			if cbRec.Code != http.StatusFound {
				t.Fatalf("callback status %d, want 302; body=%s", cbRec.Code, cbRec.Body.String())
			}
			if got := cbRec.Header().Get("Location"); got != "http://localhost:8080/xlearn/auth" {
				t.Fatalf("callback redirect = %q", got)
			}
			if !hasCookie(cbRec.Result().Cookies(), auth.SessionCookieName) {
				t.Fatalf("callback did not set session cookie")
			}
			if len(st.accounts) != 1 {
				t.Fatalf("expected 1 account created, got %d", len(st.accounts))
			}
			if len(st.outbox) != 1 || st.outbox[0].Subject != "xlearn.identity.account_created" {
				t.Fatalf("expected account_created outbox row, got %+v", st.outbox)
			}
		})
	}
}

func TestCallbackStateMismatch(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	oauth := fakeOAuth(t)
	wireFakeProviders(svc, oauth.URL)

	// tx cookie says state=good, callback presents state=evil
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=abc&state=evil", nil)
	req.SetPathValue("provider", "github")
	req.AddCookie(&http.Cookie{Name: oauthTxCookieName, Value: encodeTx(oauthTx{Provider: "github", State: "good", Verifier: "v"})})
	svc.handleCallback(rec, req)

	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "error=oauth_state") {
		t.Fatalf("want redirect with error=oauth_state, got %d %s", rec.Code, rec.Header().Get("Location"))
	}
	if hasCookie(rec.Result().Cookies(), auth.SessionCookieName) {
		t.Fatalf("state mismatch must not set a session cookie")
	}
	if len(st.accounts) != 0 {
		t.Fatalf("state mismatch must not create an account")
	}
}

func TestSessionValidateAndRevoke(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	_, _ = st.CreateSession(context.Background(), "sid-1", "acct-1", time.Now().Add(time.Hour))

	// validate valid
	rec := httptest.NewRecorder()
	svc.handleValidateSession(rec, jsonReq("/sessions/validate", `{"session_id":"sid-1"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("validate valid: %d", rec.Code)
	}
	var vr struct {
		AccountID string `json:"account_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &vr)
	if vr.AccountID != "acct-1" {
		t.Fatalf("validate returned account %q", vr.AccountID)
	}

	// validate invalid → 401 unauthenticated envelope
	rec = httptest.NewRecorder()
	svc.handleValidateSession(rec, jsonReq("/sessions/validate", `{"session_id":"nope"}`))
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), `"unauthenticated"`) {
		t.Fatalf("validate invalid: %d %s", rec.Code, rec.Body.String())
	}

	// revoke → 204, then validate fails
	rec = httptest.NewRecorder()
	svc.handleRevokeSession(rec, jsonReq("/sessions/revoke", `{"session_id":"sid-1"}`))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	svc.handleValidateSession(rec, jsonReq("/sessions/validate", `{"session_id":"sid-1"}`))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("validate after revoke: %d", rec.Code)
	}
}

func TestProtectedRoutesJWT(t *testing.T) {
	st := newFakeStore()
	// seed an account + onboarding
	acct, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{
		Provider: "github", ProviderUserID: "acct", DisplayName: "Ada",
	})
	v := fakeVerifier{token: "good-token", claims: auth.Claims{Subject: acct.ID, Audience: "identity", Roles: []string{"learner"}}}
	svc := newTestService(st, v)
	handler := svc.Handler()

	// no token → 401
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/accounts/"+acct.ID, nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", rec.Code)
	}

	// valid token, correct subject → 200 with account + onboarding
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/accounts/"+acct.ID, nil)
	req.Header.Set("Authorization", "Bearer good-token")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid token: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"account"`) || !strings.Contains(rec.Body.String(), `"onboarding"`) {
		t.Fatalf("account response missing fields: %s", rec.Body.String())
	}

	// valid token, wrong subject → 403
	vWrong := fakeVerifier{token: "good-token", claims: auth.Claims{Subject: "someone-else", Audience: "identity"}}
	svcWrong := newTestService(st, vWrong)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/accounts/"+acct.ID, nil)
	req.Header.Set("Authorization", "Bearer good-token")
	svcWrong.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("wrong subject: %d", rec.Code)
	}

	// onboarding step: set path
	rec = httptest.NewRecorder()
	stepReq := jsonReq("/onboarding/step", `{"step":"path","path_chosen":"dsa"}`)
	stepReq.Header.Set("Authorization", "Bearer good-token")
	handler.ServeHTTP(rec, stepReq)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"dsa"`) {
		t.Fatalf("onboarding step: %d %s", rec.Code, rec.Body.String())
	}
	if got := st.onboarding[acct.ID].PathChosen; got != "dsa" {
		t.Fatalf("path not persisted: %q", got)
	}

	// unknown step → 400
	rec = httptest.NewRecorder()
	stepReq = jsonReq("/onboarding/step", `{"step":"nope"}`)
	stepReq.Header.Set("Authorization", "Bearer good-token")
	handler.ServeHTTP(rec, stepReq)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown step: %d", rec.Code)
	}
}

// --- helpers ---

func encodeTx(tx oauthTx) string {
	raw, _ := json.Marshal(tx)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func jsonReq(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func hasCookie(cs []*http.Cookie, name string) bool { return findCookie(cs, name) != nil }

func findCookie(cs []*http.Cookie, name string) *http.Cookie {
	for _, c := range cs {
		if c.Name == name {
			return c
		}
	}
	return nil
}
