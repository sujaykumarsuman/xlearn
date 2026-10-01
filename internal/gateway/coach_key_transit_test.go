package gateway

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
)

// The gateway TRANSITS a coach key: it forwards the PUT body to coach, which seals it, and
// the raw key exists in the gateway only as bytes in flight. ADR-0007 asserts this in
// prose; m1-10 makes it test-backed, because it is the one claim in the coach design that
// a careless change could falsify invisibly — adding a debug log of a request body, or
// echoing an upstream error, would leak a learner's provider key into the gateway's logs
// without breaking any other test.
//
// The test puts a SENTINEL key through PUT /api/coach/key and then asserts the sentinel:
//
//	appears in what the fake coach received          (it really is forwarded)
//	appears in NO log line, at DEBUG, access log included
//	appears in NO response body, on success or error
//	appears in NO error path when coach is down      (502, and nothing echoed)
//
// It also covers GET /api/coach/models as a plain passthrough (m1-10's one new route).

// sentinelKey is distinctive enough that a substring search for it cannot false-negative,
// and is shaped like a real provider key so a redactor keyed on the prefix would be
// exercised too.
const sentinelKey = "sk-ant-api03-SENTINELdoNOTlogTHIS-0a1b2c3d4e5f"

// coachTransitHarness wires a real gateway to a fake identity and a fake coach, capturing
// every log record the gateway emits at DEBUG and above.
type coachTransitHarness struct {
	gw       *httptest.Server
	logs     *syncBuffer
	coachSrv *httptest.Server

	mu sync.Mutex
	// what the fake coach received
	gotBody   string
	gotAuth   string
	gotPath   string
	gotMethod string

	// coachHandler lets a test replace the fake coach's behaviour.
	coachHandler http.HandlerFunc
}

// syncBuffer is an io.Writer safe for the gateway's concurrent logging.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func newCoachTransitHarness(t *testing.T) *coachTransitHarness {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	signer := auth.NewSigner(key, "xlearn-gateway", 0)

	h := &coachTransitHarness{logs: &syncBuffer{}}

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

	h.coachSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		h.mu.Lock()
		h.gotBody, h.gotAuth = string(body), r.Header.Get("Authorization")
		h.gotPath, h.gotMethod = r.URL.Path, r.Method
		handler := h.coachHandler
		h.mu.Unlock()
		if handler != nil {
			handler(w, r)
			return
		}
		// The masked view coach really returns — never the raw key.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[{"provider":"anthropic","masked_key":"sk-...4e5f","default_model":"claude-sonnet-5","name":"n","enabled":true,"is_default":true}],"connected":true,"default_provider":"anthropic","defaults":{"coach":{"provider":"anthropic","model":"claude-sonnet-5"},"interview":null},"usage_month":{"messages":0,"input_tokens":0,"output_tokens":0,"est_cost_micros":0,"has_unknown_cost":false}}`))
	}))
	t.Cleanup(h.coachSrv.Close)

	// DEBUG so the test sees everything the gateway could possibly log, not just INFO.
	logger := slog.New(slog.NewJSONHandler(h.logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	gw := New(Options{
		BasePath:         "/xlearn",
		Version:          "test",
		Dist:             fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}},
		Signer:           signer,
		Logger:           logger,
		IdentityBaseURL:  identity.URL,
		AudienceIdentity: "identity",
		CoachBaseURL:     h.coachSrv.URL,
		AudienceCoach:    "coach",
	})
	// The real middleware chain, so the ACCESS LOG is in scope too — it is the most
	// likely accidental leak (it sees every request).
	handler := httpx.Chain(gw.Handler(), httpx.RequestID, httpx.AccessLog(logger), httpx.Recoverer(logger))
	h.gw = httptest.NewServer(handler)
	t.Cleanup(h.gw.Close)
	return h
}

// putKey sends PUT /api/coach/key with the sentinel key.
func (h *coachTransitHarness) putKey(t *testing.T) *http.Response {
	t.Helper()
	body := `{"provider":"anthropic","key":"` + sentinelKey + `","default_model":"claude-sonnet-5"}`
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut,
		h.gw.URL+"/xlearn/api/coach/key", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	req.Header.Set("Content-Type", "application/json")
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("PUT /coach/key: %v", err)
	}
	return resp
}

func (h *coachTransitHarness) received() (method, path, body, authz string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.gotMethod, h.gotPath, h.gotBody, h.gotAuth
}

// TestCoachKeyPutIsTransitedNotLogged is the main assertion.
func TestCoachKeyPutIsTransitedNotLogged(t *testing.T) {
	h := newCoachTransitHarness(t)

	resp := h.putKey(t)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	respBody, _ := io.ReadAll(resp.Body)

	// 1. TRANSIT: the fake coach really received the raw key, at coach's internal path.
	method, path, gotBody, authz := h.received()
	if method != http.MethodPut || path != "/keys" {
		t.Fatalf("coach saw %s %s, want PUT /keys", method, path)
	}
	if !strings.Contains(gotBody, sentinelKey) {
		t.Fatalf("coach did not receive the key; body = %q", gotBody)
	}
	// Forwarded verbatim: the gateway must not rewrite a body it does not understand.
	if gotBody != `{"provider":"anthropic","key":"`+sentinelKey+`","default_model":"claude-sonnet-5"}` {
		t.Fatalf("body was not forwarded verbatim: %q", gotBody)
	}
	// With a coach-audience JWT, not the session cookie.
	if !strings.HasPrefix(authz, "Bearer ") {
		t.Fatalf("coach got Authorization %q, want a bearer token", authz)
	}

	// 2. NOT IN THE RESPONSE: only the masked view comes back.
	if strings.Contains(string(respBody), sentinelKey) {
		t.Fatalf("the response echoed the raw key: %s", respBody)
	}

	// 3. NOT IN ANY LOG LINE, at DEBUG, access log included.
	assertNoSentinelInLogs(t, h.logs.String())

	// And the JWT itself is not logged either: it is a bearer credential for coach.
	if tok := strings.TrimPrefix(authz, "Bearer "); tok != "" && strings.Contains(h.logs.String(), tok) {
		t.Fatal("the gateway logged the coach JWT it minted")
	}
}

// TestCoachKeyPutLeaksNothingWhenCoachIsDown: the error path is the easiest place to leak
// a secret, by echoing the failed request or the upstream body.
func TestCoachKeyPutLeaksNothingWhenCoachIsDown(t *testing.T) {
	h := newCoachTransitHarness(t)
	h.coachSrv.Close() // nothing is listening any more

	resp := h.putKey(t)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
	if strings.Contains(string(body), sentinelKey) {
		t.Fatalf("the 502 body echoed the key: %s", body)
	}
	// A clean error envelope, not a dump of the failed request.
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil || env.Error.Code == "" {
		t.Fatalf("502 body is not an error envelope: %s", body)
	}
	assertNoSentinelInLogs(t, h.logs.String())
}

// TestCoachKeyPutLeaksNothingWhenCoachRejects: an upstream 4xx must not be relayed with
// the request body attached, and the gateway must not log the exchange to explain it.
func TestCoachKeyPutLeaksNothingWhenCoachRejects(t *testing.T) {
	h := newCoachTransitHarness(t)
	h.mu.Lock()
	h.coachHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_key","message":"key is too long"}}`))
	}
	h.mu.Unlock()

	resp := h.putKey(t)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (coach's status is propagated)", resp.StatusCode)
	}
	if strings.Contains(string(body), sentinelKey) {
		t.Fatalf("the 422 body echoed the key: %s", body)
	}
	assertNoSentinelInLogs(t, h.logs.String())
}

// TestCoachKeyPutUnauthenticatedNeverReachesCoach: no session → 401, and coach is never
// called at all, so an unauthenticated request cannot even be used to probe it.
func TestCoachKeyPutUnauthenticatedNeverReachesCoach(t *testing.T) {
	h := newCoachTransitHarness(t)
	body := `{"provider":"anthropic","key":"` + sentinelKey + `"}`
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut,
		h.gw.URL+"/xlearn/api/coach/key", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if _, _, got, _ := h.received(); got != "" {
		t.Fatalf("coach was called for an unauthenticated PUT: %q", got)
	}
	assertNoSentinelInLogs(t, h.logs.String())
}

// --- GET /api/coach/models (m1-10's new route) ---

func TestCoachModelsPassthrough(t *testing.T) {
	h := newCoachTransitHarness(t)
	const catalog = `{"as_of":"2026-10-01","providers":["anthropic","openai"],"models":[{"id":"claude-sonnet-5","provider":"anthropic","label":"Sonnet 5","capabilities":["chat","interview_brain"],"price":{"input_micros_per_mtok":2000000,"output_micros_per_mtok":10000000},"as_of":"2026-10-01","recommended":true,"covered_model":false}],"defaults":{"anthropic":"claude-sonnet-5","openai":"gpt-5.6-sol"}}`
	h.mu.Lock()
	h.coachHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(catalog))
	}
	h.mu.Unlock()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		h.gw.URL+"/xlearn/api/coach/models", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("GET /coach/models: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	// A passthrough: the catalog arrives byte-for-byte, so the gateway holds no second
	// copy of the model list to go stale.
	if strings.TrimSpace(string(body)) != catalog {
		t.Fatalf("catalog was not passed through verbatim:\n got %s\nwant %s", body, catalog)
	}
	if _, path, _, authz := h.received(); path != "/models" || !strings.HasPrefix(authz, "Bearer ") {
		t.Fatalf("coach saw path %q with auth %q, want /models with a bearer token", path, authz)
	}
}

// TestCoachModelsRequiresASession: the catalog carries no learner data, but keeping it
// session-gated preserves the invariant that the public profile is the ONLY unauthenticated
// route under /api.
func TestCoachModelsRequiresASession(t *testing.T) {
	h := newCoachTransitHarness(t)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		h.gw.URL+"/xlearn/api/coach/models", nil)
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if _, path, _, _ := h.received(); path != "" {
		t.Fatalf("coach was called without a session: %q", path)
	}
}

// TestCoachModelsUnavailableWhenCoachUnset: 503 rather than an empty catalog, which would
// be indistinguishable from "this provider has no models" and would render an empty
// switcher.
func TestCoachModelsUnavailableWhenCoachUnset(t *testing.T) {
	h := newBFFHarness(t) // no CoachBaseURL → g.coach == nil
	resp := h.get(t, "/xlearn/api/coach/models", &http.Cookie{Name: auth.SessionCookieName, Value: "sess-1"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil || env.Error.Code != "unavailable" {
		t.Fatalf("body = %+v (err %v), want an `unavailable` envelope", env, err)
	}
}

// assertNoSentinelInLogs fails with the offending lines when the sentinel key — or a
// recognisable fragment of it — appears anywhere in the captured log output.
func assertNoSentinelInLogs(t *testing.T, logs string) {
	t.Helper()
	// Guard against a vacuous pass: if the handler were not capturing anything, "the key
	// is not in the logs" would be trivially true and this whole file would prove nothing.
	// The access log emits a line per request, so there is always output to search.
	if strings.TrimSpace(logs) == "" {
		t.Fatal("no log output was captured; the leak assertion would be vacuous")
	}
	// Fragments as well as the whole key: a truncating logger that printed the first 20
	// characters would still have leaked the secret.
	for _, probe := range []string{sentinelKey, "SENTINELdoNOTlogTHIS", "0a1b2c3d4e5f", "sk-ant-api03-SENTINEL"} {
		if !strings.Contains(logs, probe) {
			continue
		}
		var hits []string
		for _, line := range strings.Split(logs, "\n") {
			if strings.Contains(line, probe) {
				hits = append(hits, line)
			}
		}
		t.Fatalf("the gateway logged the coach key (fragment %q) in %d line(s):\n%s",
			probe, len(hits), strings.Join(hits, "\n"))
	}
}
