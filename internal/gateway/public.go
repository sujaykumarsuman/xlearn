package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/gateway/limit"
)

// This file is the PUBLIC user dashboard aggregation (F009 / ADR-0024, ADR-0025): the
// LeetCode-style profile at projects.sujaykumar.dev/xlearn/u/<username>. It is the ONLY
// unauthenticated /api route — every other handler starts with authAccount; this one
// deliberately does not.
//
// Security model (ADR-0024). PII lives only in identity. The public path resolves the
// username via identity's ClusterIP-only /internal/accounts/by-username, which returns
// ONLY non-PII fields (id, username, display name, join date, visible courses) — it NEVER
// calls identity's PII /accounts/{id}. It then mints read-scoped service tokens for the
// RESOLVED account and fans out to assessment (projections) + curriculum (taxonomy) exactly
// as the authed Progress aggregation does, but composes only public aggregates (solved
// counts, streak, mock count, activity heatmap, per-course completion/mastery) into typed
// structs: nothing an upstream adds passes through. The response is cached per resolved
// account (public → shareable across viewers; a short TTL + the account's own
// mutating-write invalidation bound staleness).
//
// The v2 floor (m1-05; ADR-0033 §13, rollout §10):
//   - P1 (D31): the mock tile is the COUNT only; best/average never leave the gateway (the
//     authed Progress keeps them).
//   - P2: only enrolled ∩ public_visible ∩ active courses (identity's visible_courses,
//     intersected again with the gateway's own active list).
//   - P4/L4: 60/min per IP (burst 20), a 60 s negative 404 cache, ≤ 8 concurrent cold
//     composes (429 busy beyond).
//   - P10: the payload shape is pinned by an allowlist test (public_test.go); joinedAt is
//     a date (no sub-day timestamps).
//   - P11: a suspended account 404s on the very next request (never cache a positive
//     resolve; see handlePublicProfile).

// Public-profile abuse controls (L4; ADR-0035 §4).
const (
	// publicNegativeTTL is how long a 404 (unknown, suspended or malformed username) is
	// answered from memory without asking identity. A just-claimed or reactivated
	// username can therefore 404 for up to this long (accepted, sprint m1-05).
	publicNegativeTTL = 60 * time.Second
	// publicNegativeMax bounds the negative cache; on overflow it is dropped (like aggCache).
	publicNegativeMax = 4096
	// publicMaxComposes caps concurrent cold composes; the next one gets 429 busy.
	publicMaxComposes = 8
)

// publicAccount is identity's non-PII resolver payload for a username. `region` is a coarse
// UTC-offset band (never the IANA zone) — the low-identifying "Region" (F009 review).
// VisibleCourses is enrolled ∩ public_visible ∩ active (P2); identity answers only active
// accounts (P11).
type publicAccount struct {
	AccountID      string   `json:"account_id"`
	Username       string   `json:"username"`
	DisplayName    string   `json:"display_name"`
	CreatedAt      string   `json:"created_at"`
	Region         string   `json:"region"`
	VisibleCourses []string `json:"visible_courses"`
}

// --- composed public output shapes (camelCase for the SPA) ---

type publicUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	JoinedAt    string `json:"joinedAt"` // YYYY-MM-DD (P10: no sub-day timestamps)
	Region      string `json:"region"`   // coarse UTC-offset band, "" when unknown
}

type publicStreak struct {
	Current int `json:"current"`
	Longest int `json:"longest"`
}

type publicTotals struct {
	Solved int          `json:"solved"`
	Streak publicStreak `json:"streak"`
}

// publicMock is the mock tile: the count only (P1, D31).
type publicMock struct {
	Count int `json:"count"`
}

type publicHeatmapDay struct {
	Date    string `json:"date"`
	Solves  int    `json:"solves"`
	Reviews int    `json:"reviews"`
}

type publicHeatmap struct {
	Days []publicHeatmapDay `json:"days"`
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
	User    publicUser     `json:"user"`
	Totals  publicTotals   `json:"totals"`
	Mock    publicMock     `json:"mock"`
	Heatmap *publicHeatmap `json:"heatmap"` // null when the projection is unavailable
	Courses []publicCourse `json:"courses"`
}

// handlePublicProfile serves GET /api/u/{username} — the public dashboard. No session.
func (g *Gateway) handlePublicProfile(w http.ResponseWriter, r *http.Request) {
	if g.identity == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "identity not configured")
		return
	}
	// L4: 60/min per IP (burst 20).
	if !allowIP(w, r, g.limits.publicIP, "public") {
		return
	}
	username := r.PathValue("username")
	negKey := strings.ToLower(strings.TrimSpace(username))
	if g.publicNeg.has(negKey) {
		writePublicNotFound(w)
		return
	}

	// 1. Resolve username → account (non-PII fields only). A genuine 404 is "no such user"
	//    (unknown, suspended or malformed — identity answers them alike); a transport/other
	//    failure is a 502 (don't leak identity internals).
	//
	//    P11: never cache a positive resolve. Every public request asks identity again, so a
	//    suspend (run in the identity pod; the gateway is not a NATS client) hides the
	//    profile on the very next request even while the composed payload below is warm.
	body, status, err := g.identity.publicAccountByUsername(r.Context(), username)
	if err != nil {
		g.log.Warn("bff public profile: identity resolve failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "identity unavailable")
		return
	}
	if status == http.StatusNotFound {
		g.publicNeg.add(negKey)
		writePublicNotFound(w)
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
	//    across all anonymous viewers and invalidated by that account's own writes). It is
	//    reached only after a successful resolve (P11).
	const cacheName = "public-profile"
	if cached, ok := g.cache.get(acct.AccountID, cacheName); ok {
		passthrough(w, http.StatusOK, cached)
		return
	}
	cacheEpoch := g.cache.epoch(acct.AccountID)

	// 3. A cold compose fans out to 3 + 2·courses upstream calls: at most
	//    publicMaxComposes run at once (L4); the next caller gets 429 busy.
	select {
	case g.composeSem <- struct{}{}:
	default:
		limit.WriteTooMany(w, limit.CodeBusy, time.Second)
		return
	}
	profile := func() publicProfile {
		defer func() { <-g.composeSem }()
		return g.composePublicProfile(r.Context(), acct)
	}()
	out, err := json.Marshal(profile)
	if err != nil {
		g.log.Error("bff public profile: marshal failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "profile compose failed")
		return
	}
	g.cache.putFresh(acct.AccountID, cacheName, out, cacheEpoch)
	passthrough(w, http.StatusOK, out)
}

func writePublicNotFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, "not_found", "no such user")
}

// composePublicProfile fans out to assessment + curriculum for the resolved account and
// composes the public payload. Every upstream section degrades independently: a projection
// outage yields zeros / a null heatmap / empty courses rather than failing the page.
func (g *Gateway) composePublicProfile(ctx context.Context, acct publicAccount) publicProfile {
	profile := publicProfile{
		User:    publicUser{Username: acct.Username, DisplayName: acct.DisplayName, JoinedAt: publicDate(acct.CreatedAt), Region: acct.Region},
		Courses: []publicCourse{},
	}

	aToken, _ := g.mintQuiet(acct.AccountID, g.audAssessment)

	// The courses come first (a quick sequential read) so we know how many per-course
	// fan-outs to launch: only visible, active courses are shown (P2).
	courses := g.publicCourses(ctx, acct.VisibleCourses)

	var (
		summaryRaw, heatmapRaw, masteryRaw json.RawMessage
		courseRaw                          = make([]coursePair, len(courses))
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
		for i, p := range courses {
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

	// Profile totals + the mock count come from the summary projection (account-wide; P6
	// moves the totals to visible courses in M2b). best/average are never read (P1).
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
	// The heatmap is already account-wide (merged across courses) in assessment (ADR-0018).
	// It is re-shaped field by field, so nothing else in the projection can pass through.
	profile.Heatmap = publicHeatmapFrom(heatmapRaw)

	// Per-course completion + pattern mastery, composed from curriculum taxonomy ∩ the
	// solved set (the same roll-up the authed Progress screen uses; ADR-0018).
	for i, p := range courses {
		cr := courseRaw[i]
		var roadmap roadmapDoc
		_ = json.Unmarshal(cr.roadmap, &roadmap)
		var problems problemsDoc
		_ = json.Unmarshal(cr.problems, &problems)
		pc := publicCourse{
			Slug:     p.slug,
			Title:    p.title,
			Phases:   composePhaseCompletion(roadmap.Phases, problems.Problems, mastery.Problems),
			Patterns: composePatternMastery(problems.Problems, mastery.Problems),
		}
		for _, pr := range problems.Problems {
			if pr.IsReinforcement {
				continue
			}
			pc.Total++
			if solvedSet[pr.ID] {
				pc.Solved++
			}
		}
		if pc.Total > 0 {
			pc.Pct = int(100.0*float64(pc.Solved)/float64(pc.Total) + 0.5)
		}
		profile.Courses = append(profile.Courses, pc)
	}
	return profile
}

// publicDate renders identity's created_at as a date (YYYY-MM-DD, UTC): the public payload
// carries no sub-day timestamp (P10). "" when it can't be read.
func publicDate(s string) string {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC().Format(time.DateOnly)
	}
	if t, err := time.Parse(time.DateOnly, s); err == nil {
		return t.Format(time.DateOnly)
	}
	return ""
}

// publicHeatmapFrom re-shapes assessment's heatmap projection into the public shape
// (days[].{date,solves,reviews}); nil (JSON null) when it is missing or malformed. A day
// whose date isn't a plain date is truncated to one, or dropped.
func publicHeatmapFrom(raw json.RawMessage) *publicHeatmap {
	if len(raw) == 0 {
		return nil
	}
	var doc struct {
		Days []publicHeatmapDay `json:"days"`
	}
	if json.Unmarshal(raw, &doc) != nil || doc.Days == nil {
		return nil
	}
	out := &publicHeatmap{Days: make([]publicHeatmapDay, 0, len(doc.Days))}
	for _, d := range doc.Days {
		if d.Date = publicDate(d.Date); d.Date != "" {
			out.Days = append(out.Days, d)
		}
	}
	return out
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

// publicCourses is the public profile's course list (P2): the curriculum catalog's active
// courses, in catalog order, kept only when identity listed them as visible AND the
// gateway's own compiled registry says active (defence in depth: a preview course never
// shows, whatever an upstream says). Empty on any curriculum failure.
func (g *Gateway) publicCourses(ctx context.Context, visible []string) []activePathRef {
	if len(visible) == 0 {
		return nil
	}
	want := make(map[string]bool, len(visible))
	for _, s := range visible {
		want[s] = true
	}
	var out []activePathRef
	for _, p := range g.activePaths(ctx) {
		if !want[p.slug] {
			continue
		}
		if m, ok := g.courses.Lookup(p.slug); !ok || m.Status != course.StatusActive {
			continue
		}
		out = append(out, p)
	}
	return out
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

// negativeCache remembers public-profile 404s for a TTL (L4) so a scan of unknown
// usernames doesn't reach identity. Bounded: on overflow it is dropped whole (a cold
// cache only means identity is asked again). It is the ONLY resolver-side cache (P11).
type negativeCache struct {
	ttl time.Duration
	now func() time.Time

	mu sync.Mutex
	m  map[string]time.Time // key → expiry
}

func newNegativeCache(ttl time.Duration, now func() time.Time) *negativeCache {
	return &negativeCache{ttl: ttl, now: now, m: make(map[string]time.Time)}
}

// has reports whether key has a live negative entry.
func (c *negativeCache) has(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp, ok := c.m[key]
	if !ok {
		return false
	}
	if !c.now().Before(exp) {
		delete(c.m, key)
		return false
	}
	return true
}

// add records a 404 for key.
func (c *negativeCache) add(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) >= publicNegativeMax {
		c.m = make(map[string]time.Time)
	}
	c.m[key] = c.now().Add(c.ttl)
}

// len is the number of entries held (tests).
func (c *negativeCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.m)
}
