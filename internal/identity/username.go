package identity

import (
	"fmt"
	"regexp"
	"strings"
)

// Username rules (F009 / ADR-0024). A username is a URL-safe public handle that appears
// bare in the URL (projects.sujaykumar.dev/xlearn/<username>), so it must not collide with
// any current or future top-level app route, and it doubles as a login identifier
// (email OR username). We store the normalised (lower-case) form; matching is
// case-insensitive at the DB level (partial unique index on lower(username)).

const (
	usernameMinLen = 3
	usernameMaxLen = 30
)

// usernameRe is the allowed shape: lower-case letters/digits/hyphens, starting and ending
// with an alphanumeric (no leading/trailing hyphen). Length is checked separately so the
// error can be specific. Dots and other URL-significant characters are excluded, which also
// makes dotted reserved names (favicon.ico, robots.txt) unclaimable by construction.
var usernameRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// reservedUsernames are names a user may not claim because they are — or may become — a
// top-level path under /xlearn, a gateway route, a static asset, or a confusable/system
// word. This is the SINGLE source of truth for reservations: there is no web copy — the SPA's
// claim UI asks GET /username/available and shows the reason returned — and the router keeps
// the current ones as real routes. IMPORTANT: when a new top-level route or a new curriculum
// path slug is added, add it here too (documented in ADR-0024; every slug in
// curriculum/paths.json is enforced by TestReservedCoversCurriculumPathSlugs). Reserve BEFORE
// the route ships: the public-profile resolver re-validates against this list, so reserving
// a name someone already holds turns their profile into a 404.
var reservedUsernames = map[string]bool{
	// Current top-level SPA routes (a username shares the /xlearn namespace with these; React
	// Router's static routes win, so such a name would be an unreachable profile — reserve it).
	"auth": true, "settings": true, "xlearn": true,
	// Curriculum path slugs — every path in curriculum/paths.json, active or coming_soon
	// (each becomes a /xlearn/<slug> route when it goes live).
	"dsa": true, "system-design": true, "go-concurrency": true, "lld-ood": true, "sql": true,
	"behavioral": true,
	// Gateway-served prefixes / probes / assets (never the SPA shell).
	"api": true, "assets": true, "healthz": true, "readyz": true,
	"well-known": true, "favicon": true, "robots": true, "static": true, "public": true,
	// Reserved for likely future top-level routes + the profile namespace itself.
	"u": true, "user": true, "users": true, "profile": true, "profiles": true,
	"dashboard": true, "progress": true, "me": true, "account": true, "accounts": true,
	"onboarding": true, "catalog": true, "roadmap": true, "explore": true, "search": true,
	"claim-username": true,
	// Likely v2 top-level routes (judge / submissions / mock interviews / problem + course
	// browsing) — reserved ahead of time so no one can claim them before the route ships.
	"judge": true, "submissions": true, "interview": true, "interviews": true, "arena": true,
	"problems": true, "courses": true, "course": true, "paths": true, "path": true,
	// System / confusable / safety words.
	"admin": true, "root": true, "support": true, "help": true, "about": true,
	"login": true, "logout": true, "signin": true, "signup": true, "register": true,
	"terms": true, "privacy": true, "legal": true, "new": true, "edit": true, "index": true,
	"null": true, "undefined": true, "none": true, "system": true, "anonymous": true,
}

// normalizeUsername trims surrounding whitespace and lower-cases the handle. The stored +
// compared form is always the normalised one.
func normalizeUsername(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// validateUsername reports whether a normalised username is well-formed and not reserved,
// returning a human-readable reason on rejection. Callers must normalizeUsername first.
func validateUsername(s string) error {
	if len(s) < usernameMinLen || len(s) > usernameMaxLen {
		return fmt.Errorf("username must be %d–%d characters", usernameMinLen, usernameMaxLen)
	}
	if !usernameRe.MatchString(s) {
		return fmt.Errorf("use lowercase letters, numbers and hyphens (no leading, trailing or repeated hyphens)")
	}
	if reservedUsernames[s] {
		return fmt.Errorf("that username is reserved")
	}
	return nil
}
