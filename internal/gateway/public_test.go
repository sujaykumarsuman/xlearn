package gateway

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
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
	// P10: joinedAt is a date, no sub-day timestamp.
	if user["joinedAt"] != "2026-01-01" {
		t.Fatalf("user.joinedAt = %v, want 2026-01-01", user["joinedAt"])
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
	// P1 (D31): the mock tile is the count only; best/average never leave the gateway.
	mock, _ := body["mock"].(map[string]any)
	if mock == nil || mock["count"].(float64) != 2 || len(mock) != 1 {
		t.Fatalf("mock = %v, want {count: 2} only", body["mock"])
	}
	for _, k := range []string{`"best"`, `"average"`} {
		if strings.Contains(string(raw), k) {
			t.Errorf("public payload carries %s: %s", k, raw)
		}
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

// --- the v2 public floor (m1-05: P2, P4, P10, P11) ---

// sentinel marks every upstream value that must never reach the public payload.
const sentinel = "SENTINEL"

// publicHarness is a real gateway (fake clock) over sentinel-planting fakes: identity's
// resolver (with per-user visible courses and a suspend switch), assessment's projections
// and curriculum's taxonomy. Every field the public shape must not carry is planted with
// a sentinel value, an item id, or a sub-day timestamp.
type publicHarness struct {
	t     *testing.T
	clock *fakeClock
	gw    *httptest.Server

	mu        sync.Mutex
	visible   []string // ada's visible_courses
	suspended bool
	resolves  map[string]int

	summaryCalls atomic.Int64
	summaryGate  chan struct{} // when non-nil, /progress/summary blocks until it closes
	summaryIn    chan struct{} // one send per /progress/summary arrival
}

func newPublicHarness(t *testing.T, cacheTTL time.Duration) *publicHarness {
	t.Helper()
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer := auth.NewSigner(key, "xlearn-gateway", 0)
	h := &publicHarness{t: t, clock: newFakeClock(), visible: []string{"dsa"}, resolves: map[string]int{}}

	identity := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /internal/accounts/by-username/{username}": func(w http.ResponseWriter, r *http.Request) {
			h.mu.Lock()
			name := r.PathValue("username")
			h.resolves[name]++
			visible, suspended := append([]string{}, h.visible...), h.suspended
			h.mu.Unlock()
			// identity answers only an active account (P11): a suspended one is the same 404.
			if name != "ada" || suspended {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"resource not found"}}`))
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"account_id": "acct-" + sentinel, "username": "ada", "display_name": "Ada Lovelace",
				"created_at": "2026-01-01T13:45:07Z", "region": "UTC+05:30", "visible_courses": visible,
				// Fields a resolver must never have, planted to prove none pass.
				"email": sentinel + "@example.com", "timezone": sentinel + "/Zone", "status": "active",
				"role": "owner", "profile_visibility": "public", "notes": sentinel,
			})
		},
	}))
	t.Cleanup(identity.Close)

	stamp := "2026-09-20T10:11:12Z"
	assessment := httptest.NewServer(jsonMux(map[string]handlerFn{
		"GET /progress/summary": func(w http.ResponseWriter, r *http.Request) {
			h.summaryCalls.Add(1)
			if h.summaryIn != nil {
				h.summaryIn <- struct{}{}
			}
			if h.summaryGate != nil {
				<-h.summaryGate
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"solved": 2, "total": 151,
				"streak":    map[string]any{"current": 5, "longest": 9, "lastAt": stamp},
				"retention": map[string]any{"pct": 80, "resets": 1, "ladders": 5},
				"mock": map[string]any{"count": 2, "average": 22, "best": 24, "last": 24, "delta": 2,
					"scored_by": sentinel, "transcript": sentinel, "feedback": sentinel, "startedAt": stamp},
				"outcomeMix": map[string]any{"total": 3, "clean": 2},
				"arena":      map[string]any{"runs": 7, "submits": 3},
				"runCount":   7,
				"prose":      sentinel,
			})
		},
		"GET /progress/heatmap": writeJSONFn(map[string]any{
			"days": []any{
				map[string]any{"date": "2026-09-20", "solves": 1, "reviews": 2, "problemIds": []string{"p-" + sentinel}, "at": stamp},
				map[string]any{"date": stamp, "solves": 1, "reviews": 0},
			},
			"window": sentinel,
		}),
		"GET /progress/mastery": writeJSONFn(map[string]any{"problems": []any{
			map[string]any{"problemId": "p-" + sentinel + "-1", "weight": 1.0, "bestRank": 4, "bestOutcome": "clean", "pattern": "Hashing", "lastAt": stamp},
			map[string]any{"problemId": "p-" + sentinel + "-2", "weight": 0.7, "bestRank": 3, "bestOutcome": "rough"},
		}}),
	}))
	t.Cleanup(assessment.Close)

	taxonomy := func(slug string) (handlerFn, handlerFn) {
		roadmap := writeJSONFn(map[string]any{
			"path": map[string]any{"slug": slug, "problem_total": 3, "notes": sentinel},
			"phases": []any{
				map[string]any{"order": 1, "name": "Fundamentals", "theme": "Arrays and hashing", "week_from": 1, "week_to": 3, "prose": sentinel},
			},
			"weeks": []any{map[string]any{"n": 1, "title": sentinel}},
		})
		problems := writeJSONFn(map[string]any{"problems": []any{
			map[string]any{"id": "p-" + sentinel + "-1", "week_n": 1, "title": sentinel, "difficulty": "easy", "pattern": "Hashing", "is_reinforcement": false, "leetcode_url": sentinel},
			map[string]any{"id": "p-" + sentinel + "-2", "week_n": 1, "title": sentinel, "difficulty": "med", "pattern": "Two pointers", "is_reinforcement": false},
			map[string]any{"id": "p-" + sentinel + "-3", "week_n": 2, "title": sentinel, "difficulty": "hard", "pattern": "Hashing", "is_reinforcement": false},
		}})
		return roadmap, problems
	}
	routes := map[string]handlerFn{
		"GET /paths": writeJSONFn(map[string]any{"paths": []any{
			map[string]any{"slug": "dsa", "title": "Data Structures & Algorithms", "status": "active", "prose": sentinel},
			map[string]any{"slug": coursetest.FixtureActive, "title": "Fixture", "status": "active"},
			// Drift: curriculum claims the preview course is active. The gateway's own
			// registry still keeps it off the public profile.
			map[string]any{"slug": coursetest.FixturePreview, "title": "Preview", "status": "active"},
			map[string]any{"slug": coursetest.FixtureComingSoon, "title": "Soon", "status": "coming_soon"},
		}}),
	}
	for _, slug := range []string{"dsa", coursetest.FixtureActive, coursetest.FixturePreview} {
		rm, pr := taxonomy(slug)
		routes["GET /paths/"+slug] = rm
		routes["GET /paths/"+slug+"/problems"] = pr
	}
	curriculum := httptest.NewServer(jsonMux(routes))
	t.Cleanup(curriculum.Close)

	gw := New(Options{
		BasePath: "/xlearn", Version: "test", Signer: signer,
		Dist:            fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}},
		IdentityBaseURL: identity.URL, AudienceIdentity: "identity",
		CurriculumBaseURL: curriculum.URL,
		AssessmentBaseURL: assessment.URL, AudienceAssessment: "assessment",
		Courses:     coursetest.Registry(t),
		AggCacheTTL: cacheTTL,
		Now:         h.clock.Now,
	})
	h.gw = httptest.NewServer(gw.Handler())
	t.Cleanup(h.gw.Close)
	return h
}

func (h *publicHarness) setVisible(slugs ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.visible = slugs
}

func (h *publicHarness) setSuspended(v bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.suspended = v
}

func (h *publicHarness) resolveCount(name string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.resolves[name]
}

// get fetches /api/u/{name} from client IP ip.
func (h *publicHarness) get(name, ip string) (*http.Response, []byte) {
	h.t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, h.gw.URL+"/xlearn/api/v1/u/"+name, nil)
	if ip != "" {
		req.Header.Set("X-Real-Ip", ip)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatalf("GET /u/%s: %v", name, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, b
}

func courseSlugs(t *testing.T, body []byte) []string {
	t.Helper()
	var p struct {
		Courses []struct {
			Slug string `json:"slug"`
		} `json:"courses"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	out := []string{}
	for _, c := range p.Courses {
		out = append(out, c.Slug)
	}
	return out
}

// P2: only enrolled ∩ visible ∩ active courses. identity's visible_courses already applies
// all three; the gateway intersects again with its own active list, so a preview course
// never shows even if identity or curriculum got it wrong.
func TestPublicProfileVisibleCourses(t *testing.T) {
	h := newPublicHarness(t, 0)
	for _, tc := range []struct {
		visible []string
		want    string
	}{
		{[]string{"dsa"}, "dsa"},
		{[]string{}, ""}, // enrolled-but-hidden (or nothing enrolled): no course at all
		{[]string{"dsa", coursetest.FixtureActive}, "dsa," + coursetest.FixtureActive},
		{[]string{coursetest.FixtureActive, coursetest.FixturePreview}, coursetest.FixtureActive},
		{[]string{coursetest.FixtureComingSoon, coursetest.FixtureRetired}, ""},
	} {
		h.setVisible(tc.visible...)
		h.clock.Advance(time.Minute) // keep L4's bucket out of the way
		resp, body := h.get("ada", "203.0.113.1")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("visible %v: %d %s", tc.visible, resp.StatusCode, body)
		}
		if got := strings.Join(courseSlugs(t, body), ","); got != tc.want {
			t.Errorf("visible %v: courses %q, want %q", tc.visible, got, tc.want)
		}
	}
}

// P4/L4: 60/min per IP with burst 20; the 21st request is a typed 429 with Retry-After 1 s.
func TestPublicProfileRateLimit(t *testing.T) {
	h := newPublicHarness(t, time.Hour)
	for i := 0; i < 20; i++ {
		if resp, body := h.get("ada", "203.0.113.7"); resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d: %d %s", i+1, resp.StatusCode, body)
		}
	}
	resp, body := h.get("ada", "203.0.113.7")
	want429(t, resp, body, "rate_limited", 1)
	// Unknown usernames spend the same bucket (a scan is what L4 is for).
	resp, body = h.get("ghost", "203.0.113.7")
	want429(t, resp, body, "rate_limited", 1)
	if resp, _ := h.get("ada", "203.0.113.8"); resp.StatusCode != http.StatusOK {
		t.Fatalf("another IP: %d", resp.StatusCode)
	}
	h.clock.Advance(time.Second)
	if resp, _ := h.get("ada", "203.0.113.7"); resp.StatusCode != http.StatusOK {
		t.Fatalf("after 1 s: %d, want 200", resp.StatusCode)
	}
}

// P4: a 404 is cached for 60 s per normalised username, without calling identity.
func TestPublicProfileNegativeCache(t *testing.T) {
	h := newPublicHarness(t, 0)
	for i, name := range []string{"ghost", "ghost", "GHOST", "ghost"} {
		if resp, body := h.get(name, "203.0.113.9"); resp.StatusCode != http.StatusNotFound || errorCode(body) != "not_found" {
			t.Fatalf("request %d: %d %s, want 404 not_found", i+1, resp.StatusCode, body)
		}
	}
	if n := h.resolveCount("ghost"); n != 1 {
		t.Fatalf("identity resolved ghost %d times, want 1 (then the negative cache)", n)
	}
	h.clock.Advance(59 * time.Second)
	h.get("ghost", "203.0.113.9")
	if n := h.resolveCount("ghost"); n != 1 {
		t.Fatalf("identity resolved ghost %d times inside 60 s, want 1", n)
	}
	h.clock.Advance(time.Second)
	h.get("ghost", "203.0.113.9")
	if n := h.resolveCount("ghost"); n != 2 {
		t.Fatalf("identity resolved ghost %d times after 60 s, want 2", n)
	}
}

// P4: at most 8 cold composes run at once; the 9th is a typed 429 busy with Retry-After 1.
func TestPublicProfileComposeSemaphore(t *testing.T) {
	h := newPublicHarness(t, 0) // no composed cache: every request is a cold compose
	h.summaryGate = make(chan struct{})
	h.summaryIn = make(chan struct{}, 16)

	var wg sync.WaitGroup
	codes := make([]int, publicMaxComposes)
	for i := 0; i < publicMaxComposes; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, _ := h.get("ada", "203.0.113.10")
			codes[i] = resp.StatusCode
		}(i)
	}
	for i := 0; i < publicMaxComposes; i++ {
		select {
		case <-h.summaryIn: // all eight are inside a compose, holding a slot
		case <-time.After(10 * time.Second):
			t.Fatalf("only %d composes started", i)
		}
	}
	resp, body := h.get("ada", "203.0.113.11")
	want429(t, resp, body, "busy", 1)
	if resp.Header.Get("Retry-After") != "1" {
		t.Fatalf("Retry-After %q, want 1", resp.Header.Get("Retry-After"))
	}
	close(h.summaryGate)
	wg.Wait()
	for i, c := range codes {
		if c != http.StatusOK {
			t.Fatalf("compose %d: %d, want 200", i+1, c)
		}
	}
	// The slots are free again.
	if resp, body := h.get("ada", "203.0.113.11"); resp.StatusCode != http.StatusOK {
		t.Fatalf("after the composes: %d %s", resp.StatusCode, body)
	}
}

// P11: a suspended account 404s on the very next request, even with its composed payload
// warm in the cache (the gateway never caches a positive resolve); reactivated, it is
// visible again once the 60 s negative-cache entry expires.
func TestPublicProfileSuspendedIs404AtOnce(t *testing.T) {
	h := newPublicHarness(t, time.Hour)
	if resp, body := h.get("ada", "203.0.113.12"); resp.StatusCode != http.StatusOK {
		t.Fatalf("cold: %d %s", resp.StatusCode, body)
	}
	if resp, _ := h.get("ada", "203.0.113.12"); resp.StatusCode != http.StatusOK {
		t.Fatalf("warm: %d", resp.StatusCode)
	}
	if n := h.summaryCalls.Load(); n != 1 {
		t.Fatalf("assessment summary called %d times, want 1 (the second hit is warm)", n)
	}

	h.setSuspended(true)
	_, unknown := h.get("nobody-here", "203.0.113.12")
	resp, body := h.get("ada", "203.0.113.12")
	if resp.StatusCode != http.StatusNotFound || string(body) != string(unknown) {
		t.Fatalf("suspended: %d %s, want the unknown username's 404 %s", resp.StatusCode, body, unknown)
	}

	h.setSuspended(false)
	if resp, _ := h.get("ada", "203.0.113.12"); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("reactivated inside 60 s: %d, want the cached 404", resp.StatusCode)
	}
	h.clock.Advance(publicNegativeTTL)
	if resp, body := h.get("ada", "203.0.113.12"); resp.StatusCode != http.StatusOK {
		t.Fatalf("reactivated after 60 s: %d %s", resp.StatusCode, body)
	}
}

// The negative cache is bounded (memory-sum rule): past publicNegativeMax it starts over.
func TestNegativeCacheBounded(t *testing.T) {
	clk := newFakeClock()
	c := newNegativeCache(publicNegativeTTL, clk.Now)
	for i := 0; i < publicNegativeMax+10; i++ {
		c.add("user-" + strconv.Itoa(i))
		if c.len() > publicNegativeMax {
			t.Fatalf("negative cache holds %d entries, cap %d", c.len(), publicNegativeMax)
		}
	}
	if !c.has("user-" + strconv.Itoa(publicNegativeMax+9)) {
		t.Fatal("the newest entry was dropped")
	}
	clk.Advance(publicNegativeTTL)
	if c.has("user-" + strconv.Itoa(publicNegativeMax+9)) {
		t.Fatal("an entry outlived its TTL")
	}
}

// publicShapeAllowlist is every key path the public payload may carry (P10). "[]" is an
// array element. A new field needs a reviewed edit here — that is the point.
var publicShapeAllowlist = []string{
	"user", "user.username", "user.displayName", "user.joinedAt", "user.region",
	"totals", "totals.solved", "totals.streak", "totals.streak.current", "totals.streak.longest",
	"mock", "mock.count",
	"heatmap", "heatmap.days", "heatmap.days[]", "heatmap.days[].date", "heatmap.days[].solves", "heatmap.days[].reviews",
	"courses", "courses[]", "courses[].slug", "courses[].title", "courses[].solved", "courses[].total", "courses[].pct",
	"courses[].phases", "courses[].phases[]",
	"courses[].phases[].order", "courses[].phases[].name", "courses[].phases[].theme",
	"courses[].phases[].weekFrom", "courses[].phases[].weekTo",
	"courses[].phases[].solved", "courses[].phases[].total",
	"courses[].phases[].clean", "courses[].phases[].rough", "courses[].phases[].assisted", "courses[].phases[].miss",
	"courses[].patterns", "courses[].patterns[]",
	"courses[].patterns[].name", "courses[].patterns[].solved", "courses[].patterns[].total", "courses[].patterns[].pct",
}

// walkPublic records every key path of v and every string value.
func walkPublic(prefix string, v any, paths map[string]bool, strs *[]string) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			paths[p] = true
			walkPublic(p, child, paths, strs)
		}
	case []any:
		for _, child := range x {
			paths[prefix+"[]"] = true
			walkPublic(prefix+"[]", child, paths, strs)
		}
	case string:
		*strs = append(*strs, x)
	}
}

// P10: the public payload's shape is exactly the allowlist, whatever the upstreams add,
// and its values carry no item ids, no sub-day timestamps and nothing planted upstream.
func TestPublicProfileShapeAllowlist(t *testing.T) {
	h := newPublicHarness(t, 0)
	h.setVisible("dsa", coursetest.FixtureActive, coursetest.FixturePreview)
	resp, body := h.get("ada", "203.0.113.13")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d %s", resp.StatusCode, body)
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	var strs []string
	walkPublic("", payload, paths, &strs)

	allowed := map[string]bool{}
	for _, p := range publicShapeAllowlist {
		allowed[p] = true
	}
	var extra, missing []string
	for p := range paths {
		if !allowed[p] {
			extra = append(extra, p)
		}
	}
	for p := range allowed {
		if !paths[p] {
			missing = append(missing, p)
		}
	}
	sort.Strings(extra)
	sort.Strings(missing)
	if len(extra) > 0 {
		t.Errorf("key paths outside the allowlist: %v", extra)
	}
	// The fixture fills every allowed path, so the test can't pass vacuously.
	if len(missing) > 0 {
		t.Errorf("allowlisted paths the fixture didn't exercise: %v", missing)
	}

	// Named never-public keys, anywhere in the tree (belt and braces over the allowlist).
	for p := range paths {
		leaf := p[strings.LastIndex(p, ".")+1:]
		leaf = strings.TrimSuffix(leaf, "[]")
		switch leaf {
		case "id", "problemId", "problemIds", "items", "pattern", "best", "average", "scored_by", "transcript",
			"notes", "feedback", "prose", "arena", "runs", "runCount", "email", "timezone", "status", "role":
			t.Errorf("never-public key %q at %s", leaf, p)
		}
	}
	subDay := regexp.MustCompile(`T\d{2}:\d{2}`)
	for _, s := range strs {
		if strings.Contains(s, sentinel) {
			t.Errorf("a planted upstream value reached the public payload: %q", s)
		}
		if subDay.MatchString(s) {
			t.Errorf("sub-day timestamp in the public payload: %q", s)
		}
	}
	var p publicProfile
	_ = json.Unmarshal(body, &p)
	if p.User.JoinedAt != "2026-01-01" {
		t.Errorf("joinedAt = %q, want 2026-01-01", p.User.JoinedAt)
	}
	if p.Heatmap == nil || len(p.Heatmap.Days) != 2 || p.Heatmap.Days[1].Date != "2026-09-20" {
		t.Errorf("heatmap = %+v, want both days with plain dates", p.Heatmap)
	}
	if got := strings.Join(courseSlugs(t, body), ","); got != "dsa,"+coursetest.FixtureActive {
		t.Errorf("courses %q: the preview course leaked", got)
	}
}
