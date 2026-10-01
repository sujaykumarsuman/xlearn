package coach

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// --- harness helpers for L18 ---

// errEnvelope is the typed JSON error body.
type errEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Reason  string `json:"reason"`
	} `json:"error"`
}

// tokens is how many whole tokens the account's bucket holds at the harness clock.
func (h *harness) tokens() int {
	l := h.svc.limits
	l.mu.Lock()
	defer l.mu.Unlock()
	return int(l.availLocked(h.account, l.now()) / rateRefillEvery)
}

// streams is how many chat streams the account holds open.
func (h *harness) streams() int {
	l := h.svc.limits
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.streams[h.account]
}

// today is the harness clock's UTC day.
func (h *harness) today() time.Time { return h.clock.Now() }

// chatOK sends one chat turn and fails unless it streamed to done.
func (h *harness) chatOK(t *testing.T, pageContext string) {
	t.Helper()
	resp := h.do(t, http.MethodPost, "/chat", map[string]any{"context": pageContext, "message": "hi"}, nil)
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("chat: status %d: %s", resp.StatusCode, b)
	}
	if res := readSSE(t, resp); !res.done {
		t.Fatalf("chat did not finish: %+v", res)
	}
}

// refused decodes a refusal: status, typed code, Retry-After.
func refused(t *testing.T, resp *http.Response) (int, errEnvelope, string) {
	t.Helper()
	defer resp.Body.Close()
	var env errEnvelope
	b, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(b, &env); err != nil {
		t.Fatalf("status %d: body is not the typed envelope: %s", resp.StatusCode, b)
	}
	return resp.StatusCode, env, resp.Header.Get("Retry-After")
}

func wantLimit(t *testing.T, resp *http.Response, code, retryAfter string) {
	t.Helper()
	status, env, ra := refused(t, resp)
	if status != http.StatusTooManyRequests || env.Error.Code != code || env.Error.Message == "" {
		t.Fatalf("got %d %+v, want 429 %s with a message", status, env.Error, code)
	}
	if ra != retryAfter {
		t.Fatalf("%s: Retry-After %q, want %q", code, ra, retryAfter)
	}
}

func (h *harness) chat(t *testing.T, pageContext string) *http.Response {
	t.Helper()
	return h.do(t, http.MethodPost, "/chat", map[string]any{"context": pageContext, "message": "hi"}, nil)
}

func (h *harness) admission(t *testing.T) *http.Response {
	t.Helper()
	return h.do(t, http.MethodGet, "/admission", nil, nil)
}

func wantAdmitted(t *testing.T, resp *http.Response) {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("admission: status %d (%s), want 204", resp.StatusCode, b)
	}
}

// --- per minute ---

// The 21st message inside a minute is refused with coach_rate_limited and the seconds to
// the next token; the refusal writes nothing and takes nothing; a refilled token admits
// exactly one more.
func TestL18PerMinuteBucket(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")

	for i := 0; i < rateBurst; i++ {
		h.chatOK(t, "dashboard")
	}
	if h.tokens() != 0 || h.store.dailyCount(h.account, h.today()) != rateBurst {
		t.Fatalf("after %d chats: tokens %d, daily %d", rateBurst, h.tokens(), h.store.dailyCount(h.account, h.today()))
	}
	rows := h.store.messageCount(h.account)

	wantLimit(t, h.chat(t, "dashboard"), codeCoachRateLimited, "3")
	if got := h.store.messageCount(h.account); got != rows {
		t.Fatalf("a refused chat wrote %d coach_message rows", got-rows)
	}
	if h.store.dailyCount(h.account, h.today()) != rateBurst || h.streams() != 0 {
		t.Fatalf("a refused chat consumed: daily %d, streams %d", h.store.dailyCount(h.account, h.today()), h.streams())
	}

	// Two seconds later the next token is one second away.
	h.clock.Advance(2 * time.Second)
	wantLimit(t, h.chat(t, "dashboard"), codeCoachRateLimited, "1")
	// A part-second wait still rounds up to a whole second, never 0.
	h.clock.Advance(999 * time.Millisecond)
	wantLimit(t, h.chat(t, "dashboard"), codeCoachRateLimited, "1")

	// The token is back: one chat, then refused again.
	h.clock.Advance(time.Millisecond)
	h.chatOK(t, "dashboard")
	wantLimit(t, h.chat(t, "dashboard"), codeCoachRateLimited, "3")

	// A full minute refills the bucket to its burst, never beyond.
	h.clock.Advance(10 * time.Minute)
	if h.tokens() != rateBurst {
		t.Fatalf("tokens after a long idle = %d, want the burst %d", h.tokens(), rateBurst)
	}
}

// --- concurrent streams ---

// blockingProvider holds every provider call open until release is closed (or the call is
// cancelled), announcing each arrival on started.
func blockingProvider(started chan<- struct{}, release <-chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read the body to EOF first: only then does net/http watch the connection, so a
		// cancelled coach→provider call cancels r.Context() here and the handler can end.
		_, _ = io.Copy(io.Discard, r.Body)
		started <- struct{}{}
		select {
		case <-release:
			openAISSE(w)
		case <-r.Context().Done():
		}
	}
}

// postChat sends a chat from any goroutine (no t.Fatal), draining the reply.
func (h *harness) postChat(ctx context.Context, pageContext string) (int, error) {
	b, _ := json.Marshal(map[string]any{"context": pageContext, "message": "hi"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.server.URL+"/chat", bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, err
}

// A third concurrent stream is refused with coach_busy / Retry-After: 5 — and the refusal
// takes no token, no daily message and writes no row; the slots come back when the
// streams end, and when a client goes away mid-stream.
func TestL18ConcurrentStreams(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")
	started, release := make(chan struct{}, 4), make(chan struct{})
	h.provHandler = blockingProvider(started, release)

	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for i := 0; i < maxConcurrentStreams; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, err := h.postChat(context.Background(), "dashboard")
			if err != nil {
				t.Errorf("held chat: %v", err)
			}
			statuses <- code
		}()
	}
	for i := 0; i < maxConcurrentStreams; i++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("the held chats never reached the provider")
		}
	}
	if h.streams() != maxConcurrentStreams {
		t.Fatalf("streams = %d, want %d", h.streams(), maxConcurrentStreams)
	}
	tokens, daily, rows := h.tokens(), h.store.dailyCount(h.account, h.today()), h.store.messageCount(h.account)

	wantLimit(t, h.chat(t, "dashboard"), codeCoachBusy, "5")
	wantLimit(t, h.admission(t), codeCoachBusy, "5")
	if h.tokens() != tokens || h.store.dailyCount(h.account, h.today()) != daily || h.store.messageCount(h.account) != rows {
		t.Fatalf("coach_busy consumed: tokens %d→%d, daily %d→%d, rows %d→%d",
			tokens, h.tokens(), daily, h.store.dailyCount(h.account, h.today()), rows, h.store.messageCount(h.account))
	}

	close(release)
	wg.Wait()
	close(statuses)
	for code := range statuses {
		if code != http.StatusOK {
			t.Fatalf("held chat finished with %d", code)
		}
	}
	if h.streams() != 0 {
		t.Fatalf("streams = %d after both ended, want 0 (released by defer)", h.streams())
	}

	// A client that goes away mid-stream gives its slot back too.
	h.provHandler = blockingProvider(started, make(chan struct{})) // never released
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = h.postChat(ctx, "dashboard")
	}()
	<-started
	if h.streams() != 1 {
		t.Fatalf("streams = %d during the chat, want 1", h.streams())
	}
	cancel()
	<-done
	deadline := time.Now().Add(5 * time.Second)
	for h.streams() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("a cancelled chat never released its stream slot")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// --- per UTC day ---

// The 301st message of a UTC day is refused with coach_daily_cap and Retry-After to the
// next UTC midnight; the refusal gives back the token it took and the stream slot, writes
// nothing, and the next UTC day starts a fresh count.
func TestL18DailyCap(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")
	h.clock.Set(time.Date(2026, time.October, 1, 22, 30, 0, 0, time.UTC))
	day1 := h.today()
	h.store.setDailyCount(h.account, day1, dailyMessageCap-1)

	h.chatOK(t, "dashboard") // the 300th
	if n := h.store.dailyCount(h.account, day1); n != dailyMessageCap {
		t.Fatalf("daily = %d after the 300th, want %d", n, dailyMessageCap)
	}
	tokens, rows := h.tokens(), h.store.messageCount(h.account)

	wantLimit(t, h.chat(t, "dashboard"), codeCoachDailyCap, "5400") // 22:30 → 00:00 UTC
	if h.tokens() != tokens {
		t.Fatalf("coach_daily_cap kept the token it took: %d → %d", tokens, h.tokens())
	}
	if h.streams() != 0 || h.store.messageCount(h.account) != rows || h.store.dailyCount(h.account, day1) != dailyMessageCap {
		t.Fatalf("coach_daily_cap consumed: streams %d, rows %d→%d, daily %d",
			h.streams(), rows, h.store.messageCount(h.account), h.store.dailyCount(h.account, day1))
	}
	// The probe agrees and changes nothing.
	wantLimit(t, h.admission(t), codeCoachDailyCap, "5400")
	if h.store.dailyCount(h.account, day1) != dailyMessageCap || h.tokens() != tokens {
		t.Fatal("the admission probe consumed something")
	}

	// One second before midnight the wait is one second; at midnight UTC a new day starts.
	h.clock.Set(time.Date(2026, time.October, 1, 23, 59, 59, 0, time.UTC))
	wantLimit(t, h.chat(t, "dashboard"), codeCoachDailyCap, "1")
	h.clock.Set(time.Date(2026, time.October, 2, 0, 0, 0, 0, time.UTC))
	h.chatOK(t, "dashboard")
	if n := h.store.dailyCount(h.account, h.today()); n != 1 {
		t.Fatalf("the new UTC day's count = %d, want 1", n)
	}
	if n := h.store.dailyCount(h.account, day1); n != dailyMessageCap {
		t.Fatalf("yesterday's count changed: %d", n)
	}
}

// A failing daily-cap read refuses the chat (500) and still gives back the token and the
// stream slot: nothing is consumed and nothing written.
func TestL18DailyCapStoreErrorConsumesNothing(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")
	h.store.setQuotaErr(errors.New("database is down"))

	status, env, _ := refused(t, h.chat(t, "dashboard"))
	if status != http.StatusInternalServerError || env.Error.Code != "internal" {
		t.Fatalf("got %d %+v, want 500 internal", status, env.Error)
	}
	if h.tokens() != rateBurst || h.streams() != 0 || h.store.messageCount(h.account) != 0 {
		t.Fatalf("consumed: tokens %d, streams %d, rows %d", h.tokens(), h.streams(), h.store.messageCount(h.account))
	}
	if keys := h.store.threadKeys(h.account); len(keys) != 0 {
		t.Fatalf("a refused chat created threads %v", keys)
	}

	status, env, _ = refused(t, h.admission(t))
	if status != http.StatusInternalServerError || env.Error.Code != "internal" {
		t.Fatalf("admission: got %d %+v, want 500 internal", status, env.Error)
	}
}

// An admitted turn that fails before reaching the provider (here: the stored key can't be
// decrypted) gives back its token and its daily message.
func TestL18PreProviderFailureRefunds(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")
	k, _ := h.store.GetKey(context.Background(), h.account, "openai")
	k.EncKey, k.EncKeyAD = []byte("corrupt-corrupt-corrupt-corrupt-x"), []byte("corrupt-corrupt-corrupt-corrupt-x")
	h.store.mu.Lock()
	h.store.keys[h.account]["openai"] = k
	h.store.mu.Unlock()

	status, _, _ := refused(t, h.chat(t, "dashboard"))
	if status != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", status)
	}
	if h.tokens() != rateBurst || h.streams() != 0 || h.store.dailyCount(h.account, h.today()) != 0 {
		t.Fatalf("a turn that never reached the provider consumed: tokens %d, streams %d, daily %d",
			h.tokens(), h.streams(), h.store.dailyCount(h.account, h.today()))
	}
}

// --- the admission probe ---

// GET /admission returns 204 while a chat would pass and the matching typed 429 in each
// exhausted state, and never consumes anything (the bucket and n are unchanged).
func TestL18AdmissionProbe(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")

	// Fresh: admitted, any number of times, with nothing taken and no quota row made.
	for i := 0; i < 2*rateBurst; i++ {
		wantAdmitted(t, h.admission(t))
	}
	if h.tokens() != rateBurst || h.store.dailyCount(h.account, h.today()) != 0 || h.streams() != 0 {
		t.Fatalf("probing consumed: tokens %d, daily %d, streams %d", h.tokens(), h.store.dailyCount(h.account, h.today()), h.streams())
	}

	// Per minute exhausted → coach_rate_limited; probing leaves the bucket empty-but-refilling,
	// so exactly one chat passes once one token's time has gone by.
	for i := 0; i < rateBurst; i++ {
		h.chatOK(t, "dashboard")
	}
	for i := 0; i < 3; i++ {
		wantLimit(t, h.admission(t), codeCoachRateLimited, "3")
	}
	h.clock.Advance(rateRefillEvery)
	wantAdmitted(t, h.admission(t))
	wantAdmitted(t, h.admission(t))
	h.chatOK(t, "dashboard")
	wantLimit(t, h.admission(t), codeCoachRateLimited, "3")

	// Per day exhausted → coach_daily_cap with the wait to the next UTC midnight; n is
	// unchanged by the probe.
	h.clock.Set(time.Date(2026, time.October, 1, 20, 0, 0, 0, time.UTC))
	h.store.setDailyCount(h.account, h.today(), dailyMessageCap)
	for i := 0; i < 3; i++ {
		wantLimit(t, h.admission(t), codeCoachDailyCap, "14400")
	}
	if n := h.store.dailyCount(h.account, h.today()); n != dailyMessageCap {
		t.Fatalf("probing changed n: %d", n)
	}
	if h.tokens() != rateBurst {
		t.Fatalf("probing changed the bucket: %d tokens", h.tokens())
	}
	// One under the cap is admitted (n < 300), still without consuming.
	h.store.setDailyCount(h.account, h.today(), dailyMessageCap-1)
	wantAdmitted(t, h.admission(t))
	if n := h.store.dailyCount(h.account, h.today()); n != dailyMessageCap-1 {
		t.Fatalf("probing changed n: %d", n)
	}
}

// --- locked mode ---

// A locked mode reaching coach (defensive: the gateway answers it) is 409 coach_paused
// with reason "mock": nothing persisted, no limit consumed — even with no key at all.
func TestChatLockedModeIsPaused(t *testing.T) {
	for _, withKey := range []bool{true, false} {
		h := newHarness(t)
		if withKey {
			h.storeOpenAIKey(t, "sk-openai-good-key-1234")
		}
		resp := h.do(t, http.MethodPost, "/chat", map[string]any{
			"context": "problem:16", "problemId": "16", "message": "hi",
		}, map[string]string{headerCoachMode: "locked"})
		status, env, _ := refused(t, resp)
		if status != http.StatusConflict || env.Error.Code != codeCoachPaused || env.Error.Reason != "mock" || env.Error.Message == "" {
			t.Fatalf("withKey=%v: got %d %+v, want 409 coach_paused reason mock", withKey, status, env.Error)
		}
		if len(h.store.threadKeys(h.account)) != 0 || h.store.messageCount(h.account) != 0 {
			t.Fatalf("withKey=%v: a paused chat persisted something", withKey)
		}
		if h.tokens() != rateBurst || h.streams() != 0 || h.store.dailyCount(h.account, h.today()) != 0 {
			t.Fatalf("withKey=%v: a paused chat consumed a limit", withKey)
		}
	}
}

// --- what a turn records ---

// Both rows of a turn record coach-prompt@2 and, when the gateway sent one, the attempt
// id; a malformed attempt header is dropped, never refused.
func TestChatRecordsPromptVersionAndAttempt(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")
	const attempt = "3f2c8a51-7d4e-4b6a-9c1d-2e5f6a7b8c9d"

	send := func(headers map[string]string) {
		t.Helper()
		resp := h.do(t, http.MethodPost, "/chat?path=dsa", map[string]any{
			"context": "problem:16", "kind": "problem", "problemId": "16", "message": "hi",
		}, headers)
		if res := readSSE(t, resp); !res.done {
			t.Fatalf("chat: %+v", res)
		}
	}
	send(map[string]string{headerCoachMode: ModeAttempt, headerCoachAttempt: strings.ToUpper(attempt)})
	send(map[string]string{headerCoachMode: ModeReview})
	send(map[string]string{headerCoachMode: ModeAttempt, headerCoachAttempt: "not-a-uuid"})

	metas := h.store.metasFor(h.account, "problem:16")
	want := []store.MessageMeta{
		{PromptV: PromptVersion, AttemptID: attempt}, {PromptV: PromptVersion, AttemptID: attempt},
		{PromptV: PromptVersion}, {PromptV: PromptVersion},
		{PromptV: PromptVersion}, {PromptV: PromptVersion},
	}
	if len(metas) != len(want) {
		t.Fatalf("metas = %+v, want %d rows", metas, len(want))
	}
	for i := range want {
		if metas[i] != want[i] {
			t.Fatalf("row %d meta = %+v, want %+v (all: %+v)", i, metas[i], want[i], metas)
		}
	}
	if _, path, _ := h.store.thread(h.account, "problem:16"); path != course.DefaultSlug {
		t.Fatalf("path_slug = %q", path)
	}
}

// --- the prompt a chat sends ---

// The persona follows X-Coach-Course: a known course's own, else the default course's.
func TestChatPersonaFromCourseHeader(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")
	h.provHandler = echoSystemOpenAI
	dsa := coursetest.DSA(t).Coach.Persona
	fixture := coursetest.Fixtures(t)[coursetest.FixtureActive].Coach.Persona

	for _, tc := range []struct {
		course string // "" sends no header
		want   string
		not    string
	}{
		{course.DefaultSlug, dsa, fixture},
		{coursetest.FixtureActive, fixture, dsa},
		{"", dsa, fixture},
		{"no-such-course", dsa, fixture},
	} {
		headers := map[string]string{headerCoachMode: ModeGeneral}
		if tc.course != "" {
			headers[headerCoachCourse] = tc.course
		}
		h.clock.Advance(rateRefillEvery)
		res := readSSE(t, h.do(t, http.MethodPost, "/chat", map[string]any{"context": "general", "message": "hi"}, headers))
		if !strings.HasPrefix(res.text, promptFrame+"\n\n"+tc.want+"\n\n") || strings.Contains(res.text, tc.not) {
			t.Fatalf("X-Coach-Course %q: prompt persona wrong:\n%s", tc.course, res.text)
		}
	}
}

// The gateway's live_items reach the general prompt as the off-limits guard (an id may
// arrive as a number); attempt prompts never carry it, nor the body's pattern.
func TestChatLiveItemsAndPatternGuard(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")
	h.provHandler = echoSystemOpenAI
	items := []any{
		map[string]any{"id": "16", "title": "3Sum"},
		map[string]any{"id": 42, "title": "Trapping Rain Water"},
	}

	res := readSSE(t, h.do(t, http.MethodPost, "/chat", map[string]any{
		"context": "dsa:week:3", "kind": "week", "label": "Week 3", "message": "hi", "live_items": items,
	}, map[string]string{headerCoachMode: ModeGeneral, headerCoachCourse: course.DefaultSlug}))
	if !strings.Contains(res.text, "OFF-LIMITS — the learner has these items live (an open attempt or a due revision touch): #16 3Sum; #42 Trapping Rain Water.") {
		t.Fatalf("general prompt missing the live-items guard:\n%s", res.text)
	}

	h.clock.Advance(rateRefillEvery)
	res = readSSE(t, h.do(t, http.MethodPost, "/chat", map[string]any{
		"context": "problem:16", "kind": "problem", "problemId": "16", "problemTitle": "3Sum",
		"pattern": "Two Pointers", "message": "which pattern is it?", "live_items": items,
	}, map[string]string{headerCoachMode: ModeAttempt, headerCoachCourse: course.DefaultSlug}))
	if !strings.Contains(res.text, "ACTIVE ATTEMPT") || strings.Contains(res.text, "OFF-LIMITS") || strings.Contains(res.text, "Two Pointers") {
		t.Fatalf("attempt prompt carries the guard or the pattern:\n%s", res.text)
	}
}

// --- history window ---

// captureOpenAI records the turns (system excluded) of every provider request, then
// streams a normal reply.
func captureOpenAI(mu *sync.Mutex, got *[]ChatMessage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		*got = (*got)[:0]
		for _, m := range body.Messages {
			if m.Role != "system" {
				*got = append(*got, ChatMessage{Role: m.Role, Content: m.Content})
			}
		}
		mu.Unlock()
		openAISSE(w)
	}
}

// A chat replays at most the last 20 messages, trimmed to 32 KiB, always ending on the
// current turn — while GET /threads still returns the full history, as in v1.
func TestChatHistoryWindow(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")
	var mu sync.Mutex
	var got []ChatMessage
	h.provHandler = captureOpenAI(&mu, &got)

	seed := func(key string, n int, content func(i int) string) {
		t.Helper()
		id, err := h.store.EnsureThread(context.Background(), h.account, key, course.DefaultSlug)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < n; i++ {
			role := store.RoleUser
			if i%2 == 1 {
				role = store.RoleAssistant
			}
			if err := h.store.AppendMessage(context.Background(), id, role, content(i), store.MessageMeta{}); err != nil {
				t.Fatal(err)
			}
		}
	}

	// 30 short messages: the turn replays the last 20 (19 after aligning on a user turn).
	seed("dsa:week:3", 30, func(i int) string { return "m" + itoa(i) })
	resp := h.do(t, http.MethodPost, "/chat", map[string]any{"context": "week:3", "message": "current"}, nil)
	if res := readSSE(t, resp); !res.done {
		t.Fatalf("chat: %+v", res)
	}
	mu.Lock()
	turns := append([]ChatMessage(nil), got...)
	mu.Unlock()
	if len(turns) > historyLimit || len(turns) < historyLimit-1 {
		t.Fatalf("replayed %d turns, want the last %d (aligned on a user turn)", len(turns), historyLimit)
	}
	if turns[0].Role != store.RoleUser || turns[len(turns)-1].Content != "current" {
		t.Fatalf("window = first %+v … last %+v", turns[0], turns[len(turns)-1])
	}
	if _, n := h.threadMessages(t, "week:3"); n != 32 {
		t.Fatalf("GET /threads returned %d messages, want the full 32", n)
	}

	// 12 messages of ~4 KiB: the window is cut to ≤ 32 KiB and keeps the current turn.
	big := strings.Repeat("z", maxMessageLen)
	seed("dsa:roadmap", 12, func(i int) string { return big })
	h.clock.Advance(rateRefillEvery)
	resp = h.do(t, http.MethodPost, "/chat", map[string]any{"context": "roadmap", "message": "current"}, nil)
	if res := readSSE(t, resp); !res.done {
		t.Fatalf("chat: %+v", res)
	}
	mu.Lock()
	turns = append([]ChatMessage(nil), got...)
	mu.Unlock()
	total := 0
	for _, tr := range turns {
		total += len(tr.Content)
	}
	if total > historyMaxBytes || len(turns) >= 13 {
		t.Fatalf("replayed %d turns / %d bytes, want ≤ %d bytes", len(turns), total, historyMaxBytes)
	}
	if turns[0].Role != store.RoleUser || turns[len(turns)-1].Content != "current" {
		t.Fatalf("window = first %q … last %q", turns[0].Role, turns[len(turns)-1].Content)
	}
	if _, n := h.threadMessages(t, "roadmap"); n != 14 {
		t.Fatalf("GET /threads returned %d messages, want the full 14", n)
	}
}

// --- the limiter itself ---

// The limiter's order (streams before the bucket), its exact refill arithmetic, refunds,
// a clock that steps back, and pruning — against a fake clock.
func TestLimiterUnit(t *testing.T) {
	clock := newFakeClock(harnessEpoch)
	l := newLimiter()
	l.now = clock.Now
	const a = "acct-a"

	// Two slots; the third is busy even though tokens remain.
	if l.admit(a) != nil || l.admit(a) != nil {
		t.Fatal("first two admits refused")
	}
	if rej := l.admit(a); rej == nil || rej.code != codeCoachBusy || rej.retryAfterSeconds() != 5 {
		t.Fatalf("third admit = %+v, want coach_busy/5", rej)
	}
	if rej := l.probe(a); rej == nil || rej.code != codeCoachBusy {
		t.Fatalf("probe at 2 streams = %+v, want coach_busy", rej)
	}
	l.releaseStream(a)
	l.releaseStream(a)
	l.releaseStream(a) // extra releases are harmless
	if l.streams[a] != 0 {
		t.Fatalf("streams = %d", l.streams[a])
	}

	// 18 more admits drain the bucket (2 already taken); the next waits one refill.
	for i := 0; i < rateBurst-2; i++ {
		if rej := l.admit(a); rej != nil {
			t.Fatalf("admit %d refused: %+v", i, rej)
		}
		l.releaseStream(a)
	}
	rej := l.admit(a)
	if rej == nil || rej.code != codeCoachRateLimited || rej.retryAfter != rateRefillEvery {
		t.Fatalf("empty bucket = %+v, want coach_rate_limited after %s", rej, rateRefillEvery)
	}
	// A refund puts exactly one token back.
	l.refundToken(a)
	if l.admit(a) != nil {
		t.Fatal("the refunded token was not admitted")
	}
	l.releaseStream(a)
	if l.admit(a) == nil {
		t.Fatal("admitted beyond the refunded token")
	}

	// A clock stepping back mints nothing.
	clock.Advance(-time.Hour)
	if rej := l.admit(a); rej == nil || rej.code != codeCoachRateLimited {
		t.Fatalf("after the clock stepped back: %+v, want coach_rate_limited", rej)
	}
	clock.Advance(time.Hour)

	// Pruning drops only full buckets, and a pruned account reads as full.
	for i := 0; i <= pruneAbove; i++ {
		b := "idle-" + itoa(i)
		if l.admit(b) != nil {
			t.Fatal(b)
		}
		l.releaseStream(b)
	}
	clock.Advance(2 * time.Minute) // every idle bucket is full again; a too
	if l.admit("trigger") != nil {
		t.Fatal("trigger refused")
	}
	l.releaseStream("trigger")
	if len(l.empty) != 1 { // only the trigger's own, non-full bucket is left
		t.Fatalf("buckets after prune = %d, want 1", len(l.empty))
	}
	if l.probe(a) != nil {
		t.Fatal("a pruned (full) bucket reads as limited")
	}

	// Daily-cap wait: to the next UTC midnight, whatever the clock's zone.
	ist := time.FixedZone("IST", 5*3600+1800)
	at := time.Date(2026, time.October, 2, 5, 0, 0, 0, ist) // 2026-10-01 23:30 UTC
	if got := dailyCapRejection(at).retryAfterSeconds(); got != 1800 {
		t.Fatalf("daily cap Retry-After at 23:30 UTC = %d, want 1800", got)
	}
}
