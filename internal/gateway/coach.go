package gateway

import (
	"context"
	"io"
	"net/http"
	"time"
)

// coachEmptyKey is the "no key — coach off" empty state (api.md GET /coach/key). It is
// returned when coach is not deployed (S10) or has no key for the account. The masked
// read never carries a raw key.
var coachEmptyKey = []byte(`{"keys":[],"connected":false}`)

// coachClient calls the internal coach service. In S10 the only exercised call is the
// masked GET /keys read; the store/delete writes are wired in S11 (ADR-0007).
type coachClient struct {
	baseURL string
	httpc   *http.Client
}

func newCoachClient(baseURL string) *coachClient {
	return &coachClient{baseURL: baseURL, httpc: &http.Client{Timeout: 10 * time.Second}}
}

// getKey fetches the account's masked key config from coach (never the raw key). The
// external /coach/key maps to coach's internal /keys (services.md).
func (c *coachClient) getKey(ctx context.Context, token string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/keys", nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return body, resp.StatusCode, err
}

// handleCoachKey serves the masked coach-key read (api.md GET /coach/key). When coach
// is not configured (the S10 default — coach lands in S11) or returns "no key", it
// renders the empty state so the Settings API-keys section shows "No key — coach off"
// rather than an error. It never returns a raw key (the coach service masks it).
func (g *Gateway) handleCoachKey(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.coach == nil {
		// Coach is not deployed yet (S11 wires it) — the shell's empty state.
		passthrough(w, http.StatusOK, coachEmptyKey)
		return
	}
	token, ok := g.mintForCoachW(w, accountID)
	if !ok {
		return
	}
	body, status, err := g.coach.getKey(r.Context(), token)
	if err != nil {
		// Degrade to the empty state rather than failing the whole Settings screen.
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

// mintForCoachW mints a coach-scoped JWT, writing the error envelope on failure.
func (g *Gateway) mintForCoachW(w http.ResponseWriter, accountID string) (string, bool) {
	return g.mintFor(w, accountID, g.audCoach)
}
