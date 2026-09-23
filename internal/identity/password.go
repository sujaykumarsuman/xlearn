package identity

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Password policy (ADR-0023): a length floor + the bcrypt 72-byte ceiling. bcrypt silently
// truncates input beyond 72 bytes, so we reject longer passwords rather than let a suffix be
// ignored. No composition rules for now.
const (
	minPasswordLen = 8
	maxPasswordLen = 72
)

var errWeakPassword = errors.New("password must be 8–72 characters")

// validatePassword enforces the length policy.
func validatePassword(pw string) error {
	if len(pw) < minPasswordLen || len(pw) > maxPasswordLen {
		return errWeakPassword
	}
	return nil
}

// hashPassword bcrypt-hashes a (validated) password.
func hashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// checkPassword reports whether pw matches the stored bcrypt hash (constant-time via bcrypt).
func checkPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// normalizeEmail trims + lowercases for consistent storage and case-insensitive matching.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validEmail is a minimal sanity check (not RFC-complete): non-empty local + a dotted domain,
// no spaces. Real deliverability is a verification concern (deferred).
func validEmail(email string) bool {
	at := strings.IndexByte(email, '@')
	if at <= 0 || at >= len(email)-1 || strings.ContainsAny(email, " \t\r\n") {
		return false
	}
	return strings.Contains(email[at+1:], ".")
}

// displayNameFromEmail derives a friendly default name from an email local-part.
func displayNameFromEmail(email string) string {
	local := email
	if i := strings.IndexByte(email, '@'); i > 0 {
		local = email[:i]
	}
	if local = strings.TrimSpace(local); local == "" {
		return "Learner"
	}
	return local
}
