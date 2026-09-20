package curriculum

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

// seedFiles maps each JSON seed file (relative to the embedded curriculum/ root) to
// the SeedContent slice it fills. The DSA path's content lives under dsa/; paths.json
// holds every path row (the active DSA path + the coming-soon stubs) so Catalog is
// fully API-driven.
const (
	pathsFile    = "paths.json"
	phasesFile   = "dsa/phases.json"
	weeksFile    = "dsa/weeks.json"
	conceptsFile = "dsa/concepts.json"
	problemsFile = "dsa/problems.json"
)

// Seed parses the versioned curriculum files from fsys and upserts them idempotently
// (store.SeedAll runs one transaction of natural-key upserts). It logs the seeded
// problem count against each path's declared target (the 151 goal for DSA) so a
// partial sample seed is visible in the logs. Called on startup; the caller refuses
// to serve on a non-nil error (a content service with no content is broken).
func Seed(ctx context.Context, st store.Store, fsys fs.FS, log *slog.Logger) error {
	var content store.SeedContent

	if err := readJSON(fsys, pathsFile, &content.Paths); err != nil {
		return err
	}
	if err := readJSON(fsys, phasesFile, &content.Phases); err != nil {
		return err
	}
	if err := readJSON(fsys, weeksFile, &content.Weeks); err != nil {
		return err
	}
	if err := readJSON(fsys, conceptsFile, &content.Concepts); err != nil {
		return err
	}
	if err := readJSON(fsys, problemsFile, &content.Problems); err != nil {
		return err
	}

	if err := st.SeedAll(ctx, content); err != nil {
		return fmt.Errorf("apply seed: %w", err)
	}

	log.Info("curriculum seeded",
		"paths", len(content.Paths),
		"phases", len(content.Phases),
		"weeks", len(content.Weeks),
		"concepts", len(content.Concepts),
		"problems_in_seed", len(content.Problems),
	)

	// Report seeded vs the declared target per path (e.g. DSA: sample set / 151).
	for _, p := range content.Paths {
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

// readJSON reads and strictly decodes a single seed file into v.
func readJSON(fsys fs.FS, name string, v any) error {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("read seed file %s: %w", name, err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode seed file %s: %w", name, err)
	}
	return nil
}
