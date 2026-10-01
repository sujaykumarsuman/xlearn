package gateway

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// The m1-07 coach mode gate on newCoachHarness (coach_test.go): the D27 assist confirm and
// record, the admission probe, the fail-closed 503s, the mock lock, the modes and their
// headers, Retry-After relay, and the thread's `gate`.

const att16 = "att-16"

func chatBody(ctx, ack string) string {
	b := map[string]any{"context": ctx, "kind": "problem", "message": "help", "pattern": "CLIENT-PATTERN",
		"concepts": []string{"CLIENT-CONCEPT"}, "solution_facts": map[string]any{"x": "CLIENT-FACTS"}}
	if ack != "" {
		b["assist_ack"] = ack
	}
	out, _ := json.Marshal(b)
	return string(out)
}

func gateErr(t *testing.T, body string) map[string]any {
	t.Helper()
	var env struct {
		Error map[string]any `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &env); err != nil || env.Error == nil {
		t.Fatalf("not an error envelope: %s", body)
	}
	return env.Error
}

// coachCalls filters the recorded calls to those made to coach (admission, chat).
func coachCalls(calls []string) []string {
	var out []string
	for _, c := range calls {
		if strings.HasPrefix(c, "coach ") {
			out = append(out, c)
		}
	}
	return out
}

func TestCoachGateD27(t *testing.T) {
	assistCall := "practice POST /attempts/" + att16 + "/assist"
	cases := []struct {
		name  string
		setup func(h *coachHarness)
		ctx   string
		ack   string

		wantStatus     int
		wantCode       string
		wantCalls      []string // in order, among assist/admission/chat
		wantAttemptHdr string
	}{
		{
			name: "no ack → 409, nothing reaches coach", ctx: "problem:16",
			wantStatus: http.StatusConflict, wantCode: codeAssistConfirm,
		},
		{
			name: "wrong ack → 409", ctx: "problem:16", ack: "att-other",
			wantStatus: http.StatusConflict, wantCode: codeAssistConfirm,
		},
		{
			name: "ack → admission probe, then the record, then the chat", ctx: "problem:16", ack: att16,
			wantStatus: http.StatusOK, wantCalls: []string{"coach GET /admission", assistCall, "coach POST /chat"},
			wantAttemptHdr: att16,
		},
		{
			name: "ack + practice can't record → 503, coach never called", ctx: "problem:16", ack: att16,
			setup:      func(h *coachHarness) { h.assistStatus = http.StatusInternalServerError },
			wantStatus: http.StatusServiceUnavailable, wantCode: codeAssistUnavail,
			wantCalls: []string{"coach GET /admission", assistCall},
		},
		{
			name: "ack + admission 429 → relayed, the attempt is not capped", ctx: "problem:16", ack: att16,
			setup:      func(h *coachHarness) { h.admissionStatus, h.admissionRetryAfter = http.StatusTooManyRequests, "12" },
			wantStatus: http.StatusTooManyRequests, wantCode: "coach_rate_limited",
			wantCalls: []string{"coach GET /admission"},
		},
		{
			name: "already assisted → no confirm, no record, forwarded with the attempt", ctx: "problem:16",
			setup: func(h *coachHarness) {
				h.attempts = []map[string]any{openAttemptFor(att16, "16", "2026-10-01T08:30:00Z")}
			},
			wantStatus: http.StatusOK, wantCalls: []string{"coach POST /chat"}, wantAttemptHdr: att16,
		},
		{
			name: "another problem's context → no confirm (the honor-based bypass)", ctx: "problem:17",
			wantStatus: http.StatusOK, wantCalls: []string{"coach POST /chat"},
		},
		{
			name: "a general page → no confirm", ctx: "dsa:week:1",
			wantStatus: http.StatusOK, wantCalls: []string{"coach POST /chat"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newCoachHarness(t)
			h.attempts = []map[string]any{openAttemptFor(att16, "16", "")}
			if c.setup != nil {
				c.setup(h)
			}
			resp, body := h.do(t, http.MethodPost, "/coach/chat", chatBody(c.ctx, c.ack))
			if resp.StatusCode != c.wantStatus {
				t.Fatalf("status %d, want %d: %s", resp.StatusCode, c.wantStatus, body)
			}
			if c.wantCode != "" {
				e := gateErr(t, body)
				if e["code"] != c.wantCode {
					t.Fatalf("code %v, want %s: %s", e["code"], c.wantCode, body)
				}
				if c.wantCode == codeAssistConfirm && (e["attemptId"] != att16 || e["problemId"] != "16") {
					t.Errorf("409 body %s, want attemptId %s problemId 16", body, att16)
				}
			}
			var got []string
			for _, call := range h.called() {
				if call == assistCall || strings.HasPrefix(call, "coach ") {
					got = append(got, call)
				}
			}
			if !slices.Equal(got, c.wantCalls) {
				t.Errorf("calls %v, want %v", got, c.wantCalls)
			}
			if c.wantStatus == http.StatusTooManyRequests && resp.Header.Get("Retry-After") != "12" {
				t.Errorf("Retry-After %q, want 12", resp.Header.Get("Retry-After"))
			}
			if c.wantStatus == http.StatusOK {
				if got := h.chatHeaders.Get(coachAttemptHeader); got != c.wantAttemptHdr {
					t.Errorf("X-Coach-Attempt %q, want %q", got, c.wantAttemptHdr)
				}
				if strings.Contains(h.chatBody, "assist_ack") {
					t.Errorf("assist_ack forwarded to coach: %s", h.chatBody)
				}
			}
		})
	}
}

func TestCoachGateFailsClosed(t *testing.T) {
	for name, setup := range map[string]func(h *coachHarness){
		"practice down":   func(h *coachHarness) { h.practiceStatus = http.StatusInternalServerError },
		"review down":     func(h *coachHarness) { h.reviewStatus = http.StatusInternalServerError },
		"assessment down": func(h *coachHarness) { h.assessmentStatus = http.StatusBadGateway },
	} {
		for _, ctx := range []string{"problem:16", "dsa:week:1", "settings"} {
			t.Run(name+"/"+ctx, func(t *testing.T) {
				h := newCoachHarness(t)
				setup(h)
				resp, body := h.do(t, http.MethodPost, "/coach/chat", chatBody(ctx, att16))
				if resp.StatusCode != http.StatusServiceUnavailable || gateErr(t, body)["code"] != codeStateUnavailable {
					t.Fatalf("%d %s, want 503 %s", resp.StatusCode, body, codeStateUnavailable)
				}
				if cc := coachCalls(h.called()); len(cc) != 0 {
					t.Errorf("coach was called: %v", cc)
				}
			})
		}
	}
}

func TestCoachGateLockedDuringLiveMock(t *testing.T) {
	for _, ctx := range []string{"problem:16", "dsa:mock", "settings"} {
		t.Run(ctx, func(t *testing.T) {
			h := newCoachHarness(t)
			h.attempts = []map[string]any{openAttemptFor(att16, "16", "")}
			h.liveMock = map[string]any{"id": "m-1", "pathSlug": "dsa", "startedAt": "2026-10-01T09:00:00Z", "deadlineAt": "2026-10-01T09:45:00Z"}
			resp, body := h.do(t, http.MethodPost, "/coach/chat", chatBody(ctx, att16))
			if resp.StatusCode != http.StatusConflict {
				t.Fatalf("%d %s, want 409", resp.StatusCode, body)
			}
			if e := gateErr(t, body); e["code"] != codeCoachPaused || e["reason"] != "mock" {
				t.Fatalf("body %s, want coach_paused / mock", body)
			}
			if resp.Header.Get(coachModeHeader) != coachModeLocked {
				t.Errorf("X-Coach-Mode %q, want locked", resp.Header.Get(coachModeHeader))
			}
			calls := h.called()
			if slices.Contains(calls, "practice POST /attempts/"+att16+"/assist") || len(coachCalls(calls)) != 0 {
				t.Errorf("a paused chat recorded or forwarded: %v", calls)
			}
		})
	}
}

func TestCoachGateModes(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(h *coachHarness)
		ctx       string
		mode      string
		course    string
		pattern   bool   // the authoritative pattern reaches coach
		liveItems string // general: the live_items coach receives (JSON)
		query     string
	}{
		{name: "never solved → attempt", ctx: "problem:20", mode: coachModeAttempt, course: "dsa", query: "path=dsa"},
		{name: "solved, not due, no attempt → review", ctx: "problem:21", mode: coachModeReview, course: "dsa", pattern: true, query: "path=dsa",
			setup: func(h *coachHarness) { h.solved["21"] = true }},
		{name: "solved but a touch is due → attempt", ctx: "problem:22", mode: coachModeAttempt, course: "dsa", query: "path=dsa",
			setup: func(h *coachHarness) { h.solved["22"], h.due = true, []string{"22"} }},
		{name: "solved with an assisted open attempt → attempt", ctx: "problem:23", mode: coachModeAttempt, course: "dsa", query: "path=dsa",
			setup: func(h *coachHarness) {
				h.solved["23"] = true
				h.attempts = []map[string]any{openAttemptFor("att-23", "23", "2026-10-01T08:00:00Z")}
			}},
		{name: "a course page → general with the live items", ctx: "dsa:week:1", mode: coachModeGeneral, course: "dsa",
			liveItems: `[{"id":"16","title":"Problem 16"},{"id":"22","title":"Problem 22"}]`,
			setup: func(h *coachHarness) {
				h.attempts = []map[string]any{openAttemptFor(att16, "16", "")}
				h.due = []string{"22"}
			}},
		{name: "a v1 course context → general, normalized", ctx: "week:1", mode: coachModeGeneral, course: course.DefaultSlug, liveItems: `[]`},
		{name: "an account-wide page → general, no course", ctx: "settings", mode: coachModeGeneral, course: "", liveItems: `[]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newCoachHarness(t)
			if c.setup != nil {
				c.setup(h)
			}
			resp, body := h.do(t, http.MethodPost, "/coach/chat", chatBody(c.ctx, ""))
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%d %s", resp.StatusCode, body)
			}
			if got := h.chatHeaders.Get(coachModeHeader); got != c.mode {
				t.Errorf("mode to coach %q, want %q", got, c.mode)
			}
			if got := resp.Header.Get(coachModeHeader); got != c.mode {
				t.Errorf("X-Coach-Mode on the response %q, want %q", got, c.mode)
			}
			if got := h.chatHeaders.Get(coachCourseHeader); got != c.course {
				t.Errorf("X-Coach-Course %q, want %q", got, c.course)
			}
			if h.chatQuery != c.query {
				t.Errorf("chat query %q, want %q", h.chatQuery, c.query)
			}
			var fwd map[string]json.RawMessage
			if err := json.Unmarshal([]byte(h.chatBody), &fwd); err != nil {
				t.Fatalf("forwarded body: %v", err)
			}
			id := strings.TrimPrefix(c.ctx, "problem:")
			hasPattern := strings.Contains(h.chatBody, "PATTERN-"+id)
			if hasPattern != c.pattern || strings.Contains(h.chatBody, "CLIENT-PATTERN") {
				t.Errorf("pattern forwarded=%v (want %v) or a client pattern passed through: %s", hasPattern, c.pattern, h.chatBody)
			}
			if c.mode != coachModeReview && (strings.Contains(h.chatBody, "CLIENT-CONCEPT") || strings.Contains(h.chatBody, "CLIENT-FACTS")) {
				t.Errorf("concepts / solution facts reached coach in %s mode: %s", c.mode, h.chatBody)
			}
			if c.liveItems != "" {
				if got := string(fwd["live_items"]); got != c.liveItems {
					t.Errorf("live_items %s, want %s", got, c.liveItems)
				}
			} else if _, ok := fwd["live_items"]; ok {
				t.Errorf("live_items on a problem context: %s", h.chatBody)
			}
		})
	}
}

// Coach's typed 429s (its own L18 checks on the chat) relay with their Retry-After.
func TestCoachChatRelaysRetryAfter(t *testing.T) {
	h := newCoachHarness(t)
	h.chatStatus, h.chatRetryAfter = http.StatusTooManyRequests, "7"
	resp, body := h.do(t, http.MethodPost, "/coach/chat", chatBody("dsa:week:1", ""))
	if resp.StatusCode != http.StatusTooManyRequests || resp.Header.Get("Retry-After") != "7" {
		t.Fatalf("%d Retry-After %q: %s", resp.StatusCode, resp.Header.Get("Retry-After"), body)
	}
	if gateErr(t, body)["code"] != "coach_rate_limited" {
		t.Fatalf("body %s", body)
	}
}

func TestCoachThreadGate(t *testing.T) {
	type gate struct {
		Mode    string `json:"mode"`
		Reason  string `json:"reason"`
		Attempt *struct {
			AttemptID     string  `json:"attemptId"`
			CoachAssistAt *string `json:"coachAssistAt"`
		} `json:"attempt"`
	}
	read := func(t *testing.T, h *coachHarness, ctx string) *gate {
		t.Helper()
		resp, body := h.do(t, http.MethodGet, "/coach/thread?context="+ctx, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("thread %s: %d %s", ctx, resp.StatusCode, body)
		}
		var out struct {
			Messages []any `json:"messages"`
			Gate     *gate `json:"gate"`
		}
		if err := json.Unmarshal([]byte(body), &out); err != nil || out.Messages == nil {
			t.Fatalf("thread body %s: %v", body, err)
		}
		return out.Gate
	}

	h := newCoachHarness(t)
	h.attempts = []map[string]any{openAttemptFor(att16, "16", "")}
	h.solved["21"] = true

	if g := read(t, h, "problem:16"); g == nil || g.Mode != coachModeAttempt || g.Attempt == nil ||
		g.Attempt.AttemptID != att16 || g.Attempt.CoachAssistAt != nil {
		t.Errorf("open attempt: gate %+v", g)
	}
	if g := read(t, h, "problem:21"); g == nil || g.Mode != coachModeReview || g.Attempt != nil {
		t.Errorf("concluded: gate %+v", g)
	}
	if g := read(t, h, "week:1"); g == nil || g.Mode != coachModeGeneral {
		t.Errorf("week: gate %+v", g)
	}
	// The read never records an assist and never answers 409.
	if calls := h.called(); slices.Contains(calls, "practice POST /attempts/"+att16+"/assist") || len(coachCalls(calls)) != 0 {
		t.Errorf("the thread read wrote or chatted: %v", calls)
	}

	h.attempts = []map[string]any{openAttemptFor(att16, "16", "2026-10-01T08:30:00Z")}
	if g := read(t, h, "problem:16"); g == nil || g.Attempt == nil || g.Attempt.CoachAssistAt == nil ||
		*g.Attempt.CoachAssistAt != "2026-10-01T08:30:00Z" {
		t.Errorf("assisted: gate %+v", g)
	}

	h.liveMock = map[string]any{"id": "m-1", "pathSlug": "dsa"}
	if g := read(t, h, "problem:16"); g == nil || g.Mode != coachModeLocked || g.Reason != "mock" || g.Attempt != nil {
		t.Errorf("live mock: gate %+v", g)
	}
	h.liveMock = nil

	// A lookup failure omits the gate; the thread still loads.
	h.reviewStatus = http.StatusInternalServerError
	if g := read(t, h, "problem:16"); g != nil {
		t.Errorf("lookup failure: gate %+v, want omitted", g)
	}
}
