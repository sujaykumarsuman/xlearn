package gateway

import (
	"context"
	"encoding/json"
	"net/http"
)

// Curriculum gating (review round 2): starting a path is a prerequisite for solving,
// and only SCHEDULED work counts toward curriculum activity. "Scheduled" for a NEW
// problem (the /problems/{id}/outcome path) is: its week is at or before the learner's
// frontier week — the lowest week that still has an unsolved core problem (currentWeek,
// dashboard.go). Weeks ahead of the frontier are browsable + attemptable but do NOT
// count (the outcome is acknowledged, not recorded). Revisions ride a separate,
// already-due-gated path (/revision/{id}/score), so they are inherently scheduled.

// enrolledPaths returns the set of path slugs the account has started (F002), read from
// identity's /accounts/{id} (which carries enrollments). Empty on any failure — callers
// treat "unknown" as not-enrolled, which is the safe (gated) default.
func (g *Gateway) enrolledPaths(ctx context.Context, accountID string) map[string]bool {
	out := map[string]bool{}
	if g.identity == nil {
		return out
	}
	token, ok := g.mintQuiet(accountID, g.audIdentity)
	if !ok {
		return out
	}
	body, status, err := g.identity.getAccount(ctx, token, accountID)
	if err != nil || status != http.StatusOK {
		return out
	}
	var doc struct {
		Enrollments []struct {
			PathSlug string `json:"path_slug"`
		} `json:"enrollments"`
	}
	if json.Unmarshal(body, &doc) != nil {
		return out
	}
	for _, e := range doc.Enrollments {
		out[e.PathSlug] = true
	}
	return out
}

// isEnrolled reports whether the account has started the given path.
func (g *Gateway) isEnrolled(ctx context.Context, accountID, slug string) bool {
	return g.enrolledPaths(ctx, accountID)[slug]
}

// requireEnrolled gates a practice action on enrollment: if the account has not started
// the path it writes the 403 not_enrolled envelope (the SPA turns this into the "Start
// the path" gate) and returns false.
func (g *Gateway) requireEnrolled(w http.ResponseWriter, ctx context.Context, accountID, slug string) bool {
	if g.isEnrolled(ctx, accountID, slug) {
		return true
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":{"code":"not_enrolled","message":"Start the path to begin.","path":"` + slug + `"}}`))
	return false
}

// pathFrontier fetches the DSA problem index + the account's solved set (assessment
// mastery projection) and returns the frontier week + the index keyed by problem id.
// ok=false when the index can't be resolved (the caller then skips the schedule gate
// rather than blocking on a degraded upstream).
func (g *Gateway) pathFrontier(ctx context.Context, accountID string) (frontier int, byID map[string]problemIndexItem, ok bool) {
	byID = map[string]problemIndexItem{}
	if g.curriculum == nil {
		return 0, byID, false
	}
	body, status, err := g.curriculum.get(ctx, "/paths/dsa/problems")
	if err != nil || status != http.StatusOK {
		return 0, byID, false
	}
	var doc problemsDoc
	if json.Unmarshal(body, &doc) != nil || len(doc.Problems) == 0 {
		return 0, byID, false
	}
	for _, p := range doc.Problems {
		byID[p.ID] = p
	}

	solved := map[string]bool{}
	if g.assessment != nil {
		if aToken, okT := g.mintQuiet(accountID, g.audAssessment); okT {
			if mbody, mstatus, merr := g.assessment.get(ctx, aToken, "/progress/mastery"); merr == nil && mstatus == http.StatusOK {
				var m masteryDoc
				if json.Unmarshal(mbody, &m) == nil {
					for _, mp := range m.Problems {
						solved[mp.ProblemID] = true
					}
				}
			}
		}
	}
	return currentWeek(doc.Problems, solved), byID, true
}

// problemGate is the per-problem gating block injected into the Problem workspace agg so
// the SPA can render the right state (start-the-path gate / ahead-of-schedule banner).
type problemGate struct {
	Enrolled    bool `json:"enrolled"`
	Scheduled   bool `json:"scheduled"`   // false → ahead of schedule (attempt freely, won't count)
	CurrentWeek int  `json:"currentWeek"` // the learner's frontier week (0 when not enrolled)
	ProblemWeek int  `json:"problemWeek"` // the problem's week (for the banner copy)
}

// problemGateFor computes the gate for a single problem: enrolled? and is this problem
// scheduled (its week is at or before the frontier)?
func (g *Gateway) problemGateFor(ctx context.Context, accountID, slug, problemID string, problemWeek int) problemGate {
	enrolled := g.isEnrolled(ctx, accountID, slug)
	gate := problemGate{Enrolled: enrolled, ProblemWeek: problemWeek}
	if !enrolled {
		return gate // scheduled stays false; the SPA shows the start gate
	}
	frontier, _, ok := g.pathFrontier(ctx, accountID)
	if !ok {
		// Couldn't resolve the frontier — don't falsely block; treat as scheduled.
		gate.Scheduled = true
		return gate
	}
	gate.CurrentWeek = frontier
	gate.Scheduled = problemWeek <= frontier
	return gate
}
