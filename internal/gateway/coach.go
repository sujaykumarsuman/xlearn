package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// coachEmptyKey is the "no key — coach off" empty state (api.md GET /coach/key). It is
// returned when coach is not deployed or has no key for the account. The masked read
// never carries a raw key.
var coachEmptyKey = []byte(`{"keys":[],"connected":false}`)

// coachModeHeader carries the SERVER-AUTHORITATIVE behaviour gate to coach: the gateway
// derives it from practice's state (not the client), so a browser can't unlock reviewer
// mode mid-attempt and leak a solution (ADR-0007). Must match coach's headerCoachMode.
const coachModeHeader = "X-Coach-Mode"

// coach behaviour modes (mirror internal/coach).
const (
	coachModeAttempt = "attempt"
	coachModeReview  = "review"
	coachModeGeneral = "general"
)

// coachClient calls the internal coach service. The key + thread calls use a normal
// short-timeout client; the chat call streams SSE and uses a no-timeout client bounded
// by the request context (a fixed timeout would cut a long reply).
type coachClient struct {
	baseURL string
	httpc   *http.Client
	streamc *http.Client
}

func newCoachClient(baseURL string) *coachClient {
	return &coachClient{
		baseURL: baseURL,
		httpc:   &http.Client{Timeout: 10 * time.Second},
		streamc: &http.Client{Timeout: 0}, // streaming; bounded by the request context
	}
}

// getKey fetches the account's masked key config from coach (never the raw key). The
// external /coach/key maps to coach's internal /keys (services.md).
func (c *coachClient) getKey(ctx context.Context, token string) ([]byte, int, error) {
	return c.jsonReq(ctx, http.MethodGet, "/keys", token, nil)
}

// putKey stores/replaces the account's key (or toggles enabled). The gateway forwards the
// body verbatim; coach never returns the raw key.
func (c *coachClient) putKey(ctx context.Context, token string, body []byte) ([]byte, int, error) {
	return c.jsonReq(ctx, http.MethodPut, "/keys", token, body)
}

// deleteKey removes the account's key.
func (c *coachClient) deleteKey(ctx context.Context, token string) ([]byte, int, error) {
	return c.jsonReq(ctx, http.MethodDelete, "/keys", token, nil)
}

// getThread fetches the message history for a page context.
func (c *coachClient) getThread(ctx context.Context, token, pageContext string) ([]byte, int, error) {
	return c.jsonReq(ctx, http.MethodGet, "/threads?context="+url.QueryEscape(pageContext), token, nil)
}

// jsonReq issues a JSON request to coach and returns the buffered body + status.
func (c *coachClient) jsonReq(ctx context.Context, method, path, token string, body []byte) ([]byte, int, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return respBody, resp.StatusCode, err
}

// chatStream POSTs the chat request to coach and returns the raw streaming response for
// the caller to relay (it must close resp.Body). mode is the authoritative behaviour gate.
func (c *coachClient) chatStream(ctx context.Context, token, mode string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set(coachModeHeader, mode)
	return c.streamc.Do(req)
}

// handleCoachKey serves the masked coach-key read (api.md GET /coach/key). When coach is
// not configured or returns "no key" it renders the empty state so the Settings API-keys
// section shows "No key — coach off" rather than an error. It never returns a raw key.
func (g *Gateway) handleCoachKey(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.coach == nil {
		passthrough(w, http.StatusOK, coachEmptyKey)
		return
	}
	token, ok := g.mintForCoachW(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.coach.getKey(r.Context(), token)
	if err != nil {
		g.log.Warn("bff /coach/key: coach call failed; rendering empty state", "err", err)
		passthrough(w, http.StatusOK, coachEmptyKey)
		return
	}
	if status == http.StatusNotFound {
		passthrough(w, http.StatusOK, coachEmptyKey)
		return
	}
	passthrough(w, status, body)
}

// handlePutCoachKey stores/replaces the account's provider key or toggles enabled
// (api.md PUT /coach/key). The gateway never sees the raw key beyond forwarding it to
// coach, which encrypts it; coach returns only the masked view.
func (g *Gateway) handlePutCoachKey(w http.ResponseWriter, r *http.Request) {
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
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "could not read body")
		return
	}
	body, status, err := g.coach.putKey(r.Context(), token, reqBody)
	if err != nil {
		g.log.Error("bff PUT /coach/key: coach call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "coach unavailable")
		return
	}
	passthrough(w, status, body)
}

// handleDeleteCoachKey removes the account's provider key (api.md DELETE /coach/key).
func (g *Gateway) handleDeleteCoachKey(w http.ResponseWriter, r *http.Request) {
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
	body, status, err := g.coach.deleteKey(r.Context(), token)
	if err != nil {
		g.log.Error("bff DELETE /coach/key: coach call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "coach unavailable")
		return
	}
	if status == http.StatusNoContent {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	passthrough(w, status, body)
}

// handleCoachThread returns the chat history for a page context (api.md
// GET /coach/thread?context=).
func (g *Gateway) handleCoachThread(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	pageContext := r.URL.Query().Get("context")
	if pageContext == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "context is required")
		return
	}
	if g.coach == nil {
		// No coach yet → an empty thread so the panel renders its empty state.
		passthrough(w, http.StatusOK, []byte(`{"messages":[]}`))
		return
	}
	token, ok := g.mintForCoachW(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.coach.getThread(r.Context(), token, pageContext)
	if err != nil {
		g.log.Warn("bff /coach/thread: coach call failed; empty thread", "err", err)
		passthrough(w, http.StatusOK, []byte(`{"messages":[]}`))
		return
	}
	passthrough(w, status, body)
}

// handleCoachChat proxies the coach chat SSE stream (api.md POST /coach/chat). It derives
// the SERVER-AUTHORITATIVE behaviour mode from practice's state (never the client) and
// passes it to coach as a header, then relays coach's response byte-for-byte with
// flushing so the token stream reaches the browser live (no buffering). The gateway
// never sees the raw provider key — coach makes the provider call.
func (g *Gateway) handleCoachChat(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.coach == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "coach not configured")
		return
	}
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "could not read body")
		return
	}
	token, ok := g.mintForCoachW(w, accountID)
	if !ok {
		return
	}
	mode, body := g.coachEnrich(r, accountID, reqBody)

	// Clear the server WriteTimeout (60s) before dialling coach: chatStream can block up
	// to the provider limit before coach's first byte, and a transport error after 60s
	// must still be able to write its clean 502 (relayStream only clears the deadline on
	// the success path). Reaches the base writer via httpx.statusWriter's Unwrap.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	resp, err := g.coach.chatStream(r.Context(), token, mode, body)
	if err != nil {
		g.log.Error("bff /coach/chat: coach call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "coach unavailable")
		return
	}
	defer resp.Body.Close()
	relayStream(w, resp)
}

// coachEnrich derives the SERVER-AUTHORITATIVE behaviour mode for a chat request AND
// rebinds the problem it describes to the SAME id — the one in the thread's `context`
// ("problem:<id>"), never a free client field. This closes the spoiler gate: reviewer
// mode is granted only when the CONTEXT problem is solved, and the coach names that same
// problem (title/pattern sourced from curriculum), so a client can't point the
// solved-check at a solved problem B while the coach discusses unsolved problem A.
//
// Only a problem context can be an attempt or a review; every other page is the general
// tutor (body unchanged). If practice can't confirm the problem is solved (or is
// unavailable) the mode is the SAFE default — attempt (spoiler-free). Returns the mode
// header value and the (possibly rewritten) request body to forward to coach.
func (g *Gateway) coachEnrich(r *http.Request, accountID string, body []byte) (string, []byte) {
	var meta struct {
		Context string `json:"context"`
	}
	_ = json.Unmarshal(body, &meta)
	id, ok := problemIDFromContext(meta.Context)
	if !ok {
		return coachModeGeneral, body
	}

	// Authoritative problem descriptors from curriculum (best-effort). On failure we still
	// bind problemId to the context id and clear the client's title/pattern, so the coach
	// never names a spoofed problem.
	var title, pattern string
	if g.curriculum != nil {
		title, pattern = coachProblemTitlePattern(g.curriculumProblemMeta(r.Context(), id))
	}

	// Authoritative mode from practice: review ONLY when THIS problem is solved.
	mode := coachModeAttempt
	if g.practice != nil {
		if token, okp := g.mintForPractice(accountID); okp {
			if pbody, pstatus, perr := g.practice.get(r.Context(), token, "/state/"+url.PathEscape(id)); perr == nil && pstatus == http.StatusOK && practiceSolved(pbody) {
				mode = coachModeReview
			}
		}
	}

	// Rewrite the descriptive problem fields to the authoritative values so the prompt
	// names the gated problem, not a client-spoofed one. Other fields (message, context,
	// kind, stage) pass through unchanged.
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil || obj == nil {
		obj = map[string]any{}
	}
	obj["problemId"] = id
	obj["problemTitle"] = title
	obj["pattern"] = pattern
	rewritten, err := json.Marshal(obj)
	if err != nil {
		return mode, body
	}
	return mode, rewritten
}

// problemIDFromContext extracts the problem id from a "problem:<id>" thread context.
func problemIDFromContext(ctx string) (string, bool) {
	const prefix = "problem:"
	if strings.HasPrefix(ctx, prefix) && len(ctx) > len(prefix) {
		return ctx[len(prefix):], true
	}
	return "", false
}

// coachProblemTitlePattern extracts the title + pattern from curriculum's raw `problem`
// object (as returned by curriculumProblemMeta), or empty strings when it's nil/malformed.
func coachProblemTitlePattern(raw json.RawMessage) (title, pattern string) {
	if len(raw) == 0 {
		return "", ""
	}
	var p struct {
		Title   string `json:"title"`
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", ""
	}
	return p.Title, p.Pattern
}

// practiceSolved reports whether a practice GET /state/{id} response marks the problem
// solved ({"state":{"status":"solved"|firstSolvedAt:...}}).
func practiceSolved(body []byte) bool {
	var env struct {
		State struct {
			Status        string  `json:"status"`
			FirstSolvedAt *string `json:"firstSolvedAt"`
		} `json:"state"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return false
	}
	return env.State.Status == "solved" || (env.State.FirstSolvedAt != nil && *env.State.FirstSolvedAt != "")
}

// relayStream copies an upstream (coach) response to the client with per-chunk flushing
// and no write deadline, so an SSE token stream is delivered live rather than buffered.
// It passes coach's status + content-type through, so a non-SSE error (e.g. 409 no_key)
// relays cleanly too. Flush + the deadline reset reach the base writer via
// httpx.statusWriter's Unwrap.
func relayStream(w http.ResponseWriter, resp *http.Response) {
	h := w.Header()
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		h.Set("Content-Type", ct)
	} else {
		h.Set("Content-Type", "text/event-stream")
	}
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})
	w.WriteHeader(resp.StatusCode)
	_ = rc.Flush()

	buf := make([]byte, 4096)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return
			}
			_ = rc.Flush()
		}
		if rerr != nil {
			return
		}
	}
}

// mintForCoachW mints a coach-scoped JWT, writing the error envelope on failure.
func (g *Gateway) mintForCoachW(w http.ResponseWriter, accountID string) (string, bool) {
	return g.mintFor(w, accountID, g.audCoach)
}
