package curriculum

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

// Seed loads the versioned curriculum from fsys (the glob loader, LoadContent: every
// course under courses/, strictly decoded, id and slug guards applied) and applies it in
// one transaction (store.SeedAll: upserts, delete-missing per course, sections rewritten,
// re-parenting aborts). It logs a WARN for every item it had to retire because the seed
// no longer carries it, and the seeded problem count against each path's declared target
// (the 151 goal for DSA). Called on startup; the caller refuses to serve on a non-nil
// error (a content service with no content is broken).
func Seed(ctx context.Context, st store.Store, fsys fs.FS, log *slog.Logger) error {
	content, err := LoadContent(fsys)
	if err != nil {
		return fmt.Errorf("load curriculum: %w", err)
	}
	seed, err := content.SeedContent()
	if err != nil {
		return fmt.Errorf("resolve curriculum: %w", err)
	}

	report, err := st.SeedAll(ctx, seed)
	if err != nil {
		return fmt.Errorf("apply seed: %w", err)
	}
	for _, r := range report.Retired {
		log.Warn("curriculum item missing from the seed; retired (ids.lock.json should prevent this)",
			"id", r.ID, "path", r.PathSlug)
	}

	log.Info("curriculum seeded",
		"courses", len(seed.Courses),
		"paths", len(seed.Paths),
		"phases", len(seed.Phases),
		"weeks", len(seed.Weeks),
		"concepts", len(seed.Concepts),
		"problems_in_seed", len(seed.Problems),
		"retired_missing", len(report.Retired),
	)

	// Report seeded (live) vs the declared target per path (e.g. DSA: sample set / 151).
	for _, p := range seed.Paths {
		if p.ProblemTotal == 0 {
			continue
		}
		n, err := st.CountProblems(ctx, p.Slug)
		if err != nil {
			return fmt.Errorf("count problems for %q: %w", p.Slug, err)
		}
		log.Info("curriculum seed coverage", "path", p.Slug, "seeded", n, "target", p.ProblemTotal)
	}
	return nil
}
