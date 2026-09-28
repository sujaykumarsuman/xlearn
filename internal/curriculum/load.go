package curriculum

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

// The glob loader (sprint m1-09; t0 §3, t1 §3.2): it reads the embedded curriculum FS
// (rooted at curriculum/) course by course, strictly decodes every file, inlines the
// sidecars and applies the id and slug guards. It is pure (no database), so contentlint
// runs the same checks in CI that the seed runs at startup.
//
//	paths.json  ids.lock.json
//	courses/<slug>/course.json                         (course.Load)
//	courses/<slug>/{phases,weeks,concepts}.json        optional; rows without a course field
//	courses/<slug>/concepts/<slug>.md | <slug>.when.md | _code/<slug>.<lang>.snip
//	courses/<slug>/items/<id>/item.json                (course.Item + ValidateFor)
//	courses/<slug>/items/<id>/sections/<stage>/<NN>-<kind>.md
//	courses/<slug>/items/<id>/_code/<stage>-<NN>.<lang>.snip
//
// Dotfiles (any path segment starting with ".") are ignored; any other unexpected file
// is an error.

// Content is the whole curriculum, loaded, strictly decoded and guarded.
type Content struct {
	// Manifests by course slug (course.Load: validated, consistent with paths.json).
	Manifests map[string]*course.Manifest
	// Paths are the catalog rows of paths.json.
	Paths []course.PathRow
	// Lock is ids.lock.json.
	Lock course.IDsLock
	// Courses are every course's content, sorted by slug.
	Courses []CourseContent
}

// CourseContent is one course's content files, resolved.
type CourseContent struct {
	Slug     string
	Phases   []course.Phase
	Weeks    []course.Week
	Concepts []course.ResolvedConcept
	// Items are sorted by id (their directory order).
	Items []course.ResolvedItem
}

var (
	// sectionFileRe: sections/<stage>/<NN>-<kind>.md, NN = the section's order (01..99).
	sectionFileRe = regexp.MustCompile(`^(0[1-9]|[1-9][0-9])-([a-z][a-z_]*)\.md$`)
	// codeFileRe: _code/<stage>-<NN>.<lang>.snip (kind "code"; language from the extension).
	codeFileRe = regexp.MustCompile(`^(attempt|hint|solution)-(0[1-9]|[1-9][0-9])\.([a-z][a-z0-9]*)\.snip$`)
	// conceptSlugRe is a concept slug; concept sidecars are named after it.
	conceptSlugRe      = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	conceptBodyRe      = regexp.MustCompile(`^([a-z0-9]+(?:-[a-z0-9]+)*)\.md$`)
	conceptWhenRe      = regexp.MustCompile(`^([a-z0-9]+(?:-[a-z0-9]+)*)\.when\.md$`)
	conceptTemplateRe  = regexp.MustCompile(`^([a-z0-9]+(?:-[a-z0-9]+)*)\.([a-z][a-z0-9]*)\.snip$`)
	courseDirEntries   = map[string]bool{"course.json": true, course.PhasesFile: true, course.WeeksFile: true, course.ConceptsFile: true, "concepts": true, "items": true}
	errNotDirectoryish = errors.New("expected a directory")
)

// LoadContent reads and guards the whole curriculum in fsys. Every error names the file.
func LoadContent(fsys fs.FS) (*Content, error) {
	manifests, err := course.Load(fsys)
	if err != nil {
		return nil, err
	}
	c := &Content{Manifests: manifests}
	if err := readStrict(fsys, course.PathsFile, &c.Paths); err != nil {
		return nil, err
	}
	if err := readStrict(fsys, course.LockFile, &c.Lock); err != nil {
		return nil, err
	}
	if c.Lock.Items == nil {
		return nil, fmt.Errorf("%s: items is required", course.LockFile)
	}

	slugs := make([]string, 0, len(manifests))
	for s := range manifests {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)

	var errs []error
	owner := map[string]string{} // item id -> course (ids are unique across courses)
	for _, slug := range slugs {
		if err := course.CheckCourseSlug(slug); err != nil {
			errs = append(errs, fmt.Errorf("courses/%s: %w", slug, err))
			continue
		}
		cc, err := loadCourse(fsys, manifests[slug], c.Lock, owner)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		c.Courses = append(c.Courses, *cc)
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return c, nil
}

func loadCourse(fsys fs.FS, m *course.Manifest, lock course.IDsLock, owner map[string]string) (*CourseContent, error) {
	dir := path.Join("courses", m.Slug)
	cc := &CourseContent{Slug: m.Slug}
	var errs []error

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	for _, e := range entries {
		if isDot(e.Name()) {
			continue
		}
		if !courseDirEntries[e.Name()] {
			errs = append(errs, fmt.Errorf("%s/%s: unexpected file (see curriculum/README.md)", dir, e.Name()))
		}
	}

	for _, f := range []struct {
		name string
		v    any
	}{
		{course.PhasesFile, &cc.Phases}, {course.WeeksFile, &cc.Weeks},
	} {
		if err := readOptional(fsys, path.Join(dir, f.name), f.v); err != nil {
			errs = append(errs, err)
		}
	}
	errs = append(errs, checkPhasesWeeks(dir, cc.Phases, cc.Weeks)...)

	concepts, err := loadConcepts(fsys, dir, cc.Weeks)
	if err != nil {
		errs = append(errs, err)
	}
	cc.Concepts = concepts

	items, err := loadItems(fsys, dir, m, lock, owner)
	if err != nil {
		errs = append(errs, err)
	}
	cc.Items = items

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return cc, nil
}

func checkPhasesWeeks(dir string, phases []course.Phase, weeks []course.Week) []error {
	var errs []error
	seenPhase := map[int]bool{}
	for _, p := range phases {
		if seenPhase[p.Order] {
			errs = append(errs, fmt.Errorf("%s/%s: duplicate phase order %d", dir, course.PhasesFile, p.Order))
		}
		seenPhase[p.Order] = true
		if p.Order < 1 || p.WeekFrom < 1 || p.WeekTo < p.WeekFrom {
			errs = append(errs, fmt.Errorf("%s/%s: phase %d has an invalid order or week range %d..%d", dir, course.PhasesFile, p.Order, p.WeekFrom, p.WeekTo))
		}
		if strings.TrimSpace(p.Name) == "" {
			errs = append(errs, fmt.Errorf("%s/%s: phase %d has no name", dir, course.PhasesFile, p.Order))
		}
	}
	seenWeek := map[int]bool{}
	for _, w := range weeks {
		if seenWeek[w.N] {
			errs = append(errs, fmt.Errorf("%s/%s: duplicate week %d", dir, course.WeeksFile, w.N))
		}
		seenWeek[w.N] = true
		if w.N < 1 || strings.TrimSpace(w.Title) == "" {
			errs = append(errs, fmt.Errorf("%s/%s: week %d needs n >= 1 and a title", dir, course.WeeksFile, w.N))
		}
	}
	return errs
}

// loadConcepts reads concepts.json and inlines the concepts/ sidecars.
func loadConcepts(fsys fs.FS, dir string, weeks []course.Week) ([]course.ResolvedConcept, error) {
	var rows []course.Concept
	if err := readOptional(fsys, path.Join(dir, course.ConceptsFile), &rows); err != nil {
		return nil, err
	}
	var errs []error
	weekSet := map[int]bool{}
	for _, w := range weeks {
		weekSet[w.N] = true
	}
	out := make([]course.ResolvedConcept, 0, len(rows))
	index := map[string]int{}
	for _, r := range rows {
		if !conceptSlugRe.MatchString(r.Slug) {
			errs = append(errs, fmt.Errorf("%s/%s: concept slug %q is not a slug", dir, course.ConceptsFile, r.Slug))
			continue
		}
		if _, dup := index[r.Slug]; dup {
			errs = append(errs, fmt.Errorf("%s/%s: duplicate concept %q", dir, course.ConceptsFile, r.Slug))
			continue
		}
		if strings.TrimSpace(r.Title) == "" {
			errs = append(errs, fmt.Errorf("%s/%s: concept %q has no title", dir, course.ConceptsFile, r.Slug))
		}
		for _, n := range r.Weeks {
			if !weekSet[n] {
				errs = append(errs, fmt.Errorf("%s/%s: concept %q links unknown week %d", dir, course.ConceptsFile, r.Slug, n))
			}
		}
		index[r.Slug] = len(out)
		out = append(out, course.ResolvedConcept{Concept: r, Templates: map[string]string{}})
	}

	cdir := path.Join(dir, "concepts")
	err := walkFiles(fsys, cdir, func(rel string, body string) error {
		var slug string
		var set func(*course.ResolvedConcept)
		switch {
		case strings.HasPrefix(rel, "_code/"):
			sm := conceptTemplateRe.FindStringSubmatch(strings.TrimPrefix(rel, "_code/"))
			if sm == nil {
				return fmt.Errorf("%s/%s: a code template must be _code/<slug>.<lang>.snip", cdir, rel)
			}
			slug, lang := sm[1], sm[2]
			set = func(rc *course.ResolvedConcept) { rc.Templates[lang] = body }
			return setConcept(out, index, cdir, rel, slug, set)
		case conceptWhenRe.MatchString(rel):
			slug = conceptWhenRe.FindStringSubmatch(rel)[1]
			set = func(rc *course.ResolvedConcept) { rc.WhenToUseMD = body }
		case conceptBodyRe.MatchString(rel):
			slug = conceptBodyRe.FindStringSubmatch(rel)[1]
			set = func(rc *course.ResolvedConcept) { rc.BodyMD = body }
		default:
			return fmt.Errorf("%s/%s: unexpected file (want <slug>.md, <slug>.when.md or _code/<slug>.<lang>.snip)", cdir, rel)
		}
		return setConcept(out, index, cdir, rel, slug, set)
	})
	if err != nil {
		errs = append(errs, err)
	}
	return out, errors.Join(errs...)
}

func setConcept(out []course.ResolvedConcept, index map[string]int, cdir, rel, slug string, set func(*course.ResolvedConcept)) error {
	i, ok := index[slug]
	if !ok {
		return fmt.Errorf("%s/%s: no concept %q in %s", cdir, rel, slug, course.ConceptsFile)
	}
	set(&out[i])
	return nil
}

// loadItems reads every items/<id>/ directory of a course.
func loadItems(fsys fs.FS, dir string, m *course.Manifest, lock course.IDsLock, owner map[string]string) ([]course.ResolvedItem, error) {
	idir := path.Join(dir, "items")
	entries, err := fs.ReadDir(fsys, idir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", idir, err)
	}
	var errs []error
	var out []course.ResolvedItem
	for _, e := range entries {
		if isDot(e.Name()) {
			continue
		}
		if !e.IsDir() {
			errs = append(errs, fmt.Errorf("%s/%s: %w (items/<id>/)", idir, e.Name(), errNotDirectoryish))
			continue
		}
		ri, err := loadItem(fsys, path.Join(idir, e.Name()), e.Name(), m, lock, owner)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, *ri)
	}
	return out, errors.Join(errs...)
}

// loadItem reads one item directory and applies the id guard.
func loadItem(fsys fs.FS, dir, id string, m *course.Manifest, lock course.IDsLock, owner map[string]string) (*course.ResolvedItem, error) {
	b, err := fs.ReadFile(fsys, path.Join(dir, course.ItemFile))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	it, err := course.DecodeItem(b)
	if err != nil {
		return nil, fmt.Errorf("%s/%s: %w", dir, course.ItemFile, err)
	}
	if err := it.ValidateFor(m); err != nil {
		return nil, fmt.Errorf("%s/%s: %w", dir, course.ItemFile, err)
	}
	// The id guard (t1 §4).
	if it.ID != id {
		return nil, fmt.Errorf("%s/%s: id %q does not match its directory %q", dir, course.ItemFile, it.ID, id)
	}
	if other, dup := owner[it.ID]; dup {
		return nil, fmt.Errorf("%s: id %q is already used by course %q (ids are unique across courses)", dir, it.ID, other)
	}
	owner[it.ID] = m.Slug
	le, ok := lock.Items[it.ID]
	switch {
	case !ok:
		return nil, fmt.Errorf("%s: id %q is not in %s (add it; the lock is append-only)", dir, it.ID, course.LockFile)
	case le.Course != m.Slug:
		return nil, fmt.Errorf("%s: id %q is locked to course %q (ids are never re-parented)", dir, it.ID, le.Course)
	}

	ri := &course.ResolvedItem{Item: *it}
	seen := map[string]string{} // "<stage>/<order>" -> "" (prose) or the code languages
	err = walkFiles(fsys, dir, func(rel, body string) error {
		if rel == course.ItemFile {
			return nil
		}
		sec, err := parseSectionFile(rel, body)
		if err != nil {
			return fmt.Errorf("%s/%s: %w", dir, rel, err)
		}
		key := fmt.Sprintf("%s/%02d", sec.Stage, sec.Order)
		langs, taken := seen[key]
		switch {
		case taken && (sec.Language == "" || langs == ""):
			return fmt.Errorf("%s/%s: stage %s order %d is already taken", dir, rel, sec.Stage, sec.Order)
		case taken && strings.Contains(" "+langs+" ", " "+sec.Language+" "):
			return fmt.Errorf("%s/%s: duplicate %s code at stage %s order %d", dir, rel, sec.Language, sec.Stage, sec.Order)
		}
		seen[key] = strings.TrimSpace(langs + " " + sec.Language)
		ri.Sections = append(ri.Sections, sec)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sortSections(ri.Sections)
	return ri, nil
}

// parseSectionFile maps an item-relative sidecar path to its section.
func parseSectionFile(rel, body string) (course.Section, error) {
	parts := strings.Split(rel, "/")
	switch {
	case len(parts) == 3 && parts[0] == "sections":
		if course.StageRank(parts[1]) == len(course.SectionStages) {
			return course.Section{}, fmt.Errorf("unknown stage %q (want one of %v)", parts[1], course.SectionStages)
		}
		sm := sectionFileRe.FindStringSubmatch(parts[2])
		if sm == nil {
			return course.Section{}, errors.New("a prose section must be sections/<stage>/<NN>-<kind>.md")
		}
		order, _ := strconv.Atoi(sm[1])
		return course.Section{Stage: parts[1], Order: order, Kind: sm[2], BodyMD: body}, nil
	case len(parts) == 2 && parts[0] == "_code":
		sm := codeFileRe.FindStringSubmatch(parts[1])
		if sm == nil {
			return course.Section{}, errors.New("a code section must be _code/<stage>-<NN>.<lang>.snip")
		}
		order, _ := strconv.Atoi(sm[2])
		return course.Section{Stage: sm[1], Order: order, Kind: "code", Language: sm[3], Code: body}, nil
	}
	return course.Section{}, errors.New("unexpected file (want item.json, sections/<stage>/<NN>-<kind>.md or _code/<stage>-<NN>.<lang>.snip)")
}

// sortSections puts sections in canonical order: stage, order, language, kind.
func sortSections(secs []course.Section) {
	sort.SliceStable(secs, func(i, j int) bool {
		a, b := secs[i], secs[j]
		if ra, rb := course.StageRank(a.Stage), course.StageRank(b.Stage); ra != rb {
			return ra < rb
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		if a.Language != b.Language {
			return a.Language < b.Language
		}
		return a.Kind < b.Kind
	})
}

// walkFiles calls fn for every regular file under root (paths relative to root, with
// "/" separators) with its content, skipping dotfiles and dot-directories. A missing
// root is empty. Bodies must be valid UTF-8 (they become text columns).
func walkFiles(fsys fs.FS, root string, fn func(rel, body string) error) error {
	var errs []error
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == root && errors.Is(err, fs.ErrNotExist) {
				return fs.SkipDir
			}
			return err
		}
		if p != root && isDot(d.Name()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, root+"/")
		if !utf8.Valid(b) {
			errs = append(errs, fmt.Errorf("%s: not valid UTF-8", p))
			return nil
		}
		if err := fn(rel, string(b)); err != nil {
			errs = append(errs, err)
		}
		return nil
	})
	if err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func isDot(name string) bool { return strings.HasPrefix(name, ".") }

// readOptional strictly decodes an optional JSON file (absent = zero value).
func readOptional(fsys fs.FS, name string, v any) error {
	if _, err := fs.Stat(fsys, name); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return readStrict(fsys, name, v)
}

// readStrict decodes exactly one JSON value, rejecting unknown fields and trailing data.
func readStrict(fsys fs.FS, name string, v any) error {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode %s: trailing data after the JSON value", name)
	}
	return nil
}

// SeedContent flattens the loaded content to the store's seed rows: path rows with
// their manifest's id prefix, and every item with its content_hash (canon.ContentHash).
func (c *Content) SeedContent() (store.SeedContent, error) {
	var out store.SeedContent
	for _, p := range c.Paths {
		sp := store.SeedPath{
			Slug: p.Slug, Title: p.Title, Status: p.Status, Summary: p.Summary,
			ProblemTotal: p.ProblemTotal, WeekTotal: p.WeekTotal, SortOrder: p.SortOrder,
		}
		if m := c.Manifests[p.Slug]; m != nil {
			sp.IDPrefix = m.IDPrefix
		}
		out.Paths = append(out.Paths, sp)
	}
	for _, cc := range c.Courses {
		out.Courses = append(out.Courses, cc.Slug)
		for _, p := range cc.Phases {
			out.Phases = append(out.Phases, store.SeedPhase{
				PathSlug: cc.Slug, Order: p.Order, Name: p.Name, Theme: p.Theme, WeekFrom: p.WeekFrom, WeekTo: p.WeekTo,
			})
		}
		for _, w := range cc.Weeks {
			out.Weeks = append(out.Weeks, store.SeedWeek{PathSlug: cc.Slug, N: w.N, Title: w.Title, Thesis: w.Thesis})
		}
		for _, cn := range cc.Concepts {
			out.Concepts = append(out.Concepts, store.SeedConcept{
				PathSlug: cc.Slug, Slug: cn.Slug, Title: cn.Title, BodyMD: cn.BodyMD,
				WhenToUseMD: cn.WhenToUseMD, Templates: cn.Templates, Weeks: cn.Weeks,
			})
		}
		for i := range cc.Items {
			ri := &cc.Items[i]
			hash, err := canon.ContentHash(ri)
			if err != nil {
				return store.SeedContent{}, fmt.Errorf("item %s: %w", ri.Item.ID, err)
			}
			it := ri.Item
			sp := store.SeedProblem{
				ID: it.ID, PathSlug: cc.Slug, WeekN: it.WeekN, Title: it.Title, Difficulty: it.Difficulty,
				Pattern: it.Pattern, Role: it.Role, Status: it.Status, SortOrder: it.SortOrder, ContentHash: hash,
			}
			for _, l := range it.Links {
				sp.Links = append(sp.Links, store.SeedLink{Kind: l.Kind, URL: l.URL})
			}
			for _, s := range ri.Sections {
				sp.Sections = append(sp.Sections, store.SeedSection{
					Stage: s.Stage, Kind: s.Kind, Order: s.Order, Language: s.Language, BodyMD: s.BodyMD, Code: s.Code,
				})
			}
			out.Problems = append(out.Problems, sp)
		}
	}
	return out, nil
}
