// Package coursetest holds helpers for the service-side manifest golden mirror tests
// (sprint m1-01): each owning package compares its OWN constants and migration CHECK
// lists to the loaded DSA manifest, so drift on either side fails CI. Like httptest, it
// is imported only by _test.go files; no production code imports it.
package coursetest

import (
	"io/fs"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/sujaykumarsuman/xlearn/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/course"
)

var (
	loadOnce sync.Once
	loaded   map[string]*course.Manifest
	loadErr  error
)

// DSA returns the embedded DSA manifest, loaded and validated by course.Load.
func DSA(t testing.TB) *course.Manifest {
	t.Helper()
	m := All(t)[course.DSASlug]
	if m == nil {
		t.Fatal("no dsa manifest")
	}
	return m
}

// All returns every embedded course manifest keyed by slug, loaded and validated by
// course.Load (m1-02: identity mirrors each course's public_stats.default_visible). The
// map is shared: callers must not modify it.
func All(t testing.TB) map[string]*course.Manifest {
	t.Helper()
	loadOnce.Do(func() { loaded, loadErr = course.Load(curriculum.FS) })
	if loadErr != nil {
		t.Fatalf("course.Load(curriculum.FS): %v", loadErr)
	}
	return loaded
}

var quoted = regexp.MustCompile(`'([^']*)'`)

// CheckIn returns the values of the first `CHECK (<column> IN ('a', 'b', …))` in the
// named migration file, in order.
func CheckIn(t testing.TB, fsys fs.FS, file, column string) []string {
	t.Helper()
	sql := read(t, fsys, file)
	re := regexp.MustCompile(`(?s)CHECK\s*\(\s*` + regexp.QuoteMeta(column) + `\s+IN\s*\(([^)]*)\)`)
	m := re.FindStringSubmatch(sql)
	if m == nil {
		t.Fatalf("%s: no CHECK (%s IN (…))", file, column)
	}
	var out []string
	for _, q := range quoted.FindAllStringSubmatch(m[1], -1) {
		out = append(out, q[1])
	}
	return out
}

// CheckBetween returns lo and hi of the first `<column> BETWEEN lo AND hi` in the named
// migration file.
func CheckBetween(t testing.TB, fsys fs.FS, file, column string) (lo, hi int) {
	t.Helper()
	sql := read(t, fsys, file)
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(column) + `\s+BETWEEN\s+(\d+)\s+AND\s+(\d+)`)
	m := re.FindStringSubmatch(sql)
	if m == nil {
		t.Fatalf("%s: no %s BETWEEN … AND …", file, column)
	}
	lo, _ = strconv.Atoi(m[1])
	hi, _ = strconv.Atoi(m[2])
	return lo, hi
}

func read(t testing.TB, fsys fs.FS, file string) string {
	t.Helper()
	b, err := fs.ReadFile(fsys, file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	return string(b)
}
