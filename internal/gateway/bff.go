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

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
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
}

// apiRoutes is the authoritative BFF route table. Order is irrelevant to correctness
// (Go 1.22's ServeMux matches most-specific-first, so /mocks/trend beats /mocks/{id}
// regardless of registration order); it is grouped by feature for readability.
func (g *Gateway) apiRoutes() []apiRoute {
	return []apiRoute{
		// System / auth infra.
		{"GET", "/api/healthz", g.appHealth, false},
		{"GET", "/.well-known/jwks.json", g.handleJWKS, false},
		// Account + onboarding (identity-backed). OAuth start/callback are proxied so the
		// browser only ever talks to the gateway origin.
		{"GET", "/api/me", g.handleMe, true},
		{"PATCH", "/api/me", g.handlePatchMe, true},
		// Account & sign-in management (ADR-0023): set/change password + disconnect a provider.
		{"POST", "/api/me/password", g.handleSetPassword, true},
		{"DELETE", "/api/me/oauth/{provider}", g.handleUnlinkOAuth, true},
		{"POST", "/api/auth/logout", g.handleLogout, true},
		{"POST", "/api/onboarding/step", g.handleOnboardingStep, true},
		// Email/password auth (ADR-0023): fetch-based signup/login (session cookie on the JSON
		// response). OAuth start/callback are the browser-redirect flow; `?link=1` on start
		// connects the provider to the signed-in account.
		{"POST", "/api/auth/signup", g.handleAuthSignup, true},
		{"POST", "/api/auth/login", g.handleAuthLogin, true},
		{"POST", "/api/auth/{provider}/start", g.handleAuthProxy, true},
		{"GET", "/api/auth/{provider}/callback", g.handleAuthProxy, true},
		// Local-only dev login (F002 / ADR-0022): proxied to identity, which 404s them
		// unless DEV_AUTH is set. Undocumented (Doc:false) — never part of the prod surface.
		{"POST", "/api/auth/dev/login", g.handleAuthDevProxy, false},
		{"GET", "/api/auth/dev/enabled", g.handleAuthDevProxy, false},
		// Per-user path enrollment (F002): starting a path is an explicit, durable action.
		{"POST", "/api/paths/{slug}/start", g.handleStartPath, true},
		// Curriculum content (read-only, session-gated). The week route is a BFF
		// aggregation (api.md `agg`): the gateway layers per-user five-touch/solve state
		// onto curriculum content (ADR-0005/0013).
		{"GET", "/api/paths", g.handleListPaths, true},
		{"GET", "/api/paths/{slug}", g.handleGetPath, true},
		{"GET", "/api/paths/{slug}/problems", g.handleListPathProblems, true},
		{"GET", "/api/paths/{slug}/weeks/{n}", g.handleGetWeek, true},
		{"GET", "/api/concepts/{slug}", g.handleGetConcept, true},
		// Problem workspace: the GET is a BFF aggregation (content limited to unlocked
		// stages + practice state + timer); the writes proxy to practice.
		{"GET", "/api/problems/{id}", g.handleGetProblem, true},
		{"POST", "/api/problems/{id}/attempt/start", g.handleAttemptStart, true},
		{"POST", "/api/problems/{id}/reveal", g.handleReveal, true},
		{"POST", "/api/problems/{id}/outcome", g.handleOutcome, true},
		// Revision (review). External /revision maps to review's internal /revisions.
		{"GET", "/api/revision/due", g.handleRevisionDue, true},
		{"POST", "/api/revision/{itemId}/score", g.handleRevisionScore, true},
		// Mistake journal + weak-area (review), enriched with curriculum metadata.
		{"GET", "/api/mistakes", g.handleMistakes, true},
		{"POST", "/api/mistakes", g.handleCreateMistake, true},
		{"PATCH", "/api/mistakes/{id}", g.handlePatchMistake, true},
		{"GET", "/api/weak-area", g.handleWeakArea, true},
		// Mock interview (assessment). /mocks/trend is more specific than /mocks/{id},
		// so it wins regardless of order.
		{"POST", "/api/mocks", g.handleStartMock, true},
		{"GET", "/api/mocks/trend", g.handleMockTrend, true},
		{"GET", "/api/mocks/{id}", g.handleGetMock, true},
		{"POST", "/api/mocks/{id}/score", g.handleScoreMock, true},
		// Progress + Dashboard "Today" (parallel fan-out aggregations, S09).
		{"GET", "/api/progress", g.handleProgress, true},
		{"GET", "/api/dashboard", g.handleDashboard, true},
		// Coach (S11): masked key CRUD, per-page thread, and the SSE chat relay.
		{"GET", "/api/coach/key", g.handleCoachKey, true},
		{"PUT", "/api/coach/key", g.handlePutCoachKey, true},
		{"DELETE", "/api/coach/key", g.handleDeleteCoachKey, true},
		{"GET", "/api/coach/thread", g.handleCoachThread, true},
		{"POST", "/api/coach/chat", g.handleCoachChat, true},
	}
}

// newAPIMux builds the BFF routes from the authoritative apiRoutes table. The gateway
// validates the opaque session cookie with identity, mints a short-TTL JWT (ADR-0006),
// and forwards it on internal calls; OAuth start/callback are proxied through to
// identity so the browser only ever talks to the gateway origin.
func (g *Gateway) newAPIMux() *http.ServeMux {
	mux := http.NewServeMux()
	for _, rt := range g.apiRoutes() {
		mux.HandleFunc(rt.Method+" "+rt.Pattern, rt.Handler)
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
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "could not read body")
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

// handleAuthSignup / handleAuthLogin forward the email/password body to identity, which
// creates/authenticates the account and sets the session cookie on its JSON response
// (ADR-0023). The gateway passes Set-Cookie back so the browser talks only to the gateway.
func (g *Gateway) handleAuthSignup(w http.ResponseWriter, r *http.Request) {
	if g.identity == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "identity not configured")
		return
	}
	g.identity.forward(w, r, "/auth/signup")
}

func (g *Gateway) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if g.identity == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "identity not configured")
		return
	}
	g.identity.forward(w, r, "/auth/login")
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
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "could not read body")
		return
	}
	body, status, err := g.identity.setPassword(r.Context(), token, accountID, reqBody)
	if err != nil {
		g.log.Error("bff /me/password: identity call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "identity unavailable")
		return
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

// handleStartPath enrolls the caller in a path (F002 · POST /paths/{slug}/start),
// minting an identity-scoped JWT and forwarding to identity. Idempotent.
func (g *Gateway) handleStartPath(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	token, ok := g.mint(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.identity.startEnrollment(r.Context(), token, r.PathValue("slug"))
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

// handleListPathProblems proxies the whole problem index for a path (the Problems arena,
// review round 2 — a flat list you can browse + attempt any problem from).
func (g *Gateway) handleListPathProblems(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.authAccount(w, r); !ok {
		return
	}
	g.proxyCurriculum(w, r, "/paths/"+url.PathEscape(r.PathValue("slug"))+"/problems")
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
	merged = g.injectProblemGate(r.Context(), accountID, id, merged)
	passthrough(w, http.StatusOK, merged)
}

// injectProblemGate adds the `gate` block (enrolled/scheduled/currentWeek) to the Problem
// aggregate, reading the problem's week from the merged content.
func (g *Gateway) injectProblemGate(ctx context.Context, accountID, problemID string, merged []byte) []byte {
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
	obj["gate"] = mustJSON(g.problemGateFor(ctx, accountID, "dsa", problemID, week))
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
// JSON envelope straight through. The gates (review round 2):
//   - enrollment: every practice write requires the path to be started (403 not_enrolled).
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
	if !g.requireEnrolled(w, r.Context(), accountID, "dsa") {
		return
	}
	if scheduleGate {
		if frontier, byID, resolved := g.pathFrontier(r.Context(), accountID); resolved {
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
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "could not read body")
		return
	}
	body, status, err := g.practice.post(r.Context(), token, upstreamPath, reqBody)
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
	token, err := g.signer.Mint(context.Background(), accountID, audience, []string{"learner"})
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
func (g *Gateway) handleRevisionDue(w http.ResponseWriter, r *http.Request) {
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
	body, status, err := g.review.get(r.Context(), token, "/revisions/due")
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
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "could not read body")
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
	return g.mintFor(w, accountID, g.audIdentity)
}

// mintFor issues a JWT for accountID scoped to a specific downstream audience
// (ADR-0006: one token per audience).
func (g *Gateway) mintFor(w http.ResponseWriter, accountID, audience string) (string, bool) {
	if g.signer == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "signing not configured")
		return "", false
	}
	token, err := g.signer.Mint(context.Background(), accountID, audience, []string{"learner"})
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
