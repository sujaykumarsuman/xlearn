package course

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
)

// ManifestGlob is where manifests live inside the curriculum FS (ADR-0027 §2).
const ManifestGlob = "courses/*/course.json"

// PathsFile is the v1 catalog file every manifest must agree with until m1-09's loader
// owns the path rows (one catalog truth).
const PathsFile = "paths.json"

// Load reads every `courses/*/course.json` in fsys (the embedded curriculum FS, rooted
// at curriculum/), strictly decodes each one (unknown fields and trailing data are
// errors), validates it, and checks the set against `paths.json`. It returns the
// manifests keyed by slug.
func Load(fsys fs.FS) (map[string]*Manifest, error) {
	names, err := fs.Glob(fsys, ManifestGlob)
	if err != nil {
		return nil, fmt.Errorf("course: glob %s: %w", ManifestGlob, err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("course: no manifests match %s", ManifestGlob)
	}
	sort.Strings(names)

	out := make(map[string]*Manifest, len(names))
	var errs []error
	for _, name := range names {
		b, err := fs.ReadFile(fsys, name)
		if err != nil {
			errs = append(errs, fmt.Errorf("course: read %s: %w", name, err))
			continue
		}
		m, err := DecodeManifest(b)
		if err != nil {
			errs = append(errs, fmt.Errorf("course: %s: %w", name, err))
			continue
		}
		if dir := path.Base(path.Dir(name)); dir != m.Slug {
			errs = append(errs, fmt.Errorf("course: %s: slug %q does not match its directory %q", name, m.Slug, dir))
			continue
		}
		if err := m.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("course: %s: %w", name, err))
			continue
		}
		out[m.Slug] = m
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	if err := validateSet(out); err != nil {
		return nil, err
	}

	rows, err := readPaths(fsys)
	if err != nil {
		return nil, err
	}
	if err := checkPaths(out, rows); err != nil {
		return nil, err
	}
	return out, nil
}

// DecodeManifest strictly decodes one manifest: unknown fields and any trailing data
// after the JSON value are errors. It does not validate.
func DecodeManifest(b []byte) (*Manifest, error) {
	var m Manifest
	if err := decodeStrict(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// DecodeItem strictly decodes one item.json: unknown fields and any trailing data after
// the JSON value are errors. It does not validate.
func DecodeItem(b []byte) (*Item, error) {
	var it Item
	if err := decodeStrict(b, &it); err != nil {
		return nil, err
	}
	return &it, nil
}

// decodeStrict decodes exactly one JSON value into v, rejecting unknown fields and
// trailing data.
func decodeStrict(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("decode: trailing data after the JSON value")
	}
	return nil
}

// validateSet checks rules that span manifests: id_prefix is unique.
func validateSet(ms map[string]*Manifest) error {
	byPrefix := make(map[string]string, len(ms))
	slugs := make([]string, 0, len(ms))
	for s := range ms {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)
	var errs []error
	for _, s := range slugs {
		p := ms[s].IDPrefix
		if other, dup := byPrefix[p]; dup {
			errs = append(errs, fmt.Errorf("course: id_prefix %q is used by both %q and %q", p, other, s))
			continue
		}
		byPrefix[p] = s
	}
	return errors.Join(errs...)
}

// pathRow is the subset of a paths.json row the consistency check compares.
type pathRow struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

func readPaths(fsys fs.FS) ([]pathRow, error) {
	b, err := fs.ReadFile(fsys, PathsFile)
	if err != nil {
		return nil, fmt.Errorf("course: read %s: %w", PathsFile, err)
	}
	// paths.json carries more fields (summary, totals, sort order) that the manifest does
	// not restate, so this decode is deliberately not strict.
	var rows []pathRow
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil, fmt.Errorf("course: decode %s: %w", PathsFile, err)
	}
	return rows, nil
}

// checkPaths keeps the two catalog sources equal: every paths.json row has a manifest
// with the same slug, title and status, and every manifest has a row.
func checkPaths(ms map[string]*Manifest, rows []pathRow) error {
	var errs []error
	seen := make(map[string]bool, len(rows))
	for _, r := range rows {
		seen[r.Slug] = true
		m, ok := ms[r.Slug]
		if !ok {
			errs = append(errs, fmt.Errorf("course: %s row %q has no manifest", PathsFile, r.Slug))
			continue
		}
		if m.Title != r.Title {
			errs = append(errs, fmt.Errorf("course: %s: title %q != %s title %q", r.Slug, m.Title, PathsFile, r.Title))
		}
		if m.Status != r.Status {
			errs = append(errs, fmt.Errorf("course: %s: status %q != %s status %q", r.Slug, m.Status, PathsFile, r.Status))
		}
	}
	slugs := make([]string, 0, len(ms))
	for s := range ms {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)
	for _, s := range slugs {
		if !seen[s] {
			errs = append(errs, fmt.Errorf("course: manifest %q has no %s row", s, PathsFile))
		}
	}
	return errors.Join(errs...)
}
