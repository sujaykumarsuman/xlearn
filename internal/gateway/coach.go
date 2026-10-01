package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
)

// coachEmptyKey is the "no key — coach off" empty state (api.md GET /coach/key). It is
// returned when coach is not deployed or has no key for the account. The masked read
// never carries a raw key.
var coachEmptyKey = []byte(`{"keys":[],"connected":false,"default_provider":""}`)

// coachModeHeader carries the SERVER-AUTHORITATIVE behaviour gate to coach: the gateway
// derives it from the learner's state (coach_gate.go; never the client), so a browser
// can't unlock reviewer mode mid-attempt and leak a solution (ADR-0007). Must match
// coach's headerCoachMode. Since m1-07 the gateway also sets it on the relayed chat
// response, so the SPA's mode chip shows what the server decided (AB01 F1, F6, F7).
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

// deleteKey removes one provider's key (provider carried through as a query param).
func (c *coachClient) deleteKey(ctx context.Context, token, provider string) ([]byte, int, error) {
	return c.jsonReq(ctx, http.MethodDelete, "/keys?provider="+url.QueryEscape(provider), token, nil)
}

// getThread fetches the message history for a page context (already normalized).
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

// chatHeaders are the server-authoritative gate values the gateway sends coach with a
// chat (m1-07): the mode, the resolved course (persona) and, when a D27 assist is
// recorded, the attempt it is on.
type chatHeaders struct {
	mode, course, attempt string
}

// chatStream POSTs the chat request to coach and returns the raw streaming response for
// the caller to relay (it must close resp.Body). h carries the authoritative gate;
// pathSlug (optional) is a problem context's course, which coach writes on the thread
// (it defaults an absent one to course.DefaultSlug).
func (c *coachClient) chatStream(ctx context.Context, token string, h chatHeaders, pathSlug string, body []byte) (*http.Response, error) {
	target := c.baseURL + "/chat"
	if pathSlug != "" {
		target = withPath(target, pathSlug)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set(coachModeHeader, h.mode)
	req.Header.Set(coachCourseHeader, h.course)
	if h.attempt != "" {
		req.Header.Set(coachAttemptHeader, h.attempt)
	}
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
	reqBody, ok := httpx.ReadBody(w, r, httpx.BodyLimitDefault)
	if !ok {
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
	body, status, err := g.coach.deleteKey(r.Context(), token, r.URL.Query().Get("provider"))
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
// GET /coach/thread?context=). The context is normalized first with the shared dual
// parser (course.NormalizeCoachContext): a v1.6.0 tab's `week:3` reads the same thread as
// v1.7.0's `<DefaultSlug>:week:3` (m1-03); coach normalizes again, which is a no-op.
//
// m1-07: the body gains `gate` — the mode the server would apply to a chat here, the lock
// reason, and on a problem page the open attempt's D27 assist state — from the same
// lookups as the chat, run read-only (no assist write, never a 409). If a lookup fails,
// `gate` is omitted and the thread still loads: it is display-only (AB01's mode chip),
// and the chat itself stays fail-closed.
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
	cc := g.courses.NormalizeCoachContext(pageContext)
	pageContext = cc.Key
	if g.coach == nil {
		// No coach yet → an empty thread so the panel renders its empty state.
		passthrough(w, http.StatusOK, []byte(`{"messages":[]}`))
		return
	}
	token, ok := g.mintForCoachW(w, accountID)
	if !ok {
		return
	}

	var (
		wg     sync.WaitGroup
		gate   coachGate
		gateOK bool
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		gate, gateOK = g.coachGateFor(r.Context(), accountID, cc)
	}()
	body, status, err := g.coach.getThread(r.Context(), token, pageContext)
	wg.Wait()
	if err != nil {
		g.log.Warn("bff /coach/thread: coach call failed; empty thread", "err", err)
		body, status = []byte(`{"messages":[]}`), http.StatusOK
	}
	if status == http.StatusOK && gateOK {
		body = withGate(body, gate.json())
	}
	passthrough(w, status, body)
}

// withGate adds `gate` to a thread body (a JSON object); any other body is unchanged.
func withGate(body []byte, gate gateJSON) []byte {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil || obj == nil {
		return body
	}
	raw, err := json.Marshal(gate)
	if err != nil {
		return body
	}
	obj["gate"] = raw
	out, err := json.Marshal(obj)
	if err != nil {
		return body
	}
	return out
}

// handleCoachChat proxies the coach chat SSE stream (api.md POST /coach/chat). It
// resolves the SERVER-AUTHORITATIVE gate (coach_gate.go: never the client) and, in this
// order (sprint m1-07):
//
//  1. a lookup failed → 503 coach_state_unavailable, nothing sent (fails closed);
//  2. locked (a live mock the course switches the coach off for) → 409 coach_paused;
//  3. D27 — a problem:<id> chat while a counted attempt A on <id> is open:
//     A already carries an assist → no confirm (it is capped; a reload never re-prompts);
//     else assist_ack ≠ A → 409 assist_confirm_required {attemptId, problemId};
//     else coach's L18 admission probe (a 429 is relayed as is, A stays uncapped), then
//     practice records the assist (any failure → 503 assist_unavailable, nothing sent);
//  4. the body is rewritten for the mode (coach_gate.go rewriteChatBody) and forwarded
//     with X-Coach-Mode, X-Coach-Course and, after an assist, X-Coach-Attempt.
//
// Coach's response is relayed byte-for-byte with flushing so the token stream reaches
// the browser live, with X-Coach-Mode (the mode the SPA's chip shows) and coach's
// Retry-After (its L18 429s). The gateway never sees the raw provider key — coach makes
// the provider call.
func (g *Gateway) handleCoachChat(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.coach == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "coach not configured")
		return
	}
	reqBody, ok := httpx.ReadBody(w, r, httpx.BodyLimitDefault)
	if !ok {
		return
	}
	token, ok := g.mintForCoachW(w, accountID)
	if !ok {
		return
	}
	var meta struct {
		Context   string `json:"context"`
		AssistAck string `json:"assist_ack"`
	}
	_ = json.Unmarshal(reqBody, &meta)
	cc := g.courses.NormalizeCoachContext(meta.Context)

	gate, ok := g.coachGateFor(r.Context(), accountID, cc)
	if !ok {
		writeGateError(w, http.StatusServiceUnavailable, "", codeStateUnavailable,
			"the practice state could not be checked, so nothing was sent", nil)
		return
	}
	if gate.mode == coachModeLocked {
		writeGateError(w, http.StatusConflict, coachModeLocked, codeCoachPaused,
			"the coach is paused while a revision touch or mock is live", map[string]any{"reason": gate.reason})
		return
	}

	var assisted string
	if a := gate.attempt; a != nil {
		switch {
		case a.assisted():
			assisted = a.AttemptID
		case meta.AssistAck != a.AttemptID:
			writeGateError(w, http.StatusConflict, gate.mode, codeAssistConfirm,
				"using the coach during a counted attempt caps it at Assisted; resend with assist_ack to confirm",
				map[string]any{"attemptId": a.AttemptID, "problemId": gate.problemID})
			return
		default:
			if !g.coachAdmit(w, r, token, gate.mode) {
				return
			}
			if !g.recordAssist(r.Context(), accountID, a.AttemptID) {
				g.log.Warn("bff /coach/chat: assist record failed; nothing sent (fail closed)", "attempt", a.AttemptID)
				writeGateError(w, http.StatusServiceUnavailable, gate.mode, codeAssistUnavail,
					"coach use could not be recorded on the attempt, so nothing was sent", nil)
				return
			}
			assisted = a.AttemptID
		}
	}
	body := gate.rewriteChatBody(reqBody, cc.Key)

	// Clear the server WriteTimeout (60s) before dialling coach: chatStream can block up
	// to the provider limit before coach's first byte, and a transport error after 60s
	// must still be able to write its clean 502 (relayStream only clears the deadline on
	// the success path). Reaches the base writer via httpx.statusWriter's Unwrap.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	resp, err := g.coach.chatStream(r.Context(), token, chatHeaders{mode: gate.mode, course: gate.course, attempt: assisted}, gate.problemPath, body)
	if err != nil {
		g.log.Error("bff /coach/chat: coach call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "coach unavailable")
		return
	}
	defer resp.Body.Close()
	w.Header().Set(coachModeHeader, gate.mode)
	relayStream(w, resp)
}

// coachAdmit runs coach's read-only L18 admission probe before a D27 assist is recorded,
// so a chat the caps would refuse never caps the attempt. A 429 is relayed as is (its
// typed code and Retry-After); a probe that fails otherwise is a 502 (nothing recorded,
// nothing sent). It reports whether the chat may proceed.
func (g *Gateway) coachAdmit(w http.ResponseWriter, r *http.Request, token, mode string) bool {
	resp, body, err := g.coach.admission(r.Context(), token)
	if err != nil {
		g.log.Error("bff /coach/chat: admission probe failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "coach unavailable")
		return false
	}
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return true
	case http.StatusTooManyRequests:
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			w.Header().Set("Retry-After", ra)
		}
		w.Header().Set(coachModeHeader, mode)
		passthrough(w, http.StatusTooManyRequests, body)
		return false
	default:
		g.log.Error("bff /coach/chat: admission probe refused", "status", resp.StatusCode)
		writeError(w, http.StatusBadGateway, "upstream", "coach unavailable")
		return false
	}
}

// coachProblemPathSlug extracts the item's course from curriculum's raw `problem` object,
// or "" when it's nil/malformed (coach then applies its own course.DefaultSlug default).
func coachProblemPathSlug(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var p struct {
		PathSlug string `json:"path_slug"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return ""
	}
	return p.PathSlug
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

// relayStream copies an upstream (coach) response to the client with per-chunk flushing
// and no write deadline, so an SSE token stream is delivered live rather than buffered.
// It passes coach's status + content-type through, so a non-SSE error (e.g. 409 no_key)
// relays cleanly too, with coach's Retry-After (m1-07: its typed L18 429s). Flush + the
// deadline reset reach the base writer via httpx.statusWriter's Unwrap.
func relayStream(w http.ResponseWriter, resp *http.Response) {
	h := w.Header()
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		h.Set("Content-Type", ct)
	} else {
		h.Set("Content-Type", "text/event-stream")
	}
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		h.Set("Retry-After", ra)
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
