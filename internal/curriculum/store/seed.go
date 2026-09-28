package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store/gen"
)

// SeedContent is the fully resolved curriculum seed: what internal/curriculum.LoadContent
// reads from the embedded curriculum/ files (sprint m1-09), flattened to rows. SeedAll
// applies it in one transaction.
type SeedContent struct {
	// Courses are the course slugs whose content this seed fully describes: phases,
	// weeks, week<->concept links and concepts are delete-missing per course in this
	// list. Problems are never deleted (a missing one is retired).
	Courses  []string
	Paths    []SeedPath
	Phases   []SeedPhase
	Weeks    []SeedWeek
	Concepts []SeedConcept
	Problems []SeedProblem
}

// SeedPath is one path row (Catalog card metadata) plus its manifest's id prefix.
type SeedPath struct {
	Slug         string
	Title        string
	Status       string
	Summary      string
	ProblemTotal int
	WeekTotal    int
	SortOrder    int
	IDPrefix     string
}

// SeedPhase is one phase (a contiguous week range with a name + theme).
type SeedPhase struct {
	PathSlug string
	Order    int
	Name     string
	Theme    string
	WeekFrom int
	WeekTo   int
}

// SeedWeek is one week (title + thesis).
type SeedWeek struct {
	PathSlug string
	N        int
	Title    string
	Thesis   string
}

// SeedConcept is one concept/pattern reading, plus the week numbers it links to.
type SeedConcept struct {
	PathSlug    string
	Slug        string
	Title       string
	BodyMD      string
	WhenToUseMD string
	// Templates maps a language to its code template; templates["go"] is dual-written
	// to the v1 code_template column until M1c.
	Templates map[string]string
	Weeks     []int
}

// SeedProblem is one item plus its stage-scoped content sections.
type SeedProblem struct {
	ID         string
	PathSlug   string
	WeekN      int
	Title      string
	Difficulty string
	Pattern    string
	// Role is core | reinforcement | drill; is_reinforcement is dual-written from it.
	Role string
	// Status is live | retired | withdrawn. Retired items leave the index and counts;
	// withdrawn items also serve no sections.
	Status string
	// Links are the outbound links; the leetcode and neetcode ones are dual-written to
	// the v1 URL columns.
	Links       []SeedLink
	SortOrder   int
	ContentHash string
	// ContractHash is canon.ContractHash ("" on the self path); GradingSummary is the
	// derived, answer-free JSON object ({"mode":"self"} on the self path). m3-01.
	ContractHash   string
	GradingSummary json.RawMessage
	Sections       []SeedSection
}

// SeedLink is one outbound link ({kind, url}), stored in problem.links.
type SeedLink struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

// SeedSection is one stage-scoped content section. Language is the code language of a
// code section ("" for prose).
type SeedSection struct {
	Stage    string
	Kind     string
	Order    int
	Language string
	BodyMD   string
	Code     string
}

// SeedReport says what a seed did beyond upserting.
type SeedReport struct {
	// Retired are live items that were in the database but not in the seed; SeedAll
	// retired them defensively (the caller logs a WARN for each).
	Retired []RetiredProblem
}

// RetiredProblem is one defensively retired item.
type RetiredProblem struct {
	ID       string
	PathSlug string
}

// ErrReparent is returned (wrapped) when the seed would move an existing item to another
// course. Ids are never re-parented (ADR-0026 §2); the whole seed rolls back.
var ErrReparent = errors.New("curriculum: an item id would move to another course")

// SeedAll applies the whole seed in one transaction (sprint m1-09 semantics):
//   - paths, phases, weeks and concepts are upserted on their natural keys; phases,
//     weeks, week<->concept links and concepts missing from the seed are deleted per
//     course in content.Courses;
//   - each problem is upserted under the id guard (exactly 1 row, else ErrReparent and a
//     rollback), then ALL its sections are deleted and re-inserted (a withdrawn item gets
//     none). content_hash is written but never used to skip: the v1.5.2 image rewrites
//     section bodies without touching it;
//   - live problems missing from the seed are retired, never deleted;
//   - the v1 columns (is_reinforcement, leetcode_url, neetcode_url, code_template) are
//     dual-written from the v2 fields until M1c.
func (s *PgStore) SeedAll(ctx context.Context, content SeedContent) (SeedReport, error) {
	var report SeedReport
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return report, fmt.Errorf("begin seed tx: %w", err)
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
			IDPrefix:     pgtype.Text{String: p.IDPrefix, Valid: p.IDPrefix != ""},
		}); err != nil {
			return report, fmt.Errorf("upsert path %q: %w", p.Slug, err)
		}
	}

	// Delete-missing per course (content-only tables). Keep-lists are never nil: a NULL
	// array would make NOT (x = ANY(NULL)) NULL and delete nothing.
	for _, c := range content.Courses {
		phases, weeks, concepts := []int32{}, []int32{}, []string{}
		for _, ph := range content.Phases {
			if ph.PathSlug == c {
				phases = append(phases, int32(ph.Order))
			}
		}
		for _, w := range content.Weeks {
			if w.PathSlug == c {
				weeks = append(weeks, int32(w.N))
			}
		}
		for _, cn := range content.Concepts {
			if cn.PathSlug == c {
				concepts = append(concepts, cn.Slug)
			}
		}
		if err := q.DeleteMissingPhases(ctx, gen.DeleteMissingPhasesParams{PathSlug: c, Keep: phases}); err != nil {
			return report, fmt.Errorf("delete missing phases of %q: %w", c, err)
		}
		// Links are rebuilt from the concepts' weeks below.
		if err := q.DeleteWeekConceptsByPath(ctx, c); err != nil {
			return report, fmt.Errorf("unlink week concepts of %q: %w", c, err)
		}
		if err := q.DeleteMissingWeeks(ctx, gen.DeleteMissingWeeksParams{PathSlug: c, Keep: weeks}); err != nil {
			return report, fmt.Errorf("delete missing weeks of %q: %w", c, err)
		}
		if err := q.DeleteMissingConcepts(ctx, gen.DeleteMissingConceptsParams{PathSlug: c, Keep: concepts}); err != nil {
			return report, fmt.Errorf("delete missing concepts of %q: %w", c, err)
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
			return report, fmt.Errorf("upsert phase %s/%d: %w", ph.PathSlug, ph.Order, err)
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
			return report, fmt.Errorf("upsert week %s/%d: %w", w.PathSlug, w.N, err)
		}
		weekID[weekKey{w.PathSlug, w.N}] = id
	}

	for _, c := range content.Concepts {
		templates, err := jsonOrEmpty(c.Templates, "{}")
		if err != nil {
			return report, fmt.Errorf("concept %q templates: %w", c.Slug, err)
		}
		conceptID, err := q.UpsertConcept(ctx, gen.UpsertConceptParams{
			PathSlug:     c.PathSlug,
			Slug:         c.Slug,
			Title:        c.Title,
			BodyMd:       c.BodyMD,
			WhenToUseMd:  c.WhenToUseMD,
			CodeTemplate: pgtype.Text{String: c.Templates["go"], Valid: true},
			Templates:    templates,
		})
		if err != nil {
			return report, fmt.Errorf("upsert concept %s/%s: %w", c.PathSlug, c.Slug, err)
		}
		for _, n := range c.Weeks {
			wid, ok := weekID[weekKey{c.PathSlug, n}]
			if !ok {
				return report, fmt.Errorf("concept %q links unknown week %s/%d", c.Slug, c.PathSlug, n)
			}
			if err := q.LinkWeekConcept(ctx, gen.LinkWeekConceptParams{WeekID: wid, ConceptID: conceptID}); err != nil {
				return report, fmt.Errorf("link week %s/%d <-> concept %q: %w", c.PathSlug, n, c.Slug, err)
			}
		}
	}

	seeded := make([]string, 0, len(content.Problems))
	for _, pr := range content.Problems {
		links, err := jsonOrEmpty(pr.Links, "[]")
		if err != nil {
			return report, fmt.Errorf("problem %q links: %w", pr.ID, err)
		}
		summary := []byte(pr.GradingSummary)
		if len(summary) == 0 {
			summary = []byte("{}")
		}
		n, err := q.UpsertProblem(ctx, gen.UpsertProblemParams{
			ID:              pr.ID,
			PathSlug:        pr.PathSlug,
			WeekN:           int32(pr.WeekN),
			Title:           pr.Title,
			Difficulty:      pr.Difficulty,
			Pattern:         pr.Pattern,
			LeetcodeUrl:     pgtype.Text{String: linkURL(pr.Links, "leetcode"), Valid: true},
			NeetcodeUrl:     pgtype.Text{String: linkURL(pr.Links, "neetcode"), Valid: true},
			IsReinforcement: pgtype.Bool{Bool: pr.Role == "reinforcement", Valid: true},
			SortOrder:       int32(pr.SortOrder),
			Role:            pr.Role,
			Status:          pr.Status,
			Links:           links,
			ContentHash:     pr.ContentHash,
			ContractHash:    pr.ContractHash,
			GradingSummary:  summary,
		})
		if err != nil {
			return report, fmt.Errorf("upsert problem %q: %w", pr.ID, err)
		}
		if n != 1 {
			return report, fmt.Errorf("upsert problem %q into course %q affected %d rows: %w", pr.ID, pr.PathSlug, n, ErrReparent)
		}
		seeded = append(seeded, pr.ID)

		// Every seed rewrites every item's sections.
		if err := q.DeleteSectionsByProblem(ctx, pr.ID); err != nil {
			return report, fmt.Errorf("delete sections of %q: %w", pr.ID, err)
		}
		if pr.Status == "withdrawn" {
			continue // a takedown: prose blanked
		}
		for _, sec := range pr.Sections {
			if err := q.InsertSection(ctx, gen.InsertSectionParams{
				ProblemID: pr.ID,
				Stage:     sec.Stage,
				Kind:      sec.Kind,
				Order:     int32(sec.Order),
				Language:  sec.Language,
				BodyMd:    sec.BodyMD,
				Code:      sec.Code,
			}); err != nil {
				return report, fmt.Errorf("insert section %s/%s/%d: %w", pr.ID, sec.Stage, sec.Order, err)
			}
		}
	}

	retired, err := q.RetireMissingProblems(ctx, seeded)
	if err != nil {
		return report, fmt.Errorf("retire missing problems: %w", err)
	}
	for _, r := range retired {
		report.Retired = append(report.Retired, RetiredProblem{ID: r.ID, PathSlug: r.PathSlug})
	}

	if err := tx.Commit(ctx); err != nil {
		return SeedReport{}, fmt.Errorf("commit seed tx: %w", err)
	}
	return report, nil
}

// linkURL returns the first link of a kind ("" if none): the dual-write source of the
// v1 leetcode_url / neetcode_url columns.
func linkURL(links []SeedLink, kind string) string {
	for _, l := range links {
		if l.Kind == kind {
			return l.URL
		}
	}
	return ""
}

// jsonOrEmpty marshals v for a jsonb column, or returns empty (e.g. "[]", "{}") when v
// has no elements.
func jsonOrEmpty[T any](v T, empty string) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" || string(b) == empty {
		return []byte(empty), nil
	}
	return b, nil
}

// weekKey identifies a week by its natural key (path_slug, n).
type weekKey struct {
	pathSlug string
	n        int
}
