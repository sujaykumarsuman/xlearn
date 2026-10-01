package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// The coach mode gate (sprint m1-07; D27, ADR-0031 §7, t5 §9). Every chat — and, read
// only, every thread load — resolves ONE server-authoritative gate for the account and
// the page context, from three lookups run in parallel under short timeouts:
//
//   - practice GET /attempts/open[?problem_id=] — the open attempts (?problem_id= for a
//     problem:<id> context, which also returns the item's solve state; account-wide
//     otherwise). It replaces the coach path's v1 /state/{id} call;
//   - review's due queue (fetchDueSet, as m1-06's itemStates reads it);
//   - assessment GET /mocks/live — the account's live mock.
//
// The item's itemState is built from them, and the mode comes from m1-06's live()
// predicate — the same one withhold() uses, so withholding and mode can never disagree:
//
//	locked   a live mock (from M2a, an open touch) whose course's coach.off_during lists it
//	attempt  a problem context with live(s) || !s.Solved
//	review   a problem context that is concluded, not due and not open
//	general  every other page (with live_items: the account's live items, off-limits)
//
// The chat FAILS CLOSED: if any lookup can't answer, it is 503 coach_state_unavailable
// and nothing is sent (v1 fell back to attempt). The thread read omits `gate` instead.
//
// D27 (per problem): a chat in problem:<id>'s context while a counted course attempt on
// <id> is open must carry assist_ack = that attempt's id (409 assist_confirm_required
// otherwise); the gateway then probes coach's L18 admission (a 429 is relayed and the
// attempt stays uncapped) and records the assist in practice BEFORE forwarding (503
// assist_unavailable on failure). Once recorded, the confirm is skipped: the attempt is
// already capped and a reload must not re-prompt. Chats from any other page are the
// accepted, honor-based bypass.

// The gate's modes and headers beyond coach.go's (gateway → coach unless noted).
const (
	coachModeLocked = "locked"

	// coachCourseHeader carries the resolved course (the item's, or the page's) so coach
	// picks the course persona; "" for an account-wide page.
	coachCourseHeader = "X-Coach-Course"
	// coachAttemptHeader names the open attempt a recorded D27 assist is on, so coach
	// stores it on the turn (coach_message.attempt_id).
	coachAttemptHeader = "X-Coach-Attempt"

	// coachGateTimeout bounds each gate lookup: the chat waits on all of them.
	coachGateTimeout = 3 * time.Second
	// maxLiveItems caps the general-mode guard list sent to coach.
	maxLiveItems = 20

	// Attempt purposes reported by practice's /attempts/open (m2-01 adds "touch").
	purposeCourse = "course"
	purposeTouch  = "touch"

	// Lock reasons (coach_paused) — the course manifest's coach.off_during kinds.
	lockMock  = "mock"
	lockTouch = "touch"
)

// Gate error codes (api.md, POST /api/coach/chat).
const (
	codeAssistConfirm    = "assist_confirm_required"
	codeCoachPaused      = "coach_paused"
	codeStateUnavailable = "coach_state_unavailable"
	codeAssistUnavail    = "assist_unavailable"
)

// gateAttempt is one entry of practice's GET /attempts/open.
type gateAttempt struct {
	AttemptID     string  `json:"attemptId"`
	ProblemID     string  `json:"problemId"`
	PathSlug      string  `json:"pathSlug"`
	Purpose       string  `json:"purpose"`
	StageReached  string  `json:"stageReached"`
	CoachAssistAt *string `json:"coachAssistAt"`
}

// assisted reports whether a D27 assist is already recorded on the attempt.
func (a gateAttempt) assisted() bool { return a.CoachAssistAt != nil && *a.CoachAssistAt != "" }

// openAttempts is practice's GET /attempts/open body.
type openAttempts struct {
	Attempts []gateAttempt `json:"attempts"`
	Problem  *struct {
		Status        string  `json:"status"`
		FirstSolvedAt *string `json:"firstSolvedAt"`
	} `json:"problem"`
}

// liveMockRef is assessment's GET /mocks/live `live` object.
type liveMockRef struct {
	ID       string `json:"id"`
	PathSlug string `json:"pathSlug"`
}

// liveItemRef is one entry of the general-mode body's live_items.
type liveItemRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// coachGate is the resolved gate for one account and page context.
type coachGate struct {
	mode   string
	reason string // locked only: lockMock | lockTouch
	// course is X-Coach-Course: the item's course (problem), the page's (course-scoped)
	// or "" (account-wide).
	course string

	// Problem context only.
	problemID   string
	problemPath string // curriculum's raw path_slug ("" if unknown): coach's ?path=
	title       string
	pattern     string
	state       itemState
	attempt     *gateAttempt // the open counted (course) attempt on this problem

	// General context only: the account's live items (open attempts, due touches).
	liveItems []liveItemRef
}

// gateJSON is the thread read's `gate` field (api.md GET /coach/thread).
type gateJSON struct {
	Mode    string           `json:"mode"`
	Reason  string           `json:"reason,omitempty"`
	Attempt *gateAttemptJSON `json:"attempt,omitempty"`
}

type gateAttemptJSON struct {
	AttemptID     string  `json:"attemptId"`
	CoachAssistAt *string `json:"coachAssistAt"`
}

// json renders the gate for the SPA (mode, the lock reason, and the open attempt's
// assist state on a problem page — AB01 F1, F3, F5–F7).
func (cg coachGate) json() gateJSON {
	out := gateJSON{Mode: cg.mode, Reason: cg.reason}
	if cg.attempt != nil && cg.mode != coachModeLocked {
		out.Attempt = &gateAttemptJSON{AttemptID: cg.attempt.AttemptID, CoachAssistAt: cg.attempt.CoachAssistAt}
	}
	return out
}

// coachGateFor resolves the gate for accountID in page context cc. ok=false when a
// configured upstream (practice, review, assessment) could not answer: the chat then
// fails closed. An upstream that isn't configured has nothing to report (as in
// itemStates). Curriculum (title, pattern, the item's course) stays best-effort, as in v1.
func (g *Gateway) coachGateFor(ctx context.Context, accountID string, cc course.CoachContext) (coachGate, bool) {
	problemID, isProblem := problemIDFromContext(cc.Key)
	cg := coachGate{mode: coachModeGeneral, course: cc.PathSlug}
	if isProblem {
		cg.mode, cg.problemID = coachModeAttempt, problemID
	}

	var (
		wg             sync.WaitGroup
		atts           openAttempts
		attsOK         bool
		due            map[string]bool
		dueOK          bool
		mock           *liveMockRef
		mockOK         bool
		meta           json.RawMessage
		dueSlug        = cc.PathSlug
		lookup, cancel = context.WithTimeout(ctx, coachGateTimeout)
	)
	defer cancel()
	wg.Add(3)
	go func() {
		defer wg.Done()
		// The due queue is per course: the item's course comes from curriculum first.
		if isProblem && g.curriculum != nil {
			meta = g.curriculumProblemMeta(lookup, problemID)
			cg.problemPath = coachProblemPathSlug(meta)
			dueSlug = itemPathSlug(cg.problemPath)
		}
		if dueSlug == "" {
			dueSlug = course.DefaultSlug
		}
		due, dueOK = g.fetchDueSet(lookup, accountID, dueSlug)
	}()
	go func() {
		defer wg.Done()
		atts, attsOK = g.fetchOpenAttempts(lookup, accountID, problemID)
	}()
	go func() {
		defer wg.Done()
		mock, mockOK = g.fetchLiveMock(lookup, accountID)
	}()
	wg.Wait()
	if !attsOK || !dueOK || !mockOK {
		g.log.Warn("coach gate: state unavailable; failing closed",
			"practice_ok", attsOK, "review_ok", dueOK, "assessment_ok", mockOK)
		return coachGate{}, false
	}

	// locked: account-wide, whatever the page (D27 kept the lock).
	if mock != nil && g.coachOffDuring(mock.PathSlug, lockMock) {
		return coachGate{mode: coachModeLocked, reason: lockMock, course: cg.course}, true
	}
	for _, a := range atts.Attempts {
		if a.Purpose == purposeTouch && g.coachOffDuring(a.PathSlug, lockTouch) {
			return coachGate{mode: coachModeLocked, reason: lockTouch, course: cg.course}, true
		}
	}

	if !isProblem {
		cg.liveItems = g.coachLiveItems(lookup, atts.Attempts, due)
		return cg, true
	}

	// A problem context: the item's state from the same sources withhold() reads.
	cg.title, cg.pattern = coachProblemTitlePattern(meta)
	cg.course = cg.problemPath
	s := itemState{DueTouch: due[problemID]}
	if p := atts.Problem; p != nil {
		s.Solved = p.Status == "solved" || (p.FirstSolvedAt != nil && *p.FirstSolvedAt != "")
	}
	for i, a := range atts.Attempts {
		if a.ProblemID != problemID {
			continue
		}
		s.OpenAttempt = true
		switch a.Purpose {
		case purposeTouch:
			s.LiveTouch = true
		case purposeCourse:
			if cg.attempt == nil { // newest first
				cg.attempt = &atts.Attempts[i]
			}
		}
	}
	cg.state = s
	if !live(s) && s.Solved {
		cg.mode = coachModeReview
	}
	return cg, true
}

// coachOffDuring reports whether course slug's manifest switches the coach off during
// kind (coach.off_during). An unknown course, or one without a coach block, locks: the
// gate fails closed.
func (g *Gateway) coachOffDuring(slug, kind string) bool {
	m, ok := g.courses.Lookup(itemPathSlug(slug))
	if !ok || m.Coach == nil {
		return true
	}
	return slices.Contains(m.Coach.OffDuring, kind)
}

// coachLiveItems lists the account's live items for the general-mode guard: every open
// attempt's problem, then the due touches, de-duplicated and capped at maxLiveItems,
// with curriculum's titles (best-effort: an id without a title still names the item).
func (g *Gateway) coachLiveItems(ctx context.Context, atts []gateAttempt, due map[string]bool) []liveItemRef {
	ids := make([]string, 0, len(atts)+len(due))
	for _, a := range atts {
		ids = append(ids, a.ProblemID)
	}
	dueIDs := make([]string, 0, len(due))
	for id, on := range due {
		if on {
			dueIDs = append(dueIDs, id)
		}
	}
	slices.Sort(dueIDs)
	ids = dedupeIDs(append(ids, dueIDs...))
	if len(ids) > maxLiveItems {
		ids = ids[:maxLiveItems]
	}
	out := make([]liveItemRef, 0, len(ids))
	if len(ids) == 0 {
		return out
	}
	metas := g.curriculumProblemMetas(ctx, ids)
	for _, id := range ids {
		title, _ := coachProblemTitlePattern(metas[id])
		out = append(out, liveItemRef{ID: id, Title: title})
	}
	return out
}

// fetchOpenAttempts reads practice's open attempts (for one problem, with its solve
// state, or account-wide). Not configured is an empty answer; any failure is ok=false.
func (g *Gateway) fetchOpenAttempts(ctx context.Context, accountID, problemID string) (openAttempts, bool) {
	if g.practice == nil {
		return openAttempts{}, true
	}
	token, ok := g.mintForPractice(accountID)
	if !ok {
		return openAttempts{}, false
	}
	path := "/attempts/open"
	if problemID != "" {
		path += "?problem_id=" + url.QueryEscape(problemID)
	}
	body, status, err := g.practice.get(ctx, token, path)
	if err != nil || status != http.StatusOK {
		return openAttempts{}, false
	}
	var out openAttempts
	if err := json.Unmarshal(body, &out); err != nil {
		return openAttempts{}, false
	}
	return out, true
}

// fetchLiveMock reads the account's live mock from assessment (nil when none). Not
// configured is "no live mock"; any failure is ok=false.
func (g *Gateway) fetchLiveMock(ctx context.Context, accountID string) (*liveMockRef, bool) {
	if g.assessment == nil {
		return nil, true
	}
	token, ok := g.mintQuiet(accountID, g.audAssessment)
	if !ok {
		return nil, false
	}
	body, status, err := g.assessment.get(ctx, token, "/mocks/live")
	if err != nil || status != http.StatusOK {
		return nil, false
	}
	var env struct {
		Live *liveMockRef `json:"live"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, false
	}
	return env.Live, true
}

// recordAssist asks practice to record the D27 assist on attemptID (idempotent).
func (g *Gateway) recordAssist(ctx context.Context, accountID, attemptID string) bool {
	if g.practice == nil {
		return false
	}
	token, ok := g.mintForPractice(accountID)
	if !ok {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, coachGateTimeout)
	defer cancel()
	_, status, err := g.practice.post(ctx, token, "/attempts/"+url.PathEscape(attemptID)+"/assist", []byte("{}"))
	return err == nil && status/100 == 2
}

// rewriteChatBody builds the body forwarded to coach: assist_ack is stripped; a problem
// context names the gated problem with curriculum's authoritative title, and its pattern
// only in review mode (m1-06's strip of pattern / concepts / solution facts is kept for
// every other mode, so a client-sent value can't pass through); a general context drops
// them too and carries live_items. Other fields (message, context, kind, stage, …) pass
// through.
func (cg coachGate) rewriteChatBody(body []byte, normalizedContext string) []byte {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil || obj == nil {
		obj = map[string]any{}
	}
	delete(obj, "assist_ack")
	obj["context"] = normalizedContext
	if cg.problemID != "" {
		obj["problemId"] = cg.problemID
		obj["problemTitle"] = cg.title
		// review ⇔ !live(s) && s.Solved ⇔ withhold(s, surfaceCoach) withholds nothing:
		// the mode and m1-06's strip come from the same predicate.
		wh := withhold(cg.state, surfaceCoach)
		if cg.mode == coachModeReview {
			obj["pattern"] = cg.pattern
		}
		if wh.Pattern || cg.mode != coachModeReview {
			delete(obj, "pattern")
		}
		if wh.Concepts || cg.mode != coachModeReview {
			delete(obj, "concepts")
		}
		if wh.SolutionFacts || cg.mode != coachModeReview {
			delete(obj, "solution_facts")
		}
	} else {
		// Not an item: no answer-bearing field belongs here, whatever the client sent
		// (coach ignores the pattern outside review too — defence in depth).
		delete(obj, "pattern")
		delete(obj, "concepts")
		delete(obj, "solution_facts")
		delete(obj, "live_items")
		if cg.liveItems != nil {
			obj["live_items"] = cg.liveItems
		}
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return body
	}
	return out
}

// writeGateError writes the gate's typed error envelope with its extra fields
// ({"error":{"code","message",…extra}}) and the mode header when one applies.
func writeGateError(w http.ResponseWriter, status int, mode, code, message string, extra map[string]any) {
	e := map[string]any{"code": code, "message": message}
	for k, v := range extra {
		e[k] = v
	}
	if mode != "" {
		w.Header().Set(coachModeHeader, mode)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": e})
}

// admission is coach's read-only L18 probe (GET /admission): 204 when a chat would pass
// the caps now, else the typed 429 (with Retry-After), which the gateway relays.
func (c *coachClient) admission(ctx context.Context, token string) (*http.Response, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, coachGateTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/admission", nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	return resp, body, err
}
