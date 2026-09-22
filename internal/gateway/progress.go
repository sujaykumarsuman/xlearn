package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"
)

// This file is the Progress BFF aggregation (api.md GET /progress `agg`). assessment
// owns the read-model projections at the EVENT grain (per problem / day / outcome —
// ADR-0018), because the practice/review events carry only a bare problem_id. The
// gateway composes the curriculum taxonomy on top: the by-phase completion table and the
// by-pattern mastery bars. The fan-out (assessment summary/heatmap/mastery/trend +
// curriculum roadmap/problems + review weak-area) runs in parallel with per-call
// timeouts; each section degrades independently, and the summary is the one required
// call (a total assessment outage → 502, so the screen shows an honest error, not zeros).

// aggCallTimeout bounds each fan-out call so a slow upstream can't blow the p95 (the
// clients also carry a 10s transport timeout; this is the tighter budget).
const aggCallTimeout = 5 * time.Second

// --- parsed upstream shapes ---

type masteryProblem struct {
	ProblemID string  `json:"problemId"`
	Weight    float64 `json:"weight"`
}

type masteryDoc struct {
	Problems []masteryProblem `json:"problems"`
}

type problemIndexItem struct {
	ID              string `json:"id"`
	WeekN           int    `json:"week_n"`
	Title           string `json:"title"`
	Difficulty      string `json:"difficulty"`
	Pattern         string `json:"pattern"`
	IsReinforcement bool   `json:"is_reinforcement"`
}

type problemsDoc struct {
	Problems []problemIndexItem `json:"problems"`
}

type roadmapPhase struct {
	Order    int    `json:"order"`
	Name     string `json:"name"`
	Theme    string `json:"theme"`
	WeekFrom int    `json:"week_from"`
	WeekTo   int    `json:"week_to"`
}

type roadmapWeek struct {
	N     int    `json:"n"`
	Title string `json:"title"`
}

type roadmapDoc struct {
	Path struct {
		ProblemTotal int `json:"problem_total"`
	} `json:"path"`
	Phases []roadmapPhase `json:"phases"`
	Weeks  []roadmapWeek  `json:"weeks"`
}

// --- composed output shapes ---

type phaseCompletion struct {
	Order    int    `json:"order"`
	Name     string `json:"name"`
	Theme    string `json:"theme"`
	WeekFrom int    `json:"weekFrom"`
	WeekTo   int    `json:"weekTo"`
	Solved   int    `json:"solved"`
	Total    int    `json:"total"`
}

type patternMastery struct {
	Name   string `json:"name"`
	Solved int    `json:"solved"`
	Total  int    `json:"total"`
	Pct    int    `json:"pct"`
}

// handleProgress is the Progress screen aggregation. It fans out to assessment (the
// projections), curriculum (the taxonomy for the roll-ups) and review (the weak-area),
// then composes the coverage-by-phase table and the pattern-mastery bars.
func (g *Gateway) handleProgress(w http.ResponseWriter, r *http.Request) {
	accountID, ok := g.authAccount(w, r)
	if !ok {
		return
	}
	if g.assessment == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "assessment not configured")
		return
	}
	aToken, ok := g.mintQuiet(accountID, g.audAssessment)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal", "could not mint token")
		return
	}
	rToken, _ := g.mintQuiet(accountID, g.audReview)

	var (
		summaryRaw, heatmapRaw, trendRaw json.RawMessage
		masteryRaw, weakAreaRaw, dueRaw  json.RawMessage
		roadmapRaw, problemsRaw          []byte
		summaryOK                        bool
		wg                               sync.WaitGroup
	)

	// assessment: the required summary (tiles + outcome mix) + heatmap + mastery + trend.
	aget := func(path string, dst *json.RawMessage, okFlag *bool) {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(r.Context(), aggCallTimeout)
		defer cancel()
		body, status, err := g.assessment.get(ctx, aToken, path)
		if err != nil || status != http.StatusOK {
			g.log.Warn("bff progress: assessment call degraded", "path", path, "status", status, "err", err)
			return
		}
		*dst = body
		if okFlag != nil {
			*okFlag = true
		}
	}
	wg.Add(4)
	go aget("/progress/summary", &summaryRaw, &summaryOK)
	go aget("/progress/heatmap", &heatmapRaw, nil)
	go aget("/progress/mastery", &masteryRaw, nil)
	go aget("/mocks/trend", &trendRaw, nil)

	// curriculum: the roadmap (phases + true problem total) + the whole problem index.
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

	// review: the weekly weak-area (the coach-context signal) + the due-queue count (for
	// the Roadmap rail's "Revisions due").
	if g.review != nil && rToken != "" {
		wg.Add(2)
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
			if body, status, err := g.review.get(ctx, rToken, "/revisions/due"); err == nil && status == http.StatusOK {
				dueRaw = body
			}
		}()
	}

	wg.Wait()

	if !summaryOK {
		// The whole Progress screen reads from assessment; a summary outage is an honest
		// error (the screen shows retry), not a page of misleading zeros.
		writeError(w, http.StatusBadGateway, "upstream", "assessment unavailable")
		return
	}

	var roadmap roadmapDoc
	_ = json.Unmarshal(roadmapRaw, &roadmap)
	var problems problemsDoc
	_ = json.Unmarshal(problemsRaw, &problems)
	var mastery masteryDoc
	_ = json.Unmarshal(masteryRaw, &mastery)

	phases := composePhaseCompletion(roadmap.Phases, problems.Problems, mastery.Problems)
	patterns := composePatternMastery(problems.Problems, mastery.Problems)

	// Enrollment + frontier + due count drive the Roadmap rail (review round 2): the
	// "Current" week is the frontier only once the path is started, else "Not started".
	solvedSet := make(map[string]bool, len(mastery.Problems))
	for _, m := range mastery.Problems {
		solvedSet[m.ProblemID] = true
	}
	enrolled := g.isEnrolled(r.Context(), accountID, "dsa")
	cur := 0
	if enrolled {
		cur = currentWeek(problems.Problems, solvedSet)
	}
	dueCount := 0
	for _, it := range parseDueItems(dueRaw) {
		if it.Due {
			dueCount++
		}
	}

	out := map[string]json.RawMessage{
		"summary":  overrideSolvedTotal(summaryRaw, roadmap.Path.ProblemTotal),
		"heatmap":  orNull(heatmapRaw),
		"trend":    orNull(trendRaw),
		"weakArea": orNull(weakAreaRaw),
	}
	out["phases"] = mustJSON(phases)
	out["patterns"] = mustJSON(patterns)
	out["enrolled"] = mustJSON(enrolled)
	out["currentWeek"] = mustJSON(cur)
	out["revisionsDue"] = mustJSON(dueCount)

	body, err := json.Marshal(out)
	if err != nil {
		g.log.Error("bff progress: marshal failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream", "progress compose failed")
		return
	}
	passthrough(w, http.StatusOK, body)
}

// composePhaseCompletion builds the by-phase completion rows: for each phase's week
// range, `total` is the non-reinforcement problems and `solved` is how many of them the
// learner has solved (from the mastery projection).
func composePhaseCompletion(phases []roadmapPhase, index []problemIndexItem, solved []masteryProblem) []phaseCompletion {
	solvedSet := make(map[string]bool, len(solved))
	for _, m := range solved {
		solvedSet[m.ProblemID] = true
	}
	out := make([]phaseCompletion, 0, len(phases))
	for _, ph := range phases {
		pc := phaseCompletion{Order: ph.Order, Name: ph.Name, Theme: ph.Theme, WeekFrom: ph.WeekFrom, WeekTo: ph.WeekTo}
		for _, p := range index {
			if p.IsReinforcement || p.WeekN < ph.WeekFrom || p.WeekN > ph.WeekTo {
				continue
			}
			pc.Total++
			if solvedSet[p.ID] {
				pc.Solved++
			}
		}
		out = append(out, pc)
	}
	return out
}

// composePatternMastery groups the (non-reinforcement) problem index by curriculum
// pattern and layers the mastery projection: `total` is the pattern's problems, `solved`
// is how many were solved, and `pct` is the quality-weighted mastery (Σ weight / total).
// Rows are ordered strongest-first, then by name.
func composePatternMastery(index []problemIndexItem, solved []masteryProblem) []patternMastery {
	weightByID := make(map[string]float64, len(solved))
	for _, m := range solved {
		weightByID[m.ProblemID] = m.Weight
	}
	type agg struct {
		total  int
		solved int
		weight float64
	}
	byPattern := map[string]*agg{}
	order := []string{}
	for _, p := range index {
		if p.IsReinforcement || p.Pattern == "" {
			continue
		}
		a, ok := byPattern[p.Pattern]
		if !ok {
			a = &agg{}
			byPattern[p.Pattern] = a
			order = append(order, p.Pattern)
		}
		a.total++
		if w, isSolved := weightByID[p.ID]; isSolved {
			a.solved++
			a.weight += w
		}
	}
	out := make([]patternMastery, 0, len(order))
	for _, name := range order {
		a := byPattern[name]
		pct := 0
		if a.total > 0 {
			pct = int(100.0*a.weight/float64(a.total) + 0.5)
		}
		out = append(out, patternMastery{Name: name, Solved: a.solved, Total: a.total, Pct: pct})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pct != out[j].Pct {
			return out[i].Pct > out[j].Pct
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// overrideSolvedTotal replaces the summary's `total` with curriculum's real
// path.problem_total when available (assessment defaults it to the fixed 151), so the
// "solved / N" denominator always matches the live curriculum. On any parse failure it
// returns the summary unchanged.
func overrideSolvedTotal(summary json.RawMessage, problemTotal int) json.RawMessage {
	if len(summary) == 0 {
		return json.RawMessage("null")
	}
	if problemTotal <= 0 {
		return summary
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(summary, &obj); err != nil {
		return summary
	}
	obj["total"] = mustJSON(problemTotal)
	if merged, err := json.Marshal(obj); err == nil {
		return merged
	}
	return summary
}

// orNull returns raw, or JSON null when it is empty (an upstream that degraded).
func orNull(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("null")
	}
	return raw
}

// mustJSON marshals v, falling back to null (v here is always a plain composed value).
func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("null")
	}
	return b
}
