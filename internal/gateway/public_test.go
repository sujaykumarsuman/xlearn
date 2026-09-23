package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestBFFPublicProfileComposes exercises the unauthenticated public dashboard: it resolves
// a username, composes per-course + account-wide stats, and — critically — never leaks PII.
func TestBFFPublicProfileComposes(t *testing.T) {
	h := newAggHarness(t)
	// No cookie: the public endpoint must not require a session.
	body := h.get(t, "/xlearn/api/u/ada", nil)

	user, _ := body["user"].(map[string]any)
	if user == nil || user["username"] != "ada" || user["displayName"] != "Ada Lovelace" {
		t.Fatalf("user = %v", body["user"])
	}
	// Coarse region (UTC offset) passes through from the identity resolver (F009 review).
	if user["region"] != "UTC+05:30" {
		t.Fatalf("user.region = %v, want UTC+05:30", user["region"])
	}

	// PII must never appear anywhere in the public payload (defence in depth: check the
	// whole serialized body, not just known fields).
	raw, _ := json.Marshal(body)
	for _, leak := range []string{"email", "timezone", "study_budget", "reminders", "password", "linked_providers"} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("public payload leaks %q: %s", leak, raw)
		}
	}

	totals, _ := body["totals"].(map[string]any)
	if totals == nil || totals["solved"].(float64) != 3 {
		t.Fatalf("totals = %v", body["totals"])
	}
	streak, _ := totals["streak"].(map[string]any)
	if streak["current"].(float64) != 5 || streak["longest"].(float64) != 9 {
		t.Fatalf("streak = %v", streak)
	}
	mock, _ := body["mock"].(map[string]any)
	if mock == nil || mock["best"].(float64) != 24 {
		t.Fatalf("mock = %v", body["mock"])
	}

	// Exactly one active course (dsa); the coming-soon System Design path is excluded.
	courses, _ := body["courses"].([]any)
	if len(courses) != 1 {
		t.Fatalf("courses = %d, want 1", len(courses))
	}
	dsa, _ := courses[0].(map[string]any)
	// Core problems are 1,2,9,20 (#99 is reinforcement → excluded); solved 1,2,20 → 3/4 = 75%.
	if dsa["slug"] != "dsa" || dsa["total"].(float64) != 4 || dsa["solved"].(float64) != 3 || dsa["pct"].(float64) != 75 {
		t.Fatalf("dsa course = %v", dsa)
	}

	// The heatmap passes through (already merged/account-wide in assessment).
	if _, ok := body["heatmap"].(map[string]any); !ok {
		t.Fatalf("heatmap missing/null: %v", body["heatmap"])
	}
}

// TestBFFPublicProfileNotFound: an unknown username is a uniform 404 (no existence detail).
func TestBFFPublicProfileNotFound(t *testing.T) {
	h := newAggHarness(t)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, h.gwServer.URL+"/xlearn/api/u/ghost", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown user status %d, want 404", resp.StatusCode)
	}
}
