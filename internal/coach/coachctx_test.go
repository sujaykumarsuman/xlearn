package coach

import (
	"net/http"
	"net/url"
	"slices"
	"sort"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// chatOn sends one chat turn on a page context (with an optional ?path=) and fails the
// test unless the reply streamed to done. It first lets one L18 token refill (the
// harness clock only moves when told to), so a table of chats never trips the 20-a-minute
// bucket that limits_test.go exercises on purpose.
func (h *harness) chatOn(t *testing.T, pageContext, pathParam string) {
	t.Helper()
	h.clock.Advance(rateRefillEvery)
	target := "/chat"
	if pathParam != "" {
		target += "?path=" + url.QueryEscape(pathParam)
	}
	resp := h.do(t, http.MethodPost, target, map[string]any{"context": pageContext, "message": "hi"}, nil)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("chat on %q: status %d", pageContext, resp.StatusCode)
	}
	if res := readSSE(t, resp); !res.done {
		t.Fatalf("chat on %q: %+v", pageContext, res)
	}
}

// threadMessages reads GET /threads?context= and returns the echoed context and the
// message count.
func (h *harness) threadMessages(t *testing.T, pageContext string) (string, int) {
	t.Helper()
	resp := h.do(t, http.MethodGet, "/threads?context="+url.QueryEscape(pageContext), nil, nil)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("thread %q: status %d", pageContext, resp.StatusCode)
	}
	var body struct {
		Context  string           `json:"context"`
		Messages []map[string]any `json:"messages"`
	}
	decode(t, resp, &body)
	return body.Context, len(body.Messages)
}

// Every course-scoped context is keyed `<course>:<ctx>` (sprint m1-03, t0 §7), and the
// handlers run the shared dual parser end to end: each v1 form an open v1.6.0 tab sends
// lands on its DSA key, a new form is its own key, problem:<id> and the account-wide
// contexts keep theirs, and each thread carries its course in path_slug.
func TestChatContextKeysAndPathSlug(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")

	cases := []struct {
		context, path     string // the body's context and the gateway's ?path=
		wantKey, wantPath string // the thread key and its path_slug ("" = NULL)
	}{
		// v1 forms → the DSA course's key.
		{"concept:two-pointers", "", "dsa:concept:two-pointers", course.DefaultSlug},
		{"week:3", "", "dsa:week:3", course.DefaultSlug},
		{"roadmap", "", "dsa:roadmap", course.DefaultSlug},
		{"dashboard", "", "dsa:dashboard", course.DefaultSlug},
		{"revision", "", "dsa:revision", course.DefaultSlug},
		{"mistakes", "", "dsa:mistakes", course.DefaultSlug},
		{"mock", "", "dsa:mock", course.DefaultSlug},
		{"progress", "", "dsa:progress", course.DefaultSlug},
		// New forms are their own key; the prefix is the course (?path= never overrides it).
		{"dsa:concept:sliding-window", "", "dsa:concept:sliding-window", course.DefaultSlug},
		{"dsa:week:4", coursetest.FixtureActive, "dsa:week:4", course.DefaultSlug},
		{coursetest.FixtureActive + ":concept:loops", "", coursetest.FixtureActive + ":concept:loops", coursetest.FixtureActive},
		{coursetest.FixtureActive + ":revision", "", coursetest.FixtureActive + ":revision", coursetest.FixtureActive},
		// problem:<id> keeps its key; the course is the gateway's ?path=, else the default
		// (a v1.6.0 gateway sends none; an unknown value is only logged).
		{"problem:16", "", "problem:16", course.DefaultSlug},
		{"problem:zz-001", coursetest.FixtureActive, "problem:zz-001", coursetest.FixtureActive},
		{"problem:17", "no-such-course", "problem:17", course.DefaultSlug},
		// Account-wide: unchanged, no course, whatever ?path= says.
		{"catalog", "", "catalog", ""},
		{"settings", coursetest.FixtureActive, "settings", ""},
		{"general", "", "general", ""},
		// Unrecognised: passed through, no course.
		{"no-such-course:week:3", "", "no-such-course:week:3", ""},
		{"concept:", "", "concept:", ""},
	}
	var wantKeys []string
	for _, tc := range cases {
		h.chatOn(t, tc.context, tc.path)
		_, path, ok := h.store.thread(h.account, tc.wantKey)
		if !ok {
			t.Fatalf("%q: no thread under %q (threads %v)", tc.context, tc.wantKey, h.store.threadKeys(h.account))
		}
		if path != tc.wantPath {
			t.Fatalf("%q: path_slug %q, want %q", tc.context, path, tc.wantPath)
		}
		if n := len(h.store.messagesFor(h.account, tc.wantKey)); n != 2 {
			t.Fatalf("%q: %d messages, want the user turn + the reply", tc.context, n)
		}
		// The thread read normalizes the same way and echoes the context as sent.
		if echoed, n := h.threadMessages(t, tc.context); echoed != tc.context || n != 2 {
			t.Fatalf("GET /threads?context=%s: context %q, %d messages", tc.context, echoed, n)
		}
		wantKeys = append(wantKeys, tc.wantKey)
	}
	// No thread was keyed by a raw v1 form.
	sort.Strings(wantKeys)
	if got := h.store.threadKeys(h.account); !slices.Equal(got, wantKeys) {
		t.Fatalf("thread keys %v, want %v", got, wantKeys)
	}
}

// A v1 form from an open v1.6.0 tab and its `dsa:` form are one thread, read and
// written from either side.
func TestV1ContextSharesTheDSAThread(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")

	h.chatOn(t, "week:3", "")
	h.chatOn(t, "dsa:week:3", "")
	for _, ctx := range []string{"week:3", "dsa:week:3"} {
		if _, n := h.threadMessages(t, ctx); n != 4 {
			t.Fatalf("GET /threads?context=%s: %d messages, want both chats' 4", ctx, n)
		}
	}
	if keys := h.store.threadKeys(h.account); !slices.Equal(keys, []string{"dsa:week:3"}) {
		t.Fatalf("thread keys %v, want one DSA week-3 thread", keys)
	}
}

// Two courses' week 3 are two threads (the reason for the prefix): the DSA course's and
// the fixture course's, each with its own path_slug and history.
func TestTwoCoursesWeek3AreSeparateThreads(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")

	fixtureWeek3 := coursetest.FixtureActive + ":week:3"
	h.chatOn(t, "dsa:week:3", "")
	h.chatOn(t, "dsa:week:3", "")
	h.chatOn(t, fixtureWeek3, "")

	dsaID, dsaPath, ok1 := h.store.thread(h.account, "dsa:week:3")
	fxID, fxPath, ok2 := h.store.thread(h.account, fixtureWeek3)
	if !ok1 || !ok2 || dsaID == fxID {
		t.Fatalf("threads dsa=%q (%v) fixture=%q (%v), want two", dsaID, ok1, fxID, ok2)
	}
	if dsaPath != course.DefaultSlug || fxPath != coursetest.FixtureActive {
		t.Fatalf("path_slug dsa=%q fixture=%q", dsaPath, fxPath)
	}
	if _, n := h.threadMessages(t, "dsa:week:3"); n != 4 {
		t.Fatalf("DSA week 3: %d messages, want 4", n)
	}
	if _, n := h.threadMessages(t, fixtureWeek3); n != 2 {
		t.Fatalf("fixture week 3: %d messages, want 2", n)
	}
}
