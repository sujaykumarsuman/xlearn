package coach

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// The two-pair key crypto of M1b (m1-10 task 3; ADR-0031 §7). Every read and write of a
// stored provider key goes through this file, so the dual-write and the staleness rule
// cannot be implemented differently in the PUT path and the re-wrap job.
//
// Why two pairs at all: v1.6.0 opens the sealed material with NIL associated data
// (secrets.Open). A key re-sealed WITH associated data is therefore undecryptable by it,
// so re-sealing the existing columns in place would silently raise coach's rollback floor
// to 1.7.0 while v1.7.0 declares 1.6.0 (ADR-0034 §3). Instead:
//
//	legacy pair  (enc_key, enc_data_key)            unbound, v1 format, DUAL-WRITTEN
//	AD pair      (enc_key_ad, enc_data_key_ad)      bound to (account, provider), kek_id
//
// The legacy pair is retired in two later steps once the floor passes 1.7.0 — l-01
// (v1.11.0) stops reading and writing it, l-02 (v1.12.0) drops it. The AEAD binding is
// only ENFORCED from that point: until then a reader still falls back to the unbound pair
// when the AD pair is absent or stale, so someone with database write access could force
// that path and swap legacy pairs between accounts. That residual risk is accepted (v2 is
// owner-only per D35, and the database is not reachable from outside the cluster, MI-5)
// and is recorded in the decisions log.

// sealedKey is both sealed pairs of one key plus the metadata that ties them together —
// exactly what a PUT writes in a single upsert.
type sealedKey struct {
	EncKey       []byte
	EncDataKey   []byte
	EncKeyAD     []byte
	EncDataKeyAD []byte
	KEKID        string
	ADSrcDigest  []byte
}

// sealKey seals rawKey into BOTH pairs: the legacy unbound pair (so v1.6.0 can still read
// it during a rollback) and the AD pair bound to (accountID, provider) under the keyring's
// active entry.
//
// ADSrcDigest is taken from the legacy enc_data_key produced RIGHT HERE, so the digest
// always names the exact legacy seal the AD pair was made beside. Because the envelope
// draws a fresh random data key on every seal, that digest identifies one seal and no
// other — which is what lets a later reader notice that someone (a v1.6.0 image) replaced
// the legacy pair underneath.
func (s *Service) sealKey(accountID, provider, rawKey string) (sealedKey, error) {
	encKey, encDataKey, err := s.cipher.Seal([]byte(rawKey))
	if err != nil {
		return sealedKey{}, fmt.Errorf("seal legacy pair: %w", err)
	}
	ad := secrets.CoachKeyAD(accountID, provider)
	encKeyAD, encDataKeyAD, kekID, err := s.keyring().SealAD([]byte(rawKey), ad)
	if err != nil {
		return sealedKey{}, fmt.Errorf("seal ad pair: %w", err)
	}
	return sealedKey{
		EncKey:       encKey,
		EncDataKey:   encDataKey,
		EncKeyAD:     encKeyAD,
		EncDataKeyAD: encDataKeyAD,
		KEKID:        kekID,
		ADSrcDigest:  sha256Of(encDataKey),
	}, nil
}

// openKey decrypts a stored key in memory. The caller MUST secrets.Zero the result as
// soon as the provider call is done.
//
// Which pair it reads, and why:
//
//   - A CURRENT AD pair (present, and its digest matches the legacy pair beside it) is
//     opened with associated data. If that open FAILS, the error is final: falling back
//     to the legacy pair here would defeat the whole binding, since an attacker who can
//     write the database could simply corrupt the AD pair to be served the unbound one.
//   - No AD pair (the re-wrap job hasn't reached this row yet) or a STALE one (a v1.6.0
//     image replaced the key during a rollback, rewriting only the legacy columns) → the
//     legacy pair, which in the stale case is the NEWER of the two. The next re-wrap pass
//     re-derives the AD pair from it.
//
// The stale arm is the one that matters operationally: without it, a key the owner pasted
// during a rollback would be ignored in favour of the previous key after rolling forward —
// and if they rotated because the old key was revoked, the resulting 401 would disable the
// new key (KindAuth is the one kind that disables).
//
// It does NOT share Reseal's ErrUnknownKEK fallback, and that asymmetry is deliberate.
// Reseal is REPAIRING a row — it re-seals under the active KEK and hands the plaintext to
// nobody — so falling back to the legacy pair there is free. openKey is SERVING a key to a
// provider, and `kek_id` is a plain text column: if an unknown id fell back to the unbound
// legacy pair, writing `kek_id = 'k99'` would be a one-column way to opt any row out of
// the binding. So an unreadable current pair is an error here even when the cause is a
// missing keyring entry rather than tampering.
//
// The operational cost is a bounded one: reverting COACH_MASTER_KEYS after a rotation
// makes chats fail until the next re-wrap pass moves the rows back (≤ ~10 minutes). The
// distinct log below is what makes that five seconds to diagnose instead of an hour.
func (s *Service) openKey(kc store.KeyConfig) ([]byte, error) {
	if kc.ADPairCurrent() {
		ad := secrets.CoachKeyAD(kc.AccountID, kc.Provider)
		raw, err := s.keyring().OpenAD(kc.EncKeyAD, kc.EncDataKeyAD, kc.KEKID, ad)
		if errors.Is(err, secrets.ErrUnknownKEK) {
			// Not a corrupt row: this process simply does not hold the key that sealed it.
			// Naming the entry (ids carry no key material) points straight at the cause —
			// a retired or reverted COACH_MASTER_KEYS — rather than at the stored key.
			s.log.Error("coach: key sealed under a keyring entry this process does not have; "+
				"restore the entry or wait for the re-wrap pass",
				"kek_id", kc.KEKID, "active_kek", s.keyring().Active(), "provider", kc.Provider)
		}
		return raw, err
	}
	return s.cipher.Open(kc.EncKey, kc.EncDataKey)
}

// keyring returns the AD keyring, falling back to a single-entry ring over the legacy
// cipher. The fallback keeps every test and any single-key deployment working without
// threading a keyring through: cmd/coach builds the same {k0: COACH_MASTER_KEY} ring when
// COACH_MASTER_KEYS is unset, which is what production runs today (no SOPS change this
// sprint).
func (s *Service) keyring() *secrets.Keyring {
	if s.keys != nil {
		return s.keys
	}
	return secrets.KeyringOf(DefaultKEKID, s.cipher)
}

// sha256Of is ad_src_digest's one definition on the Go side: the SHA-256 of the legacy
// wrapped data key an AD pair was sealed beside. The re-wrap path computes the same value
// in Postgres (`sha256(enc_data_key)`), and TestGoAndPostgresSha256Agree proves the two
// match on real stored bytes — if they ever diverged, every row would read as stale
// forever and the job would re-wrap the same keys every ten minutes.
func sha256Of(b []byte) []byte {
	d := sha256.Sum256(b)
	return d[:]
}
