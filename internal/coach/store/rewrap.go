package store

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store/gen"
)

// The persistence half of the background AD re-wrap (m1-10 task 3; the crypto half is
// internal/coach/rewrap.go). The split is deliberate: the store owns the transaction, the
// advisory lock and the optimistic update, and never sees a plaintext key; coach owns the
// keyring and the seal, and never writes SQL.

// RewrapRow is one key config the re-wrap job has to repair, with both sealed pairs so
// the caller can decide which one to decrypt from.
type RewrapRow struct {
	ID        string
	AccountID string
	Provider  string

	EncKey     []byte
	EncDataKey []byte

	EncKeyAD     []byte
	EncDataKeyAD []byte
	KEKID        string
	ADSrcDigest  []byte
}

// ADPairCurrent reports whether this row's AD pair describes the legacy pair beside it —
// the same rule as KeyConfig.ADPairCurrent, so the job and chat agree about staleness.
func (r RewrapRow) ADPairCurrent() bool {
	if len(r.EncKeyAD) == 0 || len(r.EncDataKeyAD) == 0 {
		return false
	}
	if len(r.EncDataKey) == 0 {
		return true
	}
	return digestMatches(r.ADSrcDigest, r.EncDataKey)
}

// Resealer produces a fresh AD pair for one row, under the keyring's active entry.
// Returning an error means "skip this row": the job logs it and moves on, and the row is
// left byte-identical. It is called INSIDE the batch transaction, so it must not block.
type Resealer func(RewrapRow) (encKeyAD, encDataKeyAD []byte, kekID string, err error)

// RewrapStats is what one pass did.
type RewrapStats struct {
	// Locked is false when another runner held the lock and this pass did nothing.
	Locked bool
	// Rewrapped is the number of rows given a fresh AD pair.
	Rewrapped int
	// Skipped is the number of rows that could not be decrypted (reported, never deleted
	// or disabled — the owner re-pastes the key if it matters).
	Skipped int
	// Raced is the number of rows a concurrent PUT replaced mid-pass, so the optimistic
	// update matched nothing. The PUT already wrote a correct pair; nothing is wrong.
	Raced int
	// Pending is how many rows still need work after this batch (drives the job's
	// cadence, and l-01's `pending=0` precondition).
	Pending int64
}

// RewrapBatch repairs up to batchSize key configs in ONE transaction.
//
// The transaction takes pg_try_advisory_xact_lock first, so exactly one runner works at a
// time across replicas and across a rolling update. It must be the TRANSACTION-scoped
// lock: a session-level pg_try_advisory_lock taken through pgxpool can have its unlock
// issued on a different pooled connection, leaking the lock for the life of the pod and
// silently stopping all future passes. COMMIT or ROLLBACK releases this one on the
// connection that took it.
//
// A row that cannot be decrypted is skipped and counted, never deleted and never
// disabled: the key may be perfectly valid and merely sealed under a KEK this process
// doesn't have, and destroying or disabling it would turn an operator error into learner
// data loss.
func (s *PgStore) RewrapBatch(ctx context.Context, activeKEK string, batchSize int32, reseal Resealer) (RewrapStats, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RewrapStats{}, fmt.Errorf("begin rewrap tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	locked, err := qtx.TryLockRewrap(ctx)
	if err != nil {
		return RewrapStats{}, fmt.Errorf("try rewrap lock: %w", err)
	}
	if !locked {
		return RewrapStats{Locked: false}, nil
	}

	rows, err := qtx.ListRewrapPending(ctx, gen.ListRewrapPendingParams{
		ActiveKekID: activeKEK,
		BatchSize:   batchSize,
	})
	if err != nil {
		return RewrapStats{}, fmt.Errorf("list rewrap pending: %w", err)
	}

	stats := RewrapStats{Locked: true}
	for _, r := range rows {
		row := RewrapRow{
			ID:        uuidString(r.ID),
			AccountID: uuidString(r.AccountID),
			Provider:  r.Provider,

			EncKey:     r.EncKey,
			EncDataKey: r.EncDataKey,

			EncKeyAD:     r.EncKeyAd,
			EncDataKeyAD: r.EncDataKeyAd,
			KEKID:        r.KekID.String,
			ADSrcDigest:  r.AdSrcDigest,
		}
		encKeyAD, encDataKeyAD, kekID, err := reseal(row)
		if err != nil {
			stats.Skipped++
			continue
		}
		// Optimistic on the legacy pair: the WHERE pins enc_data_key to the bytes we
		// decrypted from, so a PUT that replaced the key during this pass wins (0 rows)
		// rather than being overwritten with a pair bound to the old key.
		n, err := qtx.UpdateApiKeyAD(ctx, gen.UpdateApiKeyADParams{
			ID:             r.ID,
			EncKeyAd:       encKeyAD,
			EncDataKeyAd:   encDataKeyAD,
			KekID:          pgtype.Text{String: kekID, Valid: kekID != ""},
			SeenEncDataKey: r.EncDataKey,
		})
		if err != nil {
			return stats, fmt.Errorf("update api key ad: %w", err)
		}
		if n == 0 {
			stats.Raced++
			continue
		}
		stats.Rewrapped++
	}

	pending, err := qtx.CountRewrapPending(ctx, activeKEK)
	if err != nil {
		return stats, fmt.Errorf("count rewrap pending: %w", err)
	}
	stats.Pending = pending

	if err := tx.Commit(ctx); err != nil {
		return stats, fmt.Errorf("commit rewrap tx: %w", err)
	}
	return stats, nil
}

// RewrapPending reports how many key configs still need an AD pair under activeKEK.
// Read on demand (D34: a log line, not an alert) and the precondition l-01 checks before
// it stops writing the legacy pair.
func (s *PgStore) RewrapPending(ctx context.Context, activeKEK string) (int64, error) {
	n, err := s.q.CountRewrapPending(ctx, activeKEK)
	if err != nil {
		return 0, fmt.Errorf("count rewrap pending: %w", err)
	}
	return n, nil
}

// LegacyPairDigest returns PG's own sha256 of a row's legacy wrapped data key — the exact
// value the re-wrap selector compares ad_src_digest against. Used by a test to prove Go's
// crypto/sha256 and Postgres's sha256(bytea) agree on real stored bytes: if they ever
// disagreed the selector would see every row as stale forever and the job would spin
// re-wrapping the same keys.
func (s *PgStore) LegacyPairDigest(ctx context.Context, keyConfigID string) ([]byte, error) {
	id, err := parseUUID(keyConfigID)
	if err != nil {
		return nil, fmt.Errorf("parse key config id: %w", err)
	}
	d, err := s.q.Sha256OfLegacyPair(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("sha256 of legacy pair: %w", err)
	}
	return d, nil
}

// digestMatches reports whether want equals sha256(encDataKey), in constant time. A
// missing or wrong-length digest never matches, so a row written before 00006 (digest
// NULL) is treated as stale and re-wrapped rather than trusted.
func digestMatches(want, encDataKey []byte) bool {
	if len(want) != sha256.Size {
		return false
	}
	got := sha256.Sum256(encDataKey)
	return subtle.ConstantTimeCompare(want, got[:]) == 1
}
