package gateway

import (
	"context"
	"net/http"
)

// getModels fetches the coach service's model catalog. It is defined HERE rather than
// beside the other coachClient methods in coach.go on purpose: m1-05 rewrites that file's
// body reads in parallel with this sprint, and m1-10's gateway change is additive only
// (one new file, one route row, one OpenAPI path, one test file). A method on a type
// declared in the same package may live in any file, so the two sprints never touch the
// same lines.
func (c *coachClient) getModels(ctx context.Context, token string) ([]byte, int, error) {
	return c.jsonReq(ctx, http.MethodGet, "/models", token, nil)
}

// handleCoachModels serves the coach's SERVER-SIDE model catalog (api.md
// GET /api/coach/models; sprint m1-10, ADR-0031 §7).
//
// It is a plain passthrough to coach's /models: the catalog lives in the coach service, so
// the switcher, the Settings key panel and the onboarding step all render what the
// deployed coach release knows rather than a list baked into the SPA, which is what went
// stale in v1 every time a provider shipped or retired a model.
//
// SESSION-GATED, even though the catalog contains no learner data at all. Keeping it
// behind authAccount costs nothing and preserves the invariant that the public profile is
// the ONLY unauthenticated route under /api.
//
// 503 rather than an empty catalog when coach is unset: an empty list is indistinguishable
// from "this provider has no models", and the client would quietly render a switcher with
// nothing in it. The SPA falls back to showing just the currently set model on failure.
func (g *Gateway) handleCoachModels(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.coach == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "coach not configured")
		return
	}
	token, ok := g.mintForCoachW(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.coach.getModels(r.Context(), token)
	if err != nil {
		g.log.Warn("bff /coach/models: coach call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "coach unavailable")
		return
	}
	passthrough(w, status, body)
}
