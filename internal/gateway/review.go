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

// reviewClient calls the internal review service (the five-touch scheduler + Revision
// queue). The gateway forwards a review-scoped JWT on every call (ADR-0006).
type reviewClient struct {
	baseURL string
	httpc   *http.Client
}

func newReviewClient(baseURL string) *reviewClient {
	return &reviewClient{
		baseURL: baseURL,
		httpc:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *reviewClient) get(ctx context.Context, token, path string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

func (c *reviewClient) post(ctx context.Context, token, path string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

func (c *reviewClient) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return body, resp.StatusCode, err
}

// --- due-queue enrichment ---
//
// review owns only bare problem ids (ADR-0005: no cross-schema reads). The Revision
// screen needs each due item's title/difficulty/pattern, so the gateway enriches the
// queue with curriculum problem metadata — the same BFF-composition role it plays for
// the Week and Problem aggregations. A problem curriculum can't resolve degrades to a
// null `problem` (the screen falls back to the id) rather than failing the queue.

// reviewDueItem is one enriched entry in the due queue. Problem is the curriculum
// problem object (id/title/difficulty/pattern/week_n…) or null when unavailable.
type reviewDueItem struct {
	ItemID     string          `json:"itemId"`
	ProblemID  string          `json:"problemId"`
	TouchLevel int             `json:"touchLevel"`
	DayLabel   string          `json:"dayLabel"`
	DueDate    string          `json:"dueDate"`
	Due        bool            `json:"due"`
	MockMode   bool            `json:"mockMode"`
	Status     string          `json:"status"`
	Problem    json.RawMessage `json:"problem"`
}

type reviewDueResponse struct {
	Items    []reviewDueItem `json:"items"`
	DueCount int             `json:"dueCount"`
}

// enrichDueQueue parses review's due-queue response and attaches each item's
// curriculum problem metadata. It fetches each distinct problem id once. Returns the
// re-marshalled JSON. On a parse failure it returns the raw body unchanged so the
// screen still gets review's data.
func (g *Gateway) enrichDueQueue(ctx context.Context, body []byte) []byte {
	var resp reviewDueResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		g.log.Warn("bff revision/due: unparseable review response; passing through", "err", err)
		return body
	}
	if g.curriculum == nil || len(resp.Items) == 0 {
		return body
	}

	metas := make(map[string]json.RawMessage)
	for i := range resp.Items {
		pid := resp.Items[i].ProblemID
		if pid == "" {
			continue
		}
		if _, seen := metas[pid]; !seen {
			metas[pid] = g.curriculumProblemMeta(ctx, pid)
		}
		resp.Items[i].Problem = metas[pid]
	}

	merged, err := json.Marshal(resp)
	if err != nil {
		g.log.Warn("bff revision/due: re-marshal failed; passing through", "err", err)
		return body
	}
	return merged
}

// curriculumProblemMeta fetches a problem's `problem` object from curriculum, or nil
// when it can't be resolved (curriculum down, unknown id, malformed body).
func (g *Gateway) curriculumProblemMeta(ctx context.Context, problemID string) json.RawMessage {
	cbody, cstatus, cerr := g.curriculum.get(ctx, "/problems/"+url.PathEscape(problemID))
	if cerr != nil || cstatus != http.StatusOK {
		return nil
	}
	var env struct {
		Problem json.RawMessage `json:"problem"`
	}
	if err := json.Unmarshal(cbody, &env); err != nil || len(env.Problem) == 0 {
		return nil
	}
	return env.Problem
}
