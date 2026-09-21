package curriculum

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

// --- JSON response shapes (api.md conventions: JSON success + error envelope) ---

type pathJSON struct {
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	Summary      string `json:"summary"`
	ProblemTotal int    `json:"problem_total"`
	WeekTotal    int    `json:"week_total"`
}

type phaseJSON struct {
	Order    int    `json:"order"`
	Name     string `json:"name"`
	Theme    string `json:"theme"`
	WeekFrom int    `json:"week_from"`
	WeekTo   int    `json:"week_to"`
}

type weekSummaryJSON struct {
	N      int    `json:"n"`
	Title  string `json:"title"`
	Thesis string `json:"thesis"`
	Easy   int    `json:"easy"`
	Med    int    `json:"med"`
	Hard   int    `json:"hard"`
	Total  int    `json:"total"`
}

type conceptRefJSON struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// weekPathJSON is the slim path context a week view needs for its eyebrow
// ("Week N of {week_total} …") without pulling the whole Roadmap payload.
type weekPathJSON struct {
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	ProblemTotal int    `json:"problem_total"`
	WeekTotal    int    `json:"week_total"`
}

type problemJSON struct {
	ID              string `json:"id"`
	PathSlug        string `json:"path_slug"`
	WeekN           int    `json:"week_n"`
	Title           string `json:"title"`
	Difficulty      string `json:"difficulty"`
	Pattern         string `json:"pattern"`
	LeetcodeURL     string `json:"leetcode_url"`
	NeetcodeURL     string `json:"neetcode_url"`
	IsReinforcement bool   `json:"is_reinforcement"`
}

type sectionJSON struct {
	Stage  string `json:"stage"`
	Kind   string `json:"kind"`
	Order  int    `json:"order"`
	BodyMD string `json:"body_md"`
	Code   string `json:"code"`
}

type conceptJSON struct {
	Slug         string `json:"slug"`
	PathSlug     string `json:"path_slug"`
	Title        string `json:"title"`
	BodyMD       string `json:"body_md"`
	WhenToUseMD  string `json:"when_to_use_md"`
	CodeTemplate string `json:"code_template"`
}

// --- handlers ---

// handleListPaths: GET /paths — Catalog (all paths + status).
func (s *Service) handleListPaths(w http.ResponseWriter, r *http.Request) {
	paths, err := s.store.ListPaths(r.Context())
	if err != nil {
		s.internal(w, "list paths", err)
		return
	}
	out := make([]pathJSON, 0, len(paths))
	for _, p := range paths {
		out = append(out, toPathJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"paths": out})
}

// handleGetPath: GET /paths/{slug} — Roadmap (phases, weeks, totals).
func (s *Service) handleGetPath(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	path, err := s.store.GetPath(r.Context(), slug)
	if err != nil {
		s.mapErr(w, "get path", err)
		return
	}
	phases, err := s.store.ListPhases(r.Context(), slug)
	if err != nil {
		s.internal(w, "list phases", err)
		return
	}
	weeks, err := s.store.ListWeeks(r.Context(), slug)
	if err != nil {
		s.internal(w, "list weeks", err)
		return
	}
	phasesOut := make([]phaseJSON, 0, len(phases))
	for _, p := range phases {
		phasesOut = append(phasesOut, phaseJSON{
			Order: p.Order, Name: p.Name, Theme: p.Theme, WeekFrom: p.WeekFrom, WeekTo: p.WeekTo,
		})
	}
	weeksOut := make([]weekSummaryJSON, 0, len(weeks))
	for _, wk := range weeks {
		weeksOut = append(weeksOut, weekSummaryJSON{
			N: wk.N, Title: wk.Title, Thesis: wk.Thesis,
			Easy: wk.Easy, Med: wk.Med, Hard: wk.Hard, Total: wk.Total,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":   toPathJSON(path),
		"phases": phasesOut,
		"weeks":  weeksOut,
	})
}

// handleListPathProblems: GET /paths/{slug}/problems — the whole problem index for a
// path (id / week_n / pattern / difficulty / reinforcement). It powers the gateway's
// Progress + Dashboard roll-ups (by-phase completion, by-pattern mastery) in one call,
// so the BFF need not fan out per problem (ADR-0005: cross-context composition in the
// gateway). Content-only; the per-user solve state is layered on in the gateway.
func (s *Service) handleListPathProblems(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	// Validate the path exists so an unknown slug is a 404 (not an empty list).
	if _, err := s.store.GetPath(r.Context(), slug); err != nil {
		s.mapErr(w, "get path", err)
		return
	}
	problems, err := s.store.ListProblemsByPath(r.Context(), slug)
	if err != nil {
		s.internal(w, "list problems by path", err)
		return
	}
	out := make([]problemJSON, 0, len(problems))
	for _, p := range problems {
		out = append(out, toProblemJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"problems": out})
}

// handleGetWeek: GET /paths/{slug}/weeks/{n} — week thesis + its phase + concepts
// + problem list. This is the CONTENT read; the gateway's `agg` version layers the
// per-user five-touch/solve state on top (ADR-0005: cross-context stitching lives in
// the gateway). The phase (resolved from the week range) and slim path context let
// the Week screen render its "Week N of {week_total} · Phase X <name>" eyebrow from
// one call.
func (s *Service) handleGetWeek(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n < 1 {
		writeError(w, http.StatusBadRequest, "bad_request", "week must be a positive integer")
		return
	}
	path, err := s.store.GetPath(r.Context(), slug)
	if err != nil {
		s.mapErr(w, "get path", err)
		return
	}
	week, err := s.store.GetWeek(r.Context(), slug, n)
	if err != nil {
		s.mapErr(w, "get week", err)
		return
	}
	phases, err := s.store.ListPhases(r.Context(), slug)
	if err != nil {
		s.internal(w, "list phases", err)
		return
	}
	concepts, err := s.store.ListConceptsByWeek(r.Context(), slug, n)
	if err != nil {
		s.internal(w, "list concepts by week", err)
		return
	}
	problems, err := s.store.ListProblemsByWeek(r.Context(), slug, n)
	if err != nil {
		s.internal(w, "list problems by week", err)
		return
	}
	conceptsOut := make([]conceptRefJSON, 0, len(concepts))
	for _, c := range concepts {
		conceptsOut = append(conceptsOut, conceptRefJSON{Slug: c.Slug, Title: c.Title})
	}
	problemsOut := make([]problemJSON, 0, len(problems))
	for _, p := range problems {
		problemsOut = append(problemsOut, toProblemJSON(p))
	}
	out := map[string]any{
		"week": map[string]any{
			"n":      week.N,
			"title":  week.Title,
			"thesis": week.Thesis,
		},
		"path": weekPathJSON{
			Slug:         path.Slug,
			Title:        path.Title,
			ProblemTotal: path.ProblemTotal,
			WeekTotal:    path.WeekTotal,
		},
		"concepts": conceptsOut,
		"problems": problemsOut,
	}
	// The phase that contains this week (order/name drive the eyebrow). A week with
	// no matching phase range simply omits it rather than guessing.
	for _, p := range phases {
		if n >= p.WeekFrom && n <= p.WeekTo {
			out["phase"] = phaseJSON{Order: p.Order, Name: p.Name, Theme: p.Theme, WeekFrom: p.WeekFrom, WeekTo: p.WeekTo}
			break
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// handleGetProblem: GET /problems/{id} — problem + its problem_sections keyed by
// `stage`. Returns all content sections now; S05 filters to only unlocked stages.
func (s *Service) handleGetProblem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	problem, err := s.store.GetProblem(r.Context(), id)
	if err != nil {
		s.mapErr(w, "get problem", err)
		return
	}
	sections, err := s.store.ListSections(r.Context(), id)
	if err != nil {
		s.internal(w, "list sections", err)
		return
	}
	sectionsOut := make([]sectionJSON, 0, len(sections))
	for _, sec := range sections {
		sectionsOut = append(sectionsOut, sectionJSON{
			Stage: sec.Stage, Kind: sec.Kind, Order: sec.Order, BodyMD: sec.BodyMD, Code: sec.Code,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"problem":  toProblemJSON(problem),
		"sections": sectionsOut,
	})
}

// handleGetConcept: GET /concepts/{slug} — concept reading + code template.
func (s *Service) handleGetConcept(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	c, err := s.store.GetConcept(r.Context(), slug)
	if err != nil {
		s.mapErr(w, "get concept", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"concept": conceptJSON{
			Slug:         c.Slug,
			PathSlug:     c.PathSlug,
			Title:        c.Title,
			BodyMD:       c.BodyMD,
			WhenToUseMD:  c.WhenToUseMD,
			CodeTemplate: c.CodeTemplate,
		},
	})
}

// --- conversions + helpers ---

func toPathJSON(p store.Path) pathJSON {
	return pathJSON{
		Slug:         p.Slug,
		Title:        p.Title,
		Status:       p.Status,
		Summary:      p.Summary,
		ProblemTotal: p.ProblemTotal,
		WeekTotal:    p.WeekTotal,
	}
}

func toProblemJSON(p store.Problem) problemJSON {
	return problemJSON{
		ID:              p.ID,
		PathSlug:        p.PathSlug,
		WeekN:           p.WeekN,
		Title:           p.Title,
		Difficulty:      p.Difficulty,
		Pattern:         p.Pattern,
		LeetcodeURL:     p.LeetcodeURL,
		NeetcodeURL:     p.NeetcodeURL,
		IsReinforcement: p.IsReinforcement,
	}
}

// mapErr maps a store error to 404 (not found) or 500 (everything else).
func (s *Service) mapErr(w http.ResponseWriter, what string, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}
	s.internal(w, what, err)
}

func (s *Service) internal(w http.ResponseWriter, what string, err error) {
	s.log.Error("curriculum store error", "op", what, "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "internal error")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}
