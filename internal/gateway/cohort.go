package gateway

import (
	"context"
	"net/http"
)

// The T-3 cohort (ADR-0034 §2; ADR-0033 §7, m1-04): the owner and testers see `preview`
// courses; everyone else gets the uniform course_not_found. The cohort is the account's
// role in identity's database as session-validate reports it — NEVER a JWT claim (JWTs
// stay ["learner"] for every account). This file is the single T-3 read: every preview
// gate (courseVisible via visibleCourse/requireCourse, courseListed via filterCatalog,
// handleStartPath) goes through Gateway.cohort.

// Account roles as identity reports them (identity/store admission.go).
const (
	roleOwner  = "owner"
	roleTester = "tester"
)

// inCohort reports whether a validated session belongs to the owner/tester cohort.
func inCohort(info sessionInfo) bool { return info.Role == roleOwner || info.Role == roleTester }

// cohort is this request's cohort bit: false unless authSession validated the session
// during THIS request and identity reported an owner or tester role.
func (g *Gateway) cohort(r *http.Request) bool {
	info, ok := requestSession(r)
	return ok && inCohort(info)
}

// requestSessionKey holds a per-request slot for the validated session. newAPIMux gives
// every API request a fresh slot; authSession fills it. It is not a cache: it dies with
// the request, and every request validates the session with identity again.
type requestSessionKey struct{}

type requestSessionSlot struct {
	info sessionInfo
	ok   bool
}

// withRequestSession returns r with an empty session slot.
func withRequestSession(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), requestSessionKey{}, &requestSessionSlot{}))
}

// rememberSession stores the validated session in r's slot (no-op without a slot).
func rememberSession(r *http.Request, info sessionInfo) {
	if slot, ok := r.Context().Value(requestSessionKey{}).(*requestSessionSlot); ok {
		slot.info, slot.ok = info, true
	}
}

// requestSession returns the session authSession validated during this request.
func requestSession(r *http.Request) (sessionInfo, bool) {
	slot, ok := r.Context().Value(requestSessionKey{}).(*requestSessionSlot)
	if !ok || !slot.ok {
		return sessionInfo{}, false
	}
	return slot.info, true
}
