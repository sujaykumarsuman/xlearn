package coach

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// The background AD re-wrap (m1-10 task 3; ADR-0031 §7).
//
// Migration 00006 adds the AD-bound sealed pair as NULLable columns and backfills
// nothing: a backfill inside a migration would have to hold the master key, which lives
// in the pod, not in goose. This goroutine does it instead — a few rows at a time, inside
// the existing coach pod, with no new container and no new memory budget (ADR-0035 §5:
// batches of 50 keep a pass well inside coach's 128 Mi).
//
// It is also the REPAIR path, which is the part that keeps earning its keep after the
// backfill is done:
//
//   - a master-key rotation (a new active entry in COACH_MASTER_KEYS) makes every row
//     pending, and this moves them onto the new KEK with no downtime and no re-paste;
//   - a v1.6.0 image replacing a key during a rollback rewrites only the LEGACY pair,
//     leaving the AD pair describing the previous key. The digest detects it, chat falls
//     back to the (newer) legacy pair meanwhile, and the next pass re-derives the AD pair.
//
// Reporting is a LOG LINE READ ON DEMAND, deliberately (D34): no alert, no push channel,
// no opscheck. An INFO summary per pass, a WARN per skipped row carrying only the
// key_config_id. Never a key, a plaintext, or a ciphertext.
//
// Nothing here ever deletes or disables a key. A row it cannot decrypt is one the owner
// may still need — most likely sealed under a KEK retired too early — so it is skipped,
// counted and reported, and the owner re-pastes the key if it matters.

// Re-wrap cadence. On start, then every rewrapBusyInterval while work remains, then
// rewrapIdleInterval once everything is current. The fast cadence exists so the backfill
// after a deploy finishes in minutes rather than hours; the slow one exists so a steady
// state costs one cheap indexed count per hour.
const (
	rewrapBatchSize      = 50
	rewrapBusyInterval   = 10 * time.Minute
	rewrapIdleInterval   = time.Hour
	rewrapStartupDelay   = 5 * time.Second
	rewrapPassTimeout    = 2 * time.Minute
	rewrapErrorBackoff   = time.Minute
	rewrapMaxBatchesPass = 20 // 1000 keys per pass; a bound, not a target
)

// RewrapStore is the persistence seam the job needs. *store.PgStore implements it; a test
// fake can too.
type RewrapStore interface {
	RewrapBatch(ctx context.Context, activeKEK string, batchSize int32, reseal store.Resealer) (store.RewrapStats, error)
	RewrapPending(ctx context.Context, activeKEK string) (int64, error)
}

// Rewrapper re-seals stored keys onto the active KEK with associated data.
type Rewrapper struct {
	store  RewrapStore
	cipher *secrets.Cipher
	keys   *secrets.Keyring
	log    *slog.Logger
}

// NewRewrapper builds the job. cipher opens the legacy pair; keys holds the KEKs.
func NewRewrapper(st RewrapStore, cipher *secrets.Cipher, keys *secrets.Keyring, log *slog.Logger) *Rewrapper {
	if keys == nil {
		keys = secrets.KeyringOf(DefaultKEKID, cipher)
	}
	return &Rewrapper{store: st, cipher: cipher, keys: keys, log: log}
}

// Run drives the job until ctx is cancelled. Call it in a goroutine from cmd/coach with
// the service context; it returns on shutdown.
func (w *Rewrapper) Run(ctx context.Context) {
	w.log.Info("coach rewrap: started", "active_kek", w.keys.Active(), "batch_size", rewrapBatchSize)

	// A short delay so the first pass doesn't compete with migrations, the pool warming
	// up and the readiness probe on a cold start.
	if !sleepCtx(ctx, rewrapStartupDelay) {
		return
	}

	for {
		interval := rewrapIdleInterval
		switch pending, err := w.Pass(ctx); {
		case err != nil:
			if ctx.Err() != nil {
				return
			}
			// A transient database failure must not end the job for the life of the pod.
			w.log.Warn("coach rewrap: pass failed; will retry", "err", err)
			interval = rewrapErrorBackoff
		case pending > 0:
			interval = rewrapBusyInterval
		}
		if !sleepCtx(ctx, interval) {
			return
		}
	}
}

// Pass runs batches until nothing is pending, another runner holds the lock, or the batch
// bound is hit. It returns how many rows still need work.
//
// A row it cannot decrypt STAYS PENDING (that is the point — the owner has to see it), and
// batches are ordered by id, so every later batch in the same pass is offered that row
// again. Left alone, one corrupt row is therefore WARNed once per batch and counted once
// per batch, and the summary contradicts itself: "skipped=2 pending=1". So a pass
// remembers which rows it has already given up on, reports each at most once, and stops
// as soon as a batch contains nothing it has not already tried.
func (w *Rewrapper) Pass(ctx context.Context) (int64, error) {
	passCtx, cancel := context.WithTimeout(ctx, rewrapPassTimeout)
	defer cancel()

	skipped := make(map[string]bool) // rows this pass has already reported
	fresh := 0                       // rows attempted for the first time in this batch

	reseal := func(row store.RewrapRow) ([]byte, []byte, string, error) {
		if skipped[row.ID] {
			// Already tried and reported in this pass: fail it again (it is still
			// unfixable) but silently, and do not count it twice.
			return nil, nil, "", errAlreadySkipped
		}
		fresh++
		encKeyAD, encDataKeyAD, kekID, err := w.Reseal(row)
		if err != nil {
			skipped[row.ID] = true
		}
		return encKeyAD, encDataKeyAD, kekID, err
	}

	var total store.RewrapStats
	for i := 0; i < rewrapMaxBatchesPass; i++ {
		fresh = 0
		stats, err := w.store.RewrapBatch(passCtx, w.keys.Active(), rewrapBatchSize, reseal)
		if err != nil {
			return 0, err
		}
		if !stats.Locked {
			// Another replica (or a pass still draining during a rolling update) owns it.
			w.log.Info("coach rewrap: another runner holds the lock; ending pass")
			return 0, nil
		}
		total.Rewrapped += stats.Rewrapped
		total.Raced += stats.Raced
		total.Pending = stats.Pending

		switch {
		case stats.Pending == 0:
			// Everything is current.
			i = rewrapMaxBatchesPass
		case fresh == 0:
			// The batch held only rows this pass has already given up on; another batch
			// would return the same ones.
			i = rewrapMaxBatchesPass
		case stats.Rewrapped == 0:
			// No progress: the remaining rows are unfixable or racing.
			i = rewrapMaxBatchesPass
		}
		if i >= rewrapMaxBatchesPass {
			break
		}
	}
	total.Skipped = len(skipped) // DISTINCT rows, so the summary agrees with `pending`

	if total.Rewrapped > 0 || total.Skipped > 0 || total.Raced > 0 || total.Pending > 0 {
		w.log.Info("coach rewrap: pass",
			"rewrapped", total.Rewrapped,
			"skipped", total.Skipped,
			"raced", total.Raced,
			"pending", total.Pending,
			"active_kek", w.keys.Active(),
		)
	}
	return total.Pending, nil
}

// errAlreadySkipped marks a row this pass has already reported as unfixable, so the batch
// treats it as a skip without a second WARN. Static and secret-free.
var errAlreadySkipped = errors.New("coach rewrap: already skipped in this pass")

// Reseal decrypts one row and seals a fresh AD pair under the active KEK. It satisfies
// store.Resealer and is exported so the store's integration tests can drive THIS decision
// tree against a real Postgres rather than a copy of it — an earlier draft reimplemented
// it in the test and silently diverged on the ErrUnknownKEK fallback below.
//
// WHICH PAIR IT READS FROM is the crux:
//
//   - a row whose AD pair is STALE, or that has none, decrypts from the LEGACY pair —
//     which in the stale case is the NEWER of the two, because only a v1.6.0 writer can
//     create that state and it writes the legacy columns. Reading the stale AD pair would
//     re-seal the PREVIOUS key and make the staleness permanent;
//   - a row whose AD pair is CURRENT is being rotated onto a new KEK, so it decrypts from
//     its own AD pair under the kek_id it was stamped with. (The legacy pair would also
//     work today, but will be gone from l-01 on, and a rotation must keep working then.)
//
// It never logs the plaintext and zeroes it before returning.
func (w *Rewrapper) Reseal(row store.RewrapRow) (encKeyAD, encDataKeyAD []byte, kekID string, err error) {
	ad := secrets.CoachKeyAD(row.AccountID, row.Provider)

	var raw []byte
	if row.ADPairCurrent() {
		raw, err = w.keys.OpenAD(row.EncKeyAD, row.EncDataKeyAD, row.KEKID, ad)
		if err != nil && errors.Is(err, secrets.ErrUnknownKEK) {
			// The entry that sealed it has been retired. The legacy pair is the only way
			// back in; if that is gone too (post-l-01), the row is genuinely unfixable
			// here and is skipped for the owner to re-paste.
			raw, err = w.openLegacy(row)
		}
	} else {
		raw, err = w.openLegacy(row)
	}
	if err != nil {
		// Only the row id — never the error's ciphertext context, never key material.
		w.log.Warn("coach rewrap: skipped key config", "key_config_id", row.ID)
		return nil, nil, "", err
	}
	defer secrets.Zero(raw)

	return w.sealAD(raw, ad)
}

// openLegacy opens the unbound v1 pair, or reports that there is none left to open.
func (w *Rewrapper) openLegacy(row store.RewrapRow) ([]byte, error) {
	if len(row.EncKey) == 0 || len(row.EncDataKey) == 0 {
		return nil, errNoLegacyPair
	}
	return w.cipher.Open(row.EncKey, row.EncDataKey)
}

// sealAD seals the plaintext under the active KEK with ad.
func (w *Rewrapper) sealAD(raw, ad []byte) (encKeyAD, encDataKeyAD []byte, kekID string, err error) {
	return w.keys.SealAD(raw, ad)
}

// errNoLegacyPair marks a row with neither a usable AD pair nor a legacy pair — only
// reachable once l-01 has dropped the legacy pair AND the row's KEK was retired early.
// Static and secret-free, like every error on this path.
var errNoLegacyPair = errors.New("coach rewrap: no readable sealed pair")

// sleepCtx waits for d, returning false if ctx was cancelled first.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
