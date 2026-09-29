package gateway

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// m1-04: the T-3 cohort bit comes from session-validate's role (never a JWT claim), is
// read fresh on every request, and every gateway-minted JWT stays ["learner"].

// cohortHarness is a gateway (test course registry: the embedded courses + the zz-*
// fixtures) in front of a fake identity whose session → role map the test changes between
// requests, a fake curriculum and a fake review. Every upstream records the bearer tokens
// it receives.
type cohortHarness struct {
	t        *testing.T
	gw       *httptest.Server
	mu       sync.Mutex
	roles    map[string]string // session id → role identity reports ("" = a v1.6.0-shaped answer)
	validate int               // POST /sessions/validate calls
	tokens   []string          // every bearer an upstream saw
	started  []string          // slugs identity's POST /paths/{slug}/start saw
}

func (h *cohortHarness) setRole(sess, role string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.roles[sess] = role
}

func (h *cohortHarness) bearer(r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); tok != "" {
		h.tokens = append(h.tokens, tok)
	}
}

func newCohortHarness(t *testing.T) *cohortHarness {
	t.Helper()
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	h := &cohortHarness{t: t, roles: map[string]string{
		"sess-owner": "owner", "sess-tester": "tester", "sess-learner": "learner", "sess-v1": "",
	}}
	echo := func(w http.ResponseWriter, r *http.Request) {
		h.bearer(r)
		_ = json.NewEncoder(w).Encode(map[string]any{"path": r.URL.Path, "course": r.URL.Query().Get("path")})
	}
	identity := httptest.NewServer(jsonMux(map[string]handlerFn{
		"POST /sessions/validate": func(w http.ResponseWriter, r *http.Request) {
			var b struct {
				SessionID string `json:"session_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			h.mu.Lock()
			h.validate++
			role, ok := h.roles[b.SessionID]
			h.mu.Unlock()
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
				return
			}
			if role == "" { // v1.6.0 identity: account_id only
				_ = json.NewEncoder(w).Encode(map[string]any{"account_id": "acct-" + b.SessionID})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"account_id": "acct-" + b.SessionID, "role": role, "status": "active",
				"accepted": false, "created_at": "2026-09-29T05:00:00Z",
			})
		},
		"GET /accounts/{id}": func(w http.ResponseWriter, r *http.Request) {
			h.bearer(r)
			_ = json.NewEncoder(w).Encode(map[string]any{"account": map[string]any{"id": r.PathValue("id")}})
		},
		"POST /paths/{slug}/start": func(w http.ResponseWriter, r *http.Request) {
			h.bearer(r)
			h.mu.Lock()
			h.started = append(h.started, r.PathValue("slug"))
			h.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"enrollment": map[string]any{"path_slug": r.PathValue("slug")}})
		},
	}))
	t.Cleanup(identity.Close)
	curriculum := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /paths": func(w http.ResponseWriter, r *http.Request) {
			var ps []any
			for _, s := range []string{course.DefaultSlug, coursetest.FixtureActive, coursetest.FixtureComingSoon, coursetest.FixturePreview, coursetest.FixtureRetired} {
				ps = append(ps, map[string]any{"slug": s})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"paths": ps})
		},
		"GET /paths/{slug}": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"path": map[string]any{"slug": r.PathValue("slug")}})
		},
	}))
	t.Cleanup(curriculum.Close)
	review := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /mistakes":  echo,
		"POST /mistakes": echo,
		"GET /weak-area/current": func(w http.ResponseWriter, r *http.Request) {
			h.bearer(r)
			_ = json.NewEncoder(w).Encode(map[string]any{"weak_area": nil})
		},
	}))
	t.Cleanup(review.Close)

	gw := New(Options{
		BasePath: "/xlearn", Version: "test", Signer: auth.NewSigner(key, "xlearn-gateway", 0),
		Dist:            fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}},
		IdentityBaseURL: identity.URL, AudienceIdentity: "identity",
		CurriculumBaseURL: curriculum.URL,
		ReviewBaseURL:     review.URL, AudienceReview: "review",
		Courses: coursetest.Registry(t),
	})
	h.gw = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gw.Close)
	return h
}

func (h *cohortHarness) do(sess, method, path string) (int, []byte) {
	h.t.Helper()
	var body io.Reader
	if method != http.MethodGet {
		body = strings.NewReader(`{}`)
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, h.gw.URL+"/xlearn/api/v1"+path, body)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sess})
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func catalogSlugs(t *testing.T, body []byte) []string {
	t.Helper()
	var env struct {
		Paths []struct {
			Slug string `json:"slug"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("catalog %s: %v", body, err)
	}
	var out []string
	for _, p := range env.Paths {
		out = append(out, p.Slug)
	}
	return out
}

// The preview fixture: the owner and testers see it in the catalog, read it and start it;
// a learner (and a v1.6.0-shaped session answer with no role) gets the uniform 404 and
// never sees it listed.
func TestPreviewVisibleToCohortOnly(t *testing.T) {
	h := newCohortHarness(t)
	preview := coursetest.FixturePreview
	_, notFound := h.do("sess-learner", http.MethodGet, "/paths/no-such-course")

	for _, sess := range []string{"sess-owner", "sess-tester"} {
		if _, body := h.do(sess, http.MethodGet, "/paths"); !slices.Contains(catalogSlugs(t, body), preview) {
			t.Errorf("%s: catalog %s lacks %s", sess, body, preview)
		}
		if code, body := h.do(sess, http.MethodGet, "/paths/"+preview); code != http.StatusOK {
			t.Errorf("%s: GET preview %d %s, want 200", sess, code, body)
		}
		if code, body := h.do(sess, http.MethodGet, "/paths/"+preview+"/mistakes"); code != http.StatusOK {
			t.Errorf("%s: preview mistakes %d %s, want 200", sess, code, body)
		}
		if code, body := h.do(sess, http.MethodPost, "/paths/"+preview+"/mistakes"); code != http.StatusOK {
			t.Errorf("%s: POST preview mistake %d %s, want 200", sess, code, body)
		}
		if code, body := h.do(sess, http.MethodPost, "/paths/"+preview+"/start"); code != http.StatusOK {
			t.Errorf("%s: start preview %d %s, want 200", sess, code, body)
		}
	}
	for _, sess := range []string{"sess-learner", "sess-v1"} {
		if _, body := h.do(sess, http.MethodGet, "/paths"); slices.Contains(catalogSlugs(t, body), preview) {
			t.Errorf("%s: catalog lists the preview course: %s", sess, body)
		}
		for _, c := range []struct{ method, path string }{
			{http.MethodGet, "/paths/" + preview},
			{http.MethodGet, "/paths/" + preview + "/mistakes"},
			{http.MethodPost, "/paths/" + preview + "/mistakes"},
			{http.MethodPost, "/paths/" + preview + "/start"},
		} {
			code, body := h.do(sess, c.method, c.path)
			if code != http.StatusNotFound || string(body) != string(notFound) {
				t.Errorf("%s: %s %s = %d %s, want the uniform 404 %s", sess, c.method, c.path, code, body, notFound)
			}
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if !slices.Equal(h.started, []string{preview, preview}) {
		t.Fatalf("identity saw starts %v, want the two cohort starts only", h.started)
	}
}

// Role and status are never cached: a role change applies on the very next request, and
// every request validates the session with identity once.
func TestCohortReadFreshEveryRequest(t *testing.T) {
	h := newCohortHarness(t)
	preview := "/paths/" + coursetest.FixturePreview
	if code, _ := h.do("sess-learner", http.MethodGet, preview); code != http.StatusNotFound {
		t.Fatalf("learner: %d, want 404", code)
	}
	h.setRole("sess-learner", "tester") // `identity admin account set-role … tester`
	if code, _ := h.do("sess-learner", http.MethodGet, preview); code != http.StatusOK {
		t.Fatalf("after set-role tester: %d, want 200", code)
	}
	h.setRole("sess-learner", "learner")
	if code, _ := h.do("sess-learner", http.MethodGet, preview); code != http.StatusNotFound {
		t.Fatalf("after set-role learner: %d, want 404", code)
	}
	h.mu.Lock()
	delete(h.roles, "sess-learner") // suspended: identity's validate now 401s
	h.mu.Unlock()
	if code, _ := h.do("sess-learner", http.MethodGet, "/paths/dsa"); code != http.StatusUnauthorized {
		t.Fatalf("after suspend: %d, want 401", code)
	}

	h.mu.Lock()
	h.validate = 0
	h.mu.Unlock()
	h.do("sess-owner", http.MethodPost, preview+"/mistakes") // auth + visibility + the proxy's auth
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.validate != 1 {
		t.Fatalf("one request validated the session %d times, want once", h.validate)
	}
}

// Every JWT the gateway mints carries exactly ["learner"] — for the owner too (ADR-0033 §7:
// no owner/tester role is ever minted).
func TestMintedRolesAreLearnerOnly(t *testing.T) {
	if got := mintedRoles(); !slices.Equal(got, []string{auth.RoleLearner}) {
		t.Fatalf("mintedRoles() = %v, want [learner]", got)
	}
	h := newCohortHarness(t)
	for _, sess := range []string{"sess-owner", "sess-tester", "sess-learner"} {
		h.do(sess, http.MethodGet, "/me")
		h.do(sess, http.MethodGet, "/paths/dsa/mistakes")
		h.do(sess, http.MethodPost, "/paths/dsa/start")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.tokens) < 9 {
		t.Fatalf("upstreams saw %d tokens, want ≥ 9", len(h.tokens))
	}
	for _, tok := range h.tokens {
		parts := strings.Split(tok, ".")
		if len(parts) != 3 {
			t.Fatalf("not a JWT: %q", tok)
		}
		raw, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			t.Fatalf("payload: %v", err)
		}
		var c struct {
			Roles []string `json:"roles"`
		}
		_ = json.Unmarshal(raw, &c)
		if !slices.Equal(c.Roles, []string{"learner"}) {
			t.Fatalf("a minted JWT carries roles %v, want exactly [learner]", c.Roles)
		}
	}
}

func TestInCohort(t *testing.T) {
	for role, want := range map[string]bool{"owner": true, "tester": true, "learner": false, "": false, "OWNER": false} {
		if got := inCohort(sessionInfo{AccountID: "a", Role: role}); got != want {
			t.Errorf("inCohort(%q) = %v, want %v", role, got, want)
		}
	}
	// No validated session in this request → not in the cohort.
	g := &Gateway{}
	r := withRequestSession(httptest.NewRequest(http.MethodGet, "/", nil))
	if g.cohort(r) {
		t.Fatal("an unvalidated request is in the cohort")
	}
	rememberSession(r, sessionInfo{AccountID: "a", Role: "owner"})
	if !g.cohort(r) {
		t.Fatal("a validated owner request is not in the cohort")
	}
	if g.cohort(withRequestSession(httptest.NewRequest(http.MethodGet, "/", nil))) {
		t.Fatal("a fresh request inherited the cohort bit")
	}
}
