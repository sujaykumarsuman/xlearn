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
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// m1-04: session-validate's new fields reach the gateway; a password change drops the
// (revoked) session cookie; L3's 429 keeps its Retry-After through the gateway.

func TestValidateSessionParsesInfo(t *testing.T) {
	srv := httptest.NewServer(jsonMux(map[string]handlerFn{
		"POST /sessions/validate": func(w http.ResponseWriter, r *http.Request) {
			var b struct {
				SessionID string `json:"session_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			switch b.SessionID {
			case "v2":
				_, _ = w.Write([]byte(`{"account_id":"a1","expires_at":"2026-10-29T00:00:00Z","role":"tester","status":"active","accepted":false,"created_at":"2026-09-29T05:00:00Z"}`))
			case "v1":
				_, _ = w.Write([]byte(`{"account_id":"a1","expires_at":"2026-10-29T00:00:00Z"}`))
			default:
				w.WriteHeader(http.StatusUnauthorized)
			}
		},
	}))
	defer srv.Close()
	c := newIdentityClient(srv.URL)

	info, err := c.validateSession(context.Background(), "v2")
	want := sessionInfo{AccountID: "a1", Role: "tester", Status: "active", CreatedAt: time.Date(2026, 9, 29, 5, 0, 0, 0, time.UTC)}
	if err != nil || info.AccountID != want.AccountID || info.Role != want.Role || info.Status != want.Status ||
		info.Accepted || !info.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("v2 validate = %+v, %v; want %+v", info, err, want)
	}
	// A v1.6.0 identity (mixed-version rollout): the account only, nobody in the cohort.
	info, err = c.validateSession(context.Background(), "v1")
	if err != nil || info.AccountID != "a1" || info.Role != "" || inCohort(info) {
		t.Fatalf("v1 validate = %+v, %v", info, err)
	}
	if _, err := c.validateSession(context.Background(), "nope"); err == nil {
		t.Fatal("a 401 validated")
	}
}

func TestPasswordChangeClearsCookieAnd429sKeepRetryAfter(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	pwStatus := http.StatusOK
	identity := httptest.NewServer(jsonMux(map[string]handlerFn{
		"POST /sessions/validate": writeJSONFn(map[string]any{"account_id": "acct-1", "role": "learner", "status": "active"}),
		"POST /accounts/{id}/password": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if pwStatus == http.StatusTooManyRequests {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":{"code":"too_many_requests"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"reauth":true}`))
		},
		"POST /auth/login": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"too_many_requests"}}`))
		},
	}))
	defer identity.Close()
	gw := httptest.NewServer(New(Options{
		BasePath: "/xlearn", Version: "test", Signer: auth.NewSigner(key, "xlearn-gateway", 0),
		Dist:            fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}},
		IdentityBaseURL: identity.URL, AudienceIdentity: "identity",
	}).Handler())
	defer gw.Close()

	post := func(path string) *http.Response {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPost, gw.URL+"/xlearn/api/v1"+path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return resp
	}

	resp := post("/me/password")
	var cleared bool
	for _, c := range resp.Cookies() {
		if c.Name == auth.SessionCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if resp.StatusCode != http.StatusOK || !cleared {
		t.Fatalf("password change: %d, session cookie cleared=%v; want 200 + cleared", resp.StatusCode, cleared)
	}

	pwStatus = http.StatusTooManyRequests
	if resp := post("/me/password"); resp.StatusCode != http.StatusTooManyRequests || resp.Header.Get("Retry-After") != "1" {
		t.Fatalf("busy password change: %d Retry-After=%q", resp.StatusCode, resp.Header.Get("Retry-After"))
	}
	if resp := post("/auth/login"); resp.StatusCode != http.StatusTooManyRequests || resp.Header.Get("Retry-After") != "1" {
		t.Fatalf("busy login: %d Retry-After=%q", resp.StatusCode, resp.Header.Get("Retry-After"))
	}
}
