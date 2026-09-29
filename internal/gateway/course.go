package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// Course resolution (sprint m1-03, M1b; ADR-0026 §5, t0 §7). DSA is no longer special:
// every course-scoped route resolves its {slug} against the manifests compiled into the
// gateway (internal/course, never fetched), and every item route resolves the item's
// course from its path_slug. Course-scoped aggregates live under the EXISTING
// /api/paths/{slug}/… prefix (t0 §7) — one prefix, no parallel /api/courses tree. The v1
// routes without a course stay as DSA aliases (Alias on apiRoute) until the recorded
// removal tag (ADR-0034 §1.1: ≥ 1 release after the SPA stops calling them).
//
// Visibility (ADR-0034 §2):
//   - active: visible.
//   - coming_soon: listed in the catalog (GET /api/paths) only; every data route 404s.
//   - preview: visible only to the owner/tester cohort. m1-04 passes the cohort bit from
//     session-validate; until then it is false for everyone, so preview 404s everywhere.
//   - retired, or unknown: 404.
//
// Every refusal is the SAME uniform 404 course_not_found envelope, so a response never
// hints whether a hidden course exists.

// errCourseNotFound is resolveCourse's miss.
var errCourseNotFound = errors.New("gateway: course not found")

// resolveCourse returns the manifest compiled in for slug, whatever its status.
func (g *Gateway) resolveCourse(slug string) (*course.Manifest, error) {
	m, ok := g.courses.Lookup(slug)
	if !ok {
		return nil, errCourseNotFound
	}
	return m, nil
}

// courseVisible reports whether the course's data routes serve this caller. cohort is
// the owner/tester bit (m1-04); coming_soon is catalog-only, so it is not visible here.
func courseVisible(m *course.Manifest, cohort bool) bool {
	switch m.Status {
	case course.StatusActive:
		return true
	case course.StatusPreview:
		return cohort
	default:
		return false
	}
}

// courseListed reports whether the course appears in the catalog for this caller:
// every visible course, plus coming_soon (the catalog teaser).
func courseListed(m *course.Manifest, cohort bool) bool {
	return courseVisible(m, cohort) || m.Status == course.StatusComingSoon
}

// inCohort is the caller's owner/tester bit. m1-04 derives it from session-validate's
// role; in m1-03 nobody is in the cohort, so preview courses are hidden from everyone.
func (g *Gateway) inCohort(*http.Request) bool { return false }

// visibleCourse resolves slug and reports whether its data routes serve this request.
func (g *Gateway) visibleCourse(r *http.Request, slug string) (*course.Manifest, bool) {
	m, err := g.resolveCourse(slug)
	if err != nil || !courseVisible(m, g.inCohort(r)) {
		return nil, false
	}
	return m, true
}

// requireCourse resolves a course-scoped route's slug, writing the uniform 404
// course_not_found envelope unless the course is visible to the caller.
func (g *Gateway) requireCourse(w http.ResponseWriter, r *http.Request, slug string) (*course.Manifest, bool) {
	m, ok := g.visibleCourse(r, slug)
	if !ok {
		writeCourseNotFound(w)
		return nil, false
	}
	return m, true
}

// writeCourseNotFound writes the uniform course 404 (unknown, retired, coming_soon data
// routes and preview outside the cohort all look the same).
func writeCourseNotFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, "course_not_found", "no such course")
}

// withPath appends the course to an internal service path as `?path=<slug>` (or
// `&path=`). practice, review and assessment default an absent param to
// course.DefaultSlug, so a v1.6.0 service that ignores it and a v1.7.0 one that reads it
// agree for DSA during a rolling update.
func withPath(p, slug string) string {
	sep := "?"
	if strings.Contains(p, "?") {
		sep = "&"
	}
	return p + sep + "path=" + url.QueryEscape(slug)
}

// itemCourse resolves an item's course from curriculum's metadata (the item's
// path_slug). found=false when curriculum doesn't know the id; err on an upstream failure.
// An item row without a path_slug predates courses, so it is course.DefaultSlug.
func (g *Gateway) itemCourse(ctx context.Context, itemID string) (slug string, found bool, err error) {
	if g.curriculum == nil {
		return "", false, errors.New("curriculum not configured")
	}
	body, status, err := g.curriculum.get(ctx, "/problems?ids="+url.QueryEscape(itemID))
	if err != nil {
		return "", false, err
	}
	if status != http.StatusOK {
		return "", false, errors.New("curriculum: status " + http.StatusText(status))
	}
	var env struct {
		Problems []struct {
			ID       string `json:"id"`
			PathSlug string `json:"path_slug"`
		} `json:"problems"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return "", false, err
	}
	for _, p := range env.Problems {
		if p.ID == itemID {
			return itemPathSlug(p.PathSlug), true, nil
		}
	}
	return "", false, nil
}

// itemPathSlug is an item's course: its path_slug, or course.DefaultSlug for an item
// that carries none (a v1-shaped row: v1 was DSA-only).
func itemPathSlug(pathSlug string) string {
	if pathSlug == "" {
		return course.DefaultSlug
	}
	return pathSlug
}

// filterCatalog drops the courses this caller must not see from curriculum's GET /paths
// body ({"paths":[…]}): preview outside the cohort, retired, and any slug with no
// compiled-in manifest. Each kept entry's bytes pass through untouched. On a body it
// can't parse it returns the body unchanged (curriculum's own error envelope).
func (g *Gateway) filterCatalog(r *http.Request, body []byte) []byte {
	var env map[string]json.RawMessage
	if err := json.Unmarshal(body, &env); err != nil {
		return body
	}
	raw, ok := env["paths"]
	if !ok {
		return body
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return body
	}
	cohort := g.inCohort(r)
	kept := make([]json.RawMessage, 0, len(entries))
	for _, e := range entries {
		var p struct {
			Slug string `json:"slug"`
		}
		if json.Unmarshal(e, &p) != nil {
			continue
		}
		if m, err := g.resolveCourse(p.Slug); err == nil && courseListed(m, cohort) {
			kept = append(kept, e)
		}
	}
	env["paths"] = mustJSON(kept)
	out, err := json.Marshal(env)
	if err != nil {
		return body
	}
	return out
}
