package store

// v2 M1a (sprint m1-02; ADR-0033 §4, D7): identity's new write-side constants. Nothing
// READS role, status, admitted_via or public_visible in v1.6.0 — sessions, RequireRole,
// the admin CLI and the public filters are M1b (m1-04, m1-05).
//
// path_enrollment.public_visible is the course manifest's public_stats.default_visible
// (D7). Since m1-03 identity compiles the manifests in (internal/course): the enrollment
// handler reads the default from the registry it validates the slug against and passes
// it to StartEnrollment, so no table here mirrors the manifests any more.

// admitted_via values (identity.account.admitted_via, CHECK in migration 00005).
const (
	// AdmittedViaGrandfathered marks every account that existed when M1a migrated
	// (the owner's included; ADR-0033 §2). Only the migration writes it.
	AdmittedViaGrandfathered = "grandfathered"
	// AdmittedViaDev is written by every create path v1.6.0 has: dev login and
	// open-mode signup, both compose-only (production signup is closed).
	AdmittedViaDev = "dev"
	// AdmittedViaCLI arrives with m1-04's admin CLI; AdmittedViaInvite with l-03.
	AdmittedViaCLI    = "cli"
	AdmittedViaInvite = "invite"
)
