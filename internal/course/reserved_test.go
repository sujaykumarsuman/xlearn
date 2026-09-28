package course_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"slices"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// update rewrites generated files and goldens in this package's tests.
var update = flag.Bool("update", false, "rewrite generated files (web/src/lib/reservedSegments.json)")

// reservedSegmentsJSON is the SPA's generated copy of course.ReservedSegments. The SPA
// course-slug guard (web/src/lib/courseSlugGuard.ts, m1-03) imports it, so the Go list
// and the SPA list cannot drift: this test is the parity test.
const reservedSegmentsJSON = "../../web/src/lib/reservedSegments.json"

type reservedFile struct {
	Comment  string   `json:"comment"`
	Reserved []string `json:"reserved"`
}

func renderReserved(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(reservedFile{
		Comment: "Generated from internal/course/reserved.go (ReservedSegments) by " +
			"`go test ./internal/course -run TestReservedSegmentsJSON -update`. Do not edit by hand.",
		Reserved: course.ReservedSegments,
	}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestReservedSegmentsJSON(t *testing.T) {
	want := renderReserved(t)
	if *update {
		if err := os.WriteFile(reservedSegmentsJSON, want, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(reservedSegmentsJSON)
	if err != nil {
		t.Fatalf("read %s (run with -update to generate it): %v", reservedSegmentsJSON, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s drifted from course.ReservedSegments; edit internal/course/reserved.go and re-run with -update", reservedSegmentsJSON)
	}
}

func TestReservedSegmentsSortedAndComplete(t *testing.T) {
	if !slices.IsSorted(course.ReservedSegments) {
		t.Errorf("ReservedSegments is not sorted: %v", course.ReservedSegments)
	}
	// ADR-0033 §6 and t1 §4: the static SPA segments plus the gateway's probes.
	for _, s := range []string{"u", "auth", "settings", "api", "assets", "healthz", "readyz", "privacy"} {
		if !slices.Contains(course.ReservedSegments, s) {
			t.Errorf("ReservedSegments lacks %q", s)
		}
	}
}

func TestCheckCourseSlug(t *testing.T) {
	for _, ok := range []string{"dsa", "system-design", "go-concurrency", "sql", "a1"} {
		if err := course.CheckCourseSlug(ok); err != nil {
			t.Errorf("CheckCourseSlug(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"api", "privacy", "u", "Dsa", "dsa_x", "-dsa", "dsa-", "a--b", "", ".well-known"} {
		if err := course.CheckCourseSlug(bad); err == nil {
			t.Errorf("CheckCourseSlug(%q) = nil, want an error", bad)
		}
	}
}
