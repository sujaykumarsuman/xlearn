package identity

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// m1-04, the identity security floor: status-joined sessions, revoke-all, L3, RequireRole
// on every user route, and preview enrollment for the owner/tester cohort.

// --- sessions ---

// Suspending an account kills every one of its sessions at once (the validate join), and a
// suspended account can start none: email login is the uniform 401, OAuth redirects to
// account_unavailable, and the link flow finds no session.
func TestSuspendedAccountHasNoSession(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	wireFakeProviders(svc, fakeOAuth(t).URL)
	cb := oauthSignIn(t, svc) // GitHub user 4242, ada@example.com
	acct := sessionAccount(t, st, cb)
	if _, _, err := st.SetAccountPassword(context.Background(), acct, mustHash(t, "hunter2hunter")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"s-a", "s-b"} {
		_, _ = st.CreateSession(context.Background(), id, acct, time.Now().Add(time.Hour))
	}
	validate := func(sid string) int {
		rec := httptest.NewRecorder()
		svc.handleValidateSession(rec, jsonReq("/sessions/validate", `{"session_id":"`+sid+`"}`))
		return rec.Code
	}
	if validate("s-a") != http.StatusOK || validate("s-b") != http.StatusOK {
		t.Fatal("sessions not valid before the suspend")
	}

	st.setRoleStatus(acct, "", store.StatusSuspended)
	if a, b := validate("s-a"), validate("s-b"); a != http.StatusUnauthorized || b != http.StatusUnauthorized {
		t.Fatalf("after suspend validate = %d/%d, want 401/401", a, b)
	}

	// Email login with the RIGHT password: the same 401 body as an unknown identifier.
	right := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "ada@example.com", "password": "hunter2hunter"}, nil, nil)
	unknown := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "nobody@example.com", "password": "hunter2hunter"}, nil, nil)
	if right.Code != http.StatusUnauthorized || right.Body.String() != unknown.Body.String() {
		t.Fatalf("suspended login: %d %s, want the uniform 401 %s", right.Code, right.Body.String(), unknown.Body.String())
	}
	if hasCookie(right.Result().Cookies(), auth.SessionCookieName) {
		t.Fatal("suspended login set a session cookie")
	}

	// OAuth: a returning suspended user is sent back to /auth with account_unavailable.
	cb = oauthSignIn(t, svc)
	if got := cb.Header().Get("Location"); got != "http://localhost:8080/xlearn/auth?error=account_unavailable" {
		t.Fatalf("suspended OAuth redirect = %q", got)
	}
	if hasCookie(cb.Result().Cookies(), auth.SessionCookieName) {
		t.Fatal("suspended OAuth sign-in set a session cookie")
	}

	// Link mode needs a live session, which a suspended account no longer has.
	req := httptest.NewRequest(http.MethodPost, "/auth/github/start?link=1", nil)
	req.SetPathValue("provider", "github")
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "s-a"})
	rec := httptest.NewRecorder()
	svc.handleStart(rec, req)
	if got := rec.Header().Get("Location"); !strings.HasSuffix(got, "/auth?error=link_auth") {
		t.Fatalf("link with a suspended session redirected to %q", got)
	}

	// Reactivated: sign-in works again.
	st.setRoleStatus(acct, "", store.StatusActive)
	if rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "ada@example.com", "password": "hunter2hunter"}, nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("login after reactivate: %d", rec.Code)
	}
}

// A password change (or first set) revokes every session, the caller's included, and says so.
func TestSetPasswordRevokesAllSessions(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	acct, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{Provider: "github", ProviderUserID: "7", DisplayName: "Ada", Email: "ada@example.com"})
	other, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{Provider: "github", ProviderUserID: "8", DisplayName: "Bob"})
	for _, id := range []string{"caller", "laptop", "phone"} {
		_, _ = st.CreateSession(context.Background(), id, acct.ID, time.Now().Add(time.Hour))
	}
	_, _ = st.CreateSession(context.Background(), "bobs", other.ID, time.Now().Add(time.Hour))

	rec := doJSON(t, svc.handleSetPassword, http.MethodPost, "/x", map[string]string{"new_password": "first-pass-123"}, &auth.Claims{Subject: acct.ID}, map[string]string{"id": acct.ID})
	if rec.Code != http.StatusOK {
		t.Fatalf("set password: %d %s", rec.Code, rec.Body.String())
	}
	var body struct{ OK, Reauth bool }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || !body.OK || !body.Reauth {
		t.Fatalf("set password body %s, want {ok:true, reauth:true}", rec.Body.String())
	}
	for _, id := range []string{"caller", "laptop", "phone"} {
		if _, err := st.GetValidSession(context.Background(), id); err == nil {
			t.Fatalf("session %s survived the password change", id)
		}
	}
	if _, err := st.GetValidSession(context.Background(), "bobs"); err != nil {
		t.Fatal("another account's session was revoked")
	}
}

// --- L3 ---

// countingBcrypt counts compares; its hash is the password itself (tests only).
type countingBcrypt struct{ compares atomic.Int64 }

func (c *countingBcrypt) Generate(pw []byte) ([]byte, error) { return append([]byte("h:"), pw...), nil }
func (c *countingBcrypt) Compare(hash, pw []byte) error {
	c.compares.Add(1)
	if string(hash) == "h:"+string(pw) {
		return nil
	}
	return errors.New("mismatch")
}

// Every login failure runs exactly ONE compare — unknown identifier, password-less account,
// suspended account and a wrong password alike — so the uniform 401 is uniform in time.
func TestLoginOneCompareWhateverTheIdentifier(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	cb := &countingBcrypt{}
	svc.pw = newPasswords(cb, bcryptSlots)
	if err := svc.pw.warm(); err != nil {
		t.Fatal(err)
	}
	pwAcct, _ := st.CreateEmailAccount(context.Background(), "pw@example.com", "h:right-pass-1", "pw")
	_, _, _ = st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{Provider: "github", ProviderUserID: "9", DisplayName: "OAuth only", Email: "oauth@example.com"})
	susp, _ := st.CreateEmailAccount(context.Background(), "susp@example.com", "h:right-pass-1", "s")
	st.setRoleStatus(susp.ID, "", store.StatusSuspended)

	var uniform string
	for _, id := range []string{"unknown@example.com", "no-such-username", "oauth@example.com", "susp@example.com", "pw@example.com"} {
		pw := "wrong-pass-1"
		if id == "susp@example.com" {
			pw = "right-pass-1" // the right password still fails, after one compare
		}
		before := cb.compares.Load()
		rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": id, "password": pw}, nil, nil)
		if n := cb.compares.Load() - before; n != 1 {
			t.Fatalf("%s: %d compares, want exactly 1", id, n)
		}
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status %d, want 401", id, rec.Code)
		}
		if uniform == "" {
			uniform = rec.Body.String()
		} else if rec.Body.String() != uniform {
			t.Fatalf("%s: body %s differs from %s", id, rec.Body.String(), uniform)
		}
	}
	before := cb.compares.Load()
	if rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "pw@example.com", "password": "right-pass-1"}, nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("right password: %d", rec.Code)
	}
	if n := cb.compares.Load() - before; n != 1 {
		t.Fatalf("successful login: %d compares, want 1", n)
	}
	_ = pwAcct
}

// blockingBcrypt parks every operation until release is closed.
type blockingBcrypt struct {
	entered chan struct{}
	release chan struct{}
}

func (b *blockingBcrypt) Generate(pw []byte) ([]byte, error) {
	b.entered <- struct{}{}
	<-b.release
	return append([]byte("h:"), pw...), nil
}

func (b *blockingBcrypt) Compare(hash, pw []byte) error {
	b.entered <- struct{}{}
	<-b.release
	if string(hash) == "h:"+string(pw) {
		return nil
	}
	return errors.New("mismatch")
}

// With two bcrypt operations in flight, a third — login, signup or a password change —
// is refused at once with 429 too_many_requests and Retry-After: 1.
func TestBcryptGateThirdIs429(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	bb := &blockingBcrypt{entered: make(chan struct{}, 8), release: make(chan struct{})}
	svc.pw = newPasswords(bb, bcryptSlots)
	svc.pw.dummyOnce.Do(func() { svc.pw.dummy = []byte("h:dummy") }) // skip the blocking warm
	acct, _ := st.CreateEmailAccount(context.Background(), "ada@example.com", "h:right-pass-1", "ada")

	var wg sync.WaitGroup
	for range bcryptSlots {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "ada@example.com", "password": "right-pass-1"}, nil, nil)
		}()
	}
	for range bcryptSlots {
		<-bb.entered // both slots are now held
	}

	check := func(name string, rec *httptest.ResponseRecorder) {
		t.Helper()
		if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "1" ||
			!strings.Contains(rec.Body.String(), `"too_many_requests"`) {
			t.Fatalf("%s: %d Retry-After=%q %s, want 429 too_many_requests Retry-After: 1", name, rec.Code, rec.Header().Get("Retry-After"), rec.Body.String())
		}
	}
	check("login (known)", doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "ada@example.com", "password": "x-pass-123"}, nil, nil))
	check("login (unknown)", doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "nobody@example.com", "password": "x-pass-123"}, nil, nil))
	check("signup", doJSON(t, svc.handleSignup, http.MethodPost, "/auth/signup", map[string]string{"email": "new@example.com", "password": "hunter2hunter"}, nil, nil))
	check("set password", doJSON(t, svc.handleSetPassword, http.MethodPost, "/x", map[string]string{"current_password": "right-pass-1", "new_password": "next-pass-123"}, &auth.Claims{Subject: acct.ID}, map[string]string{"id": acct.ID}))
	if len(st.accounts) != 1 {
		t.Fatal("a refused signup created an account")
	}

	close(bb.release)
	wg.Wait()
	// Slots free again: a login goes through.
	if rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "ada@example.com", "password": "right-pass-1"}, nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("login after release: %d", rec.Code)
	}
}

// The real gate at the real cost: the process-wide gate has exactly bcryptSlots slots and
// the dummy hash is a real bcrypt hash.
func TestProcessPasswordsGate(t *testing.T) {
	p := processPasswords()
	if cap(p.slots) != 2 {
		t.Fatalf("bcrypt slots = %d, want 2 (L3)", cap(p.slots))
	}
	if err := p.warm(); err != nil || !strings.HasPrefix(string(p.dummy), "$2a$10$") {
		t.Fatalf("dummy hash %q (err %v), want a cost-10 bcrypt hash", p.dummy, err)
	}
	if ok, err := p.check("", "anything"); ok || err != nil {
		t.Fatalf("dummy compare = %v, %v; want false, nil", ok, err)
	}
}

// --- RequireRole on every user route ---

// rolesVerifier accepts any non-empty token and returns the configured subject + roles.
type rolesVerifier struct {
	subject string
	roles   []string
}

func (v rolesVerifier) Verify(_ context.Context, token string) (auth.Claims, error) {
	if token == "" {
		return auth.Claims{}, auth.ErrUnauthenticated
	}
	return auth.Claims{Subject: v.subject, Audience: "identity", Roles: v.roles}, nil
}

// Every JWT route of identity refuses a public-read token (403), serves a learner token
// (neither 401 nor 403) and 401s without a token (ADR-0033 §12 row 5).
func TestUserRoutesRequireLearner(t *testing.T) {
	st := newFakeStore()
	acct, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{Provider: "github", ProviderUserID: "1", DisplayName: "Ada"})
	routes := (&Service{}).userRoutes()
	if len(routes) == 0 {
		t.Fatal("no user routes")
	}
	call := func(roles []string, withToken bool, rt userRoute) *httptest.ResponseRecorder {
		svc := newTestService(st, rolesVerifier{subject: acct.ID, roles: roles})
		path := strings.NewReplacer("{id}", acct.ID, "{slug}", "dsa", "{provider}", "google").Replace(rt.Pattern)
		var body io.Reader
		if rt.Method != http.MethodGet {
			body = strings.NewReader(`{}`)
		}
		req := httptest.NewRequest(rt.Method, path, body)
		req.Header.Set("Content-Type", "application/json")
		if withToken {
			req.Header.Set("Authorization", "Bearer tok")
		}
		rec := httptest.NewRecorder()
		svc.Handler().ServeHTTP(rec, req)
		return rec
	}
	for _, rt := range routes {
		name := rt.Method + " " + rt.Pattern
		if rec := call([]string{auth.RolePublicRead}, true, rt); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), `"forbidden"`) {
			t.Errorf("%s with public-read: %d %s, want 403 forbidden", name, rec.Code, rec.Body.String())
		}
		if rec := call([]string{auth.RoleLearner}, true, rt); rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden ||
			strings.Contains(rec.Body.String(), "404 page not found") {
			t.Errorf("%s with learner: %d %s, want served", name, rec.Code, rec.Body.String())
		}
		if rec := call(nil, false, rt); rec.Code != http.StatusUnauthorized || rec.Body.String() != `{"error":{"code":"unauthenticated"}}` {
			t.Errorf("%s without a token: %d %s, want the 401 envelope", name, rec.Code, rec.Body.String())
		}
	}
}

// --- preview enrollment (the T-3 cohort) ---

// A preview course is enrollable for the owner and testers (their DB role, never a JWT
// claim) and the uniform 404 for a learner — even one whose token says "owner".
func TestPreviewEnrollmentForCohortOnly(t *testing.T) {
	st := newFakeStore()
	svc := NewService(testConfig(), st, nil, coursetest.Registry(t), slog.New(slog.NewJSONHandler(io.Discard, nil)))
	mk := func(uid, role string) string {
		a, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{Provider: "github", ProviderUserID: uid, DisplayName: uid})
		st.setRoleStatus(a.ID, role, "")
		return a.ID
	}
	owner, tester, learner := mk("o", store.RoleOwner), mk("t", store.RoleTester), mk("l", store.RoleLearner)

	for _, id := range []string{owner, tester} {
		if code, body := startEnrollment(t, svc, id, coursetest.FixturePreview); code != http.StatusOK {
			t.Fatalf("cohort %s: preview enroll %d %s, want 200", id, code, body)
		}
	}
	_, unknown := startEnrollment(t, svc, learner, "no-such-course")
	claims := auth.Claims{Subject: learner, Roles: []string{"learner", "owner"}} // a forged claim changes nothing
	rec := doJSON(t, svc.handleStartEnrollment, http.MethodPost, "/x", nil, &claims, map[string]string{"slug": coursetest.FixturePreview})
	if rec.Code != http.StatusNotFound || rec.Body.String() != unknown {
		t.Fatalf("learner preview enroll: %d %s, want the uniform 404 %s", rec.Code, rec.Body.String(), unknown)
	}
	if n := len(st.enrollments[learner]); n != 0 {
		t.Fatalf("learner got %d enrollments", n)
	}
	// Active courses are unchanged for everyone.
	if code, _ := startEnrollment(t, svc, learner, "dsa"); code != http.StatusOK {
		t.Fatalf("learner dsa enroll: %d", code)
	}
}
