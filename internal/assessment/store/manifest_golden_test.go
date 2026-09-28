package store

import (
	"reflect"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// TestManifestGoldenMirror is the assessment/store side of the DSA manifest golden test
// (sprint m1-01; side one is internal/course/golden_test.go): the rubric enum, its
// scale, the /35 total and the mock window — constants and migration CHECKs — must
// equal the manifest.
func TestManifestGoldenMirror(t *testing.T) {
	m := coursetest.DSA(t)
	rb := m.Mock.Rubric
	const initSQL = "migrations/00001_init.sql"

	if !reflect.DeepEqual(Dimensions, m.DimensionIDs()) {
		t.Errorf("Dimensions %v, manifest rubric dims %v", Dimensions, m.DimensionIDs())
	}
	if got := coursetest.CheckIn(t, migrationsFS, initSQL, "dimension"); !reflect.DeepEqual(got, m.DimensionIDs()) {
		t.Errorf("rubric_score.dimension CHECK %v, manifest rubric dims %v", got, m.DimensionIDs())
	}
	if NumDimensions != len(rb.Dims) {
		t.Errorf("NumDimensions = %d, manifest has %d dims", NumDimensions, len(rb.Dims))
	}

	if MinScore != rb.Scale[0] || MaxScore != rb.Scale[1] {
		t.Errorf("score range [%d, %d], manifest scale %v", MinScore, MaxScore, rb.Scale)
	}
	if lo, hi := coursetest.CheckBetween(t, migrationsFS, initSQL, "score"); lo != rb.Scale[0] || hi != rb.Scale[1] {
		t.Errorf("rubric_score.score CHECK [%d, %d], manifest scale %v", lo, hi, rb.Scale)
	}
	lo, hi := coursetest.CheckBetween(t, migrationsFS, initSQL, "total_35")
	if wantLo := rb.Scale[0] * len(rb.Dims); lo != wantLo || hi != rb.MaxTotal() {
		t.Errorf("mock_session.total_35 CHECK [%d, %d], manifest rubric [%d, %d]", lo, hi, wantLo, rb.MaxTotal())
	}

	if MockDuration != time.Duration(m.Mock.DurationS)*time.Second {
		t.Errorf("MockDuration = %v, manifest mock.duration_s = %ds", MockDuration, m.Mock.DurationS)
	}
	if got := coursetest.CheckIn(t, migrationsFS, initSQL, "difficulty"); !reflect.DeepEqual(got, course.Difficulties) {
		t.Errorf("mock_session.difficulty CHECK %v, item difficulties %v", got, course.Difficulties)
	}
}
