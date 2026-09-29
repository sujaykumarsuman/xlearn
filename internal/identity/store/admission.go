package store

// v2 M1a (sprint m1-02; ADR-0033 §4, D7): identity's admission constants. Since m1-04
// (M1b) role and status are read: session-validate joins status = 'active' and returns
// the role (the gateway's T-3 cohort bit), preview enrollment checks the role, and the
// admin CLI (store/admin.go) changes both. Neither is ever minted into a JWT (§7).
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

// account.role values (CHECK in migration 00005; ADR-0033 §7). The owner and testers
// form the T-3 cohort (ADR-0034 §2) and sit outside SEAT_CAP.
const (
	RoleLearner = "learner"
	RoleTester  = "tester"
	RoleOwner   = "owner"
)

// account.status values (CHECK in migration 00005). A suspended account's sessions all
// fail validation at once (the GetValidSession join) and it cannot start a new one.
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
)

// InCohort reports whether role is in the owner/tester cohort (preview courses).
func InCohort(role string) bool { return role == RoleOwner || role == RoleTester }
