package course

// This file is the ONE place the DSA slug is spelled out in non-test Go code (sprint
// m1-03). hack/lint-course-literals.sh fails CI on the literal anywhere else under
// internal/ or web/src/ outside its allowlist, so a DSA-only code path can't creep back.

// DefaultSlug is the course a v1 caller means: v1 was DSA-only, so a request, event,
// context or row that predates courses belongs to it. Every mixed-version default and
// alias site uses it, never a bare literal:
//   - the gateway's DSA alias routes (/api/dashboard → /api/paths/<DefaultSlug>/dashboard, …),
//   - the internal `?path=` defaults in practice, review and assessment (a v1.6.0 gateway
//     calls them without one during a rolling update),
//   - curriculum's GET /concepts/{slug} alias,
//   - coach's legacy page-context parser (concept:<slug>, week:<n>, dashboard, …).
//
// The frozen v1-envelope meaning (events.V1PathSlug) is a separate constant in a platform
// package that doesn't import this one; a test pins the two equal.
const DefaultSlug = "dsa"

// DSASlug is the one course whose item ids stay bare (m1-09's id guard). It names the
// same course as DefaultSlug but a different rule, so it keeps its own name.
const DSASlug = DefaultSlug
