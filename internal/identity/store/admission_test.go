package store

import (
	"reflect"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// The admitted_via constants equal the migration's CHECK list. (Each course's
// public_visible default comes from its manifest since m1-03; the identity package's
// TestEnrollmentPublicVisibleFromManifest proves it flows through for every course.)
func TestAdmittedViaMatchesMigrationCheck(t *testing.T) {
	got := coursetest.CheckIn(t, migrationsFS, "migrations/00005_m1a_roles_admission.sql", "admitted_via")
	want := []string{AdmittedViaGrandfathered, AdmittedViaInvite, AdmittedViaCLI, AdmittedViaDev}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("admitted_via CHECK %v, constants %v", got, want)
	}
}
