package secrets

import (
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"
)

// This file adds the two things ADR-0031 §7 asks of the envelope seam beyond v1's
// single unbound master key:
//
//   - ASSOCIATED DATA. SealAD/OpenAD bind a sealed pair to the identity it belongs to
//     (for coach: the account and provider) on BOTH AEAD layers. Associated data is
//     authenticated but not encrypted, so a pair moved to another account or provider —
//     by anyone with DB write access — fails to open instead of silently decrypting
//     someone else's key. The v1 pair passes nil ad and is NOT bound; the binding is
//     enforced only once that legacy pair is contracted (l-01 / l-02).
//
//   - A KEYRING. Keys are named (k0, k1, …) and a sealed pair records WHICH key wrapped
//     it (kek_id), so a master key can be rotated by adding a new active entry and
//     letting a background job re-wrap, with the old entry still able to open what it
//     sealed. Production runs a one-entry keyring (k0 = today's COACH_MASTER_KEY), so
//     rotation needs code, not a secret change, until it is actually performed.
//
// Nothing here logs, formats or returns key material; every failure is a static error.

// kekIDPattern is the accepted keyring-entry id: k followed by one or more digits. The
// id is persisted next to the ciphertext and appears in logs, so it is deliberately
// opaque and carries no key material.
var kekIDPattern = regexp.MustCompile(`^k[0-9]+$`)

// Keyring errors. Like ErrDecrypt they are static and secret-free.
var (
	// ErrUnknownKEK is returned when a sealed pair names a keyring entry this process
	// does not have (a key retired too early, or a spec missing an entry). It is NOT
	// ErrDecrypt: the ciphertext may be perfectly good, we just cannot unwrap it.
	ErrUnknownKEK = errors.New("secrets: unknown kek id")
	// ErrEmptyKeyring is returned when a keyring spec parses to no entries.
	ErrEmptyKeyring = errors.New("secrets: keyring is empty")
	// ErrBadKeyringSpec is returned for a malformed spec (a missing id, a bad id, or a
	// duplicate id). It never echoes the spec, which contains key material.
	ErrBadKeyringSpec = errors.New("secrets: malformed keyring spec")
)

// coachKeyADPrefix is the versioned associated-data domain for a coach provider key
// (ADR-0031 §7). The version lets a future binding change be told apart from this one.
const coachKeyADPrefix = "xlearn/coach/key/v1"

// CoachKeyAD builds the associated data that binds a coach provider key's sealed pair to
// its owner: `xlearn/coach/key/v1|<account_id>|<provider>`. Both the PUT path and the
// re-wrap job derive the ad from the row through this one function, so they can never
// disagree about what a pair is bound to.
func CoachKeyAD(accountID, provider string) []byte {
	return []byte(coachKeyADPrefix + "|" + accountID + "|" + provider)
}

// SealAD is Seal with AEAD associated data bound to both layers: the secret under the
// fresh data key, and the data key under the master key. Open requires byte-identical
// ad. A nil or empty ad is accepted but binds nothing — callers that want binding must
// pass one (coach uses CoachKeyAD).
func (c *Cipher) SealAD(plaintext, ad []byte) (encKey, encDataKey []byte, err error) {
	dataKey := make([]byte, chacha20poly1305.KeySize)
	if _, err = rand.Read(dataKey); err != nil {
		return nil, nil, fmt.Errorf("secrets: generate data key: %w", err)
	}
	defer Zero(dataKey)

	encKey, err = sealWith(dataKey, plaintext, ad)
	if err != nil {
		return nil, nil, err
	}
	encDataKey, err = sealWith(c.master, dataKey, ad)
	if err != nil {
		return nil, nil, err
	}
	return encKey, encDataKey, nil
}

// OpenAD reverses SealAD. It returns ErrDecrypt — with no secret bytes and no hint of
// which layer failed — when the ad does not match, the master key is wrong, or the
// ciphertext was tampered with. The returned plaintext is live secret material: Zero it
// as soon as the provider call is done.
func (c *Cipher) OpenAD(encKey, encDataKey, ad []byte) ([]byte, error) {
	dataKey, err := openWith(c.master, encDataKey, ad)
	if err != nil {
		return nil, ErrDecrypt
	}
	defer Zero(dataKey)

	plaintext, err := openWith(dataKey, encKey, ad)
	if err != nil {
		return nil, ErrDecrypt
	}
	return plaintext, nil
}

// Keyring is a named set of master keys with one ACTIVE entry. New material is always
// sealed under the active entry and stamped with its id; an existing pair is opened
// under the id it was stamped with, so a rotation is: add the new entry at the front,
// deploy, let the re-wrap job move every row onto it, then retire the old entry.
//
// It is immutable after ParseKeyring/NewKeyring and safe for concurrent use.
type Keyring struct {
	active  string
	ciphers map[string]*Cipher
}

// NewKeyring builds a keyring from id → master key, with active as the sealing entry.
// The keys are copied into their Ciphers, so the caller may zero its own buffers.
func NewKeyring(active string, keys map[string][]byte) (*Keyring, error) {
	if len(keys) == 0 {
		return nil, ErrEmptyKeyring
	}
	if _, ok := keys[active]; !ok {
		return nil, ErrUnknownKEK
	}
	ciphers := make(map[string]*Cipher, len(keys))
	for id, k := range keys {
		if !kekIDPattern.MatchString(id) {
			return nil, ErrBadKeyringSpec
		}
		c, err := NewCipher(k)
		if err != nil {
			return nil, err
		}
		ciphers[id] = c
	}
	return &Keyring{active: active, ciphers: ciphers}, nil
}

// ParseKeyring parses a COACH_MASTER_KEYS spec: `k1:<base64>,k0:<base64>` — a
// comma-separated list of `<id>:<key>` pairs whose FIRST entry is active (so rotating
// is prepending). Each key is decoded by ParseMasterKey, so base64, hex or raw 32 bytes
// all work. Ids must match ^k[0-9]+$ and be unique. The spec is never echoed in an
// error: it is key material.
func ParseKeyring(spec string) (*Keyring, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, ErrEmptyKeyring
	}
	keys := make(map[string][]byte)
	active := ""
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, raw, ok := strings.Cut(part, ":")
		if !ok {
			return nil, ErrBadKeyringSpec
		}
		id = strings.TrimSpace(id)
		if !kekIDPattern.MatchString(id) {
			return nil, ErrBadKeyringSpec
		}
		if _, dup := keys[id]; dup {
			return nil, ErrBadKeyringSpec
		}
		key, err := ParseMasterKey(raw)
		if err != nil {
			return nil, err
		}
		keys[id] = key
		if active == "" {
			active = id // the first entry seals
		}
	}
	if active == "" {
		return nil, ErrEmptyKeyring
	}
	kr, err := NewKeyring(active, keys)
	for _, k := range keys {
		Zero(k) // NewKeyring copied what it needed
	}
	return kr, err
}

// KeyringOf wraps one already-built Cipher as a single-entry keyring under id. It is the
// bridge for a deployment (or a test) that has exactly one master key and no rotation in
// flight: the AD pair still records which entry sealed it, so adding a second entry later
// is a configuration change, not a migration.
func KeyringOf(id string, c *Cipher) *Keyring {
	return &Keyring{active: id, ciphers: map[string]*Cipher{id: c}}
}

// Active is the id of the entry new material is sealed under. It is safe to log.
func (r *Keyring) Active() string { return r.active }

// IDs returns every entry id, for a startup log line ("keyring loaded: active=k1
// entries=[k1 k0]"). Ids carry no key material.
func (r *Keyring) IDs() []string {
	out := make([]string, 0, len(r.ciphers))
	for id := range r.ciphers {
		out = append(out, id)
	}
	return out
}

// Has reports whether the keyring holds an entry (a cheap pre-check before a re-wrap
// decides it cannot repair a row).
func (r *Keyring) Has(kekID string) bool {
	_, ok := r.ciphers[kekID]
	return ok
}

// SealAD seals plaintext under the ACTIVE entry with associated data ad and returns the
// pair plus the id to persist alongside it.
func (r *Keyring) SealAD(plaintext, ad []byte) (encKey, encDataKey []byte, kekID string, err error) {
	c := r.ciphers[r.active]
	if c == nil {
		return nil, nil, "", ErrUnknownKEK
	}
	encKey, encDataKey, err = c.SealAD(plaintext, ad)
	if err != nil {
		return nil, nil, "", err
	}
	return encKey, encDataKey, r.active, nil
}

// OpenAD opens a pair sealed under the entry named by kekID, with associated data ad.
// An id the keyring does not hold is ErrUnknownKEK (distinguishable from a genuine
// decrypt failure, so a re-wrap can report "retired key" rather than "corrupt row").
func (r *Keyring) OpenAD(encKey, encDataKey []byte, kekID string, ad []byte) ([]byte, error) {
	c := r.ciphers[kekID]
	if c == nil {
		return nil, ErrUnknownKEK
	}
	return c.OpenAD(encKey, encDataKey, ad)
}
