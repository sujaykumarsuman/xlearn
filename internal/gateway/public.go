package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
)

// This file is the PUBLIC user dashboard aggregation (F009 / ADR-0024, ADR-0025): the
// LeetCode-style profile at projects.sujaykumar.dev/xlearn/u/<username>. It is the ONLY
// unauthenticated /api route — every other handler starts with authAccount; this one
// deliberately does not.
//
// Security model (ADR-0024). PII lives only in identity. The public path resolves the
// username via identity's ClusterIP-only /internal/accounts/by-username, which returns
// ONLY non-PII fields (id, username, display name, join date) — it NEVER calls identity's
// PII /accounts/{id}. It then mints read-scoped service tokens for the RESOLVED account and
// fans out to assessment (projections) + curriculum (taxonomy) exactly as the authed
// Progress aggregation does, but composes only public aggregates (solved counts, streak,
// mock stats, activity heatmap, per-course completion/mastery). Assessment + curriculum
// hold no PII, so nothing private can surface no matter what we compose. The response is
// cached per resolved account (public → shareable across viewers; a short TTL + the
// account's own mutating-write invalidation bound staleness and blunt abuse).

// publicAccount is identity's non-PII resolver payload for a username. `region` is a coarse
// UTC-offset band (never the IANA zone) — the low-identifying "Region" (F009 review).
type publicAccount struct {
	AccountID   string `json:"account_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
	Region      string `json:"region"`
}

// --- composed public output shapes (camelCase for the SPA) ---

type publicUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	JoinedAt    string `json:"joinedAt"`
	Region      string `json:"region"` // coarse UTC-offset band, "" when unknown
}

type publicStreak struct {
	Current int `json:"current"`
	Longest int `json:"longest"`
}

type publicTotals struct {
	Solved int          `json:"solved"`
	Streak publicStreak `json:"streak"`
}

type publicMock struct {
	Count   int `json:"count"`
	Best    int `json:"best"`
	Average int `json:"average"`
}

type publicCourse struct {
	Slug     string            `json:"slug"`
	Title    string            `json:"title"`
	Solved   int               `json:"solved"`
	Total    int               `json:"total"`
	Pct      int               `json:"pct"`
	Phases   []phaseCompletion `json:"phases"`
	Patterns []patternMastery  `json:"patterns"`
}

type publicProfile struct {
	User    publicUser      `json:"user"`
	Totals  publicTotals    `json:"totals"`
	Mock    publicMock      `json:"mock"`
	Heatmap json.RawMessage `json:"heatmap"` // {days:[...]} or null
	Courses []publicCourse  `json:"courses"`
}

// handlePublicProfile serves GET /api/u/{username} — the public dashboard. No session.
func (g *Gateway) handlePublicProfile(w http.ResponseWriter, r *http.Request) {
	if g.identity == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "identity not configured")
		return
	}
	username := r.PathValue("username")

	// 1. Resolve username → account (non-PII fields only). A genuine 404 is "no such user";
	//    a transport/other failure is a 502 (don't leak identity internals).
	body, status, err := g.identity.publicAccountByUsername(r.Context(), username)
	if err != nil {
		g.log.Warn("bff public profile: identity resolve failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "identity unavailable")
		return
	}
	if status == http.StatusNotFound {
		writeError(w, http.StatusNotFound, "not_found", "no such user")
		return
	}
	if status != http.StatusOK {
		passthrough(w, status, body)
		return
	}
	var acct publicAccount
	if json.Unmarshal(body, &acct) != nil || acct.AccountID == "" {
		writeError(w, http.StatusBadGateway, "upstream", "identity returned malformed account")
		return
	}

	// 2. Serve a cached snapshot when warm (keyed by the RESOLVED account, so it is shared
	//    across all anonymous viewers and invalidated by that account's own writes).
	const cacheName = "public-profile"
	if cached, ok := g.cache.get(acct.AccountID, cacheName); ok {
		passthrough(w, http.StatusOK, cached)
		return
	}
	cacheEpoch := g.cache.epoch(acct.AccountID)

	profile := g.composePublicProfile(r.Context(), acct)
	out, err := json.Marshal(profile)
	if err != nil {
		g.log.Error("bff public profile: marshal failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "profile compose failed")
		return
	}
	g.cache.putFresh(acct.AccountID, cacheName, out, cacheEpoch)
	passthrough(w, http.StatusOK, out)
}

// composePublicProfile fans out to assessment + curriculum for the resolved account and
// composes the public payload. Every upstream section degrades independently: a projection
// outage yields zeros / a null heatmap / empty courses rather than failing the page.
func (g *Gateway) composePublicProfile(ctx context.Context, acct publicAccount) publicProfile {
	profile := publicProfile{
		User:    publicUser{Username: acct.Username, DisplayName: acct.DisplayName, JoinedAt: acct.CreatedAt, Region: acct.Region},
		Heatmap: json.RawMessage("null"),
		Courses: []publicCourse{},
	}

	aToken, _ := g.mintQuiet(acct.AccountID, g.audAssessment)

	// The active courses come first (a quick sequential read) so we know how many per-course
	// fan-outs to launch; only real (active) paths with content are shown.
	activePaths := g.activePaths(ctx)

	var (
		summaryRaw, heatmapRaw, masteryRaw json.RawMessage
		courseRaw                          = make([]coursePair, len(activePaths))
		wg                                 sync.WaitGroup
	)

	if g.assessment != nil && aToken != "" {
		aget := func(path string, dst *json.RawMessage) {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, aggCallTimeout)
			defer cancel()
			if b, st, err := g.assessment.get(cctx, aToken, path); err == nil && st == http.StatusOK {
				*dst = b
			}
		}
		wg.Add(3)
		go aget("/progress/summary", &summaryRaw)
		go aget("/progress/heatmap", &heatmapRaw)
		go aget("/progress/mastery", &masteryRaw)
	}

	if g.curriculum != nil {
		for i, p := range activePaths {
			wg.Add(1)
			go func(i int, slug string) {
				defer wg.Done()
				courseRaw[i] = g.courseTaxonomy(ctx, slug)
			}(i, p.slug)
		}
	}
	wg.Wait()

	// Solved set + per-problem weights from the mastery projection.
	var mastery masteryDoc
	_ = json.Unmarshal(masteryRaw, &mastery)
	solvedSet := make(map[string]bool, len(mastery.Problems))
	for _, m := range mastery.Problems {
		solvedSet[m.ProblemID] = true
	}

	// Profile totals + mock come from the summary projection (account-wide); the heatmap is
	// already account-wide (merged across courses) in assessment (ADR-0018).
	profile.Totals.Solved = len(solvedSet)
	if len(summaryRaw) > 0 {
		var s struct {
			Solved int          `json:"solved"`
			Streak publicStreak `json:"streak"`
			Mock   publicMock   `json:"mock"`
		}
		if json.Unmarshal(summaryRaw, &s) == nil {
			profile.Totals.Solved = s.Solved
			profile.Totals.Streak = s.Streak
			profile.Mock = s.Mock
		}
	}
	if len(heatmapRaw) > 0 {
		profile.Heatmap = heatmapRaw
	}

	// Per-course completion + pattern mastery, composed from curriculum taxonomy ∩ the
	// solved set (the same roll-up the authed Progress screen uses; ADR-0018).
	for i, p := range activePaths {
		cr := courseRaw[i]
		var roadmap roadmapDoc
		_ = json.Unmarshal(cr.roadmap, &roadmap)
		var problems problemsDoc
		_ = json.Unmarshal(cr.problems, &problems)
		course := publicCourse{
			Slug:     p.slug,
			Title:    p.title,
			Phases:   composePhaseCompletion(roadmap.Phases, problems.Problems, mastery.Problems),
			Patterns: composePatternMastery(problems.Problems, mastery.Problems),
		}
		for _, pr := range problems.Problems {
			if pr.IsReinforcement {
				continue
			}
			course.Total++
			if solvedSet[pr.ID] {
				course.Solved++
			}
		}
		if course.Total > 0 {
			course.Pct = int(100.0*float64(course.Solved)/float64(course.Total) + 0.5)
		}
		profile.Courses = append(profile.Courses, course)
	}
	return profile
}

// coursePair holds a course's raw roadmap + problem-index bodies for composition.
type coursePair struct {
	roadmap  []byte
	problems []byte
}

// activePathRef is the slug+title of a real (active) course.
type activePathRef struct {
	slug  string
	title string
}

// activePaths reads the catalog and returns the active paths (coming-soon paths have no
// content, so they carry no public stats). Empty on any curriculum failure.
func (g *Gateway) activePaths(ctx context.Context) []activePathRef {
	if g.curriculum == nil {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, aggCallTimeout)
	defer cancel()
	body, status, err := g.curriculum.get(cctx, "/paths")
	if err != nil || status != http.StatusOK {
		return nil
	}
	var doc struct {
		Paths []struct {
			Slug   string `json:"slug"`
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"paths"`
	}
	if json.Unmarshal(body, &doc) != nil {
		return nil
	}
	out := make([]activePathRef, 0, len(doc.Paths))
	for _, p := range doc.Paths {
		if p.Status == "active" {
			out = append(out, activePathRef{slug: p.Slug, title: p.Title})
		}
	}
	return out
}

// courseTaxonomy fetches a course's roadmap (phases + total) and problem index in parallel.
func (g *Gateway) courseTaxonomy(ctx context.Context, slug string) coursePair {
	var pair coursePair
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		cctx, cancel := context.WithTimeout(ctx, aggCallTimeout)
		defer cancel()
		if b, st, err := g.curriculum.get(cctx, "/paths/"+url.PathEscape(slug)); err == nil && st == http.StatusOK {
			pair.roadmap = b
		}
	}()
	go func() {
		defer wg.Done()
		cctx, cancel := context.WithTimeout(ctx, aggCallTimeout)
		defer cancel()
		if b, st, err := g.curriculum.get(cctx, "/paths/"+url.PathEscape(slug)+"/problems"); err == nil && st == http.StatusOK {
			pair.problems = b
		}
	}()
	wg.Wait()
	return pair
}
