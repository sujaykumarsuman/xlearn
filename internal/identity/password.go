package identity

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"sync"

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

// --- L3: bounded bcrypt (ADR-0035 §4, ADR-0033 §9) ---
//
// bcrypt at the default cost is tens of milliseconds of CPU under identity's 250m limit.
// Unbounded, a burst of logins would starve /sessions/validate, which every API call
// needs. So every hash and compare runs through a process-wide gate of bcryptSlots
// slots; a caller that can't get one immediately gets errBcryptBusy, which handlers map
// to 429 too_many_requests with Retry-After: 1 (never a queue). Login also compares
// against a dummy hash made once at startup whenever there is no real hash to compare
// (unknown identifier, password-less or suspended account), so the uniform 401 is
// uniform in time as well as in body (the enumeration row of ADR-0033 §9).

// bcryptSlots is L3's bound: at most 2 bcrypt operations in flight per identity process.
const bcryptSlots = 2

// errBcryptBusy is returned when every bcrypt slot is taken.
var errBcryptBusy = errors.New("identity: bcrypt busy")

// bcryptImpl is the raw primitive, swapped in tests for a counting or blocking one.
type bcryptImpl interface {
	Generate(pw []byte) ([]byte, error)
	Compare(hash, pw []byte) error
}

// realBcrypt is golang.org/x/crypto/bcrypt at bcrypt.DefaultCost.
type realBcrypt struct{}

func (realBcrypt) Generate(pw []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(pw, bcrypt.DefaultCost)
}

func (realBcrypt) Compare(hash, pw []byte) error { return bcrypt.CompareHashAndPassword(hash, pw) }

// passwords is the gated bcrypt plus the dummy hash.
type passwords struct {
	slots chan struct{}
	impl  bcryptImpl

	dummyOnce sync.Once
	dummy     []byte
	dummyErr  error
}

func newPasswords(impl bcryptImpl, slots int) *passwords {
	return &passwords{slots: make(chan struct{}, slots), impl: impl}
}

// processPasswords is the process-wide gate every Service and the admin CLI share.
var (
	processPasswordsOnce sync.Once
	processPasswordsV    *passwords
)

func processPasswords() *passwords {
	processPasswordsOnce.Do(func() { processPasswordsV = newPasswords(realBcrypt{}, bcryptSlots) })
	return processPasswordsV
}

// acquire takes a slot without blocking; release gives it back.
func (p *passwords) acquire() bool {
	select {
	case p.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (p *passwords) release() { <-p.slots }

// warm makes the dummy hash, once (NewService calls it at startup). It hashes 32 random
// bytes at the real cost; it is never stored or logged and matches no password.
func (p *passwords) warm() error {
	p.dummyOnce.Do(func() {
		var pw [32]byte
		if _, err := rand.Read(pw[:]); err != nil {
			p.dummyErr = err
			return
		}
		p.dummy, p.dummyErr = p.impl.Generate([]byte(base64.RawURLEncoding.EncodeToString(pw[:])))
	})
	return p.dummyErr
}

// hash bcrypt-hashes a (validated) password through the gate.
func (p *passwords) hash(pw string) (string, error) {
	if !p.acquire() {
		return "", errBcryptBusy
	}
	defer p.release()
	h, err := p.impl.Generate([]byte(pw))
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// check reports whether pw matches hash, through the gate. An empty hash (no account, no
// password, or an account that must not sign in) is compared against the dummy hash, so
// the time matches a real compare; it never matches. errBcryptBusy when no slot is free.
func (p *passwords) check(hash, pw string) (bool, error) {
	h := []byte(hash)
	if hash == "" {
		if err := p.warm(); err != nil {
			return false, err
		}
		h = p.dummy
	}
	if !p.acquire() {
		return false, errBcryptBusy
	}
	defer p.release()
	match := p.impl.Compare(h, []byte(pw)) == nil
	return match && hash != "", nil
}

// HashPassword bcrypt-hashes pw through the process-wide L3 gate (the admin CLI's
// `account create`). It returns errBcryptBusy only if both slots are in use.
func HashPassword(pw string) (string, error) { return processPasswords().hash(pw) }

// normalizeEmail trims + lowercases for consistent storage and case-insensitive matching.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// NormalizeEmail is normalizeEmail for the admin CLI.
func NormalizeEmail(email string) string { return normalizeEmail(email) }

// validEmail is a minimal sanity check (not RFC-complete): non-empty local + a dotted domain,
// no spaces. Real deliverability is a verification concern (deferred).
func validEmail(email string) bool {
	at := strings.IndexByte(email, '@')
	if at <= 0 || at >= len(email)-1 || strings.ContainsAny(email, " \t\r\n") {
		return false
	}
	return strings.Contains(email[at+1:], ".")
}

// ValidEmail is validEmail for the admin CLI.
func ValidEmail(email string) bool { return validEmail(email) }

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

// DisplayNameFromEmail is displayNameFromEmail for the admin CLI.
func DisplayNameFromEmail(email string) string { return displayNameFromEmail(email) }
