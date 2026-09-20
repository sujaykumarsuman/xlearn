package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store/gen"
)

// SeedContent is the fully-parsed versioned curriculum seed (the files under
// curriculum/). The JSON tags let the loader unmarshal the seed files straight
// into these types; SeedAll upserts them idempotently in one transaction.
type SeedContent struct {
	Paths    []SeedPath    `json:"paths"`
	Phases   []SeedPhase   `json:"phases"`
	Weeks    []SeedWeek    `json:"weeks"`
	Concepts []SeedConcept `json:"concepts"`
	Problems []SeedProblem `json:"problems"`
}

// SeedPath is one path row (Catalog card metadata).
type SeedPath struct {
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	Summary      string `json:"summary"`
	ProblemTotal int    `json:"problem_total"`
	WeekTotal    int    `json:"week_total"`
	SortOrder    int    `json:"sort_order"`
}

// SeedPhase is one phase (a contiguous week range with a name + theme).
type SeedPhase struct {
	PathSlug string `json:"path_slug"`
	Order    int    `json:"order"`
	Name     string `json:"name"`
	Theme    string `json:"theme"`
	WeekFrom int    `json:"week_from"`
	WeekTo   int    `json:"week_to"`
}

// SeedWeek is one week (title + thesis).
type SeedWeek struct {
	PathSlug string `json:"path_slug"`
	N        int    `json:"n"`
	Title    string `json:"title"`
	Thesis   string `json:"thesis"`
}

// SeedConcept is one concept/pattern reading, plus the week numbers it links to.
type SeedConcept struct {
	PathSlug     string `json:"path_slug"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	BodyMD       string `json:"body_md"`
	WhenToUseMD  string `json:"when_to_use_md"`
	CodeTemplate string `json:"code_template"`
	Weeks        []int  `json:"weeks"`
}

// SeedProblem is one problem plus its stage-scoped content sections.
type SeedProblem struct {
	ID              string        `json:"id"`
	PathSlug        string        `json:"path_slug"`
	WeekN           int           `json:"week_n"`
	Title           string        `json:"title"`
	Difficulty      string        `json:"difficulty"`
	Pattern         string        `json:"pattern"`
	LeetcodeURL     string        `json:"leetcode_url"`
	NeetcodeURL     string        `json:"neetcode_url"`
	IsReinforcement bool          `json:"is_reinforcement"`
	SortOrder       int           `json:"sort_order"`
	Sections        []SeedSection `json:"sections"`
}

// SeedSection is one stage-scoped content section.
type SeedSection struct {
	Stage  string `json:"stage"`
	Kind   string `json:"kind"`
	Order  int    `json:"order"`
	BodyMD string `json:"body_md"`
	Code   string `json:"code"`
}

// SeedAll applies the whole seed in one transaction. Every write is an upsert on a
// natural key, so re-running on every boot (or from multiple replicas) never
// duplicates rows. Paths are upserted first (phases/weeks/concepts/problems FK them),
// then week<->concept links are resolved from the just-upserted ids.
func (s *PgStore) SeedAll(ctx context.Context, content SeedContent) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)

	for _, p := range content.Paths {
		if err := q.UpsertPath(ctx, gen.UpsertPathParams{
			Slug:         p.Slug,
			Title:        p.Title,
			Status:       p.Status,
			Summary:      p.Summary,
			ProblemTotal: int32(p.ProblemTotal),
			WeekTotal:    int32(p.WeekTotal),
			SortOrder:    int32(p.SortOrder),
		}); err != nil {
			return fmt.Errorf("upsert path %q: %w", p.Slug, err)
		}
	}

	for _, ph := range content.Phases {
		if err := q.UpsertPhase(ctx, gen.UpsertPhaseParams{
			PathSlug: ph.PathSlug,
			Order:    int32(ph.Order),
			Name:     ph.Name,
			Theme:    ph.Theme,
			WeekFrom: int32(ph.WeekFrom),
			WeekTo:   int32(ph.WeekTo),
		}); err != nil {
			return fmt.Errorf("upsert phase %s/%d: %w", ph.PathSlug, ph.Order, err)
		}
	}

	// weekID maps (path_slug, n) -> the week's uuid so week_concept can be linked.
	weekID := make(map[weekKey]pgtype.UUID, len(content.Weeks))
	for _, w := range content.Weeks {
		id, err := q.UpsertWeek(ctx, gen.UpsertWeekParams{
			PathSlug: w.PathSlug,
			N:        int32(w.N),
			Title:    w.Title,
			Thesis:   w.Thesis,
		})
		if err != nil {
			return fmt.Errorf("upsert week %s/%d: %w", w.PathSlug, w.N, err)
		}
		weekID[weekKey{w.PathSlug, w.N}] = id
	}

	for _, c := range content.Concepts {
		conceptID, err := q.UpsertConcept(ctx, gen.UpsertConceptParams{
			PathSlug:     c.PathSlug,
			Slug:         c.Slug,
			Title:        c.Title,
			BodyMd:       c.BodyMD,
			WhenToUseMd:  c.WhenToUseMD,
			CodeTemplate: c.CodeTemplate,
		})
		if err != nil {
			return fmt.Errorf("upsert concept %q: %w", c.Slug, err)
		}
		for _, n := range c.Weeks {
			wid, ok := weekID[weekKey{c.PathSlug, n}]
			if !ok {
				return fmt.Errorf("concept %q links unknown week %s/%d", c.Slug, c.PathSlug, n)
			}
			if err := q.LinkWeekConcept(ctx, gen.LinkWeekConceptParams{WeekID: wid, ConceptID: conceptID}); err != nil {
				return fmt.Errorf("link week %s/%d <-> concept %q: %w", c.PathSlug, n, c.Slug, err)
			}
		}
	}

	for _, pr := range content.Problems {
		if err := q.UpsertProblem(ctx, gen.UpsertProblemParams{
			ID:              pr.ID,
			PathSlug:        pr.PathSlug,
			WeekN:           int32(pr.WeekN),
			Title:           pr.Title,
			Difficulty:      pr.Difficulty,
			Pattern:         pr.Pattern,
			LeetcodeUrl:     pr.LeetcodeURL,
			NeetcodeUrl:     pr.NeetcodeURL,
			IsReinforcement: pr.IsReinforcement,
			SortOrder:       int32(pr.SortOrder),
		}); err != nil {
			return fmt.Errorf("upsert problem %q: %w", pr.ID, err)
		}
		for _, sec := range pr.Sections {
			if err := q.UpsertSection(ctx, gen.UpsertSectionParams{
				ProblemID: pr.ID,
				Stage:     sec.Stage,
				Kind:      sec.Kind,
				Order:     int32(sec.Order),
				BodyMd:    sec.BodyMD,
				Code:      sec.Code,
			}); err != nil {
				return fmt.Errorf("upsert section %s/%s/%d: %w", pr.ID, sec.Stage, sec.Order, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed tx: %w", err)
	}
	return nil
}

// weekKey identifies a week by its natural key (path_slug, n).
type weekKey struct {
	pathSlug string
	n        int
}
