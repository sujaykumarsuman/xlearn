package identity

import (
	"fmt"
	"regexp"
	"strings"
)

// Username rules (F009 / ADR-0024, ADR-0025). A username is a URL-safe public handle that
// appears in the profile URL (projects.sujaykumar.dev/xlearn/u/<username>) and doubles as a
// login identifier (email OR username). We store the normalised (lower-case) form; matching is
// case-insensitive at the DB level (partial unique index on lower(username)).

const (
	usernameMinLen = 3
	usernameMaxLen = 30
)

// usernameRe is the allowed shape: lower-case letters/digits/hyphens, starting and ending
// with an alphanumeric (no leading/trailing hyphen). Length is checked separately so the
// error can be specific. Dots and other URL-significant characters are excluded, which also
// makes dotted names (favicon.ico, robots.txt) unclaimable by construction.
var usernameRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// reservedUsernames are the only names a user may not claim: handles that would impersonate
// the platform or read as a system account (@admin, @support, @xlearn). Profiles live under
// their own /u/ prefix and courses at /xlearn/<course-id> (ADR-0025), so a username can never
// shadow a route — course slugs and route words are deliberately NOT reserved (owner
// direction, ADR-0025 2026-09-23 update); keeping a course slug out of the /u/ and app-route
// segments is a curriculum-side check, not a username rule. This is the SINGLE source of
// truth: there is no web copy — the SPA's claim UI asks GET /username/available and shows the
// reason returned. Removing a word is always safe; before ADDING one, check no account holds
// it: the public-profile resolver re-validates against this list, so reserving a held name
// turns that profile into a 404.
var reservedUsernames = map[string]bool{
	// Platform / brand / staff impersonation.
	"xlearn": true, "admin": true, "root": true, "support": true, "help": true, "about": true,
	"system": true,
	// Auth-flow words (confusable in a sign-in form that accepts email OR username).
	"login": true, "logout": true, "signin": true, "signup": true, "register": true,
	// Legal / generic system words.
	"terms": true, "privacy": true, "legal": true, "new": true, "edit": true, "index": true,
	"null": true, "undefined": true, "none": true, "anonymous": true,
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
