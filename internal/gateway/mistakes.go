package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"

	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
)

// This file is the gateway's mistake-journal / weak-area / dashboard surface (S07).
// GET reads are BFF aggregations: review owns only bare problem ids (ADR-0005), so the
// gateway enriches each entry with curriculum problem metadata (title/difficulty),
// the same composition it does for the Revision due queue. Writes proxy to review with
// a minted review-scoped JWT (ADR-0006).

// handleMistakes: GET /paths/{slug}/mistakes?status= (and the DSA alias /mistakes) — the
// course's journal, each entry enriched with its curriculum problem metadata.
func (g *Gateway) handleMistakes(w http.ResponseWriter, r *http.Request) {
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
	upstream := "/mistakes"
	if status := r.URL.Query().Get("status"); status != "" {
		upstream += "?status=" + url.QueryEscape(status)
	}
	body, status, err := g.review.get(r.Context(), token, withPath(upstream, slug))
	if err != nil {
		g.log.Error("bff mistakes: review call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "review unavailable")
		return
	}
	if status != http.StatusOK {
		passthrough(w, status, body)
		return
	}
	passthrough(w, http.StatusOK, g.composeMistakeEnvelope(r, accountID, slug, body, "mistakes"))
}

// composeMistakeEnvelope enriches review's mistake entries (under arrayKey) with their
// curriculum problems and applies the list withholding (m1-06): a live item loses its
// pattern and concepts on both the curriculum join and review's own entry `pattern`.
// The item states are resolved while curriculum enriches.
func (g *Gateway) composeMistakeEnvelope(r *http.Request, accountID, slug string, body []byte, arrayKey string) []byte {
	var (
		states map[string]itemState
		wg     sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		states, _ = g.itemStates(r.Context(), accountID, slug, listItemIDs(body, arrayKey), stateInputs{})
	}()
	enriched := g.enrichMistakeEnvelope(r.Context(), body, arrayKey)
	wg.Wait()
	return withholdListBody(enriched, arrayKey, states)
}

// handleCreateMistake: POST /paths/{slug}/mistakes (and the DSA alias POST /mistakes) —
// proxy the create to review, in the route's course.
func (g *Gateway) handleCreateMistake(w http.ResponseWriter, r *http.Request) {
	// Auth first, as every route: an unauthenticated caller gets 401, not a course hint,
	// and the validated session carries the cohort bit the visibility check reads.
	if _, ok := g.authSession(w, r); !ok {
		return
	}
	slug := r.PathValue("slug")
	if _, ok := g.requireCourse(w, r, slug); !ok {
		return
	}
	g.proxyReviewWrite(w, r, http.MethodPost, withPath("/mistakes", slug))
}

// handlePatchMistake: PATCH /mistakes/{id} — proxy the edit to review.
func (g *Gateway) handlePatchMistake(w http.ResponseWriter, r *http.Request) {
	g.proxyReviewWrite(w, r, http.MethodPatch, "/mistakes/"+url.PathEscape(r.PathValue("id")))
}

// handleWeakArea: GET /paths/{slug}/weak-area (and the DSA alias /weak-area) — the
// course's weekly weak-area banner, with its supporting entries enriched with curriculum
// problem metadata.
func (g *Gateway) handleWeakArea(w http.ResponseWriter, r *http.Request) {
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
	body, status, err := g.review.get(r.Context(), token, withPath("/weak-area/current", slug))
	if err != nil {
		g.log.Error("bff weak-area: review call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "review unavailable")
		return
	}
	if status != http.StatusOK {
		passthrough(w, status, body)
		return
	}
	passthrough(w, http.StatusOK, g.composeMistakeEnvelope(r, accountID, slug, body, "entries"))
}

// proxyReviewWrite validates the session, mints a review-scoped JWT, and forwards a
// write (POST/PATCH with body) to review, passing its status + JSON envelope through.
func (g *Gateway) proxyReviewWrite(w http.ResponseWriter, r *http.Request, method, upstreamPath string) {
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
	var (
		body   []byte
		status int
		err    error
	)
	switch method {
	case http.MethodPatch:
		body, status, err = g.review.patch(r.Context(), token, upstreamPath, reqBody)
	default:
		body, status, err = g.review.post(r.Context(), token, upstreamPath, reqBody)
	}
	if err != nil {
		g.log.Error("bff review write failed", "path", upstreamPath, "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "review unavailable")
		return
	}
	// A mistake create/edit changes the weak-area the Dashboard shows → invalidate.
	g.invalidateAgg(status, accountID)
	passthrough(w, status, body)
}

// enrichMistakeEnvelope parses a review response object, attaches each entry's
// curriculum problem metadata under `problem`, and re-marshals. arrayKey names the
// array of mistake-shaped objects ("mistakes" for the journal, "entries" for the
// weak-area supporting list). On any parse failure it returns the body unchanged.
func (g *Gateway) enrichMistakeEnvelope(ctx context.Context, body []byte, arrayKey string) []byte {
	if g.curriculum == nil {
		return body
	}
	var env map[string]json.RawMessage
	if err := json.Unmarshal(body, &env); err != nil {
		g.log.Warn("bff mistakes: unparseable review response; passing through", "err", err)
		return body
	}
	rawArr, ok := env[arrayKey]
	if !ok || len(rawArr) == 0 {
		return body
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(rawArr, &items); err != nil {
		return body
	}
	// First pass: collect each entry's problem id. Second pass: attach the metadata,
	// resolving all ids in ONE bulk curriculum call (no N+1).
	pids := make([]string, len(items))
	ids := make([]string, 0, len(items))
	for i := range items {
		var pid string
		if raw, ok := items[i]["problemId"]; ok && json.Unmarshal(raw, &pid) == nil && pid != "" {
			pids[i] = pid
			ids = append(ids, pid)
		}
	}
	metas := g.curriculumProblemMetas(ctx, ids)
	for i := range items {
		meta := metas[pids[i]] // "" or unresolved ⇒ nil
		if meta == nil {
			meta = json.RawMessage("null")
		}
		items[i]["problem"] = meta
	}
	newArr, err := json.Marshal(items)
	if err != nil {
		return body
	}
	env[arrayKey] = newArr
	out, err := json.Marshal(env)
	if err != nil {
		return body
	}
	return out
}
