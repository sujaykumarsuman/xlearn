package gateway

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
)

// fakeClock is the limits' injectable clock; tests advance it instead of sleeping.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock { return &fakeClock{t: time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// tickingClock advances one second on every read, so no token bucket ever runs dry: for
// harnesses that sweep many routes for one account and aren't about the limits.
func tickingClock() func() time.Time {
	var n atomic.Int64
	base := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	return func() time.Time { return base.Add(time.Duration(n.Add(1)) * time.Second) }
}

// limitsHarness is a real gateway (fake clock) in front of a recording fake identity.
type limitsHarness struct {
	t     *testing.T
	clock *fakeClock
	gw    *httptest.Server

	mu          sync.Mutex
	loginBodies []string // bodies identity's /auth/login received

	signups    atomic.Int64   // signup handler invocations
	signupRecs chan signupRec // what each signup handler read, sent before it answers
	starts     atomic.Int64
}

// logins returns a copy of the login bodies identity received.
func (h *limitsHarness) logins() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.loginBodies...)
}

// signupRec is what one invocation of the fake identity's signup handler read.
type signupRec struct {
	n   int64
	err error
}

func newLimitsHarness(t *testing.T) *limitsHarness {
	t.Helper()
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer := auth.NewSigner(key, "xlearn-gateway", 0)
	h := &limitsHarness{t: t, clock: newFakeClock(), signupRecs: make(chan signupRec, 64)}

	identity := httptest.NewServer(jsonMux(map[string]handlerFn{
		"POST /sessions/validate": func(w http.ResponseWriter, r *http.Request) {
			var b struct {
				SessionID string `json:"session_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			acct, ok := map[string]string{"sess-1": "acct-1", "sess-2": "acct-2"}[b.SessionID]
			if !ok {
				w.WriteHeader(401)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"account_id": acct, "role": "learner", "status": "active"})
		},
		"POST /auth/login": func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			h.mu.Lock()
			h.loginBodies = append(h.loginBodies, string(raw))
			h.mu.Unlock()
			var b struct{ Password string }
			_ = json.Unmarshal(raw, &b)
			if b.Password != "right-password" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"code":"invalid_credentials"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		},
		"POST /auth/signup": func(w http.ResponseWriter, r *http.Request) {
			n, err := io.Copy(io.Discard, r.Body)
			h.signups.Add(1)
			select {
			case h.signupRecs <- signupRec{n: n, err: err}:
			default:
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		},
		"POST /auth/{provider}/start": func(w http.ResponseWriter, r *http.Request) {
			h.starts.Add(1)
			w.Header().Set("Location", "https://github.com/login/oauth/authorize")
			w.WriteHeader(http.StatusFound)
		},
		"GET /auth/{provider}/callback": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "/xlearn/")
			w.WriteHeader(http.StatusFound)
		},
		"POST /onboarding/step": writeJSONFn(map[string]any{"onboarding": map[string]any{}}),
	}))
	t.Cleanup(identity.Close)
	curriculum := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /paths": writeJSONFn(map[string]any{"paths": []any{}}),
	}))
	t.Cleanup(curriculum.Close)

	gw := New(Options{
		BasePath: "/xlearn", Version: "test", Signer: signer,
		Dist:            fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}, "assets/app.js": {Data: []byte("x")}},
		IdentityBaseURL: identity.URL, AudienceIdentity: "identity",
		CurriculumBaseURL: curriculum.URL,
		Courses:           coursetest.Registry(t),
		Now:               h.clock.Now,
	})
	h.gw = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gw.Close)
	return h
}

// noRedirectClient returns the OAuth 302s to the test instead of following them off-host.
var noRedirectClient = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

type limitsReq struct {
	method, path, body, ip, session string
	raw                             bool              // path is the full URL path (no /xlearn/api/v1 prefix)
	chunked                         bool              // send the body with an unknown length
	headers                         map[string]string // extra request headers
}

// do issues a request from client IP ip (X-Real-Ip, as Traefik sets it).
func (h *limitsHarness) do(q limitsReq) (*http.Response, []byte) {
	h.t.Helper()
	var rdr io.Reader
	if q.body != "" {
		rdr = strings.NewReader(q.body)
		if q.chunked {
			rdr = io.MultiReader(rdr) // hides the length: sent chunked
		}
	}
	u := h.gw.URL + "/xlearn/api/v1" + q.path
	if q.raw {
		u = h.gw.URL + q.path
	}
	req, _ := http.NewRequestWithContext(context.Background(), q.method, u, rdr)
	if q.ip != "" {
		req.Header.Set("X-Real-Ip", q.ip)
	}
	for k, v := range q.headers {
		req.Header.Set(k, v)
	}
	if q.session != "" {
		req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: q.session})
	}
	if q.method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := noRedirectClient.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", q.method, q.path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, b
}

// want429 asserts the typed 429: code, Retry-After and the envelope's retry_after agree.
func want429(t *testing.T, resp *http.Response, body []byte, code string, retryAfter int) {
	t.Helper()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status %d %s, want 429", resp.StatusCode, body)
	}
	var env struct {
		Error struct {
			Code       string `json:"code"`
			RetryAfter int    `json:"retry_after"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &env)
	if env.Error.Code != code {
		t.Fatalf("code %q, want %q (%s)", env.Error.Code, code, body)
	}
	if got := resp.Header.Get("Retry-After"); got != strconv.Itoa(retryAfter) || env.Error.RetryAfter != retryAfter {
		t.Fatalf("Retry-After %q / retry_after %d, want %d", got, env.Error.RetryAfter, retryAfter)
	}
}

func loginBody(identifier, password string) string {
	b, _ := json.Marshal(map[string]string{"email": identifier, "password": password})
	return string(b)
}

// L1 per IP: 10/min, burst 5. The 6th login inside the burst window is a typed 429 with
// Retry-After 6 s (one token per 6 s); another IP is unaffected; a token refills in 6 s.
func TestL1LoginPerIP(t *testing.T) {
	h := newLimitsHarness(t)
	for i := 0; i < 5; i++ {
		resp, body := h.do(limitsReq{method: "POST", path: "/auth/login", ip: "203.0.113.1", body: loginBody("user"+strconv.Itoa(i), "wrong")})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("login %d: %d %s, want identity's 401", i+1, resp.StatusCode, body)
		}
	}
	resp, body := h.do(limitsReq{method: "POST", path: "/auth/login", ip: "203.0.113.1", body: loginBody("user9", "wrong")})
	want429(t, resp, body, "rate_limited", 6)
	if resp, _ := h.do(limitsReq{method: "POST", path: "/auth/login", ip: "203.0.113.2", body: loginBody("user9", "wrong")}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("another IP: %d, want 401", resp.StatusCode)
	}
	h.clock.Advance(6 * time.Second)
	if resp, _ := h.do(limitsReq{method: "POST", path: "/auth/login", ip: "203.0.113.1", body: loginBody("user9", "wrong")}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("after 6 s: %d, want 401", resp.StatusCode)
	}
	// The bucket is empty again; client-supplied forwarding headers don't move the key.
	resp, body = h.do(limitsReq{method: "POST", path: "/auth/login", ip: "203.0.113.1", body: loginBody("user9", "wrong"),
		headers: map[string]string{"X-Forwarded-For": "192.0.2.77", "Forwarded": "for=192.0.2.77", "True-Client-Ip": "192.0.2.77"}})
	want429(t, resp, body, "rate_limited", 6)
}

// L1 per identifier: 5 failures / 15 min. The 6th attempt is a 429 WITHOUT calling
// identity until the oldest failure ages out; the identifier is normalised (trim, lower);
// a success clears the window; identity receives the body the client sent.
func TestL1LoginFailureWindow(t *testing.T) {
	h := newLimitsHarness(t)
	ip := 0
	login := func(identifier, password string) (*http.Response, []byte) {
		ip++ // a fresh IP each time so L1's per-IP bucket stays out of the way
		return h.do(limitsReq{method: "POST", path: "/auth/login", ip: "198.51.100." + strconv.Itoa(ip), body: loginBody(identifier, password)})
	}
	spellings := []string{"ada@example.com", "Ada@Example.com", "  ADA@example.com ", "ada@example.COM", "ada@example.com"}
	for i, id := range spellings {
		if resp, body := login(id, "wrong"); resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("failure %d: %d %s, want 401", i+1, resp.StatusCode, body)
		}
		h.clock.Advance(time.Minute)
	}
	if bodies := h.logins(); len(bodies) != 5 || bodies[0] != loginBody("ada@example.com", "wrong") {
		t.Fatalf("identity saw %d bodies (%v), want 5 identical to the client's", len(bodies), bodies)
	}
	// Failures at t0..t0+4m; now t0+5m: blocked for 10 more minutes, identity not called.
	resp, body := login("ADA@example.com", "right-password")
	want429(t, resp, body, "rate_limited", 600)
	if len(h.logins()) != 5 {
		t.Fatal("identity was called for a blocked identifier")
	}
	// Another identifier still reaches identity.
	if resp, _ := login("bo@example.com", "wrong"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("another identifier: %d, want 401", resp.StatusCode)
	}
	// The oldest failure ages out after 10 more minutes: one attempt gets through.
	h.clock.Advance(10 * time.Minute)
	if resp, body := login("ada@example.com", "right-password"); resp.StatusCode != http.StatusOK {
		t.Fatalf("after the window: %d %s, want 200", resp.StatusCode, body)
	}
	// The success cleared the window: five fresh failures are allowed again.
	for i := 0; i < 5; i++ {
		if resp, _ := login("ada@example.com", "wrong"); resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("after a success, failure %d: %d, want 401", i+1, resp.StatusCode)
		}
	}
	resp, body = login("ada@example.com", "wrong")
	want429(t, resp, body, "rate_limited", 900)
}

// L1's buffered body: an oversize login body is the typed 413 (8 KiB cap), identity untouched.
func TestL1LoginOversizeBodyIs413(t *testing.T) {
	h := newLimitsHarness(t)
	resp, body := h.do(limitsReq{method: "POST", path: "/auth/login", ip: "203.0.113.9", body: loginBody("ada", strings.Repeat("x", 9<<10))})
	if resp.StatusCode != http.StatusRequestEntityTooLarge || errorCode(body) != httpx.CodeBodyTooLarge {
		t.Fatalf("oversize login: %d %s, want 413 body_too_large", resp.StatusCode, body)
	}
	if len(h.logins()) != 0 {
		t.Fatal("identity saw an oversize login")
	}
}

// L2: signup and POST /auth/{provider}/start are 5/min per IP each; the OAuth callback
// (the provider redirecting the browser back) is not limited.
func TestL2SignupAndOAuthStartPerIP(t *testing.T) {
	h := newLimitsHarness(t)
	for _, path := range []string{"/auth/signup", "/auth/github/start"} {
		for i := 0; i < 5; i++ {
			if resp, body := h.do(limitsReq{method: "POST", path: path, ip: "203.0.113.5", body: `{}`}); resp.StatusCode == http.StatusTooManyRequests {
				t.Fatalf("%s %d: 429 inside the limit (%s)", path, i+1, body)
			}
		}
		resp, body := h.do(limitsReq{method: "POST", path: path, ip: "203.0.113.5", body: `{}`})
		want429(t, resp, body, "rate_limited", 12)
		if resp, _ := h.do(limitsReq{method: "POST", path: path, ip: "203.0.113.6", body: `{}`}); resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("%s: another IP limited", path)
		}
	}
	if h.signups.Load() != 6 || h.starts.Load() != 6 {
		t.Fatalf("identity saw %d signups / %d starts, want 6 / 6", h.signups.Load(), h.starts.Load())
	}
	for i := 0; i < 20; i++ {
		if resp, _ := h.do(limitsReq{method: "GET", path: "/auth/github/callback?code=x", ip: "203.0.113.5"}); resp.StatusCode == http.StatusTooManyRequests {
			t.Fatal("the OAuth callback was limited")
		}
	}
}

// L5 per account: every /api request 20/s burst 40, mutating methods also 5/s burst 10.
func TestL5PerAccount(t *testing.T) {
	h := newLimitsHarness(t)
	for i := 0; i < 40; i++ {
		if resp, body := h.do(limitsReq{method: "GET", path: "/paths", session: "sess-1"}); resp.StatusCode != http.StatusOK {
			t.Fatalf("read %d: %d %s", i+1, resp.StatusCode, body)
		}
	}
	resp, body := h.do(limitsReq{method: "GET", path: "/paths", session: "sess-1"})
	want429(t, resp, body, "rate_limited", 1)
	// Keyed by account, not IP: acct-2 on the same IP is unaffected.
	if resp, _ := h.do(limitsReq{method: "GET", path: "/paths", session: "sess-2"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("another account: %d", resp.StatusCode)
	}
	h.clock.Advance(50 * time.Millisecond) // one token at 20/s
	if resp, _ := h.do(limitsReq{method: "GET", path: "/paths", session: "sess-1"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("after 50 ms: %d, want 200", resp.StatusCode)
	}

	// Writes: 10 in a burst, the 11th is refused; reads still have their own budget.
	for i := 0; i < 10; i++ {
		if resp, body := h.do(limitsReq{method: "POST", path: "/onboarding/step", session: "sess-2", body: `{}`}); resp.StatusCode != http.StatusOK {
			t.Fatalf("write %d: %d %s", i+1, resp.StatusCode, body)
		}
	}
	resp, body = h.do(limitsReq{method: "POST", path: "/onboarding/step", session: "sess-2", body: `{}`})
	want429(t, resp, body, "rate_limited", 1)
	if resp, _ := h.do(limitsReq{method: "GET", path: "/paths", session: "sess-2"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("a read after the write limit: %d, want 200", resp.StatusCode)
	}
	// No session: the 401 comes first (L5 keys only validated accounts).
	if resp, _ := h.do(limitsReq{method: "GET", path: "/paths"}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no session: %d, want 401", resp.StatusCode)
	}
}

// Probes, app health, JWKS and static assets are exempt from every limit.
func TestLimitsExemptRoutes(t *testing.T) {
	h := newLimitsHarness(t)
	for _, path := range []string{"/healthz", "/readyz", "/xlearn/api/v1/healthz", "/xlearn/api/healthz", "/.well-known/jwks.json", "/xlearn/assets/app.js", "/xlearn/"} {
		for i := 0; i < 100; i++ {
			resp, _ := h.do(limitsReq{method: "GET", path: path, raw: true, ip: "203.0.113.50", session: "sess-1"})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%s request %d: %d, want 200 (exempt)", path, i+1, resp.StatusCode)
			}
		}
	}
}

// L6 through the identity proxy: an oversize signup is the typed 413 and identity never
// sees the overrun — a declared Content-Length is refused before dialling, a chunked
// body is cut at the cap.
func TestOversizeSignupIs413(t *testing.T) {
	h := newLimitsHarness(t)
	big := `{"email":"a@b.c","pad":"` + strings.Repeat("x", int(httpx.BodyLimitDefault)) + `"}`

	resp, body := h.do(limitsReq{method: "POST", path: "/auth/signup", ip: "203.0.113.20", body: big})
	if resp.StatusCode != http.StatusRequestEntityTooLarge || errorCode(body) != httpx.CodeBodyTooLarge {
		t.Fatalf("oversize signup: %d %s, want 413 body_too_large", resp.StatusCode, body)
	}
	if n := h.signups.Load(); n != 0 {
		t.Fatalf("identity was dialled %d time(s) for a declared oversize body", n)
	}

	// A cut stream: identity's handler may start reading before the gateway aborts it (and
	// may finish after the client already has its 413). Whenever it records, it must have
	// read at most the cap and then failed — never the whole overrun.
	overrun := func(rec signupRec) {
		t.Helper()
		if rec.n > httpx.BodyLimitDefault || rec.err == nil {
			t.Errorf("identity read %d bytes (err %v): it saw the overrun", rec.n, rec.err)
		}
	}
	resp, body = h.do(limitsReq{method: "POST", path: "/auth/signup", ip: "203.0.113.21", body: big, chunked: true})
	if resp.StatusCode != http.StatusRequestEntityTooLarge || errorCode(body) != httpx.CodeBodyTooLarge {
		t.Fatalf("chunked oversize signup: %d %s, want 413 body_too_large", resp.StatusCode, body)
	}
	select {
	case rec := <-h.signupRecs:
		overrun(rec)
	case <-time.After(2 * time.Second): // identity never started on it: fine
	}

	// A normal signup still streams through untouched (a late record of the cut stream
	// may arrive first; it is checked the same way).
	normal := `{"email":"a@b.c","password":"hunter2hunter"}`
	resp, body = h.do(limitsReq{method: "POST", path: "/auth/signup", ip: "203.0.113.22", body: normal})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("normal signup: %d %s", resp.StatusCode, body)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case rec := <-h.signupRecs:
			if rec.err != nil {
				overrun(rec)
				continue
			}
			if rec.n != int64(len(normal)) {
				t.Fatalf("identity read %d bytes of the normal signup, want %d", rec.n, len(normal))
			}
			return
		case <-deadline:
			t.Fatal("identity never recorded the normal signup")
		}
	}
}

// L6: every one of the 11 request-body sites answers an oversize body with the typed 413.
func TestOversizeBodyIs413AtEveryBodySite(t *testing.T) {
	h := newCourseHarness(t, 0)
	big := `{"pad":"` + strings.Repeat("x", int(httpx.BodyLimitDefault)) + `"}`
	for _, rt := range []struct{ method, path string }{
		{"PATCH", "/me"},                           // bff handlePatchMe
		{"POST", "/onboarding/step"},               // bff handleOnboardingStep
		{"POST", "/me/password"},                   // bff handleSetPassword
		{"POST", "/me/username"},                   // bff handleSetUsername
		{"POST", "/problems/zz-001/attempt/start"}, // bff proxyPracticeWrite
		{"POST", "/revision/r1/score"},             // bff handleRevisionScore
		{"POST", "/paths/zz-fixture/mocks"},        // assessment handleStartMock
		{"POST", "/mocks/m1/score"},                // assessment handleScoreMock
		{"PUT", "/coach/key"},                      // coach handlePutCoachKey
		{"POST", "/coach/chat"},                    // coach handleCoachChat
		{"POST", "/paths/zz-fixture/mistakes"},     // mistakes proxyReviewWrite
	} {
		h.reset()
		status, body := h.do(rt.method, rt.path, big)
		if status != http.StatusRequestEntityTooLarge || errorCode(body) != httpx.CodeBodyTooLarge {
			t.Errorf("%s %s: %d %s, want 413 body_too_large", rt.method, rt.path, status, body)
		}
		var env struct {
			Error struct {
				Limit int64 `json:"limit"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &env) == nil && env.Error.Limit != httpx.BodyLimitDefault {
			t.Errorf("%s %s: limit %d, want %d", rt.method, rt.path, env.Error.Limit, httpx.BodyLimitDefault)
		}
		// Nothing past the gateway saw the body.
		for _, c := range h.calls {
			if strings.HasPrefix(c, "practice POST") || strings.HasPrefix(c, "review POST") || strings.HasPrefix(c, "assessment POST") || strings.HasPrefix(c, "coach P") {
				t.Errorf("%s %s: an upstream was called: %s", rt.method, rt.path, c)
			}
		}
	}
}
