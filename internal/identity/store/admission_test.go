package store

import (
	"reflect"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// The public-visibility table mirrors every course manifest's
// public_stats.default_visible (m1-02, D7), and the admitted_via constants equal the
// migration's CHECK list.
func TestCoursePublicDefaultsMirrorManifests(t *testing.T) {
	want := map[string]bool{}
	for slug, m := range coursetest.All(t) {
		want[slug] = m.PublicStats.Visible()
	}
	if !reflect.DeepEqual(coursePublicDefault, want) {
		t.Fatalf("coursePublicDefault %v, manifests %v", coursePublicDefault, want)
	}
	if !PublicVisibleDefault("dsa") || PublicVisibleDefault("no-such-course") {
		t.Fatal("PublicVisibleDefault: dsa must be public, an unknown course private")
	}
}

func TestAdmittedViaMatchesMigrationCheck(t *testing.T) {
	got := coursetest.CheckIn(t, migrationsFS, "migrations/00005_m1a_roles_admission.sql", "admitted_via")
	want := []string{AdmittedViaGrandfathered, AdmittedViaInvite, AdmittedViaCLI, AdmittedViaDev}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("admitted_via CHECK %v, constants %v", got, want)
	}
}
