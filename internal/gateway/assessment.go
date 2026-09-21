package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

// assessmentClient calls the internal assessment service (the timed mock aggregate).
// The gateway forwards an assessment-scoped JWT on every call (ADR-0006).
type assessmentClient struct {
	baseURL string
	httpc   *http.Client
}

func newAssessmentClient(baseURL string) *assessmentClient {
	return &assessmentClient{
		baseURL: baseURL,
		httpc:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *assessmentClient) get(ctx context.Context, token, path string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

func (c *assessmentClient) post(ctx context.Context, token, path string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

func (c *assessmentClient) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return body, resp.StatusCode, err
}

// --- mock handlers (api.md: /mocks*) ---
//
// assessment owns only a bare problem_id (ADR-0005: no cross-schema reads). The Mock
// screen's live/results header needs the problem's title/pattern, so the gateway
// enriches the single-mock view with curriculum problem metadata — the same BFF-
// composition role it plays for the Week / Problem / Revision aggregations. A problem
// curriculum can't resolve degrades to a null `problem` (the screen falls back to the
// id) rather than failing the response.

// handleStartMock proxies POST /mocks (start a 45-min session) to assessment and
// enriches the returned live view with the problem metadata.
func (g *Gateway) handleStartMock(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.assessment == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "assessment not configured")
		return
	}
	token, ok := g.mintForAssessmentW(w, accountID)
	if !ok {
		return
	}
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "could not read body")
		return
	}
	body, status, err := g.assessment.post(r.Context(), token, "/mocks", reqBody)
	if err != nil {
		g.log.Error("bff mocks start: assessment call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "assessment unavailable")
		return
	}
	if status != http.StatusCreated && status != http.StatusOK {
		passthrough(w, status, body)
		return
	}
	passthrough(w, status, g.enrichMockView(r.Context(), body))
}

// handleGetMock proxies GET /mocks/{id} (live session + phase-rail state) to assessment
// and enriches the view with the problem metadata.
func (g *Gateway) handleGetMock(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.assessment == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "assessment not configured")
		return
	}
	token, ok := g.mintForAssessmentW(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.assessment.get(r.Context(), token, "/mocks/"+url.PathEscape(r.PathValue("id")))
	if err != nil {
		g.log.Error("bff mocks get: assessment call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "assessment unavailable")
		return
	}
	if status != http.StatusOK {
		passthrough(w, status, body)
		return
	}
	passthrough(w, http.StatusOK, g.enrichMockView(r.Context(), body))
}

// handleScoreMock proxies POST /mocks/{id}/score (7-dim rubric -> /35) to assessment
// and enriches the scored view with the problem metadata.
func (g *Gateway) handleScoreMock(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.assessment == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "assessment not configured")
		return
	}
	token, ok := g.mintForAssessmentW(w, accountID)
	if !ok {
		return
	}
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "could not read body")
		return
	}
	upstream := "/mocks/" + url.PathEscape(r.PathValue("id")) + "/score"
	body, status, err := g.assessment.post(r.Context(), token, upstream, reqBody)
	if err != nil {
		g.log.Error("bff mocks score: assessment call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "assessment unavailable")
		return
	}
	if status != http.StatusOK {
		passthrough(w, status, body)
		return
	}
	passthrough(w, http.StatusOK, g.enrichMockView(r.Context(), body))
}

// handleMockTrend proxies GET /mocks/trend (scored /35 history vs targets) to
// assessment. The trend is a line chart of totals — no problem enrichment needed.
func (g *Gateway) handleMockTrend(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.assessment == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "assessment not configured")
		return
	}
	token, ok := g.mintForAssessmentW(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.assessment.get(r.Context(), token, "/mocks/trend")
	if err != nil {
		g.log.Error("bff mocks trend: assessment call failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "assessment unavailable")
		return
	}
	passthrough(w, status, body)
}

// enrichMockView attaches the mock's curriculum problem metadata (title/difficulty/
// pattern) as a `problem` key, resolved from the view's problemId. A missing/unknown
// problem leaves `problem` absent (the screen falls back to the id). On any parse
// failure it returns the raw body unchanged so the screen still gets assessment's data.
func (g *Gateway) enrichMockView(ctx context.Context, body []byte) []byte {
	if g.curriculum == nil {
		return body
	}
	var view map[string]json.RawMessage
	if err := json.Unmarshal(body, &view); err != nil {
		g.log.Warn("bff mock view: unparseable assessment response; passing through", "err", err)
		return body
	}
	var problemID string
	if raw, ok := view["problemId"]; ok {
		_ = json.Unmarshal(raw, &problemID)
	}
	if problemID == "" {
		return body
	}
	meta := g.curriculumProblemMeta(ctx, problemID)
	if len(meta) == 0 {
		return body
	}
	view["problem"] = meta
	merged, err := json.Marshal(view)
	if err != nil {
		g.log.Warn("bff mock view: re-marshal failed; passing through", "err", err)
		return body
	}
	return merged
}

// mintForAssessmentW mints an assessment-scoped JWT, writing the error envelope on failure.
func (g *Gateway) mintForAssessmentW(w http.ResponseWriter, accountID string) (string, bool) {
	return g.mintFor(w, accountID, g.audAssessment)
}
