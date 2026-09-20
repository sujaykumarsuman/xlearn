package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// newAPIMux builds the BFF routes. The gateway validates the opaque session cookie
// with identity, mints a short-TTL JWT (ADR-0006), and forwards it on internal
// calls; OAuth start/callback are proxied through to identity so the browser only
// ever talks to the gateway origin.
func (g *Gateway) newAPIMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", g.appHealth)
	mux.HandleFunc("GET /.well-known/jwks.json", g.handleJWKS)
	mux.HandleFunc("GET /api/me", g.handleMe)
	mux.HandleFunc("POST /api/auth/logout", g.handleLogout)
	mux.HandleFunc("POST /api/onboarding/step", g.handleOnboardingStep)
	mux.HandleFunc("POST /api/auth/{provider}/start", g.handleAuthProxy)
	mux.HandleFunc("GET /api/auth/{provider}/callback", g.handleAuthProxy)
	// Curriculum content (read-only). Session-gated. Most routes are a straight proxy
	// (curriculum has no per-user state, so no user JWT is forwarded). The week route
	// is the exception: it is a BFF aggregation (api.md `agg`) — the gateway layers a
	// per-user five-touch/solve state onto the curriculum content (ADR-0005). That
	// state is a stable placeholder until practice/review exist (S05/S06, ADR-0013).
	mux.HandleFunc("GET /api/paths", g.handleListPaths)
	mux.HandleFunc("GET /api/paths/{slug}", g.handleGetPath)
	mux.HandleFunc("GET /api/paths/{slug}/weeks/{n}", g.handleGetWeek)
	mux.HandleFunc("GET /api/problems/{id}", g.handleGetProblem)
	mux.HandleFunc("GET /api/concepts/{slug}", g.handleGetConcept)
	// Catch-all: unknown /api/* is a 404 envelope, never the SPA shell.
	mux.HandleFunc("/api/", g.apiNotFound)
	return mux
}

// handleJWKS publishes the gateway's public verification keys (ADR-0006). The
// public key is not secret; a short cache is safe.
func (g *Gateway) handleJWKS(w http.ResponseWriter, r *http.Request) {
	if g.signer == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "signing not configured")
		return
	}
	auth.JWKSHandler(g.signer)(w, r)
}

// handleMe returns the current account + onboarding state (api.md GET /me). It
// validates the session cookie, mints an identity-scoped JWT, and passes through
// identity's response. Unauthenticated → the 401 envelope the SPA redirects on.
func (g *Gateway) handleMe(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	token, ok := g.mint(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.identity.getAccount(r.Context(), token, accountID)
	if err != nil {
		g.log.Error("bff /me: identity call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "identity unavailable")
		return
	}
	passthrough(w, status, body)
}

// handleOnboardingStep advances onboarding (api.md POST /onboarding/step),
// forwarding the JWT + body to identity.
func (g *Gateway) handleOnboardingStep(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	token, ok := g.mint(w, accountID)
	if !ok {
		return
	}
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "could not read body")
		return
	}
	body, status, err := g.identity.onboardingStep(r.Context(), token, reqBody)
	if err != nil {
		g.log.Error("bff /onboarding/step: identity call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "identity unavailable")
		return
	}
	passthrough(w, status, body)
}

// handleLogout revokes the session server-side and clears the cookie (api.md
// POST /auth/logout). Idempotent: no cookie still succeeds.
func (g *Gateway) handleLogout(w http.ResponseWriter, r *http.Request) {
	if g.identity == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "identity not configured")
		return
	}
	if c, err := r.Cookie(auth.SessionCookieName); err == nil && c.Value != "" {
		if rerr := g.identity.revokeSession(r.Context(), c.Value); rerr != nil {
			g.log.Warn("bff /auth/logout: revoke failed", "err", rerr)
		}
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// handleAuthProxy forwards OAuth start/callback to identity, passing cookies and
// copying Set-Cookie/Location back so the browser talks only to the gateway.
func (g *Gateway) handleAuthProxy(w http.ResponseWriter, r *http.Request) {
	if g.identity == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "identity not configured")
		return
	}
	// r.URL.Path here is the stripped app path, e.g. /api/auth/github/start.
	upstreamPath := "/auth/" + r.PathValue("provider") + "/" + lastSegment(r.URL.Path)
	g.identity.forward(w, r, upstreamPath)
}

func (g *Gateway) apiNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "no such endpoint")
}

// --- curriculum content proxy (read-only, session-gated) ---
//
// Content is read-only with no per-user state, so the gateway validates the session
// (consistent with the rest of /api) but does NOT forward a user JWT — curriculum
// has no user-scoped logic. It simply proxies; screen aggregation is a later sprint.

func (g *Gateway) handleListPaths(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.authAccount(w, r); !ok {
		return
	}
	g.proxyCurriculum(w, r, "/paths")
}

func (g *Gateway) handleGetPath(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.authAccount(w, r); !ok {
		return
	}
	g.proxyCurriculum(w, r, "/paths/"+url.PathEscape(r.PathValue("slug")))
}

// handleGetWeek is the week BFF aggregation (api.md `agg`): it fetches the curriculum
// week content, then layers a per-user five-touch/solve state (placeholder until
// practice/review land — S05/S06, ADR-0013). Curriculum's own status/envelope for a
// bad path or unknown week is propagated unchanged.
func (g *Gateway) handleGetWeek(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.authAccount(w, r); !ok {
		return
	}
	n := r.PathValue("n")
	if _, err := strconv.Atoi(n); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "week must be an integer")
		return
	}
	if g.curriculum == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "curriculum not configured")
		return
	}
	upstream := "/paths/" + url.PathEscape(r.PathValue("slug")) + "/weeks/" + url.PathEscape(n)
	body, status, err := g.curriculum.get(r.Context(), upstream)
	if err != nil {
		g.log.Error("bff week aggregation: curriculum call failed", "path", upstream, "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "curriculum unavailable")
		return
	}
	if status != http.StatusOK {
		// Propagate curriculum's own status + envelope (e.g. 404 for an unknown week).
		passthrough(w, status, body)
		return
	}
	merged, err := aggregateWeek(body)
	if err != nil {
		g.log.Error("bff week aggregation: merge failed", "path", upstream, "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "curriculum returned malformed content")
		return
	}
	passthrough(w, http.StatusOK, merged)
}

func (g *Gateway) handleGetProblem(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.authAccount(w, r); !ok {
		return
	}
	g.proxyCurriculum(w, r, "/problems/"+url.PathEscape(r.PathValue("id")))
}

func (g *Gateway) handleGetConcept(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.authAccount(w, r); !ok {
		return
	}
	g.proxyCurriculum(w, r, "/concepts/"+url.PathEscape(r.PathValue("slug")))
}

// proxyCurriculum forwards a GET to the curriculum service and passes its JSON
// response (including its own 404/error envelopes) straight through.
func (g *Gateway) proxyCurriculum(w http.ResponseWriter, r *http.Request, upstreamPath string) {
	if g.curriculum == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "curriculum not configured")
		return
	}
	body, status, err := g.curriculum.get(r.Context(), upstreamPath)
	if err != nil {
		g.log.Error("bff curriculum call failed", "path", upstreamPath, "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "curriculum unavailable")
		return
	}
	passthrough(w, status, body)
}

// authAccount resolves the account id from the session cookie via identity, or
// writes the 401 envelope and returns ok=false.
func (g *Gateway) authAccount(w http.ResponseWriter, r *http.Request) (string, bool) {
	if g.identity == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "identity not configured")
		return "", false
	}
	c, err := r.Cookie(auth.SessionCookieName)
	if err != nil || c.Value == "" {
		writeUnauthenticated(w)
		return "", false
	}
	accountID, err := g.identity.validateSession(r.Context(), c.Value)
	if err != nil {
		writeUnauthenticated(w)
		return "", false
	}
	return accountID, true
}

// mint issues an identity-scoped JWT for accountID (roles: learner).
func (g *Gateway) mint(w http.ResponseWriter, accountID string) (string, bool) {
	if g.signer == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "signing not configured")
		return "", false
	}
	token, err := g.signer.Mint(context.Background(), accountID, g.audIdentity, []string{"learner"})
	if err != nil {
		g.log.Error("mint jwt", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not mint token")
		return "", false
	}
	return token, true
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/xlearn",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func passthrough(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": code, "message": message}})
}

func writeUnauthenticated(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
}

func lastSegment(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[i+1:]
		}
	}
	return p
}

// --- identity client ---

// identityClient calls the internal identity service.
type identityClient struct {
	baseURL string
	httpc   *http.Client
	// proxyc does not follow redirects, so OAuth 302s pass through to the browser.
	proxyc *http.Client
}

func newIdentityClient(baseURL string) *identityClient {
	return &identityClient{
		baseURL: baseURL,
		httpc:   &http.Client{Timeout: 10 * time.Second},
		proxyc: &http.Client{
			Timeout:       15 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (c *identityClient) validateSession(ctx context.Context, sessionID string) (string, error) {
	body, status, err := c.postJSON(ctx, "/sessions/validate", "", map[string]string{"session_id": sessionID})
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("validate session: status %d", status)
	}
	var vr struct {
		AccountID string `json:"account_id"`
	}
	if err := json.Unmarshal(body, &vr); err != nil || vr.AccountID == "" {
		return "", fmt.Errorf("validate session: bad response")
	}
	return vr.AccountID, nil
}

func (c *identityClient) revokeSession(ctx context.Context, sessionID string) error {
	_, _, err := c.postJSON(ctx, "/sessions/revoke", "", map[string]string{"session_id": sessionID})
	return err
}

func (c *identityClient) getAccount(ctx context.Context, token, accountID string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/accounts/"+accountID, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

func (c *identityClient) onboardingStep(ctx context.Context, token string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/onboarding/step", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *identityClient) postJSON(ctx context.Context, path, token string, payload any) ([]byte, int, error) {
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.do(req)
}

func (c *identityClient) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return body, resp.StatusCode, err
}

// --- curriculum client ---

// curriculumClient calls the internal curriculum service (read-only content).
type curriculumClient struct {
	baseURL string
	httpc   *http.Client
}

func newCurriculumClient(baseURL string) *curriculumClient {
	return &curriculumClient{
		baseURL: baseURL,
		httpc:   &http.Client{Timeout: 10 * time.Second},
	}
}

// get issues a GET to the curriculum service and returns the raw body + status.
func (c *curriculumClient) get(ctx context.Context, path string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return body, resp.StatusCode, err
}

// forward proxies an OAuth request to identity, passing cookies + query and
// copying status, Location and Set-Cookie back (no redirect following).
func (c *identityClient) forward(w http.ResponseWriter, r *http.Request, upstreamPath string) {
	target := c.baseURL + upstreamPath
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "proxy build failed")
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := c.proxyc.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream", "identity unavailable")
		return
	}
	defer resp.Body.Close()
	for _, sc := range resp.Header.Values("Set-Cookie") {
		w.Header().Add("Set-Cookie", sc)
	}
	if loc := resp.Header.Get("Location"); loc != "" {
		w.Header().Set("Location", loc)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 4<<20))
}
