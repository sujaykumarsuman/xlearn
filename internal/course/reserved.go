package course

import (
	"fmt"
	"slices"
)

// ReservedSegments are the first URL path segments the SPA and the gateway own, so no
// course slug may take them (the SPA routes courses at `/:course/*`). This is the ONLY
// list: `web/src/lib/reservedSegments.json` is generated from it by
// `go test ./internal/course -run TestReservedSegmentsJSON -update` (the test fails on
// drift), and the SPA's course-slug guard imports that JSON. A new static SPA segment is
// added here only (ADR-0033 §6 added `privacy`), then regenerated. Kept sorted.
var ReservedSegments = []string{"api", "assets", "auth", "healthz", "privacy", "readyz", "settings", "u"}

// CheckCourseSlug is the course-slug guard (t1 §4): the slug shape, and not a reserved
// segment. The seed checks every manifest slug at startup and contentlint checks it in CI.
func CheckCourseSlug(slug string) error {
	if !slugRe.MatchString(slug) {
		return fmt.Errorf("course slug %q must match %s", slug, slugRe)
	}
	if slices.Contains(ReservedSegments, slug) {
		return fmt.Errorf("course slug %q is a reserved URL segment %v", slug, ReservedSegments)
	}
	return nil
}
