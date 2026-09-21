package gateway

import (
	"net/http"
	"strings"
	"testing"
)

func TestBFFMistakesEnrichesWithCurriculum(t *testing.T) {
	h := newRevisionHarness(t)
	resp := h.do(t, http.MethodGet, "/xlearn/api/mistakes", "", true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decode(t, resp)
	if h.reviewAuthErr != nil {
		t.Fatalf("review rejected the minted review-aud JWT: %v", h.reviewAuthErr)
	}
	if oc, _ := body["openCount"].(float64); oc != 1 {
		t.Fatalf("openCount = %v, want 1", body["openCount"])
	}
	items, _ := body["mistakes"].([]any)
	if len(items) != 1 {
		t.Fatalf("mistakes = %d, want 1", len(items))
	}
	first, _ := items[0].(map[string]any)
	prob, _ := first["problem"].(map[string]any)
	if prob == nil || prob["title"] != "Minimum Window Substring" {
		t.Fatalf("mistake not enriched with curriculum metadata: %v", first["problem"])
	}
}

func TestBFFMistakesStatusFilterForwarded(t *testing.T) {
	h := newRevisionHarness(t)
	// The status query must reach review (the fake ignores it, but the path/query
	// forwarding is what we assert doesn't 500).
	resp := h.do(t, http.MethodGet, "/xlearn/api/mistakes?status=open", "", true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestBFFMistakeCreateProxies(t *testing.T) {
	h := newRevisionHarness(t)
	resp := h.do(t, http.MethodPost, "/xlearn/api/mistakes", `{"problemId":"18","category":"off_by_one"}`, true)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if !strings.Contains(h.lastCreateBody, `"problemId":"18"`) {
		t.Fatalf("create body not forwarded: %q", h.lastCreateBody)
	}
}

func TestBFFMistakePatchProxies(t *testing.T) {
	h := newRevisionHarness(t)
	resp := h.do(t, http.MethodPatch, "/xlearn/api/mistakes/m1", `{"category":"off_by_one"}`, true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if h.lastPatchPath != "/mistakes/m1" {
		t.Fatalf("proxied patch path = %q, want /mistakes/m1", h.lastPatchPath)
	}
}

func TestBFFWeakAreaEnriches(t *testing.T) {
	h := newRevisionHarness(t)
	resp := h.do(t, http.MethodGet, "/xlearn/api/weak-area", "", true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decode(t, resp)
	if body["topCategory"] != "off_by_one" {
		t.Fatalf("topCategory = %v, want off_by_one", body["topCategory"])
	}
	entries, _ := body["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	first, _ := entries[0].(map[string]any)
	prob, _ := first["problem"].(map[string]any)
	if prob == nil || prob["title"] != "Minimum Window Substring" {
		t.Fatalf("weak-area entry not enriched: %v", first["problem"])
	}
}

func TestBFFDashboardComposes(t *testing.T) {
	h := newRevisionHarness(t)
	resp := h.do(t, http.MethodGet, "/xlearn/api/dashboard", "", true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decode(t, resp)

	// revisions: the enriched due queue.
	revisions, _ := body["revisions"].(map[string]any)
	if revisions == nil {
		t.Fatalf("dashboard missing revisions: %v", body)
	}
	items, _ := revisions["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("revisions.items = %d, want 1", len(items))
	}

	// reminders: the array from review.
	reminders, _ := body["reminders"].([]any)
	if len(reminders) != 1 {
		t.Fatalf("reminders = %d, want 1", len(reminders))
	}

	// weakArea: enriched entries.
	weakArea, _ := body["weakArea"].(map[string]any)
	if weakArea == nil || weakArea["topCategory"] != "off_by_one" {
		t.Fatalf("dashboard weakArea = %v", body["weakArea"])
	}
	entries, _ := weakArea["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("weakArea.entries = %d, want 1", len(entries))
	}
	first, _ := entries[0].(map[string]any)
	if prob, _ := first["problem"].(map[string]any); prob == nil || prob["title"] != "Minimum Window Substring" {
		t.Fatalf("dashboard weak-area entry not enriched: %v", first["problem"])
	}
}

func TestBFFMistakesRequiresSession(t *testing.T) {
	h := newRevisionHarness(t)
	for _, path := range []string{"/xlearn/api/mistakes", "/xlearn/api/weak-area", "/xlearn/api/dashboard"} {
		resp := h.do(t, http.MethodGet, path, "", false)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("GET %s status = %d, want 401", path, resp.StatusCode)
		}
	}
}
