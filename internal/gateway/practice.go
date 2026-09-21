package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// practiceClient calls the internal practice service (per-user guided flow). The
// gateway forwards a practice-scoped JWT on every call (ADR-0006).
type practiceClient struct {
	baseURL string
	httpc   *http.Client
}

func newPracticeClient(baseURL string) *practiceClient {
	return &practiceClient{
		baseURL: baseURL,
		httpc:   &http.Client{Timeout: 10 * time.Second},
	}
}

// get issues an authenticated GET to practice and returns the raw body + status.
func (c *practiceClient) get(ctx context.Context, token, path string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

// post issues an authenticated POST (with body) to practice.
func (c *practiceClient) post(ctx context.Context, token, path string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

func (c *practiceClient) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return body, resp.StatusCode, err
}

// --- practice-state parsing helpers ---

// parseProblemState extracts the raw `state` object and the unlocked-stage set from a
// practice GET /state/{id} response ({"state": {...}}). The unlocked set always
// includes at least the attempt stage so the statement is never withheld.
func parseProblemState(body []byte) (json.RawMessage, map[string]bool, bool) {
	var env struct {
		State json.RawMessage `json:"state"`
	}
	if err := json.Unmarshal(body, &env); err != nil || len(env.State) == 0 {
		return nil, nil, false
	}
	var meta struct {
		UnlockedStages []string `json:"unlockedStages"`
	}
	if err := json.Unmarshal(env.State, &meta); err != nil {
		return nil, nil, false
	}
	unlocked := map[string]bool{}
	for _, s := range meta.UnlockedStages {
		unlocked[s] = true
	}
	if len(unlocked) == 0 {
		unlocked["attempt"] = true
	}
	return env.State, unlocked, true
}

// defaultProblemState is the state embedded when practice is unavailable: available,
// statement-only, no timer. Its shape matches practice's own state JSON.
func defaultProblemState(id string) (json.RawMessage, map[string]bool) {
	raw, _ := json.Marshal(map[string]any{
		"problemId":      id,
		"status":         "available",
		"stageReached":   "",
		"unlockedStages": []string{"attempt"},
		"currentTouch":   0,
		"lastOutcome":    nil,
		"firstSolvedAt":  nil,
		"revealedEarly":  false,
		"timer":          nil,
	})
	return raw, map[string]bool{"attempt": true}
}

// parseWeekStates parses a practice GET /state?ids=… response into the slim per-problem
// state the week rollup needs.
func parseWeekStates(body []byte) (map[string]practiceProblemState, bool) {
	var env struct {
		States map[string]struct {
			Status       string  `json:"status"`
			LastOutcome  *string `json:"lastOutcome"`
			CurrentTouch int     `json:"currentTouch"`
		} `json:"states"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, false
	}
	out := make(map[string]practiceProblemState, len(env.States))
	for id, s := range env.States {
		out[id] = practiceProblemState{Status: s.Status, LastOutcome: s.LastOutcome, CurrentTouch: s.CurrentTouch}
	}
	return out, true
}

// problemIDsFromWeek pulls the problem ids out of a curriculum week response so the
// gateway can ask practice for exactly those states.
func problemIDsFromWeek(content []byte) []string {
	var obj struct {
		Problems []struct {
			ID string `json:"id"`
		} `json:"problems"`
	}
	if err := json.Unmarshal(content, &obj); err != nil {
		return nil
	}
	ids := make([]string, 0, len(obj.Problems))
	for _, p := range obj.Problems {
		if p.ID != "" {
			ids = append(ids, p.ID)
		}
	}
	return ids
}
