package gateway

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// TestAggCacheServesAndInvalidates proves the per-account BFF cache end-to-end: a
// second Week read within the TTL is served from cache (curriculum is NOT re-hit), and
// a mutating practice write for that account invalidates it so the next read recomputes.
func TestAggCacheServesAndInvalidates(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer := auth.NewSigner(key, "xlearn-gateway", time.Minute)

	identity := httptest.NewServer(jsonMux(map[string]handlerFn{
		"POST /sessions/validate": func(w http.ResponseWriter, r *http.Request) {
			var b struct {
				SessionID string `json:"session_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b.SessionID != "sess-1" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"account_id": "acct-1"})
		},
	}))
	t.Cleanup(identity.Close)

	var weekHits int32
	curriculum := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /paths/dsa/weeks/1": func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt32(&weekHits, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"week":     map[string]any{"n": 1, "title": "Arrays", "thesis": "t"},
				"problems": []any{map[string]any{"id": "1", "difficulty": "easy", "is_reinforcement": false}},
			})
		},
	}))
	t.Cleanup(curriculum.Close)

	practice := httptest.NewServer(jsonMux(map[string]handlerFn{
		// Week state lookup — a populated (non-degraded) response so the aggregation is
		// cacheable (the cache skips a practice-degraded placeholder).
		"GET /state": writeJSONFn(map[string]any{"states": map[string]any{"1": map[string]any{"status": "solved"}}}),
		// The mutating write whose 2xx must invalidate the cache.
		"POST /problems/{id}/outcome": writeJSONFn(map[string]any{"state": map[string]any{"status": "solved"}}),
	}))
	t.Cleanup(practice.Close)

	gw := New(Options{
		BasePath: "/xlearn", Version: "test",
		Dist:            fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}},
		Signer:          signer,
		IdentityBaseURL: identity.URL, AudienceIdentity: "identity",
		CurriculumBaseURL: curriculum.URL,
		PracticeBaseURL:   practice.URL, AudiencePractice: "practice",
		AggCacheTTL: time.Minute, // caching ON
	})
	gwServer := httptest.NewServer(gw.Handler())
	t.Cleanup(gwServer.Close)

	cookie := &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"}
	doGet := func() {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, gwServer.URL+"/xlearn/api/paths/dsa/weeks/1", nil)
		req.AddCookie(cookie)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("week get: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("week get status = %d", resp.StatusCode)
		}
		resp.Body.Close()
	}

	doGet() // miss → curriculum hit #1, cached
	doGet() // hit → served from cache, no curriculum call
	if got := atomic.LoadInt32(&weekHits); got != 1 {
		t.Fatalf("expected 1 curriculum hit after two cached reads, got %d", got)
	}

	// A practice outcome write for acct-1 must invalidate the cache.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, gwServer.URL+"/xlearn/api/problems/1/outcome", nil)
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("outcome write: %v", err)
	}
	resp.Body.Close()

	doGet() // cache was invalidated → curriculum hit #2
	if got := atomic.LoadInt32(&weekHits); got != 2 {
		t.Fatalf("expected curriculum re-fetch after invalidation, got %d hits", got)
	}
}

// TestAggCacheSkipsDegradedWeek proves a practice-degraded week (the honest
// placeholder, populated:false) is NOT cached, so it self-heals on the next read
// instead of being pinned for the TTL.
func TestAggCacheSkipsDegradedWeek(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer := auth.NewSigner(key, "xlearn-gateway", time.Minute)

	identity := httptest.NewServer(jsonMux(map[string]handlerFn{
		"POST /sessions/validate": func(w http.ResponseWriter, r *http.Request) {
			var b struct {
				SessionID string `json:"session_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			_ = json.NewEncoder(w).Encode(map[string]string{"account_id": "acct-1"})
			_ = b
		},
	}))
	t.Cleanup(identity.Close)

	var weekHits int32
	curriculum := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /paths/dsa/weeks/1": func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt32(&weekHits, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"week":     map[string]any{"n": 1, "title": "Arrays"},
				"problems": []any{map[string]any{"id": "1", "difficulty": "easy", "is_reinforcement": false}},
			})
		},
	}))
	t.Cleanup(curriculum.Close)

	// Practice is DOWN for the state lookup → the week degrades to the placeholder.
	practice := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /state": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	}))
	t.Cleanup(practice.Close)

	gw := New(Options{
		BasePath: "/xlearn", Version: "test",
		Dist:            fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}},
		Signer:          signer,
		IdentityBaseURL: identity.URL, AudienceIdentity: "identity",
		CurriculumBaseURL: curriculum.URL,
		PracticeBaseURL:   practice.URL, AudiencePractice: "practice",
		AggCacheTTL: time.Minute,
	})
	gwServer := httptest.NewServer(gw.Handler())
	t.Cleanup(gwServer.Close)

	cookie := &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"}
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, gwServer.URL+"/xlearn/api/paths/dsa/weeks/1", nil)
		req.AddCookie(cookie)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("week get: %v", err)
		}
		resp.Body.Close()
	}
	// Both reads recomputed (degraded responses are never cached).
	if got := atomic.LoadInt32(&weekHits); got != 2 {
		t.Fatalf("degraded week should not be cached: want 2 curriculum hits, got %d", got)
	}
}
