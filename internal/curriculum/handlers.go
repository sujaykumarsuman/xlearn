package curriculum

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

// --- JSON response shapes (api.md conventions: JSON success + error envelope) ---
//
// The shapes are additive only (m1-03): every v1 field keeps its name, position and,
// for the default course, its value. New fields go after the v1 ones.

type pathJSON struct {
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	Summary      string `json:"summary"`
	ProblemTotal int    `json:"problem_total"`
	WeekTotal    int    `json:"week_total"`
	// Course is the learner-safe manifest view (m1-03; course.View: nav, labels, stage
	// timers, short code, nothing answer-bearing). Absent when the binary has no
	// manifest for the slug.
	Course *course.View `json:"course,omitempty"`
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

// problemJSON is a problem's metadata. The v1 fields leetcode_url, neetcode_url and
// is_reinforcement are derived from the v2 role and links (m1-03; toProblemJSON), which
// are served beside them.
type problemJSON struct {
	ID              string     `json:"id"`
	PathSlug        string     `json:"path_slug"`
	WeekN           int        `json:"week_n"`
	Title           string     `json:"title"`
	Difficulty      string     `json:"difficulty"`
	Pattern         string     `json:"pattern"`
	LeetcodeURL     string     `json:"leetcode_url"`
	NeetcodeURL     string     `json:"neetcode_url"`
	IsReinforcement bool       `json:"is_reinforcement"`
	Role            string     `json:"role"`  // core | reinforcement | drill
	Links           []linkJSON `json:"links"` // content order; [] when none
}

// linkJSON is one outbound link of a problem.
type linkJSON struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

// problemDetailJSON is GET /problems/{id}'s problem: the list shape plus two additive,
// answer-free fields (m3-01) that list and bulk routes do not carry. Humans see a
// contract-hash prefix, never the full hash (t1 §3.4).
type problemDetailJSON struct {
	problemJSON
	// ContractHashPrefix is the first 12 hex chars of contract_hash ("" on the self path).
	ContractHashPrefix string `json:"contract_hash_prefix"`
	// GradingSummary is the seed-derived {"mode": "self"|"auto"|"mixed", ...}.
	GradingSummary json.RawMessage `json:"grading_summary"`
}

type sectionJSON struct {
	Stage  string `json:"stage"`
	Kind   string `json:"kind"`
	Order  int    `json:"order"`
	BodyMD string `json:"body_md"`
	Code   string `json:"code"`
	// Language is a code section's language ("" for prose; m1-03).
	Language string `json:"language"`
}

// conceptJSON is a concept's reading. The v1 code_template is the course's primary
// language's entry of templates (m1-03), which is served beside it.
type conceptJSON struct {
	Slug         string            `json:"slug"`
	PathSlug     string            `json:"path_slug"`
	Title        string            `json:"title"`
	BodyMD       string            `json:"body_md"`
	WhenToUseMD  string            `json:"when_to_use_md"`
	CodeTemplate string            `json:"code_template"`
	Templates    map[string]string `json:"templates"` // language -> template; {} when none
}

// The sources of the derived v1 problem fields: a reinforcement item has role
// "reinforcement" (course.Roles), and the two v1 URLs are the first link of each kind
// (course.LinkKinds).
const (
	roleReinforcement = "reinforcement"
	linkKindLeetcode  = "leetcode"
	linkKindNeetcode  = "neetcode"
)

// --- handlers ---

// handleListPaths: GET /paths — Catalog (every non-retired path with its status and
// course view, in catalog order). A retired course is invisible; `preview` courses are
// listed and the gateway, which knows the viewer, filters them (m1-03).
func (s *Service) handleListPaths(w http.ResponseWriter, r *http.Request) {
	paths, err := s.store.ListPaths(r.Context())
	if err != nil {
		s.internal(w, "list paths", err)
		return
	}
	out := make([]pathJSON, 0, len(paths))
	for _, p := range paths {
		if s.courseStatus(p) == course.StatusRetired {
			continue
		}
		out = append(out, s.toPathJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"paths": out})
}

// handleGetPath: GET /paths/{slug} — Roadmap (phases, weeks, totals, course view). Any
// status resolves: the gateway decides which courses a viewer may open.
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
		"path":   s.toPathJSON(path),
		"phases": phasesOut,
		"weeks":  weeksOut,
	})
}

// handleListPathProblems: GET /paths/{slug}/problems — the whole problem index for a
// path (id / week_n / pattern / difficulty / role). It powers the gateway's
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

// maxBulkProblemIDs caps a single bulk problem-metadata request. It bounds a
// malformed/abusive query while leaving room for every course: v2 grows past DSA's 151
// problems (t1 §4: ~345 items at v2.0), so a due queue / journal enrichment across
// courses still fits in one call.
const maxBulkProblemIDs = 1024

// handleGetProblemsByIDs: GET /problems?ids=a,b,c — bulk problem metadata (no
// sections). This is the gateway's one-call enrichment path for the Revision due
// queue and the mistake journal, replacing N per-id GETs (ADR-0005). Unknown ids are
// silently absent from the result; an empty/missing `ids` returns an empty list.
func (s *Service) handleGetProblemsByIDs(w http.ResponseWriter, r *http.Request) {
	ids := dedupeNonEmpty(strings.Split(r.URL.Query().Get("ids"), ","))
	if len(ids) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"problems": []problemJSON{}})
		return
	}
	if len(ids) > maxBulkProblemIDs {
		writeError(w, http.StatusBadRequest, "bad_request", "too many ids")
		return
	}
	problems, err := s.store.GetProblemsByIDs(r.Context(), ids)
	if err != nil {
		s.internal(w, "get problems by ids", err)
		return
	}
	out := make([]problemJSON, 0, len(problems))
	for _, p := range problems {
		out = append(out, toProblemJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"problems": out})
}

// dedupeNonEmpty trims each element and returns the distinct non-empty ones, order
// preserved (first occurrence wins).
func dedupeNonEmpty(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
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
			Language: sec.Language,
		})
	}
	summary := problem.GradingSummary
	if len(summary) == 0 {
		summary = json.RawMessage(`{}`)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"problem": problemDetailJSON{
			problemJSON:        toProblemJSON(problem),
			ContractHashPrefix: canon.Prefix(problem.ContractHash),
			GradingSummary:     summary,
		},
		"sections": sectionsOut,
	})
}

// handleGetConcept: GET /paths/{slug}/concepts/{c} — a concept's reading and code
// templates, keyed on (course, concept slug) (m1-03): a concept resolves only under its
// own course.
func (s *Service) handleGetConcept(w http.ResponseWriter, r *http.Request) {
	s.serveConcept(w, r, r.PathValue("slug"), r.PathValue("c"))
}

// handleGetConceptAlias: GET /concepts/{slug} — v1's route, now the DSA alias: the
// concept under course.DefaultSlug, byte-identical to
// GET /paths/<course.DefaultSlug>/concepts/{slug}.
func (s *Service) handleGetConceptAlias(w http.ResponseWriter, r *http.Request) {
	s.serveConcept(w, r, course.DefaultSlug, r.PathValue("slug"))
}

func (s *Service) serveConcept(w http.ResponseWriter, r *http.Request, pathSlug, slug string) {
	c, err := s.store.GetConcept(r.Context(), pathSlug, slug)
	if err != nil {
		s.mapErr(w, "get concept", err)
		return
	}
	templates := c.Templates
	if templates == nil {
		templates = map[string]string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"concept": conceptJSON{
			Slug:         c.Slug,
			PathSlug:     c.PathSlug,
			Title:        c.Title,
			BodyMD:       c.BodyMD,
			WhenToUseMD:  c.WhenToUseMD,
			CodeTemplate: templates[s.primaryLanguage(c.PathSlug)],
			Templates:    templates,
		},
	})
}

// --- conversions + helpers ---

// courseStatus is a path's status as its manifest states it, falling back to the row's
// when the registry has no manifest for the slug.
func (s *Service) courseStatus(p store.Path) string {
	if m, ok := s.courses.Lookup(p.Slug); ok {
		return m.Status
	}
	return p.Status
}

// primaryLanguage is a course's primary language (its manifest's
// coach.primary_language: the v1 concept template's language), "" when the registry
// has no manifest for the slug or the manifest names none.
func (s *Service) primaryLanguage(slug string) string {
	m, ok := s.courses.Lookup(slug)
	if !ok || m.Coach == nil {
		return ""
	}
	return m.Coach.PrimaryLanguage
}

func (s *Service) toPathJSON(p store.Path) pathJSON {
	out := pathJSON{
		Slug:         p.Slug,
		Title:        p.Title,
		Status:       p.Status,
		Summary:      p.Summary,
		ProblemTotal: p.ProblemTotal,
		WeekTotal:    p.WeekTotal,
	}
	if m, ok := s.courses.Lookup(p.Slug); ok {
		v := m.LearnerView()
		out.Course = &v
	}
	return out
}

// toProblemJSON serves a problem with its v1 fields derived from role and links
// (m1-03): the values v1 read from the columns M1c drops.
func toProblemJSON(p store.Problem) problemJSON {
	links := make([]linkJSON, 0, len(p.Links))
	for _, l := range p.Links {
		links = append(links, linkJSON{Kind: l.Kind, URL: l.URL})
	}
	return problemJSON{
		ID:              p.ID,
		PathSlug:        p.PathSlug,
		WeekN:           p.WeekN,
		Title:           p.Title,
		Difficulty:      p.Difficulty,
		Pattern:         p.Pattern,
		LeetcodeURL:     firstLinkURL(p.Links, linkKindLeetcode),
		NeetcodeURL:     firstLinkURL(p.Links, linkKindNeetcode),
		IsReinforcement: p.Role == roleReinforcement,
		Role:            p.Role,
		Links:           links,
	}
}

// firstLinkURL is the URL of the first link of kind ("" if none).
func firstLinkURL(links []store.Link, kind string) string {
	for _, l := range links {
		if l.Kind == kind {
			return l.URL
		}
	}
	return ""
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
