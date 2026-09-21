// Package store is the curriculum service's persistence layer: pgx/pgxpool over the
// sqlc-generated queries in ./gen and goose migrations in ./migrations (ADR-0005).
// It exposes small domain types with plain Go scalars so the HTTP layer never
// touches pgtype, keeps the xlearn_curriculum role scoped to schema curriculum (all
// SQL is schema-qualified), and applies the versioned seed idempotently in one
// transaction (SeedAll). Curriculum emits no events, so there is no outbox.
package store

import (
	"context"
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

// Concept is a full concept/pattern reading + code template.
type Concept struct {
	Slug         string
	PathSlug     string
	Title        string
	BodyMD       string
	WhenToUseMD  string
	CodeTemplate string
}

// Problem is a problem's metadata (natural id, difficulty drives the UI tokens).
type Problem struct {
	ID              string
	PathSlug        string
	WeekN           int
	Title           string
	Difficulty      string // "easy" | "med" | "hard"
	Pattern         string
	LeetcodeURL     string
	NeetcodeURL     string
	IsReinforcement bool
}

// Section is one stage-scoped content section of a problem.
type Section struct {
	Stage  string // "attempt" | "hint" | "solution"
	Kind   string
	Order  int
	BodyMD string
	Code   string
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
	ListSections(ctx context.Context, problemID string) ([]Section, error)
	GetConcept(ctx context.Context, slug string) (Concept, error)
	CountProblems(ctx context.Context, pathSlug string) (int, error)
	// SeedAll upserts the entire versioned seed in one transaction, idempotently
	// (ON CONFLICT ... DO UPDATE on natural keys) so re-running on every boot never
	// duplicates rows.
	SeedAll(ctx context.Context, content SeedContent) error
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
		out = append(out, Problem{
			ID:              r.ID,
			PathSlug:        r.PathSlug,
			WeekN:           int(r.WeekN),
			Title:           r.Title,
			Difficulty:      r.Difficulty,
			Pattern:         r.Pattern,
			LeetcodeURL:     r.LeetcodeUrl,
			NeetcodeURL:     r.NeetcodeUrl,
			IsReinforcement: r.IsReinforcement,
		})
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
		out = append(out, Problem{
			ID:              r.ID,
			PathSlug:        r.PathSlug,
			WeekN:           int(r.WeekN),
			Title:           r.Title,
			Difficulty:      r.Difficulty,
			Pattern:         r.Pattern,
			LeetcodeURL:     r.LeetcodeUrl,
			NeetcodeURL:     r.NeetcodeUrl,
			IsReinforcement: r.IsReinforcement,
		})
	}
	return out, nil
}

func (s *PgStore) GetProblem(ctx context.Context, id string) (Problem, error) {
	r, err := s.q.GetProblem(ctx, id)
	if err != nil {
		return Problem{}, mapErr(err)
	}
	return Problem{
		ID:              r.ID,
		PathSlug:        r.PathSlug,
		WeekN:           int(r.WeekN),
		Title:           r.Title,
		Difficulty:      r.Difficulty,
		Pattern:         r.Pattern,
		LeetcodeURL:     r.LeetcodeUrl,
		NeetcodeURL:     r.NeetcodeUrl,
		IsReinforcement: r.IsReinforcement,
	}, nil
}

func (s *PgStore) ListSections(ctx context.Context, problemID string) ([]Section, error) {
	rows, err := s.q.ListSectionsByProblem(ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("list sections: %w", err)
	}
	out := make([]Section, 0, len(rows))
	for _, r := range rows {
		out = append(out, Section{
			Stage:  r.Stage,
			Kind:   r.Kind,
			Order:  int(r.Order),
			BodyMD: r.BodyMd,
			Code:   r.Code,
		})
	}
	return out, nil
}

func (s *PgStore) GetConcept(ctx context.Context, slug string) (Concept, error) {
	r, err := s.q.GetConcept(ctx, slug)
	if err != nil {
		return Concept{}, mapErr(err)
	}
	return Concept{
		Slug:         r.Slug,
		PathSlug:     r.PathSlug,
		Title:        r.Title,
		BodyMD:       r.BodyMd,
		WhenToUseMD:  r.WhenToUseMd,
		CodeTemplate: r.CodeTemplate,
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
