package review

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// This file holds the review service's two internal HTTP clients for SOFT
// cross-context references (ADR-0005/0016), both used by BACKGROUND paths that have no
// user request context / JWT:
//
//   - CurriculumClient resolves a problem's `pattern` (mistake pre-fill), called from
//     the store when opening a mistake. curriculum's content reads are unauthenticated
//     on the ClusterIP (the gateway proxies them without a user JWT too).
//   - IdentityClient resolves an account's timezone + study-budget for the weekly
//     weak-area recompute and the notifications worker. It calls identity's internal,
//     ClusterIP-only /internal/accounts/{id} endpoint (no user JWT — the same
//     network-isolation trust model as identity's session endpoints, ADR-0006/0016).

// clientTimeout bounds each internal call so a slow dependency can't wedge a consumer
// handler or hold a DB transaction open (the pattern resolver runs inside the
// mistake-open tx).
const clientTimeout = 5 * time.Second

// CurriculumClient resolves problem metadata from the curriculum service.
type CurriculumClient struct {
	baseURL string
	httpc   *http.Client
}

// NewCurriculumClient builds the client, or nil when no base URL is configured (local
// dev / tests: the store then pre-fills an empty pattern, which is acceptable).
func NewCurriculumClient(baseURL string) *CurriculumClient {
	if baseURL == "" {
		return nil
	}
	return &CurriculumClient{baseURL: baseURL, httpc: &http.Client{Timeout: clientTimeout}}
}

// Pattern resolves a problem's pattern (store.PatternResolver). A nil client, a
// non-200, or a malformed body yields "" — pattern pre-fill is best-effort.
func (c *CurriculumClient) Pattern(ctx context.Context, problemID string) (string, error) {
	if c == nil {
		return "", nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/problems/"+url.PathEscape(problemID), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var env struct {
		Problem struct {
			Pattern string `json:"pattern"`
		} `json:"problem"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return "", nil
	}
	return env.Problem.Pattern, nil
}

// IdentityClient resolves an account's timezone + study budget from identity's
// internal endpoint.
type IdentityClient struct {
	baseURL string
	httpc   *http.Client
}

// NewIdentityClient builds the client, or nil when no base URL is configured (local
// dev / tests: the weak-area recompute then falls back to UTC and the notifications
// worker schedules reminders immediately).
func NewIdentityClient(baseURL string) *IdentityClient {
	if baseURL == "" {
		return nil
	}
	return &IdentityClient{baseURL: baseURL, httpc: &http.Client{Timeout: clientTimeout}}
}

// ResolveAccount returns an account's IANA timezone and raw study_budget JSON. A nil
// client or any failure yields ("UTC", nil, err?) so callers degrade gracefully; the
// timezone is never empty.
func (c *IdentityClient) ResolveAccount(ctx context.Context, accountID string) (string, []byte, error) {
	if c == nil {
		return "UTC", nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/internal/accounts/"+url.PathEscape(accountID), nil)
	if err != nil {
		return "UTC", nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return "UTC", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "UTC", nil, fmt.Errorf("identity internal accounts: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "UTC", nil, err
	}
	var out struct {
		Timezone    string          `json:"timezone"`
		StudyBudget json.RawMessage `json:"study_budget"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "UTC", nil, err
	}
	tz := out.Timezone
	if tz == "" {
		tz = "UTC"
	}
	return tz, out.StudyBudget, nil
}
