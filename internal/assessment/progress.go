package assessment

import (
	"net/http"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
)

// This file is the S09 progress read API: GET /progress/summary (the four Progress /
// Dashboard tiles + outcome mix), GET /progress/heatmap (the revision-activity grid) and
// GET /progress/mastery (per-problem solve quality the gateway rolls up by pattern +
// phase). All three read the assessment projections only (ADR-0018); the gateway
// composes the curriculum taxonomy on top.

// problemTotal is the fixed v1 DSA target (151 problems) shown as the "solved / 151"
// denominator. The gateway may override it with curriculum's real path.problem_total.
const problemTotal = 151

// streakWindowDays bounds the summary's current/longest streak scan; heatmapWindowDays
// bounds the displayed grid ("last 15 weeks" on the artboard, a little headroom).
const (
	streakWindowDays  = 400
	heatmapWindowDays = 16 * 7
)

// --- JSON response shapes ---

type streakJSON struct {
	Current int `json:"current"`
	Longest int `json:"longest"`
}

type retentionJSON struct {
	Pct     int `json:"pct"`
	Resets  int `json:"resets"`
	Ladders int `json:"ladders"`
}

type mockSummaryJSON struct {
	Count   int `json:"count"`
	Average int `json:"average"`
	Best    int `json:"best"`
	Last    int `json:"last"`
	Delta   int `json:"delta"`
}

type summaryJSON struct {
	Solved     int             `json:"solved"`
	Total      int             `json:"total"`
	Streak     streakJSON      `json:"streak"`
	Retention  retentionJSON   `json:"retention"`
	Mock       mockSummaryJSON `json:"mock"`
	OutcomeMix outcomeMixJSON  `json:"outcomeMix"`
}

// outcomeMixJSON is the first-solve outcome mix; Total is the sum (== solved count).
type outcomeMixJSON struct {
	Total    int `json:"total"`
	Clean    int `json:"clean"`
	Rough    int `json:"rough"`
	Assisted int `json:"assisted"`
	Miss     int `json:"miss"`
}

type heatmapDayJSON struct {
	Date    string `json:"date"` // YYYY-MM-DD
	Solves  int    `json:"solves"`
	Reviews int    `json:"reviews"`
}

type heatmapJSON struct {
	Days []heatmapDayJSON `json:"days"`
}

type masteryProblemJSON struct {
	ProblemID     string  `json:"problemId"`
	BestOutcome   string  `json:"bestOutcome"`
	BestRank      int     `json:"bestRank"`
	Weight        float64 `json:"weight"` // 0..1 mastery weight from the best outcome
	CleanSolves   int     `json:"cleanSolves"`
	SolveCount    int     `json:"solveCount"`
	FirstSolvedAt *string `json:"firstSolvedAt"`
}

type masteryJSON struct {
	Problems []masteryProblemJSON `json:"problems"`
}

// masteryWeight maps a best-outcome rank (clean=4 … miss=1) to a 0..1 quality weight so
// the gateway can compute a pattern's mastery percentage.
func masteryWeight(bestRank int) float64 {
	switch bestRank {
	case 4:
		return 1.0
	case 3:
		return 0.7
	case 2:
		return 0.45
	case 1:
		return 0.25
	default:
		return 0.0
	}
}

// --- handlers ---

// handleProgressSummary: GET /progress/summary — the four tiles (solved, streak, Day-7
// retention, mock average) + the first-solve outcome mix, all from the projections.
func (s *Service) handleProgressSummary(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	ctx := r.Context()

	solved, err := s.store.SolvedCount(ctx, accountID)
	if err != nil {
		s.mapErr(w, "solved count", err)
		return
	}
	ladders, resets, err := s.store.Retention(ctx, accountID)
	if err != nil {
		s.mapErr(w, "retention", err)
		return
	}
	mix, err := s.store.OutcomeMix(ctx, accountID)
	if err != nil {
		s.mapErr(w, "outcome mix", err)
		return
	}
	mock, err := s.store.MockStats(ctx, accountID)
	if err != nil {
		s.mapErr(w, "mock stats", err)
		return
	}
	trend, err := s.store.Trend(ctx, accountID)
	if err != nil {
		s.mapErr(w, "trend", err)
		return
	}
	days, err := s.store.Heatmap(ctx, accountID, time.Now().AddDate(0, 0, -streakWindowDays))
	if err != nil {
		s.mapErr(w, "heatmap", err)
		return
	}

	current, longest := computeStreak(days, time.Now())
	last, delta := lastAndDelta(trend)

	out := summaryJSON{
		Solved:    solved,
		Total:     problemTotal,
		Streak:    streakJSON{Current: current, Longest: longest},
		Retention: retentionJSON{Pct: retentionPct(ladders, resets), Resets: resets, Ladders: ladders},
		Mock: mockSummaryJSON{
			Count: mock.Count, Average: mock.Average, Best: mock.Best, Last: last, Delta: delta,
		},
		OutcomeMix: outcomeMix(mix),
	}
	writeJSON(w, http.StatusOK, out)
}

// handleProgressHeatmap: GET /progress/heatmap — the per-day revision activity for the
// display window (the teal-ramp grid).
func (s *Service) handleProgressHeatmap(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	days, err := s.store.Heatmap(r.Context(), accountID, time.Now().AddDate(0, 0, -heatmapWindowDays))
	if err != nil {
		s.mapErr(w, "heatmap", err)
		return
	}
	out := heatmapJSON{Days: make([]heatmapDayJSON, 0, len(days))}
	for _, d := range days {
		out.Days = append(out.Days, heatmapDayJSON{
			Date: d.Date.UTC().Format("2006-01-02"), Solves: d.Solves, Reviews: d.Reviews,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleProgressMastery: GET /progress/mastery — every solved problem with its solve
// quality; the gateway groups these by curriculum pattern (mastery bars) and week ->
// phase (completion table).
func (s *Service) handleProgressMastery(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	rows, err := s.store.Mastery(r.Context(), accountID)
	if err != nil {
		s.mapErr(w, "mastery", err)
		return
	}
	out := masteryJSON{Problems: make([]masteryProblemJSON, 0, len(rows))}
	for _, m := range rows {
		p := masteryProblemJSON{
			ProblemID:   m.ProblemID,
			BestOutcome: m.BestOutcome,
			BestRank:    m.BestRank,
			Weight:      masteryWeight(m.BestRank),
			CleanSolves: m.CleanSolves,
			SolveCount:  m.SolveCount,
		}
		if m.FirstSolvedAt != nil {
			v := m.FirstSolvedAt.UTC().Format(time.RFC3339)
			p.FirstSolvedAt = &v
		}
		out.Problems = append(out.Problems, p)
	}
	writeJSON(w, http.StatusOK, out)
}

// --- pure helpers ---

// retentionPct is the Day-7 retention: the share of started ladders not reset by a
// failed re-solve. 100% when no ladder has started yet (nothing to retain against).
func retentionPct(ladders, resets int) int {
	if ladders <= 0 {
		return 100
	}
	pct := 100.0 * (1.0 - float64(resets)/float64(ladders))
	if pct < 0 {
		pct = 0
	}
	return int(pct + 0.5)
}

// outcomeMix folds the projection map into the fixed four-slot shape + total.
func outcomeMix(mix map[string]int) outcomeMixJSON {
	o := outcomeMixJSON{
		Clean:    mix[store.OutcomeClean],
		Rough:    mix[store.OutcomeRough],
		Assisted: mix[store.OutcomeAssisted],
		Miss:     mix[store.OutcomeMiss],
	}
	o.Total = o.Clean + o.Rough + o.Assisted + o.Miss
	return o
}

// lastAndDelta returns the most recent scored /35 and its change from the previous mock
// (0 when fewer than one/two scored mocks exist). trend is oldest-first.
func lastAndDelta(trend []store.TrendPoint) (last, delta int) {
	n := len(trend)
	if n == 0 {
		return 0, 0
	}
	last = trend[n-1].Total35
	if n >= 2 {
		delta = last - trend[n-2].Total35
	}
	return last, delta
}

// computeStreak derives the current and longest active-day streaks from the per-day
// heatmap rows (a day is "active" if it has any solve or review). It is computed at read
// time from the stored rows + the clock, keeping the projection itself pure. The current
// streak counts consecutive active days ending today, or yesterday when today is idle
// (so the streak isn't reported broken before the day is over). Dates are UTC days.
func computeStreak(days []store.HeatmapDay, now time.Time) (current, longest int) {
	active := make(map[string]bool, len(days))
	for _, d := range days {
		if d.Solves+d.Reviews > 0 {
			active[dayKey(d.Date)] = true
		}
	}

	// longest: the longest run of consecutive active days.
	run := 0
	var prev time.Time
	havePrev := false
	for _, d := range days {
		if d.Solves+d.Reviews == 0 {
			continue
		}
		day := truncDay(d.Date)
		if havePrev && day.Equal(prev.AddDate(0, 0, 1)) {
			run++
		} else {
			run = 1
		}
		if run > longest {
			longest = run
		}
		prev, havePrev = day, true
	}

	// current: walk back from today (or yesterday, if today is idle) over active days.
	cursor := truncDay(now)
	if !active[dayKey(cursor)] {
		cursor = cursor.AddDate(0, 0, -1)
	}
	for active[dayKey(cursor)] {
		current++
		cursor = cursor.AddDate(0, 0, -1)
	}
	return current, longest
}

func truncDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func dayKey(t time.Time) string { return truncDay(t).Format("2006-01-02") }
