package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// doJSON drives one handler directly: JSON body, optional path values, and optional JWT
// claims injected into the context the way requireJWT would.
func doJSON(t *testing.T, h http.HandlerFunc, method, target string, body any, claims *auth.Claims, pathVals map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range pathVals {
		req.SetPathValue(k, v)
	}
	if claims != nil {
		req = req.WithContext(context.WithValue(req.Context(), claimsCtxKey{}, *claims))
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestSignupAndLogin(t *testing.T) {
	svc := newTestService(newFakeStore(), nil)

	rec := doJSON(t, svc.handleSignup, http.MethodPost, "/auth/signup", map[string]string{"email": "New.User@Example.com", "password": "hunter2hunter"}, nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("signup status %d, want 200", rec.Code)
	}
	if !hasCookie(rec.Result().Cookies(), auth.SessionCookieName) {
		t.Fatal("signup set no session cookie")
	}

	// Duplicate email is rejected case-insensitively.
	rec = doJSON(t, svc.handleSignup, http.MethodPost, "/auth/signup", map[string]string{"email": "new.user@example.com", "password": "another-pass"}, nil, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate signup status %d, want 409", rec.Code)
	}
	// Weak password + malformed email are 422.
	if rec := doJSON(t, svc.handleSignup, http.MethodPost, "/auth/signup", map[string]string{"email": "x@y.com", "password": "short"}, nil, nil); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("weak password status %d, want 422", rec.Code)
	}
	if rec := doJSON(t, svc.handleSignup, http.MethodPost, "/auth/signup", map[string]string{"email": "not-an-email", "password": "hunter2hunter"}, nil, nil); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad email status %d, want 422", rec.Code)
	}

	// Login is case-insensitive on email.
	if rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "NEW.USER@example.com", "password": "hunter2hunter"}, nil, nil); rec.Code != http.StatusOK || !hasCookie(rec.Result().Cookies(), auth.SessionCookieName) {
		t.Fatalf("login status %d (want 200 + cookie)", rec.Code)
	}
	// Wrong password + unknown email both return a uniform 401 (no user enumeration).
	if rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "new.user@example.com", "password": "wrongwrong"}, nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status %d, want 401", rec.Code)
	}
	if rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "nobody@example.com", "password": "whatever12"}, nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unknown email status %d, want 401", rec.Code)
	}
}

func TestSetPassword(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	acct, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{Provider: "github", ProviderUserID: "42", DisplayName: "Ada", Email: "ada@example.com"})
	claims := &auth.Claims{Subject: acct.ID}
	pv := map[string]string{"id": acct.ID}

	// First set (OAuth-native account) needs no current password.
	if rec := doJSON(t, svc.handleSetPassword, http.MethodPost, "/x", map[string]string{"new_password": "first-pass-123"}, claims, pv); rec.Code != http.StatusOK {
		t.Fatalf("first set status %d, want 200", rec.Code)
	}
	// Changing it now requires the correct current password.
	if rec := doJSON(t, svc.handleSetPassword, http.MethodPost, "/x", map[string]string{"current_password": "nope", "new_password": "second-pass-123"}, claims, pv); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong current status %d, want 401", rec.Code)
	}
	if rec := doJSON(t, svc.handleSetPassword, http.MethodPost, "/x", map[string]string{"current_password": "first-pass-123", "new_password": "second-pass-123"}, claims, pv); rec.Code != http.StatusOK {
		t.Fatalf("change status %d, want 200", rec.Code)
	}
	// The account can now sign in with email + the new password.
	if rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "ada@example.com", "password": "second-pass-123"}, nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("login after set-password status %d, want 200", rec.Code)
	}
	// Cross-account is forbidden.
	if rec := doJSON(t, svc.handleSetPassword, http.MethodPost, "/x", map[string]string{"new_password": "whatever12"}, claims, map[string]string{"id": "someone-else"}); rec.Code != http.StatusForbidden {
		t.Fatalf("cross-account status %d, want 403", rec.Code)
	}
}

func TestUnlinkGuardsLastMethod(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	acct, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{Provider: "github", ProviderUserID: "42", DisplayName: "Ada", Email: "ada@example.com"})
	claims := &auth.Claims{Subject: acct.ID}
	pv := map[string]string{"id": acct.ID, "provider": "github"}

	// GitHub is the only sign-in method → refuse to disconnect it.
	if rec := doJSON(t, svc.handleUnlinkOAuth, http.MethodDelete, "/x", nil, claims, pv); rec.Code != http.StatusConflict {
		t.Fatalf("unlink last-method status %d, want 409", rec.Code)
	}
	// With a password set, disconnecting is allowed.
	if _, err := st.SetAccountPassword(context.Background(), acct.ID, "bcrypt-ish"); err != nil {
		t.Fatalf("set password: %v", err)
	}
	if rec := doJSON(t, svc.handleUnlinkOAuth, http.MethodDelete, "/x", nil, claims, pv); rec.Code != http.StatusNoContent {
		t.Fatalf("unlink status %d, want 204", rec.Code)
	}
}

// With SIGNUP_MODE closed, sign-up is a 403 whether or not the email is registered (no email
// oracle), while an existing account still signs in.
func TestSignupClosed(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	svc.cfg.Auth.Signup = SignupClosed
	if _, err := st.CreateEmailAccount(context.Background(), "ada@example.com", mustHash(t, "hunter2hunter"), "ada"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	for _, email := range []string{"new@example.com", "ada@example.com"} {
		rec := doJSON(t, svc.handleSignup, http.MethodPost, "/auth/signup", map[string]string{"email": email, "password": "hunter2hunter"}, nil, nil)
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), `"signup_closed"`) {
			t.Fatalf("signup %s: status %d body %s, want 403 signup_closed", email, rec.Code, rec.Body.String())
		}
		if hasCookie(rec.Result().Cookies(), auth.SessionCookieName) {
			t.Fatalf("signup %s set a session cookie while closed", email)
		}
	}
	if len(st.accounts) != 1 {
		t.Fatalf("closed signup created an account: %d accounts", len(st.accounts))
	}

	if rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "ada@example.com", "password": "hunter2hunter"}, nil, nil); rec.Code != http.StatusOK || !hasCookie(rec.Result().Cookies(), auth.SessionCookieName) {
		t.Fatalf("existing-account login while closed: status %d, want 200 + cookie", rec.Code)
	}
}

func mustHash(t *testing.T, pw string) string {
	t.Helper()
	h, err := hashPassword(pw)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return h
}
