// Package store is the curriculum service's persistence layer: pgx/pgxpool over the
// sqlc-generated queries in ./gen and goose migrations in ./migrations (ADR-0005).
// It exposes small domain types with plain Go scalars so the HTTP layer never
// touches pgtype, keeps the xlearn_curriculum role scoped to schema curriculum (all
// SQL is schema-qualified), and applies the versioned seed idempotently in one
// transaction (SeedAll). Curriculum emits no events, so there is no outbox.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store/gen"
)

// ErrNotFound is returned when a lookup matches no row (mapped to 404 above).
var ErrNotFound = errors.New("curriculum: not found")

// --- read-side domain types ---

// Path is a learning path (Catalog card + Roadmap header).
type Path struct {
	Slug         string
	Title        string
	Status       string // "active" | "coming_soon"
	Summary      string
	ProblemTotal int
	WeekTotal    int
	SortOrder    int
}

// Phase groups a contiguous week range within a path.
type Phase struct {
	Order    int
	Name     string
	Theme    string
	WeekFrom int
	WeekTo   int
}

// WeekSummary is a week with the difficulty mix of its seeded problems (Roadmap rail).
type WeekSummary struct {
	N      int
	Title  string
	Thesis string
	Easy   int
	Med    int
	Hard   int
	Total  int
}

// Week is a week's content header (Week screen).
type Week struct {
	N      int
	Title  string
	Thesis string
}

// ConceptRef is a lightweight concept reference (for a week's concept list).
type ConceptRef struct {
	Slug  string
	Title string
}

// Concept is a full concept/pattern reading + its code templates.
type Concept struct {
	Slug        string
	PathSlug    string
	Title       string
	BodyMD      string
	WhenToUseMD string
	// Templates maps a language to its code template (concept.templates; never nil).
	// The v1 single template is the course's primary language's entry (handlers.go).
	Templates map[string]string
}

// Link is one outbound link of a problem ({kind, url}; kind is one of
// course.LinkKinds). It is also the jsonb shape of problem.links.
type Link struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

// Problem is a problem's metadata (natural id, difficulty drives the UI tokens).
type Problem struct {
	ID         string
	PathSlug   string
	WeekN      int
	Title      string
	Difficulty string // "easy" | "med" | "hard"
	Pattern    string
	// Role is core | reinforcement | drill (course.Roles).
	Role string
	// Links are the outbound links in content order (never nil). The v1 URL fields are
	// derived from them (handlers.go).
	Links []Link
	// ContractHash and GradingSummary are read by GetProblem and GetProblemsByIDs only
	// (m3-01): canon.ContractHash ("" on the self path) and the derived, answer-free
	// grading summary (a JSON object).
	ContractHash   string
	GradingSummary json.RawMessage
	// SolutionFacts are the item's public solution facts (the frozen item schema's
	// solution_facts, a JSON object), served only inside the solution-stage block of
	// GET /problems/{id} (m1-06). Not persisted yet: m2-01's `spec` column fills it; nil
	// means none.
	SolutionFacts json.RawMessage
}

// Section is one stage-scoped content section of a problem.
type Section struct {
	Stage  string // "attempt" | "hint" | "solution"
	Kind   string
	Order  int
	BodyMD string
	Code   string
	// Language is a code section's language ("" for prose).
	Language string
}

// Store is the curriculum persistence seam. Handlers depend on this interface so
// they can be unit-tested against an in-memory fake.
type Store interface {
	ListPaths(ctx context.Context) ([]Path, error)
	GetPath(ctx context.Context, slug string) (Path, error)
	ListPhases(ctx context.Context, pathSlug string) ([]Phase, error)
	ListWeeks(ctx context.Context, pathSlug string) ([]WeekSummary, error)
	GetWeek(ctx context.Context, pathSlug string, n int) (Week, error)
	ListConceptsByWeek(ctx context.Context, pathSlug string, n int) ([]ConceptRef, error)
	ListProblemsByWeek(ctx context.Context, pathSlug string, n int) ([]Problem, error)
	ListProblemsByPath(ctx context.Context, pathSlug string) ([]Problem, error)
	GetProblem(ctx context.Context, id string) (Problem, error)
	GetProblemsByIDs(ctx context.Context, ids []string) ([]Problem, error)
	ListSections(ctx context.Context, problemID string) ([]Section, error)
	// GetConcept resolves a concept by its course and slug (a slug is unique per course).
	GetConcept(ctx context.Context, pathSlug, slug string) (Concept, error)
	CountProblems(ctx context.Context, pathSlug string) (int, error)
	// SeedAll applies the entire versioned seed in one transaction, idempotently
	// (natural-key upserts, delete-missing per course, sections rewritten, the id guard)
	// so re-running on every boot never duplicates rows. See PgStore.SeedAll.
	SeedAll(ctx context.Context, content SeedContent) (SeedReport, error)
	Ping(ctx context.Context) error
}

// PgStore is the pgxpool-backed Store.
type PgStore struct {
	pool *pgxpool.Pool
	q    *gen.Queries
}

// New wraps a pgxpool with the generated queries.
func New(pool *pgxpool.Pool) *PgStore {
	return &PgStore{pool: pool, q: gen.New(pool)}
}

var _ Store = (*PgStore)(nil)

// Ping verifies the database is reachable (drives /readyz).
func (s *PgStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// --- reads ---

func (s *PgStore) ListPaths(ctx context.Context) ([]Path, error) {
	rows, err := s.q.ListPaths(ctx)
	if err != nil {
		return nil, fmt.Errorf("list paths: %w", err)
	}
	out := make([]Path, 0, len(rows))
	for _, r := range rows {
		out = append(out, Path{
			Slug:         r.Slug,
			Title:        r.Title,
			Status:       r.Status,
			Summary:      r.Summary,
			ProblemTotal: int(r.ProblemTotal),
			WeekTotal:    int(r.WeekTotal),
			SortOrder:    int(r.SortOrder),
		})
	}
	return out, nil
}

func (s *PgStore) GetPath(ctx context.Context, slug string) (Path, error) {
	r, err := s.q.GetPath(ctx, slug)
	if err != nil {
		return Path{}, mapErr(err)
	}
	return Path{
		Slug:         r.Slug,
		Title:        r.Title,
		Status:       r.Status,
		Summary:      r.Summary,
		ProblemTotal: int(r.ProblemTotal),
		WeekTotal:    int(r.WeekTotal),
		SortOrder:    int(r.SortOrder),
	}, nil
}

func (s *PgStore) ListPhases(ctx context.Context, pathSlug string) ([]Phase, error) {
	rows, err := s.q.ListPhasesByPath(ctx, pathSlug)
	if err != nil {
		return nil, fmt.Errorf("list phases: %w", err)
	}
	out := make([]Phase, 0, len(rows))
	for _, r := range rows {
		out = append(out, Phase{
			Order:    int(r.Order),
			Name:     r.Name,
			Theme:    r.Theme,
			WeekFrom: int(r.WeekFrom),
			WeekTo:   int(r.WeekTo),
		})
	}
	return out, nil
}

func (s *PgStore) ListWeeks(ctx context.Context, pathSlug string) ([]WeekSummary, error) {
	rows, err := s.q.ListWeeksWithCounts(ctx, pathSlug)
	if err != nil {
		return nil, fmt.Errorf("list weeks: %w", err)
	}
	out := make([]WeekSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, WeekSummary{
			N:      int(r.N),
			Title:  r.Title,
			Thesis: r.Thesis,
			Easy:   int(r.Easy),
			Med:    int(r.Med),
			Hard:   int(r.Hard),
			Total:  int(r.Total),
		})
	}
	return out, nil
}

func (s *PgStore) GetWeek(ctx context.Context, pathSlug string, n int) (Week, error) {
	r, err := s.q.GetWeek(ctx, gen.GetWeekParams{PathSlug: pathSlug, N: int32(n)})
	if err != nil {
		return Week{}, mapErr(err)
	}
	return Week{N: int(r.N), Title: r.Title, Thesis: r.Thesis}, nil
}

func (s *PgStore) ListConceptsByWeek(ctx context.Context, pathSlug string, n int) ([]ConceptRef, error) {
	rows, err := s.q.ListConceptsByWeek(ctx, gen.ListConceptsByWeekParams{PathSlug: pathSlug, N: int32(n)})
	if err != nil {
		return nil, fmt.Errorf("list concepts by week: %w", err)
	}
	out := make([]ConceptRef, 0, len(rows))
	for _, r := range rows {
		out = append(out, ConceptRef{Slug: r.Slug, Title: r.Title})
	}
	return out, nil
}

func (s *PgStore) ListProblemsByWeek(ctx context.Context, pathSlug string, n int) ([]Problem, error) {
	rows, err := s.q.ListProblemsByWeek(ctx, gen.ListProblemsByWeekParams{PathSlug: pathSlug, WeekN: int32(n)})
	if err != nil {
		return nil, fmt.Errorf("list problems by week: %w", err)
	}
	out := make([]Problem, 0, len(rows))
	for _, r := range rows {
		// The week and path index select the same columns.
		p, err := indexProblem(gen.ListProblemsByPathRow(r))
		if err != nil {
			return nil, fmt.Errorf("list problems by week: %w", err)
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *PgStore) ListProblemsByPath(ctx context.Context, pathSlug string) ([]Problem, error) {
	rows, err := s.q.ListProblemsByPath(ctx, pathSlug)
	if err != nil {
		return nil, fmt.Errorf("list problems by path: %w", err)
	}
	out := make([]Problem, 0, len(rows))
	for _, r := range rows {
		p, err := indexProblem(r)
		if err != nil {
			return nil, fmt.Errorf("list problems by path: %w", err)
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *PgStore) GetProblem(ctx context.Context, id string) (Problem, error) {
	r, err := s.q.GetProblem(ctx, id)
	if err != nil {
		return Problem{}, mapErr(err)
	}
	return detailProblem(r)
}

// GetProblemsByIDs resolves many problems in one query (the gateway's due-queue /
// mistake-journal enrichment). An empty id list short-circuits to no rows; unknown
// ids are simply absent from the result (not an error), so the caller maps what it
// got and degrades the rest.
func (s *PgStore) GetProblemsByIDs(ctx context.Context, ids []string) ([]Problem, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.q.GetProblemsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get problems by ids: %w", err)
	}
	out := make([]Problem, 0, len(rows))
	for _, r := range rows {
		// The bulk read selects the same columns as GetProblem.
		p, err := detailProblem(gen.GetProblemRow(r))
		if err != nil {
			return nil, fmt.Errorf("get problems by ids: %w", err)
		}
		out = append(out, p)
	}
	return out, nil
}

// indexProblem maps an index row (the week and path lists).
func indexProblem(r gen.ListProblemsByPathRow) (Problem, error) {
	links, err := decodeLinks(r.Links)
	if err != nil {
		return Problem{}, fmt.Errorf("problem %s: links: %w", r.ID, err)
	}
	return Problem{
		ID:         r.ID,
		PathSlug:   r.PathSlug,
		WeekN:      int(r.WeekN),
		Title:      r.Title,
		Difficulty: r.Difficulty,
		Pattern:    r.Pattern,
		Role:       r.Role,
		Links:      links,
	}, nil
}

// detailProblem maps a by-id row: the index columns plus the contract fields.
func detailProblem(r gen.GetProblemRow) (Problem, error) {
	links, err := decodeLinks(r.Links)
	if err != nil {
		return Problem{}, fmt.Errorf("problem %s: links: %w", r.ID, err)
	}
	return Problem{
		ID:             r.ID,
		PathSlug:       r.PathSlug,
		WeekN:          int(r.WeekN),
		Title:          r.Title,
		Difficulty:     r.Difficulty,
		Pattern:        r.Pattern,
		Role:           r.Role,
		Links:          links,
		ContractHash:   r.ContractHash,
		GradingSummary: json.RawMessage(r.GradingSummary),
	}, nil
}

// decodeLinks decodes problem.links (a jsonb array of {kind, url}); never nil.
func decodeLinks(b []byte) ([]Link, error) {
	var out []Link
	if len(b) > 0 {
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, err
		}
	}
	if out == nil {
		out = []Link{}
	}
	return out, nil
}

func (s *PgStore) ListSections(ctx context.Context, problemID string) ([]Section, error) {
	rows, err := s.q.ListSectionsByProblem(ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("list sections: %w", err)
	}
	out := make([]Section, 0, len(rows))
	for _, r := range rows {
		out = append(out, Section{
			Stage:    r.Stage,
			Kind:     r.Kind,
			Order:    int(r.Order),
			BodyMD:   r.BodyMd,
			Code:     r.Code,
			Language: r.Language,
		})
	}
	return out, nil
}

func (s *PgStore) GetConcept(ctx context.Context, pathSlug, slug string) (Concept, error) {
	r, err := s.q.GetConcept(ctx, gen.GetConceptParams{PathSlug: pathSlug, Slug: slug})
	if err != nil {
		return Concept{}, mapErr(err)
	}
	var templates map[string]string
	if len(r.Templates) > 0 {
		if err := json.Unmarshal(r.Templates, &templates); err != nil {
			return Concept{}, fmt.Errorf("concept %s/%s: templates: %w", pathSlug, slug, err)
		}
	}
	if templates == nil {
		templates = map[string]string{}
	}
	return Concept{
		Slug:        r.Slug,
		PathSlug:    r.PathSlug,
		Title:       r.Title,
		BodyMD:      r.BodyMd,
		WhenToUseMD: r.WhenToUseMd,
		Templates:   templates,
	}, nil
}

func (s *PgStore) CountProblems(ctx context.Context, pathSlug string) (int, error) {
	n, err := s.q.CountProblemsByPath(ctx, pathSlug)
	if err != nil {
		return 0, fmt.Errorf("count problems: %w", err)
	}
	return int(n), nil
}

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
