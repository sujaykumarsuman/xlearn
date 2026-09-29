package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// wantCSP pins the exact shell policy (m1-04 task 6). It is spelled out here rather than
// read from contentSecurityPolicy so a change to the constant has to change this test too.
const wantCSP = "default-src 'self'; script-src 'self'; style-src 'self' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; object-src 'none'; " +
	"base-uri 'self'; frame-ancestors 'none'; form-action 'self' https://github.com"

// doWith sends a request through Handler() with the given headers (name → value).
func doWith(g *Gateway, method, target string, headers map[string]string) *http.Response {
	r := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	g.Handler().ServeHTTP(w, r)
	return w.Result()
}

// respErrorCode decodes the error envelope's code ("" when the body is not an envelope).
func respErrorCode(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, _ := io.ReadAll(resp.Body)
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(b, &env)
	return env.Error.Code
}

func TestSecurity_ShellCSP(t *testing.T) {
	for _, c := range []struct {
		base, path string
	}{
		{"/xlearn", "/"},                         // pod root behind Traefik stripPrefix
		{"/xlearn", "/xlearn"},                   // the base itself
		{"/xlearn", "/xlearn/"},                  // base with trailing slash
		{"/xlearn", "/xlearn/dsa"},               // a course home
		{"/xlearn", "/dsa"},                      // same route, prefix already stripped
		{"/xlearn", "/xlearn/no/such/spa/route"}, // unknown SPA route → shell fallback
		{"", "/dsa/week/2"},                      // root-mounted gateway
	} {
		resp := do(testGateway(c.base), http.MethodGet, c.path)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("base %q %s = %d, want 200", c.base, c.path, resp.StatusCode)
		}
		if got := resp.Header.Get("Content-Security-Policy"); got != wantCSP {
			t.Errorf("base %q %s CSP =\n  %q\nwant\n  %q", c.base, c.path, got, wantCSP)
		}
		if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("base %q %s nosniff = %q", c.base, c.path, got)
		}
		if got := resp.Header.Get("Referrer-Policy"); got != "same-origin" {
			t.Errorf("base %q %s Referrer-Policy = %q, want same-origin", c.base, c.path, got)
		}
	}
}

func TestSecurity_CSPScriptSrcHasNoUnsafeInline(t *testing.T) {
	resp := do(testGateway("/xlearn"), http.MethodGet, "/xlearn/")
	csp := resp.Header.Get("Content-Security-Policy")
	var scriptSrc string
	for _, d := range strings.Split(csp, ";") {
		if d = strings.TrimSpace(d); strings.HasPrefix(d, "script-src ") {
			scriptSrc = d
		}
	}
	if scriptSrc == "" {
		t.Fatalf("CSP has no script-src directive: %q", csp)
	}
	if strings.Contains(scriptSrc, "'unsafe-inline'") || strings.Contains(scriptSrc, "'unsafe-eval'") {
		t.Fatalf("script-src must not allow inline/eval: %q", scriptSrc)
	}
	if !strings.Contains(csp, "form-action 'self' https://github.com") {
		t.Fatalf("form-action must allow the OAuth redirect to github.com: %q", csp)
	}
}

// Every response carries nosniff: probes, API (success and error), assets and the shell.
func TestSecurity_NosniffEverywhere(t *testing.T) {
	g := testGateway("/xlearn")
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/healthz"},
		{http.MethodGet, "/readyz"},
		{http.MethodGet, "/xlearn/api/healthz"},
		{http.MethodGet, "/xlearn/api/v1/healthz"},
		{http.MethodGet, "/xlearn/api/v1/does-not-exist"},
		{http.MethodGet, "/xlearn/assets/index-abc123.js"},
		{http.MethodGet, "/xlearn/assets/index-abc123.css"},
		{http.MethodGet, "/xlearn/favicon.svg"},
		{http.MethodGet, "/xlearn/"},
		{http.MethodPost, "/xlearn/api/v1/auth/logout"}, // refused with 415 — still nosniff
	} {
		resp := do(g, c.method, c.path)
		if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s %s (%d) X-Content-Type-Options = %q, want nosniff", c.method, c.path, resp.StatusCode, got)
		}
	}
	// The CSP is a shell header: API JSON and hashed assets don't need it.
	if got := do(g, http.MethodGet, "/xlearn/api/healthz").Header.Get("Content-Security-Policy"); got != "" {
		t.Errorf("API response carries the shell CSP: %q", got)
	}
}

// A cross-site (or same-site sibling host) write is refused before it reaches a handler,
// on every spelling of the API path (base-prefixed or not, versioned or not).
func TestSecurity_CrossSiteWriteForbidden(t *testing.T) {
	g := testGateway("/xlearn")
	jsonCT := map[string]string{"Content-Type": "application/json"}
	for _, site := range []string{"cross-site", "same-site", "Cross-Site"} {
		for _, c := range []struct{ method, path string }{
			{http.MethodPost, "/xlearn/api/v1/auth/logout"},
			{http.MethodPost, "/xlearn/api/auth/logout"},
			{http.MethodPost, "/api/v1/auth/logout"},
			{http.MethodPost, "/api/auth/logout"},
			{http.MethodPatch, "/xlearn/api/v1/me"},
			{http.MethodPut, "/xlearn/api/v1/coach/key"},
			{http.MethodDelete, "/xlearn/api/v1/me/oauth/github"},
			{http.MethodPost, "/xlearn/api/v1/does-not-exist"},
			// The OAuth start form POST keeps the Sec-Fetch-Site check.
			{http.MethodPost, "/xlearn/api/v1/auth/github/start"},
			{http.MethodPost, "/xlearn/api/v1/auth/github/start?link=1"},
		} {
			h := map[string]string{"Sec-Fetch-Site": site}
			for k, v := range jsonCT {
				h[k] = v
			}
			resp := doWith(g, c.method, c.path, h)
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("%s %s Sec-Fetch-Site=%s = %d, want 403", c.method, c.path, site, resp.StatusCode)
			}
			if code := respErrorCode(t, resp); code != "cross_site_request" {
				t.Fatalf("%s %s Sec-Fetch-Site=%s code = %q, want cross_site_request", c.method, c.path, site, code)
			}
		}
	}
	// The origin check runs first: a cross-site non-JSON write is a 403, not a 415.
	resp := doWith(g, http.MethodPost, "/xlearn/api/v1/auth/logout", map[string]string{"Sec-Fetch-Site": "cross-site", "Content-Type": "text/plain"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-site text/plain write = %d, want 403", resp.StatusCode)
	}
}

// Same-origin and user-initiated (none) writes pass the origin check, as do non-browser
// clients that send no Sec-Fetch-Site at all.
func TestSecurity_SameOriginWriteAllowed(t *testing.T) {
	g := testGateway("/xlearn")
	for _, site := range []string{"same-origin", "none", ""} {
		h := map[string]string{"Content-Type": "application/json"}
		if site != "" {
			h["Sec-Fetch-Site"] = site
		}
		for _, path := range []string{"/xlearn/api/v1/auth/logout", "/api/auth/logout"} {
			resp := doWith(g, http.MethodPost, path, h)
			if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnsupportedMediaType {
				t.Fatalf("POST %s Sec-Fetch-Site=%q = %d (%s), want it to reach the handler", path, site, resp.StatusCode, respErrorCode(t, resp))
			}
		}
	}
}

// A mutating call must be JSON: no Content-Type, text/plain or a form encoding is a 415 on
// every method; application/json with parameters passes.
func TestSecurity_WriteRequiresJSON(t *testing.T) {
	g := testGateway("/xlearn")
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, ct := range []string{"", "text/plain", "text/plain;charset=UTF-8", "application/x-www-form-urlencoded", "multipart/form-data; boundary=x", "application/jsonx", "application/json-patch+json"} {
			h := map[string]string{"Sec-Fetch-Site": "same-origin"}
			if ct != "" {
				h["Content-Type"] = ct
			}
			for _, path := range []string{"/xlearn/api/v1/me", "/api/me"} {
				resp := doWith(g, method, path, h)
				if resp.StatusCode != http.StatusUnsupportedMediaType {
					t.Fatalf("%s %s Content-Type=%q = %d, want 415", method, path, ct, resp.StatusCode)
				}
				if code := respErrorCode(t, resp); code != "unsupported_media_type" {
					t.Fatalf("%s %s Content-Type=%q code = %q, want unsupported_media_type", method, path, ct, code)
				}
			}
		}
	}
	// JSON — any case, with parameters — passes the check and reaches the mux.
	for _, ct := range []string{"application/json", "application/json; charset=utf-8", "Application/JSON; charset=UTF-8"} {
		resp := doWith(g, http.MethodPost, "/xlearn/api/v1/auth/logout", map[string]string{"Content-Type": ct})
		if resp.StatusCode == http.StatusUnsupportedMediaType || resp.StatusCode == http.StatusForbidden {
			t.Fatalf("POST with Content-Type=%q = %d, want it to pass the JSON check", ct, resp.StatusCode)
		}
	}
}

// Reads are unaffected: no Content-Type needed, and Sec-Fetch-Site is not checked.
func TestSecurity_ReadsUnaffected(t *testing.T) {
	g := testGateway("/xlearn")
	for _, h := range []map[string]string{nil, {"Sec-Fetch-Site": "cross-site"}} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			resp := doWith(g, method, "/xlearn/api/v1/healthz", h)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%s /api/v1/healthz headers=%v = %d, want 200", method, h, resp.StatusCode)
			}
		}
	}
	// The JWKS (pod-root, internal) is not an /api route and is never write-checked.
	if resp := doWith(g, http.MethodGet, "/.well-known/jwks.json", map[string]string{"Sec-Fetch-Site": "cross-site"}); resp.StatusCode == http.StatusForbidden {
		t.Fatalf("JWKS GET = 403")
	}
}

// The OAuth start is a top-level HTML form POST (sign-in and Settings' "Connect GitHub"):
// it can't be JSON, so it keeps only the Sec-Fetch-Site check and reaches identity's proxy.
func TestSecurity_OAuthStartFormPostAllowed(t *testing.T) {
	h := newBFFHarness(t)
	for _, path := range []string{"/xlearn/api/v1/auth/github/start", "/xlearn/api/auth/github/start", "/xlearn/api/v1/auth/github/start?link=1"} {
		req, _ := http.NewRequest(http.MethodPost, h.gwServer.URL+path, strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		resp, err := noRedirect().Do(req)
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusFound {
			t.Fatalf("POST %s = %d, want 302 from the identity proxy", path, resp.StatusCode)
		}
		if !strings.Contains(resp.Header.Get("Location"), "github.com/login/oauth/authorize") {
			t.Fatalf("POST %s did not reach the proxy: Location %q", path, resp.Header.Get("Location"))
		}
	}
	// The exemption is the start route only — a form POST anywhere else is still a 415.
	g := testGateway("/xlearn")
	for _, path := range []string{"/xlearn/api/v1/auth/login", "/xlearn/api/v1/auth/signup", "/xlearn/api/v1/auth/github/callback", "/xlearn/api/v1/auth/github/start/x"} {
		resp := doWith(g, http.MethodPost, path, map[string]string{"Content-Type": "application/x-www-form-urlencoded", "Sec-Fetch-Site": "same-origin"})
		if resp.StatusCode != http.StatusUnsupportedMediaType {
			t.Fatalf("form POST %s = %d, want 415", path, resp.StatusCode)
		}
	}
}

func TestIsOAuthStart(t *testing.T) {
	for _, c := range []struct {
		method, path string
		want         bool
	}{
		{http.MethodPost, "/api/auth/github/start", true},
		{http.MethodPost, "/api/auth/google/start", true},
		{http.MethodGet, "/api/auth/github/start", false},
		{http.MethodPost, "/api/auth/github/callback", false},
		{http.MethodPost, "/api/auth//start", false},
		{http.MethodPost, "/api/auth/start", false},
		{http.MethodPost, "/api/auth/github/start/x", false},
		{http.MethodPost, "/api/auth/logout", false},
		{http.MethodPost, "/api/v1/auth/github/start", false}, // callers pass the rewritten path
	} {
		if got := isOAuthStart(c.method, c.path); got != c.want {
			t.Errorf("isOAuthStart(%s, %s) = %v, want %v", c.method, c.path, got, c.want)
		}
	}
}
