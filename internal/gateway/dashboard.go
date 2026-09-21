package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// This file is the Dashboard "Today" BFF aggregation (api.md GET /dashboard `agg`,
// S09). It fans out — in parallel, with per-call timeouts — to assessment (the streak /
// solved / mock projection stats + the solved set), review (the due queue, weak-area and
// reminders) and curriculum (the week/problem taxonomy), then composes the daily plan.
// The reviews-before-new-work rule (R-SR5) is baked into the plan ordering: due
// revisions lead, then the next unsolved problems in the current week. Each section
// degrades independently so the page still renders when one upstream is down.

// newWorkLimit caps how many new problems the plan suggests after the reviews.
const newWorkLimit = 3

// mockW13Target is the R-MK3 readiness target used for the "Mock best" tile's check.
const mockW13Target = 24

// --- composed output shapes ---

type dashStreak struct {
	Current int `json:"current"`
	Longest int `json:"longest"`
}

type dashSolved struct {
	Count int `json:"count"`
	Total int `json:"total"`
}

type dashMock struct {
	Best   int `json:"best"`
	Last   int `json:"last"`
	Count  int `json:"count"`
	Target int `json:"target"`
}

type dashStats struct {
	Streak       dashStreak `json:"streak"`
	Solved       dashSolved `json:"solved"`
	RevisionsDue int        `json:"revisionsDue"`
	Mock         dashMock   `json:"mock"`
}

type dashWeekProblem struct {
	ProblemID string `json:"problemId"`
	Title     string `json:"title"`
	Status    string `json:"status"` // solved | attempting | available
}

type dashWeek struct {
	N        int               `json:"n"`
	Title    string            `json:"title"`
	Solved   int               `json:"solved"`
	Total    int               `json:"total"`
	Problems []dashWeekProblem `json:"problems"`
}

// planReview / planProblem are the two daily-plan card kinds (a heterogeneous list).
type planReview struct {
	Kind       string `json:"kind"` // "review"
	ItemID     string `json:"itemId"`
	ProblemID  string `json:"problemId"`
	Title      string `json:"title"`
	DayLabel   string `json:"dayLabel"`
	TouchLevel int    `json:"touchLevel"`
	MockMode   bool   `json:"mockMode"`
}

type planProblem struct {
	Kind       string `json:"kind"` // "problem"
	ProblemID  string `json:"problemId"`
	Title      string `json:"title"`
	Difficulty string `json:"difficulty"`
	Pattern    string `json:"pattern"`
	Status     string `json:"status"` // available | attempting
}

// summaryDoc is the slice of assessment /progress/summary the dashboard reads.
type summaryDoc struct {
	Streak dashStreak `json:"streak"`
	Solved int        `json:"solved"`
	Total  int        `json:"total"`
	Mock   struct {
		Best  int `json:"best"`
		Last  int `json:"last"`
		Count int `json:"count"`
	} `json:"mock"`
}

// handleDashboard composes the "Today" home from the assessment projections, the review
// queue/weak-area, and the curriculum taxonomy.
func (g *Gateway) handleDashboard(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	aToken, _ := g.mintQuiet(accountID, g.audAssessment)
	rToken, _ := g.mintQuiet(accountID, g.audReview)

	var (
		summaryRaw   json.RawMessage
		masteryRaw   json.RawMessage
		dueRaw       json.RawMessage
		weakAreaRaw  json.RawMessage
		reminimalRaw json.RawMessage
		roadmapRaw   []byte
		problemsRaw  []byte
		wg           sync.WaitGroup
	)

	if g.assessment != nil && aToken != "" {
		wg.Add(2)
		go g.fanGet(r, &wg, g.assessment.get, aToken, "/progress/summary", &summaryRaw)
		go g.fanGet(r, &wg, g.assessment.get, aToken, "/progress/mastery", &masteryRaw)
	}
	if g.review != nil && rToken != "" {
		wg.Add(3)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(r.Context(), aggCallTimeout)
			defer cancel()
			if body, status, err := g.review.get(ctx, rToken, "/revisions/due"); err == nil && status == http.StatusOK {
				dueRaw = g.enrichDueQueue(ctx, body)
			}
		}()
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(r.Context(), aggCallTimeout)
			defer cancel()
			if body, status, err := g.review.get(ctx, rToken, "/weak-area/current"); err == nil && status == http.StatusOK {
				weakAreaRaw = g.enrichMistakeEnvelope(ctx, body, "entries")
			}
		}()
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(r.Context(), aggCallTimeout)
			defer cancel()
			if body, status, err := g.review.get(ctx, rToken, "/reminders"); err == nil && status == http.StatusOK {
				reminimalRaw = body
			}
		}()
	}
	if g.curriculum != nil {
		wg.Add(2)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(r.Context(), aggCallTimeout)
			defer cancel()
			if body, status, err := g.curriculum.get(ctx, "/paths/dsa"); err == nil && status == http.StatusOK {
				roadmapRaw = body
			}
		}()
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(r.Context(), aggCallTimeout)
			defer cancel()
			if body, status, err := g.curriculum.get(ctx, "/paths/dsa/problems"); err == nil && status == http.StatusOK {
				problemsRaw = body
			}
		}()
	}
	wg.Wait()

	var summary summaryDoc
	_ = json.Unmarshal(summaryRaw, &summary)
	var mastery masteryDoc
	_ = json.Unmarshal(masteryRaw, &mastery)
	var roadmap roadmapDoc
	_ = json.Unmarshal(roadmapRaw, &roadmap)
	var problems problemsDoc
	_ = json.Unmarshal(problemsRaw, &problems)

	solvedSet := make(map[string]bool, len(mastery.Problems))
	for _, m := range mastery.Problems {
		solvedSet[m.ProblemID] = true
	}

	// Current week: the first (lowest-n) week with an unsolved core problem.
	curWeek := currentWeek(problems.Problems, solvedSet)
	weekProblems := coreProblemsInWeek(problems.Problems, curWeek)

	// Per-problem practice status for the current week (attempting/solved/available) — a
	// single call after the week is known; degrades to the projection's solved set.
	statuses := g.currentWeekStatuses(r, accountID, curWeek, weekProblems, solvedSet)

	dueItems := parseDueItems(dueRaw)
	dueCount := 0
	for _, it := range dueItems {
		if it.Due {
			dueCount++
		}
	}

	stats := dashStats{
		Streak:       summary.Streak,
		Solved:       dashSolved{Count: summary.Solved, Total: pick(roadmap.Path.ProblemTotal, summary.Total)},
		RevisionsDue: dueCount,
		Mock:         dashMock{Best: summary.Mock.Best, Last: summary.Mock.Last, Count: summary.Mock.Count, Target: mockW13Target},
	}

	plan := buildPlan(dueItems, weekProblems, statuses)
	week := buildWeek(curWeek, roadmap.Weeks, weekProblems, statuses)

	out := map[string]json.RawMessage{
		"stats":     mustJSON(stats),
		"plan":      mustJSON(plan),
		"week":      orNull(mustJSONOrNull(week)),
		"revisions": orNull(dueRaw),
		"weakArea":  orNull(weakAreaRaw),
		"reminders": remindersOrEmpty(reminimalRaw),
	}
	body, err := json.Marshal(out)
	if err != nil {
		g.log.Error("bff dashboard: marshal failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "dashboard compose failed")
		return
	}
	passthrough(w, http.StatusOK, body)
}

// fanGet runs one authenticated GET into dst under a per-call timeout (the assessment
// fan-out legs).
func (g *Gateway) fanGet(r *http.Request, wg *sync.WaitGroup, get func(context.Context, string, string) ([]byte, int, error), token, path string, dst *json.RawMessage) {
	defer wg.Done()
	ctx, cancel := context.WithTimeout(r.Context(), aggCallTimeout)
	defer cancel()
	if body, status, err := get(ctx, token, path); err == nil && status == http.StatusOK {
		*dst = body
	} else {
		g.log.Warn("bff dashboard: call degraded", "path", path, "status", status, "err", err)
	}
}

// currentWeekStatuses asks practice for the current week's per-problem status. It falls
// back to the assessment solved set (solved vs available) when practice is unavailable.
func (g *Gateway) currentWeekStatuses(r *http.Request, accountID string, week int, weekProblems []problemIndexItem, solvedSet map[string]bool) map[string]string {
	out := make(map[string]string, len(weekProblems))
	for _, p := range weekProblems {
		if solvedSet[p.ID] {
			out[p.ID] = "solved"
		} else {
			out[p.ID] = "available"
		}
	}
	if g.practice == nil || len(weekProblems) == 0 {
		return out
	}
	token, ok := g.mintForPractice(accountID)
	if !ok {
		return out
	}
	ids := make([]string, 0, len(weekProblems))
	for _, p := range weekProblems {
		ids = append(ids, p.ID)
	}
	ctx, cancel := context.WithTimeout(r.Context(), aggCallTimeout)
	defer cancel()
	body, status, err := g.practice.get(ctx, token,
		"/state?week="+url.QueryEscape(strconv.Itoa(week))+"&ids="+url.QueryEscape(strings.Join(ids, ",")))
	if err != nil || status != http.StatusOK {
		return out
	}
	states, ok := parseWeekStates(body)
	if !ok {
		return out
	}
	for id, st := range states {
		if st.Status != "" {
			out[id] = st.Status
		}
	}
	return out
}

// currentWeek returns the lowest week number that still has an unsolved core problem, or
// the highest week seen when every core problem is solved (falling to 1 when empty).
func currentWeek(index []problemIndexItem, solved map[string]bool) int {
	coreTotal := map[int]int{}
	coreSolved := map[int]int{}
	maxWeek := 0
	for _, p := range index {
		if p.IsReinforcement {
			continue
		}
		coreTotal[p.WeekN]++
		if solved[p.ID] {
			coreSolved[p.WeekN]++
		}
		if p.WeekN > maxWeek {
			maxWeek = p.WeekN
		}
	}
	if maxWeek == 0 {
		return 1
	}
	for n := 1; n <= maxWeek; n++ {
		if coreTotal[n] > 0 && coreSolved[n] < coreTotal[n] {
			return n
		}
	}
	return maxWeek
}

// coreProblemsInWeek returns the week's non-reinforcement problems in seed order.
func coreProblemsInWeek(index []problemIndexItem, week int) []problemIndexItem {
	out := make([]problemIndexItem, 0, 8)
	for _, p := range index {
		if p.WeekN == week && !p.IsReinforcement {
			out = append(out, p)
		}
	}
	return out
}

// buildPlan orders the day: due revisions first (reviews before new work, R-SR5), then
// the next unsolved problems in the current week (capped).
func buildPlan(due []reviewDueItem, weekProblems []problemIndexItem, statuses map[string]string) []any {
	plan := make([]any, 0, len(due)+newWorkLimit)
	for _, it := range due {
		if !it.Due {
			continue
		}
		plan = append(plan, planReview{
			Kind: "review", ItemID: it.ItemID, ProblemID: it.ProblemID,
			Title: problemTitle(it.Problem, it.ProblemID), DayLabel: it.DayLabel,
			TouchLevel: it.TouchLevel, MockMode: it.MockMode,
		})
	}
	added := 0
	for _, p := range weekProblems {
		if added >= newWorkLimit {
			break
		}
		if statuses[p.ID] == "solved" {
			continue
		}
		plan = append(plan, planProblem{
			Kind: "problem", ProblemID: p.ID, Title: p.Title,
			Difficulty: p.Difficulty, Pattern: p.Pattern, Status: statusOr(statuses[p.ID]),
		})
		added++
	}
	return plan
}

// buildWeek assembles the week-progress panel: solved/total core + each problem's status
// (for the segmented bar).
func buildWeek(week int, weeks []roadmapWeek, weekProblems []problemIndexItem, statuses map[string]string) *dashWeek {
	if len(weekProblems) == 0 {
		return nil
	}
	dw := &dashWeek{N: week, Title: weekTitle(weeks, week), Total: len(weekProblems)}
	dw.Problems = make([]dashWeekProblem, 0, len(weekProblems))
	for _, p := range weekProblems {
		st := statusOr(statuses[p.ID])
		if st == "solved" {
			dw.Solved++
		}
		dw.Problems = append(dw.Problems, dashWeekProblem{ProblemID: p.ID, Title: p.Title, Status: st})
	}
	return dw
}

// --- small helpers ---

// parseDueItems reads the enriched due-queue body into its items (empty on any failure).
func parseDueItems(raw json.RawMessage) []reviewDueItem {
	if len(raw) == 0 {
		return nil
	}
	var resp reviewDueResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil
	}
	// Keep a stable order: due items by touch day then id (matches the panel grouping).
	sort.SliceStable(resp.Items, func(i, j int) bool {
		if resp.Items[i].TouchLevel != resp.Items[j].TouchLevel {
			return resp.Items[i].TouchLevel < resp.Items[j].TouchLevel
		}
		return resp.Items[i].ProblemID < resp.Items[j].ProblemID
	})
	return resp.Items
}

// problemTitle pulls a title out of an enriched curriculum problem object, or "Problem
// {id}" when it is null/unresolved.
func problemTitle(problem json.RawMessage, id string) string {
	if len(problem) > 0 {
		var p struct {
			Title string `json:"title"`
		}
		if json.Unmarshal(problem, &p) == nil && p.Title != "" {
			return p.Title
		}
	}
	return "Problem " + id
}

func weekTitle(weeks []roadmapWeek, n int) string {
	for _, wk := range weeks {
		if wk.N == n {
			return wk.Title
		}
	}
	return ""
}

// statusOr defaults a missing/blank practice status to "available".
func statusOr(s string) string {
	if s == "" {
		return "available"
	}
	return s
}

// pick returns primary when > 0, else fallback (the curriculum total wins over the
// assessment default).
func pick(primary, fallback int) int {
	if primary > 0 {
		return primary
	}
	return fallback
}

// remindersOrEmpty extracts the reminders array from review's envelope, or "[]".
func remindersOrEmpty(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("[]")
	}
	var env struct {
		Reminders json.RawMessage `json:"reminders"`
	}
	if json.Unmarshal(raw, &env) == nil && len(env.Reminders) > 0 {
		return env.Reminders
	}
	return json.RawMessage("[]")
}

// mustJSONOrNull marshals a pointer value, returning empty (→ null via orNull) when nil.
func mustJSONOrNull(v *dashWeek) json.RawMessage {
	if v == nil {
		return nil
	}
	return mustJSON(v)
}
