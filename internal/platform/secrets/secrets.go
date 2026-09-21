// Package secrets is xLearn's envelope-encryption seam for user-supplied secret
// material — specifically the AI-coach's BYO provider API keys (ADR-0007). It codifies
// the handling rules so no caller can get them wrong:
//
//   - A raw secret is sealed with a fresh per-record 32-byte DATA KEY using
//     XChaCha20-Poly1305 (golang.org/x/crypto/chacha20poly1305, 24-byte random nonce
//     prefixed to the ciphertext — an AEAD, so tampering is detected on open).
//   - That data key is itself sealed ("wrapped") with the service MASTER KEY (the same
//     AEAD). Persist enc_key (sealed secret) + enc_data_key (wrapped data key). The
//     master key lives OUTSIDE the database (a SOPS-managed secret), so a DB/backup leak
//     alone cannot decrypt anything.
//   - Decryption happens IN MEMORY ONLY; the caller zeroes the returned plaintext with
//     Zero after the provider call. This package never logs a key or a buffer and never
//     puts secret bytes in an error — open failures return the static ErrDecrypt.
//
// Only Mask output is ever safe to return to a client (e.g. "sk-...3f2a").
package secrets

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"
)

// MasterKeySize is the required master-key length (XChaCha20-Poly1305 key size).
const MasterKeySize = chacha20poly1305.KeySize // 32

// Errors. ErrDecrypt is deliberately generic (no plaintext/key/ciphertext bytes) so an
// error path can never leak secret material (ADR-0007 redaction rule).
var (
	// ErrDecrypt is returned by Open for any failure (wrong master key, corrupt or
	// tampered ciphertext, truncated nonce). It carries no secret bytes.
	ErrDecrypt = errors.New("secrets: decrypt failed")
	// ErrBadMasterKey is returned when a master key is not exactly MasterKeySize bytes.
	ErrBadMasterKey = fmt.Errorf("secrets: master key must be %d bytes", MasterKeySize)
	// ErrEmptyMasterKey is returned when no master key material is supplied.
	ErrEmptyMasterKey = errors.New("secrets: master key is empty")
)

// Cipher performs envelope seal/open under a fixed master key. It is safe for
// concurrent use (it holds only the immutable master key and constructs a fresh AEAD
// per call).
type Cipher struct {
	master []byte // exactly MasterKeySize bytes; never logged
}

// NewCipher builds a Cipher from a MasterKeySize-byte master key. The key is copied so
// the caller may zero its own buffer afterwards.
func NewCipher(masterKey []byte) (*Cipher, error) {
	if len(masterKey) == 0 {
		return nil, ErrEmptyMasterKey
	}
	if len(masterKey) != MasterKeySize {
		return nil, ErrBadMasterKey
	}
	m := make([]byte, MasterKeySize)
	copy(m, masterKey)
	return &Cipher{master: m}, nil
}

// Seal envelope-encrypts plaintext: it generates a fresh 32-byte data key, seals the
// plaintext under it (enc_key), wraps the data key under the master key (enc_data_key),
// and zeroes the data key before returning. Both outputs are nonce-prefixed ciphertext.
func (c *Cipher) Seal(plaintext []byte) (encKey, encDataKey []byte, err error) {
	dataKey := make([]byte, chacha20poly1305.KeySize)
	if _, err = rand.Read(dataKey); err != nil {
		return nil, nil, fmt.Errorf("secrets: generate data key: %w", err)
	}
	defer Zero(dataKey)

	encKey, err = sealWith(dataKey, plaintext)
	if err != nil {
		return nil, nil, err
	}
	encDataKey, err = sealWith(c.master, dataKey)
	if err != nil {
		return nil, nil, err
	}
	return encKey, encDataKey, nil
}

// Open reverses Seal: it unwraps the data key with the master key, then decrypts
// enc_key with it. The data key is zeroed before returning. The returned plaintext is
// live secret material — the caller MUST Zero it once the provider call is done. Any
// failure returns ErrDecrypt with no secret bytes.
func (c *Cipher) Open(encKey, encDataKey []byte) ([]byte, error) {
	dataKey, err := openWith(c.master, encDataKey)
	if err != nil {
		return nil, ErrDecrypt
	}
	defer Zero(dataKey)

	plaintext, err := openWith(dataKey, encKey)
	if err != nil {
		return nil, ErrDecrypt
	}
	return plaintext, nil
}

// sealWith seals plaintext under a 32-byte key: nonce (24 random bytes) || AEAD sealed
// ciphertext. The random nonce makes reuse of the same key safe.
func sealWith(key, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("secrets: new aead: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("secrets: generate nonce: %w", err)
	}
	// Seal appends the ciphertext+tag to the nonce, so the result is nonce||ct.
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

// openWith reverses sealWith. It returns a bare error (callers map it to ErrDecrypt so
// no secret bytes escape).
func openWith(key, blob []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	ns := aead.NonceSize()
	if len(blob) < ns {
		return nil, errors.New("secrets: ciphertext too short")
	}
	nonce, ct := blob[:ns], blob[ns:]
	return aead.Open(nil, nonce, ct, nil)
}

// Zero overwrites b with zeros — call it on a decrypted key buffer as soon as the
// provider call is done (defence in depth; ADR-0007). It is a no-op on nil.
func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Mask redacts a raw secret for display: a short leading hint + the last four
// characters (e.g. "sk-...3f2a"), never enough to reconstruct the key. Anything eight
// characters or shorter is fully redacted (too little to reveal safely). It is the only
// key-derived value the API ever returns.
func Mask(raw string) string {
	raw = strings.TrimSpace(raw)
	n := len(raw)
	if n == 0 {
		return ""
	}
	if n <= 8 {
		return "..."
	}
	return raw[:3] + "..." + raw[n-4:]
}

// ParseMasterKey decodes a master key from its transport encoding: standard or URL
// base64, hex, or (last resort) raw bytes — whichever yields exactly MasterKeySize
// bytes. This lets the SOPS secret carry the key in whatever form is convenient
// (base64 is the recommended shape). Surrounding whitespace/newlines are trimmed.
func ParseMasterKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrEmptyMasterKey
	}
	for _, dec := range []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.URLEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
		hex.DecodeString,
	} {
		if b, err := dec(s); err == nil && len(b) == MasterKeySize {
			return b, nil
		}
	}
	// Raw bytes (e.g. a mounted 32-byte file read as a string).
	if len(s) == MasterKeySize {
		return []byte(s), nil
	}
	return nil, ErrBadMasterKey
}

// ConstantTimeEqual reports whether two byte slices are equal in constant time. Exposed
// for tests / any equality check on secret material.
func ConstantTimeEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}
