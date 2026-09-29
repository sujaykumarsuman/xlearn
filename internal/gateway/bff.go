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
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
)

// apiRoute is one registered BFF endpoint. The set of routes is a first-class value
// (apiRoutes) so both the mux and the OpenAPI drift check read the SAME source of
// truth — a route can't be added without the spec-drift test noticing (see
// openapi_drift_test.go).
type apiRoute struct {
	Method  string // GET|POST|PUT|PATCH|DELETE
	Pattern string // the app path, e.g. "/api/paths/{slug}/weeks/{n}"
	Handler http.HandlerFunc
	// Doc is false for routes intentionally excluded from the public OpenAPI contract
	// (ops/JWKS): they are served but not part of the documented /xlearn/api surface.
	Doc bool
	// Alias marks a v1 route without a course: a DSA alias that runs the course-scoped
	// handler with course.DefaultSlug (m1-03). OpenAPI marks it deprecated: true (the drift
	// test checks). The SPA stops calling aliases in v1.7.0; they stay for open v1.6.0
	// tabs at least through v1.8.0 (ADR-0034 §1.1; status.md records the removal tag).
	Alias bool
}

// apiRoutes is the authoritative BFF route table. Order is irrelevant to correctness
// (Go 1.22's ServeMux matches most-specific-first, so /mocks/trend beats /mocks/{id}
// regardless of registration order); it is grouped by feature for readability.
func (g *Gateway) apiRoutes() []apiRoute {
	return []apiRoute{
		// System / auth infra.
		{Method: "GET", Pattern: "/api/healthz", Handler: g.appHealth},
		{Method: "GET", Pattern: "/.well-known/jwks.json", Handler: g.handleJWKS},
		// Account + onboarding (identity-backed). OAuth start/callback are proxied so the
		// browser only ever talks to the gateway origin.
		{Method: "GET", Pattern: "/api/me", Handler: g.handleMe, Doc: true},
		{Method: "PATCH", Pattern: "/api/me", Handler: g.handlePatchMe, Doc: true},
		// Account & sign-in management (ADR-0023): set/change password + disconnect a provider.
		{Method: "POST", Pattern: "/api/me/password", Handler: g.handleSetPassword, Doc: true},
		{Method: "DELETE", Pattern: "/api/me/oauth/{provider}", Handler: g.handleUnlinkOAuth, Doc: true},
		// Username (F009 / ADR-0024): claim/change + availability check (session-gated).
		{Method: "POST", Pattern: "/api/me/username", Handler: g.handleSetUsername, Doc: true},
		{Method: "GET", Pattern: "/api/username/available", Handler: g.handleUsernameAvailable, Doc: true},
		{Method: "POST", Pattern: "/api/auth/logout", Handler: g.handleLogout, Doc: true},
		{Method: "POST", Pattern: "/api/onboarding/step", Handler: g.handleOnboardingStep, Doc: true},
		// Email/password auth (ADR-0023): fetch-based signup/login (session cookie on the JSON
		// response). OAuth start/callback are the browser-redirect flow; `?link=1` on start
		// connects the provider to the signed-in account.
		{Method: "POST", Pattern: "/api/auth/signup", Handler: g.handleAuthSignup, Doc: true},
		{Method: "POST", Pattern: "/api/auth/login", Handler: g.handleAuthLogin, Doc: true},
		{Method: "POST", Pattern: "/api/auth/{provider}/start", Handler: g.handleAuthProxy, Doc: true},
		{Method: "GET", Pattern: "/api/auth/{provider}/callback", Handler: g.handleAuthProxy, Doc: true},
		// Local-only dev login (F002 / ADR-0022): proxied to identity, which 404s them
		// unless DEV_AUTH is set. Undocumented (Doc:false) — never part of the prod surface.
		{Method: "POST", Pattern: "/api/auth/dev/login", Handler: g.handleAuthDevProxy},
		{Method: "GET", Pattern: "/api/auth/dev/enabled", Handler: g.handleAuthDevProxy},
		// Per-user course enrollment (F002): starting a course is an explicit, durable
		// action; only an active course can be started (ADR-0033 §12 row 8).
		{Method: "POST", Pattern: "/api/paths/{slug}/start", Handler: g.handleStartPath, Doc: true},
		// The course catalog + course content (read-only, session-gated). Every
		// /api/paths/{slug}/… route resolves {slug} against the compiled-in manifests
		// (course.go). The week route is a BFF aggregation (api.md `agg`): the gateway
		// layers per-user five-touch/solve state onto curriculum content (ADR-0005/0013).
		{Method: "GET", Pattern: "/api/paths", Handler: g.handleListPaths, Doc: true},
		{Method: "GET", Pattern: "/api/paths/{slug}", Handler: g.handleGetPath, Doc: true},
		{Method: "GET", Pattern: "/api/paths/{slug}/problems", Handler: g.handleListPathProblems, Doc: true},
		{Method: "GET", Pattern: "/api/paths/{slug}/weeks/{n}", Handler: g.handleGetWeek, Doc: true},
		{Method: "GET", Pattern: "/api/paths/{slug}/concepts/{c}", Handler: g.handleGetConcept, Doc: true},
		// Course-scoped aggregates (m1-03, t0 §7): the same handlers the DSA aliases below run.
		{Method: "GET", Pattern: "/api/paths/{slug}/dashboard", Handler: g.handleDashboard, Doc: true},
		{Method: "GET", Pattern: "/api/paths/{slug}/progress", Handler: g.handleProgress, Doc: true},
		{Method: "GET", Pattern: "/api/paths/{slug}/revision/due", Handler: g.handleRevisionDue, Doc: true},
		{Method: "GET", Pattern: "/api/paths/{slug}/mistakes", Handler: g.handleMistakes, Doc: true},
		{Method: "POST", Pattern: "/api/paths/{slug}/mistakes", Handler: g.handleCreateMistake, Doc: true},
		{Method: "GET", Pattern: "/api/paths/{slug}/weak-area", Handler: g.handleWeakArea, Doc: true},
		{Method: "POST", Pattern: "/api/paths/{slug}/mocks", Handler: g.handleStartMock, Doc: true},
		{Method: "GET", Pattern: "/api/paths/{slug}/mocks/trend", Handler: g.handleMockTrend, Doc: true},
		// Items by GLOBAL id: the course comes from the item's path_slug (curriculum) or,
		// for revision items, mistakes and mocks, from the owning service's row. The
		// Problem GET is a BFF aggregation (content limited to unlocked stages + practice
		// state + timer); the writes proxy to practice.
		{Method: "GET", Pattern: "/api/problems/{id}", Handler: g.handleGetProblem, Doc: true},
		{Method: "POST", Pattern: "/api/problems/{id}/attempt/start", Handler: g.handleAttemptStart, Doc: true},
		{Method: "POST", Pattern: "/api/problems/{id}/reveal", Handler: g.handleReveal, Doc: true},
		{Method: "POST", Pattern: "/api/problems/{id}/outcome", Handler: g.handleOutcome, Doc: true},
		// Revision (review). External /revision maps to review's internal /revisions.
		{Method: "POST", Pattern: "/api/revision/{itemId}/score", Handler: g.handleRevisionScore, Doc: true},
		// Mistake journal edits (review).
		{Method: "PATCH", Pattern: "/api/mistakes/{id}", Handler: g.handlePatchMistake, Doc: true},
		// Mock interview (assessment). /mocks/trend (an alias below) is more specific than
		// /mocks/{id}, so it wins regardless of order.
		{Method: "GET", Pattern: "/api/mocks/{id}", Handler: g.handleGetMock, Doc: true},
		{Method: "POST", Pattern: "/api/mocks/{id}/score", Handler: g.handleScoreMock, Doc: true},
		// DSA aliases (ADR-0034 §1.1): the v1 routes without a course, served by the
		// course-scoped handler with course.DefaultSlug (the course a v1 caller means).
		{Method: "GET", Pattern: "/api/concepts/{slug}", Handler: g.aliasConcept, Doc: true, Alias: true},
		{Method: "GET", Pattern: "/api/revision/due", Handler: g.alias(g.handleRevisionDue), Doc: true, Alias: true},
		{Method: "GET", Pattern: "/api/mistakes", Handler: g.alias(g.handleMistakes), Doc: true, Alias: true},
		{Method: "POST", Pattern: "/api/mistakes", Handler: g.alias(g.handleCreateMistake), Doc: true, Alias: true},
		{Method: "GET", Pattern: "/api/weak-area", Handler: g.alias(g.handleWeakArea), Doc: true, Alias: true},
		{Method: "POST", Pattern: "/api/mocks", Handler: g.alias(g.handleStartMock), Doc: true, Alias: true},
		{Method: "GET", Pattern: "/api/mocks/trend", Handler: g.alias(g.handleMockTrend), Doc: true, Alias: true},
		{Method: "GET", Pattern: "/api/progress", Handler: g.alias(g.handleProgress), Doc: true, Alias: true},
		{Method: "GET", Pattern: "/api/dashboard", Handler: g.alias(g.handleDashboard), Doc: true, Alias: true},
		// PUBLIC user dashboard (F009 / ADR-0024): the ONLY unauthenticated /api route —
		// resolves a username to non-PII public stats + a merged activity heatmap.
		{Method: "GET", Pattern: "/api/u/{username}", Handler: g.handlePublicProfile, Doc: true},
		// Coach (S11): masked key CRUD, per-page thread, and the SSE chat relay.
		{Method: "GET", Pattern: "/api/coach/key", Handler: g.handleCoachKey, Doc: true},
		{Method: "PUT", Pattern: "/api/coach/key", Handler: g.handlePutCoachKey, Doc: true},
		{Method: "DELETE", Pattern: "/api/coach/key", Handler: g.handleDeleteCoachKey, Doc: true},
		{Method: "GET", Pattern: "/api/coach/thread", Handler: g.handleCoachThread, Doc: true},
		{Method: "POST", Pattern: "/api/coach/chat", Handler: g.handleCoachChat, Doc: true},
	}
}

// alias wraps a course-scoped handler as its v1 DSA alias: the request runs with the
// {slug} path value set to course.DefaultSlug, so the alias and the course-scoped route
// share one code path (and return byte-identical bodies for the same upstream state).
func (g *Gateway) alias(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("slug", course.DefaultSlug)
		h(w, r)
	}
}

// aliasConcept is GET /api/concepts/{slug}'s alias: v1's {slug} is the CONCEPT slug, so it
// moves to {c} and the course becomes course.DefaultSlug.
func (g *Gateway) aliasConcept(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("c", r.PathValue("slug"))
	r.SetPathValue("slug", course.DefaultSlug)
	g.handleGetConcept(w, r)
}

// newAPIMux builds the BFF routes from the authoritative apiRoutes table. The gateway
// validates the opaque session cookie with identity, mints a short-TTL JWT (ADR-0006),
// and forwards it on internal calls; OAuth start/callback are proxied through to
// identity so the browser only ever talks to the gateway origin.
func (g *Gateway) newAPIMux() *http.ServeMux {
	mux := http.NewServeMux()
	for _, rt := range g.apiRoutes() {
		h := rt.Handler
		// Each request gets its own session slot (cohort.go): authSession fills it, the
		// course-visibility checks read it. Nothing outlives the request.
		mux.HandleFunc(rt.Method+" "+rt.Pattern, func(w http.ResponseWriter, r *http.Request) {
			h(w, withRequestSession(r))
		})
	}
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

// handlePatchMe updates the caller's own profile / study budget / timezone / reminders
// (api.md PATCH /me), forwarding the JWT + body to identity's PATCH /accounts/{id}
// (identity enforces ownership from the token subject; ADR-0006).
func (g *Gateway) handlePatchMe(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	token, ok := g.mint(w, accountID)
	if !ok {
		return
	}
	reqBody, ok := httpx.ReadBody(w, r, httpx.BodyLimitDefault)
	if !ok {
		return
	}
	body, status, err := g.identity.patchAccount(r.Context(), token, accountID, reqBody)
	if err != nil {
		g.log.Error("bff PATCH /me: identity call failed", "err", err)
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
	reqBody, ok := httpx.ReadBody(w, r, httpx.BodyLimitDefault)
	if !ok {
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
	// L2: POST /auth/{provider}/start is 5/min per IP. The GET callback is not limited: it
	// is the provider redirecting the browser back.
	if r.Method == http.MethodPost && !allowIP(w, r, g.limits.signupIP, "start") {
		return
	}
	// r.URL.Path here is the stripped app path, e.g. /api/auth/github/start.
	upstreamPath := "/auth/" + r.PathValue("provider") + "/" + lastSegment(r.URL.Path)
	g.identity.forward(w, r, upstreamPath)
}

// handleAuthSignup / handleAuthLogin forward the email/password body to identity, which
// creates/authenticates the account and sets the session cookie on its JSON response
// (ADR-0023). The gateway passes Set-Cookie back so the browser talks only to the gateway.
// Signup is L2 (5/min per IP); login is L1 (limits.go: per IP plus the per-identifier
// failure window).
func (g *Gateway) handleAuthSignup(w http.ResponseWriter, r *http.Request) {
	if g.identity == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "identity not configured")
		return
	}
	if !allowIP(w, r, g.limits.signupIP, "signup") {
		return
	}
	g.identity.forward(w, r, "/auth/signup")
}

func (g *Gateway) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if g.identity == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "identity not configured")
		return
	}
	g.loginWithLimits(w, r)
}

// handleSetPassword forwards a set/change-password request to identity's
// POST /accounts/{id}/password (JWT-gated; identity enforces the current-password check).
func (g *Gateway) handleSetPassword(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	token, ok := g.mint(w, accountID)
	if !ok {
		return
	}
	reqBody, ok := httpx.ReadBody(w, r, httpx.BodyLimitDefault)
	if !ok {
		return
	}
	body, status, err := g.identity.setPassword(r.Context(), token, accountID, reqBody)
	if err != nil {
		g.log.Error("bff /me/password: identity call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "identity unavailable")
		return
	}
	switch status {
	case http.StatusOK:
		// identity revoked every session, this one included (m1-04): drop the dead cookie
		// too; the SPA sends the learner to sign in again ({reauth:true}).
		clearSessionCookie(w)
	case http.StatusTooManyRequests:
		w.Header().Set("Retry-After", "1") // L3: bcrypt busy
	}
	passthrough(w, status, body)
}

// handleUnlinkOAuth forwards a disconnect-provider request to identity's
// DELETE /accounts/{id}/oauth/{provider} (JWT-gated; identity guards the last login method).
func (g *Gateway) handleUnlinkOAuth(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	token, ok := g.mint(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.identity.unlinkOAuth(r.Context(), token, accountID, r.PathValue("provider"))
	if err != nil {
		g.log.Error("bff DELETE /me/oauth: identity call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "identity unavailable")
		return
	}
	if status == http.StatusNoContent {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	passthrough(w, status, body)
}

// handleSetUsername forwards a username claim/change to identity's POST /accounts/{id}/username
// (JWT-gated; identity validates format + reserved words + case-insensitive uniqueness). F009.
func (g *Gateway) handleSetUsername(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	token, ok := g.mint(w, accountID)
	if !ok {
		return
	}
	reqBody, ok := httpx.ReadBody(w, r, httpx.BodyLimitDefault)
	if !ok {
		return
	}
	body, status, err := g.identity.setUsername(r.Context(), token, accountID, reqBody)
	if err != nil {
		g.log.Error("bff /me/username: identity call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "identity unavailable")
		return
	}
	passthrough(w, status, body)
}

// handleUsernameAvailable forwards the availability check to identity (session-gated; only
// signed-in users claim usernames). F009.
func (g *Gateway) handleUsernameAvailable(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	token, ok := g.mint(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.identity.usernameAvailable(r.Context(), token, r.URL.Query().Get("u"))
	if err != nil {
		g.log.Error("bff /username/available: identity call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "identity unavailable")
		return
	}
	passthrough(w, status, body)
}

// handleAuthDevProxy forwards the LOCAL-ONLY dev-login endpoints to identity, which
// gates them on DEV_AUTH (they 404 in prod). Cookies + Set-Cookie pass through so the
// minted dev session lands in the browser (F002 / ADR-0022).
func (g *Gateway) handleAuthDevProxy(w http.ResponseWriter, r *http.Request) {
	if g.identity == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "identity not configured")
		return
	}
	// r.URL.Path is the stripped app path, e.g. /api/auth/dev/login → /auth/dev/login.
	g.identity.forward(w, r, "/auth/dev/"+lastSegment(r.URL.Path))
}

// handleStartPath enrolls the caller in a course (F002 · POST /paths/{slug}/start),
// minting an identity-scoped JWT and forwarding to identity. Idempotent. Only an active
// course can be started (ADR-0033 §12 row 8): an unknown or preview course is the uniform
// 404 course_not_found — except a preview course for the owner/tester cohort (m1-04) —
// and a coming_soon or retired one is 409 course_not_available. identity enforces the
// same rule itself, from its own row.
func (g *Gateway) handleStartPath(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	slug := r.PathValue("slug")
	m, err := g.resolveCourse(slug)
	switch {
	case err != nil, m.Status == course.StatusPreview && !g.cohort(r):
		writeCourseNotFound(w)
		return
	case m.Status == course.StatusComingSoon, m.Status == course.StatusRetired:
		writeError(w, http.StatusConflict, "course_not_available", "this course is not open for enrollment")
		return
	}
	token, ok := g.mint(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.identity.startEnrollment(r.Context(), token, slug)
	if err != nil {
		g.log.Error("bff /paths/{slug}/start: identity call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "identity unavailable")
		return
	}
	passthrough(w, status, body)
}

func (g *Gateway) apiNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "no such endpoint")
}

// --- curriculum content proxy (read-only, session-gated) ---
//
// Content is read-only with no per-user state, so the gateway validates the session
// (consistent with the rest of /api) but does NOT forward a user JWT — curriculum
// has no user-scoped logic. It simply proxies; screen aggregation is a later sprint.

// handleListPaths is the session-gated course catalog: curriculum's GET /paths (every
// non-retired course with its learner-safe `course` view), filtered to what this caller
// may see — active courses, coming_soon teasers, and preview only for the cohort.
func (g *Gateway) handleListPaths(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.authAccount(w, r); !ok {
		return
	}
	if g.curriculum == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "curriculum not configured")
		return
	}
	body, status, err := g.curriculum.get(r.Context(), "/paths")
	if err != nil {
		g.log.Error("bff curriculum call failed", "path", "/paths", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "curriculum unavailable")
		return
	}
	if status != http.StatusOK {
		passthrough(w, status, body)
		return
	}
	passthrough(w, http.StatusOK, g.filterCatalog(r, body))
}

// handleGetPath is a course's Roadmap content (GET /paths/{slug}): visible courses only.
func (g *Gateway) handleGetPath(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.authAccount(w, r); !ok {
		return
	}
	slug := r.PathValue("slug")
	if _, ok := g.requireCourse(w, r, slug); !ok {
		return
	}
	g.proxyCurriculum(w, r, "/paths/"+url.PathEscape(slug))
}

// handleListPathProblems proxies the whole problem index for a course (the Problems
// arena, review round 2 — a flat list you can browse + attempt any problem from).
func (g *Gateway) handleListPathProblems(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.authAccount(w, r); !ok {
		return
	}
	slug := r.PathValue("slug")
	if _, ok := g.requireCourse(w, r, slug); !ok {
		return
	}
	g.proxyCurriculum(w, r, "/paths/"+url.PathEscape(slug)+"/problems")
}

// handleGetWeek is the week BFF aggregation (api.md `agg`): it fetches the curriculum
// week content, then layers the learner's per-problem solve state from practice
// (S05) — falling back to the honest placeholder when practice is unavailable
// (ADR-0013; the five-touch schedule itself lands with review, S06). Curriculum's own
// status/envelope for a bad path or unknown week is propagated unchanged.
func (g *Gateway) handleGetWeek(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if _, ok := g.requireCourse(w, r, r.PathValue("slug")); !ok {
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
	cacheName := "week:" + r.PathValue("slug") + ":" + n
	if cached, ok := g.cache.get(accountID, cacheName); ok {
		passthrough(w, http.StatusOK, cached)
		return
	}
	cacheEpoch := g.cache.epoch(accountID)
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
	states, populated := g.weekPracticeStates(r, accountID, body)
	merged, err := aggregateWeek(body, states, populated)
	if err != nil {
		g.log.Error("bff week aggregation: merge failed", "path", upstream, "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "curriculum returned malformed content")
		return
	}
	// Only cache when practice actually sourced the solve state. When practice is
	// degraded the merge is the honest placeholder (every problem "available",
	// populated:false); caching that would pin the learner's progress as unsolved for
	// the whole TTL after practice recovers, so recompute next time instead.
	if populated {
		g.cache.putFresh(accountID, cacheName, merged, cacheEpoch)
	}
	passthrough(w, http.StatusOK, merged)
}

// weekPracticeStates fetches the learner's practice states for the week's problems.
// It parses the problem ids out of the curriculum content, calls practice
// GET /state?ids=…, and returns the states keyed by id plus a `populated` flag
// (false when practice is unavailable — the honest placeholder path, ADR-0013).
func (g *Gateway) weekPracticeStates(r *http.Request, accountID string, content []byte) (map[string]practiceProblemState, bool) {
	if g.practice == nil {
		return nil, false
	}
	ids := problemIDsFromWeek(content)
	if len(ids) == 0 {
		// No problems to look up: still "populated" (practice exists; nothing to fill).
		return map[string]practiceProblemState{}, true
	}
	token, ok := g.mintForPractice(accountID)
	if !ok {
		return nil, false
	}
	n := r.PathValue("n")
	pbody, pstatus, perr := g.practice.get(r.Context(), token, "/state?week="+url.QueryEscape(n)+"&ids="+url.QueryEscape(strings.Join(ids, ",")))
	if perr != nil {
		g.log.Warn("bff week aggregation: practice call failed; placeholder state", "err", perr)
		return nil, false
	}
	if pstatus != http.StatusOK {
		g.log.Warn("bff week aggregation: practice non-200; placeholder state", "status", pstatus)
		return nil, false
	}
	states, ok := parseWeekStates(pbody)
	if !ok {
		g.log.Warn("bff week aggregation: malformed practice states; placeholder state")
		return nil, false
	}
	return states, true
}

// handleGetProblem is the Problem workspace BFF aggregation (api.md `agg`): it
// composes curriculum content, filtered to the learner's UNLOCKED stages (R-PF1), and
// the practice state + active timer. Practice is the authority on which stages are
// unlocked; the gateway drops locked-stage sections server-side so hint/solution
// content is never delivered before its stage. If practice is unavailable the problem
// still renders with only the statement (attempt stage) and a default state.
func (g *Gateway) handleGetProblem(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.curriculum == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "curriculum not configured")
		return
	}
	id := r.PathValue("id")
	body, status, err := g.curriculum.get(r.Context(), "/problems/"+url.PathEscape(id))
	if err != nil {
		g.log.Error("bff problem aggregation: curriculum call failed", "id", id, "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "curriculum unavailable")
		return
	}
	if status != http.StatusOK {
		// Propagate curriculum's own status + envelope (e.g. 404 for an unknown problem).
		passthrough(w, status, body)
		return
	}
	// Items are addressed by global id; the course is the item's path_slug. An item whose
	// course the caller can't see is the same uniform 404 as an unknown course.
	slug := problemPathSlug(body)
	if _, ok := g.requireCourse(w, r, slug); !ok {
		return
	}

	// Practice arena (?practice=1): a study view. Deliver ALL sections (all stages) with a
	// default (available, no-timer) state and DON'T touch practice — so opening a problem
	// in the arena creates no course-affecting state. The course flow reads practice state
	// + gates sections to the unlocked stages as usual.
	if r.URL.Query().Get("practice") == "1" {
		stateRaw, _ := defaultProblemState(id)
		merged, err := aggregateProblem(body, stateRaw, map[string]bool{"attempt": true, "hint": true, "solution": true})
		if err != nil {
			g.log.Error("bff problem aggregation (practice): merge failed", "id", id, "err", err)
			writeError(w, http.StatusBadGateway, "upstream", "curriculum returned malformed content")
			return
		}
		passthrough(w, http.StatusOK, merged)
		return
	}

	stateRaw, unlocked := g.problemState(r, accountID, id)
	merged, err := aggregateProblem(body, stateRaw, unlocked)
	if err != nil {
		g.log.Error("bff problem aggregation: merge failed", "id", id, "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "curriculum returned malformed content")
		return
	}
	// Layer the curriculum gate (enrolled? scheduled?) so the workspace can render the
	// "Start the path" gate or the "ahead of schedule" banner (review round 2).
	merged = g.injectProblemGate(r.Context(), accountID, slug, id, merged)
	passthrough(w, http.StatusOK, merged)
}

// problemPathSlug reads the item's course from curriculum's GET /problems/{id} body
// ({"problem":{"path_slug":…}}), via itemPathSlug's v1 default.
func problemPathSlug(body []byte) string {
	var env struct {
		Problem struct {
			PathSlug string `json:"path_slug"`
		} `json:"problem"`
	}
	_ = json.Unmarshal(body, &env)
	return itemPathSlug(env.Problem.PathSlug)
}

// injectProblemGate adds the `gate` block (enrolled/scheduled/currentWeek) to the Problem
// aggregate, reading the problem's week from the merged content. slug is the item's course.
func (g *Gateway) injectProblemGate(ctx context.Context, accountID, slug, problemID string, merged []byte) []byte {
	var obj map[string]json.RawMessage
	if json.Unmarshal(merged, &obj) != nil {
		return merged
	}
	week := 0
	if pr, ok := obj["problem"]; ok {
		var p struct {
			WeekN int `json:"week_n"`
		}
		if json.Unmarshal(pr, &p) == nil {
			week = p.WeekN
		}
	}
	obj["gate"] = mustJSON(g.problemGateFor(ctx, accountID, slug, problemID, week))
	if out, err := json.Marshal(obj); err == nil {
		return out
	}
	return merged
}

// problemState fetches the learner's practice state for a problem, returning the raw
// state JSON to embed and the set of unlocked stages to filter sections by. When
// practice is unavailable it degrades to the default state (available, statement-only)
// so the workspace still renders.
func (g *Gateway) problemState(r *http.Request, accountID, id string) (json.RawMessage, map[string]bool) {
	if g.practice != nil {
		if token, ok := g.mintForPractice(accountID); ok {
			pbody, pstatus, perr := g.practice.get(r.Context(), token, "/state/"+url.PathEscape(id))
			if perr != nil {
				g.log.Warn("bff problem aggregation: practice call failed; using default state", "id", id, "err", perr)
			} else if pstatus == http.StatusOK {
				if raw, unlocked, ok := parseProblemState(pbody); ok {
					return raw, unlocked
				}
				g.log.Warn("bff problem aggregation: malformed practice state; using default", "id", id)
			} else {
				g.log.Warn("bff problem aggregation: practice returned non-200; using default state", "id", id, "status", pstatus)
			}
		}
	}
	return defaultProblemState(id)
}

// handleAttemptStart proxies POST /problems/{id}/attempt/start to practice (enrollment-gated).
func (g *Gateway) handleAttemptStart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g.proxyPracticeWrite(w, r, id, "/problems/"+url.PathEscape(id)+"/attempt/start", false)
}

// handleReveal proxies POST /problems/{id}/reveal to practice (enrollment-gated).
func (g *Gateway) handleReveal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g.proxyPracticeWrite(w, r, id, "/problems/"+url.PathEscape(id)+"/reveal", false)
}

// handleOutcome proxies POST /problems/{id}/outcome to practice. Enrollment-gated AND
// schedule-gated: an ahead-of-schedule outcome is acknowledged without counting.
func (g *Gateway) handleOutcome(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g.proxyPracticeWrite(w, r, id, "/problems/"+url.PathEscape(id)+"/outcome", true)
}

// proxyPracticeWrite validates the session, enforces the curriculum gates, mints a
// practice-scoped JWT, and forwards a POST (with body) to practice, passing its status +
// JSON envelope straight through. The item's course is resolved from curriculum (its
// path_slug) and passed to practice as `?path=` so the rows and events carry it (m1-03).
// The gates (review round 2), per the item's course:
//   - visibility: an item whose course the caller can't see is 404 course_not_found.
//   - enrollment: every practice write requires the course to be started (403 not_enrolled).
//   - schedule (scheduleGate=true, the outcome): a NEW-problem solve counts only when the
//     problem is at/before the frontier week; an ahead solve is acknowledged (counted:false)
//     and NOT forwarded, so it records no solve, emits no events, and schedules no revision.
func (g *Gateway) proxyPracticeWrite(w http.ResponseWriter, r *http.Request, problemID, upstreamPath string, scheduleGate bool) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.practice == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "practice not configured")
		return
	}
	// The Problems ARENA (?practice=1) is a client-side study view, decoupled from the
	// course: it NEVER creates server-side practice state (no attempt, no timer, no
	// outcome), so an arena visit can't leak "in progress"/timer state into the course.
	// Every arena write is a benign no-op. The COURSE flow keeps the enrollment gate +
	// the schedule check.
	if r.URL.Query().Get("practice") == "1" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"practice":true}`))
		return
	}
	slug, found, err := g.itemCourse(r.Context(), problemID)
	if err != nil {
		g.log.Error("bff practice write: curriculum lookup failed", "id", problemID, "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "curriculum unavailable")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}
	if _, ok := g.requireCourse(w, r, slug); !ok {
		return
	}
	if !g.requireEnrolled(w, r.Context(), accountID, slug) {
		return
	}
	if scheduleGate {
		if frontier, byID, resolved := g.pathFrontier(r.Context(), accountID, slug); resolved {
			if p, found := byID[problemID]; found && p.WeekN > frontier {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(w, `{"counted":false,"scheduledWeek":%d,"currentWeek":%d}`, p.WeekN, frontier)
				return
			}
		}
	}
	token, ok := g.mintForPracticeW(w, accountID)
	if !ok {
		return
	}
	reqBody, ok := httpx.ReadBody(w, r, httpx.BodyLimitDefault)
	if !ok {
		return
	}
	body, status, err := g.practice.post(r.Context(), token, withPath(upstreamPath, slug), reqBody)
	if err != nil {
		g.log.Error("bff practice write failed", "path", upstreamPath, "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "practice unavailable")
		return
	}
	// A practice mutation (attempt/reveal/outcome) changes the learner's Week/Dashboard
	// state; drop this account's cached aggregations so the next read recomputes fresh.
	g.invalidateAgg(status, accountID)
	passthrough(w, status, body)
}

// invalidateAgg drops the account's cached Dashboard/Week aggregations after a
// successful (2xx) mutating write, so a post-write read never serves stale composed
// state. Non-2xx writes changed nothing, so nothing is invalidated.
func (g *Gateway) invalidateAgg(status int, accountID string) {
	if status/100 == 2 {
		g.cache.invalidate(accountID)
	}
}

// mintForPractice mints a practice-scoped JWT without writing an error response (used
// on the best-effort agg read path); ok reports success.
func (g *Gateway) mintForPractice(accountID string) (string, bool) {
	return g.mintQuiet(accountID, g.audPractice)
}

// mintQuiet mints an audience-scoped JWT without writing an error response, for the
// best-effort fan-out read paths (Progress / Dashboard aggregations) where a single
// section degrades rather than failing the whole response. ok reports success.
func (g *Gateway) mintQuiet(accountID, audience string) (string, bool) {
	if g.signer == nil {
		return "", false
	}
	token, err := g.signer.Mint(context.Background(), accountID, audience, mintedRoles())
	if err != nil {
		g.log.Error("mint jwt", "aud", audience, "err", err)
		return "", false
	}
	return token, true
}

// mintForPracticeW mints a practice-scoped JWT, writing the error envelope on failure.
func (g *Gateway) mintForPracticeW(w http.ResponseWriter, accountID string) (string, bool) {
	return g.mintFor(w, accountID, g.audPractice)
}

// --- revision (review service) ---

// handleRevisionDue is the Revision-queue BFF aggregation (api.md): it fetches the
// learner's prioritised due queue from review, then enriches each bare-id item with
// its curriculum problem metadata (title/difficulty/pattern) so the screen renders
// full cards. If curriculum can't resolve a problem the item degrades to id-only.
// Course-scoped: GET /paths/{slug}/revision/due (and the DSA alias /revision/due).
func (g *Gateway) handleRevisionDue(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	slug := r.PathValue("slug")
	if _, ok := g.requireCourse(w, r, slug); !ok {
		return
	}
	if g.review == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "review not configured")
		return
	}
	token, ok := g.mintForReviewW(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.review.get(r.Context(), token, withPath("/revisions/due", slug))
	if err != nil {
		g.log.Error("bff revision/due: review call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "review unavailable")
		return
	}
	if status != http.StatusOK {
		passthrough(w, status, body)
		return
	}
	passthrough(w, http.StatusOK, g.enrichDueQueue(r.Context(), body))
}

// handleRevisionScore proxies the auto-score submission to review (POST
// /revisions/{id}/score), passing its status + JSON envelope straight through.
func (g *Gateway) handleRevisionScore(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.review == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "review not configured")
		return
	}
	token, ok := g.mintForReviewW(w, accountID)
	if !ok {
		return
	}
	reqBody, ok := httpx.ReadBody(w, r, httpx.BodyLimitDefault)
	if !ok {
		return
	}
	upstream := "/revisions/" + url.PathEscape(r.PathValue("itemId")) + "/score"
	body, status, err := g.review.post(r.Context(), token, upstream, reqBody)
	if err != nil {
		g.log.Error("bff revision score: review call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "review unavailable")
		return
	}
	// A re-solve advances/resets the touch ladder + due queue → invalidate this
	// account's cached Dashboard/Week.
	g.invalidateAgg(status, accountID)
	passthrough(w, status, body)
}

// mintForReviewW mints a review-scoped JWT, writing the error envelope on failure.
func (g *Gateway) mintForReviewW(w http.ResponseWriter, accountID string) (string, bool) {
	return g.mintFor(w, accountID, g.audReview)
}

// handleGetConcept is a course's concept reading (GET /paths/{slug}/concepts/{c}, keyed
// on (course, concept slug); the DSA alias is GET /concepts/{slug}).
func (g *Gateway) handleGetConcept(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.authAccount(w, r); !ok {
		return
	}
	slug := r.PathValue("slug")
	if _, ok := g.requireCourse(w, r, slug); !ok {
		return
	}
	g.proxyCurriculum(w, r, "/paths/"+url.PathEscape(slug)+"/concepts/"+url.PathEscape(r.PathValue("c")))
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
// writes the 401 envelope and returns ok=false (authSession, keeping the v1 signature).
func (g *Gateway) authAccount(w http.ResponseWriter, r *http.Request) (string, bool) {
	info, ok := g.authSession(w, r)
	return info.AccountID, ok
}

// authSession validates the session cookie with identity and returns what
// session-validate reports (sessionInfo), or writes the 401 envelope and returns
// ok=false. It asks identity on EVERY request and the gateway never caches status or
// role: a suspend or a set-role applies on the caller's next request. The answer is
// remembered for the rest of THIS request only (rememberSession), which is where the
// course-visibility cohort bit reads it, and a second authSession in the same request
// reuses it instead of calling identity twice.
//
// It is also L5's single enforcement point (m1-05): right after validation, the account's
// token buckets (limits.go allowAccount) run once per request. A refusal is the typed 429
// and ok=false, like any other auth failure.
func (g *Gateway) authSession(w http.ResponseWriter, r *http.Request) (sessionInfo, bool) {
	if info, ok := requestSession(r); ok {
		return info, true
	}
	if g.identity == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "identity not configured")
		return sessionInfo{}, false
	}
	c, err := r.Cookie(auth.SessionCookieName)
	if err != nil || c.Value == "" {
		writeUnauthenticated(w)
		return sessionInfo{}, false
	}
	info, err := g.identity.validateSession(r.Context(), c.Value)
	if err != nil {
		writeUnauthenticated(w)
		return sessionInfo{}, false
	}
	if !g.allowAccount(w, r, info.AccountID) {
		return sessionInfo{}, false
	}
	rememberSession(r, info)
	return info, true
}

// mintedRoles is the role set of EVERY gateway-minted JWT: exactly ["learner"], for every
// account (ADR-0033 §7). The owner/tester cohort is never a JWT role — the gateway reads
// it from session-validate — and "public-read" is minted only by M2b's public route.
func mintedRoles() []string { return []string{auth.RoleLearner} }

// mint issues an identity-scoped JWT for accountID (roles: learner).
func (g *Gateway) mint(w http.ResponseWriter, accountID string) (string, bool) {
	return g.mintFor(w, accountID, g.audIdentity)
}

// mintFor issues a JWT for accountID scoped to a specific downstream audience
// (ADR-0006: one token per audience).
func (g *Gateway) mintFor(w http.ResponseWriter, accountID, audience string) (string, bool) {
	if g.signer == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "signing not configured")
		return "", false
	}
	token, err := g.signer.Mint(context.Background(), accountID, audience, mintedRoles())
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

// sessionInfo is identity's session-validate answer (m1-04, ADR-0033 §6): the account,
// its role and status from identity's DB (never a JWT claim), whether it passed the
// acceptance step (always false until L-A; not enforced until l-05) and the SESSION's
// created_at (L8's fresh-session check). A v1.6.0 identity sends only account_id: the
// rest stays zero, so nobody is in the cohort during a mixed-version rollout.
type sessionInfo struct {
	AccountID string
	Role      string
	Status    string
	Accepted  bool
	CreatedAt time.Time
}

func (c *identityClient) validateSession(ctx context.Context, sessionID string) (sessionInfo, error) {
	body, status, err := c.postJSON(ctx, "/sessions/validate", "", map[string]string{"session_id": sessionID})
	if err != nil {
		return sessionInfo{}, err
	}
	if status != http.StatusOK {
		return sessionInfo{}, fmt.Errorf("validate session: status %d", status)
	}
	var vr struct {
		AccountID string    `json:"account_id"`
		Role      string    `json:"role"`
		Status    string    `json:"status"`
		Accepted  bool      `json:"accepted"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.Unmarshal(body, &vr); err != nil || vr.AccountID == "" {
		return sessionInfo{}, fmt.Errorf("validate session: bad response")
	}
	return sessionInfo{AccountID: vr.AccountID, Role: vr.Role, Status: vr.Status, Accepted: vr.Accepted, CreatedAt: vr.CreatedAt}, nil
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

func (c *identityClient) patchAccount(ctx context.Context, token, accountID string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+"/accounts/"+accountID, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
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

// setPassword forwards a set/change-password request (ADR-0023 · POST /accounts/{id}/password).
func (c *identityClient) setPassword(ctx context.Context, token, accountID string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/accounts/"+accountID+"/password", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

// unlinkOAuth disconnects a provider (ADR-0023 · DELETE /accounts/{id}/oauth/{provider}).
func (c *identityClient) unlinkOAuth(ctx context.Context, token, accountID, provider string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/accounts/"+accountID+"/oauth/"+url.PathEscape(provider), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

// startEnrollment enrolls the caller in a path (F002 · POST /paths/{slug}/start).
func (c *identityClient) startEnrollment(ctx context.Context, token, slug string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/paths/"+url.PathEscape(slug)+"/start", nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

// setUsername forwards a username claim/change to identity (F009 · POST /accounts/{id}/username).
func (c *identityClient) setUsername(ctx context.Context, token, accountID string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/accounts/"+accountID+"/username", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

// usernameAvailable forwards the availability check to identity (F009 · GET /username/available?u=).
func (c *identityClient) usernameAvailable(ctx context.Context, token, u string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/username/available?u="+url.QueryEscape(u), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

// publicAccountByUsername resolves a username to the account's non-PII public fields via
// identity's ClusterIP-only internal endpoint (NO user JWT; F009). This is the only identity
// call the public dashboard makes — it never touches the PII /accounts/{id}.
func (c *identityClient) publicAccountByUsername(ctx context.Context, username string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/internal/accounts/by-username/"+url.PathEscape(username), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
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

// forward proxies an auth request (signup, login, OAuth start/callback, dev login) to
// identity, passing cookies + query and copying status, Location and Set-Cookie back (no
// redirect following).
//
// The body is capped (L6): it streams through http.MaxBytesReader at BodyLimitDefault,
// never as the raw request body. A declared Content-Length over the cap is a 413 before
// identity is dialled; a chunked body that overruns surfaces from the client as a
// *http.MaxBytesError and becomes the same typed 413, not a 502.
func (c *identityClient) forward(w http.ResponseWriter, r *http.Request, upstreamPath string) {
	if r.ContentLength > httpx.BodyLimitDefault {
		httpx.WriteBodyTooLarge(w, httpx.BodyLimitDefault)
		return
	}
	target := c.baseURL + upstreamPath
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	var body io.Reader
	if r.Body != nil && r.Body != http.NoBody {
		body = http.MaxBytesReader(w, r.Body, httpx.BodyLimitDefault)
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "proxy build failed")
		return
	}
	// Keep the declared length so identity sees the framing the client sent (NewRequest
	// can't see through the MaxBytesReader); an unknown length stays chunked.
	if body != nil {
		req.ContentLength = r.ContentLength
		if r.ContentLength == 0 {
			req.Body, req.ContentLength = http.NoBody, 0
		}
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := c.proxyc.Do(req)
	if err != nil {
		if httpx.IsBodyTooLarge(err) {
			httpx.WriteBodyTooLarge(w, httpx.BodyLimitDefault)
			return
		}
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
	// L3's 429 (bcrypt busy on login/signup) tells the browser when to retry.
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		w.Header().Set("Retry-After", ra)
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 4<<20))
}
