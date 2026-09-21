package gateway

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// revisionHarness wires a real gateway to fake identity + curriculum + review
// services. The fake review verifies the gateway-minted REVIEW-scoped JWT against the
// gateway's real JWKS (ADR-0006), so these tests exercise the mint→forward→verify path
// for the review audience, plus the Revision-queue enrichment and the score proxy.
type revisionHarness struct {
	gwServer       *httptest.Server
	reviewAuthErr  error
	lastScoreBody  string
	lastScorePath  string
	lastCreateBody string
	lastPatchPath  string
}

func newRevisionHarness(t *testing.T) *revisionHarness {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	signer := auth.NewSigner(key, "xlearn-gateway", 0)

	h := &revisionHarness{}
	var gwJWKSURL string

	identityMux := http.NewServeMux()
	identityMux.HandleFunc("POST /sessions/validate", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SessionID string `json:"session_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.SessionID != "sess-1" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"account_id": "acct-1"})
	})
	identity := httptest.NewServer(identityMux)
	t.Cleanup(identity.Close)

	// Fake curriculum: problem metadata for enrichment (no user JWT).
	titles := map[string]string{"3": "Two Sum", "16": "3Sum", "18": "Minimum Window Substring"}
	curriculumMux := http.NewServeMux()
	curriculumMux.HandleFunc("GET /problems/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		title, ok := titles[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"not_found"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"problem": map[string]any{"id": id, "title": title, "difficulty": "easy", "pattern": "Complement lookup", "week_n": 1},
		})
	})
	curriculum := httptest.NewServer(curriculumMux)
	t.Cleanup(curriculum.Close)

	verify := func(r *http.Request) bool {
		v := auth.NewJWKSVerifier(gwJWKSURL, "review", "xlearn-gateway")
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		claims, err := v.Verify(r.Context(), token)
		h.reviewAuthErr = err
		return err == nil && claims.Subject == "acct-1"
	}
	reviewMux := http.NewServeMux()
	reviewMux.HandleFunc("GET /revisions/due", func(w http.ResponseWriter, r *http.Request) {
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"itemId": "it-1", "problemId": "3", "touchLevel": 1, "dayLabel": "Day 1", "dueDate": "2026-09-20T00:00:00Z", "due": true, "mockMode": false, "status": "pending", "problem": nil},
			},
			"dueCount": 1,
		})
	})
	reviewMux.HandleFunc("POST /revisions/{id}/score", func(w http.ResponseWriter, r *http.Request) {
		h.lastScorePath = r.URL.Path
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		b, _ := io.ReadAll(r.Body)
		h.lastScoreBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"itemId": r.PathValue("id"), "problemId": "3", "touchLevel": 1,
			"autoPass": true, "status": "passed", "mockMode": false, "reset": false,
			"nextTouchLevel": 2, "nextDayLabel": "Day 3", "nextDueDate": "2026-09-24T00:00:00Z",
		})
	})
	// S07: mistake journal (bare-id, gateway enriches with curriculum problem meta).
	reviewMux.HandleFunc("GET /mistakes", func(w http.ResponseWriter, r *http.Request) {
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"mistakes": []any{
				map[string]any{"id": "m1", "problemId": "18", "pattern": "Sliding window", "mistake": "", "rootCause": "", "insight": "", "category": "off_by_one", "status": "open", "revisitCount": 1, "revisitDate": nil, "createdAt": "2026-09-21T00:00:00Z"},
			},
			"openCount": 1, "closedCount": 0, "closeThreshold": 2, "categories": []any{"off_by_one"},
		})
	})
	reviewMux.HandleFunc("POST /mistakes", func(w http.ResponseWriter, r *http.Request) {
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		b, _ := io.ReadAll(r.Body)
		h.lastCreateBody = string(b)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "new", "problemId": "18", "status": "open"})
	})
	reviewMux.HandleFunc("PATCH /mistakes/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.lastPatchPath = r.URL.Path
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": r.PathValue("id"), "category": "off_by_one", "status": "open"})
	})
	reviewMux.HandleFunc("GET /weak-area/current", func(w http.ResponseWriter, r *http.Request) {
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"weekOf": "2026-09-21", "topCategory": "off_by_one", "topCount": 1,
			"counts":  map[string]any{"off_by_one": 1},
			"entries": []any{map[string]any{"id": "m1", "problemId": "18", "category": "off_by_one", "status": "open"}},
		})
	})
	reviewMux.HandleFunc("GET /reminders", func(w http.ResponseWriter, r *http.Request) {
		if !verify(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"reminders": []any{map[string]any{"id": "r1", "kind": "revision_due", "dueAt": "2026-09-21T00:00:00Z"}},
		})
	})
	review := httptest.NewServer(reviewMux)
	t.Cleanup(review.Close)

	dist := fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>xLearn</title>")}}
	gw := New(Options{
		BasePath:          "/xlearn",
		Version:           "test",
		Dist:              dist,
		Signer:            signer,
		IdentityBaseURL:   identity.URL,
		AudienceIdentity:  "identity",
		CurriculumBaseURL: curriculum.URL,
		ReviewBaseURL:     review.URL,
		AudienceReview:    "review",
	})
	h.gwServer = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gwServer.Close)
	gwJWKSURL = h.gwServer.URL + "/.well-known/jwks.json"
	return h
}

func (h *revisionHarness) do(t *testing.T, method, path, body string, cookie bool) *http.Response {
	t.Helper()
	var r *http.Request
	if body == "" {
		r, _ = http.NewRequestWithContext(context.Background(), method, h.gwServer.URL+path, nil)
	} else {
		r, _ = http.NewRequestWithContext(context.Background(), method, h.gwServer.URL+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if cookie {
		r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	}
	resp, err := noRedirect().Do(r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var m map[string]any
	if b, _ := io.ReadAll(resp.Body); len(b) > 0 {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

func TestBFFRevisionDueEnrichesWithCurriculum(t *testing.T) {
	h := newRevisionHarness(t)
	resp := h.do(t, http.MethodGet, "/xlearn/api/revision/due", "", true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decode(t, resp)
	if h.reviewAuthErr != nil {
		t.Fatalf("review rejected the minted review-aud JWT: %v", h.reviewAuthErr)
	}
	if dc, _ := body["dueCount"].(float64); dc != 1 {
		t.Fatalf("dueCount = %v, want 1", body["dueCount"])
	}
	items, _ := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	first, _ := items[0].(map[string]any)
	prob, _ := first["problem"].(map[string]any)
	if prob == nil || prob["title"] != "Two Sum" {
		t.Fatalf("item not enriched with curriculum metadata: %v", first["problem"])
	}
}

func TestBFFRevisionScoreProxies(t *testing.T) {
	h := newRevisionHarness(t)
	resp := h.do(t, http.MethodPost, "/xlearn/api/revision/it-1/score", `{"namedPatternSecs":45,"solvedInTimer":true,"statedComplexity":true}`, true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decode(t, resp)
	if h.reviewAuthErr != nil {
		t.Fatalf("review rejected the minted review-aud JWT: %v", h.reviewAuthErr)
	}
	if body["autoPass"] != true || body["nextDayLabel"] != "Day 3" {
		t.Fatalf("score result = %v", body)
	}
	if h.lastScorePath != "/revisions/it-1/score" {
		t.Fatalf("proxied path = %q, want /revisions/it-1/score", h.lastScorePath)
	}
	if !strings.Contains(h.lastScoreBody, `"namedPatternSecs":45`) {
		t.Fatalf("score body not forwarded: %q", h.lastScoreBody)
	}
}

func TestBFFRevisionDueRequiresSession(t *testing.T) {
	h := newRevisionHarness(t)
	resp := h.do(t, http.MethodGet, "/xlearn/api/revision/due", "", false)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
