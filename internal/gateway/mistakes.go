package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// This file is the gateway's mistake-journal / weak-area / dashboard surface (S07).
// GET reads are BFF aggregations: review owns only bare problem ids (ADR-0005), so the
// gateway enriches each entry with curriculum problem metadata (title/difficulty),
// the same composition it does for the Revision due queue. Writes proxy to review with
// a minted review-scoped JWT (ADR-0006).

// handleMistakes: GET /mistakes?status= — the journal, each entry enriched with its
// curriculum problem metadata.
func (g *Gateway) handleMistakes(w http.ResponseWriter, r *http.Request) {
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
	upstream := "/mistakes"
	if status := r.URL.Query().Get("status"); status != "" {
		upstream += "?status=" + url.QueryEscape(status)
	}
	body, status, err := g.review.get(r.Context(), token, upstream)
	if err != nil {
		g.log.Error("bff mistakes: review call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "review unavailable")
		return
	}
	if status != http.StatusOK {
		passthrough(w, status, body)
		return
	}
	passthrough(w, http.StatusOK, g.enrichMistakeEnvelope(r.Context(), body, "mistakes"))
}

// handleCreateMistake: POST /mistakes — proxy the create to review.
func (g *Gateway) handleCreateMistake(w http.ResponseWriter, r *http.Request) {
	g.proxyReviewWrite(w, r, http.MethodPost, "/mistakes")
}

// handlePatchMistake: PATCH /mistakes/{id} — proxy the edit to review.
func (g *Gateway) handlePatchMistake(w http.ResponseWriter, r *http.Request) {
	g.proxyReviewWrite(w, r, http.MethodPatch, "/mistakes/"+url.PathEscape(r.PathValue("id")))
}

// handleWeakArea: GET /weak-area — the weekly weak-area banner, with its supporting
// entries enriched with curriculum problem metadata.
func (g *Gateway) handleWeakArea(w http.ResponseWriter, r *http.Request) {
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
	body, status, err := g.review.get(r.Context(), token, "/weak-area/current")
	if err != nil {
		g.log.Error("bff weak-area: review call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "review unavailable")
		return
	}
	if status != http.StatusOK {
		passthrough(w, status, body)
		return
	}
	passthrough(w, http.StatusOK, g.enrichMistakeEnvelope(r.Context(), body, "entries"))
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
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "could not read body")
		return
	}
	var body []byte
	var status int
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
	metas := make(map[string]json.RawMessage)
	for i := range items {
		var pid string
		if raw, ok := items[i]["problemId"]; !ok || json.Unmarshal(raw, &pid) != nil || pid == "" {
			items[i]["problem"] = json.RawMessage("null")
			continue
		}
		if _, seen := metas[pid]; !seen {
			metas[pid] = g.curriculumProblemMeta(ctx, pid)
		}
		meta := metas[pid]
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
