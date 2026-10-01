package store_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/coach"
	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// The background AD re-wrap, against a real Postgres (m1-10 task 3 / task 6).
//
// These cases exist because every one of them is a way the job could lose or mis-serve a
// learner's key, and none of them is reachable through the HTTP surface:
//
//	legacy → k0            the backfill migration 00006 deliberately does not do
//	rotation k0 → k1       a master-key rotation with no downtime and no re-paste
//	corrupted row          must be SKIPPED and left byte-identical, never deleted
//	racing PUT             a key replaced mid-pass must win over the job
//	stale pair             a v1.6.0 key replace during a rollback, then roll forward
//	Go vs PG sha256        the selector compares the two; disagreement = an infinite loop
//	single runner          the advisory lock, and that it is not leaked on the pool
//
// Gated on XLEARN_TEST_DATABASE_URL, like the other store integration tests.
//
// Two deliberate choices about how these run:
//
//   - They drive the REAL job (coach.Rewrapper), not a local copy of the reseal logic. An
//     earlier draft reimplemented the resealer here and silently diverged from production
//     on the ErrUnknownKEK fallback — the test passed while the real path went untested.
//     package store_test may import coach: coach imports store, and this external test
//     package is distinct from store, so there is no cycle.
//   - Each test TRUNCATES coach.api_key_config first. The other store tests coexist by
//     using random account ids, but the re-wrap job is inherently whole-table — it scans
//     every pending row — so rows left by a sibling test, sealed under that test's own
//     random master key and therefore undecryptable here, would surface as skipped and
//     make a pass's statistics meaningless.

// rewrapEnv is one test's database, store, cipher and keyring.
type rewrapEnv struct {
	ctx    context.Context
	pool   *pgxpool.Pool
	st     *store.PgStore
	cipher *secrets.Cipher // the legacy (unbound) master key
	k0     []byte
	t      *testing.T
}

func newRewrapEnv(t *testing.T) *rewrapEnv {
	t.Helper()
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the coach re-wrap integration test")
	}
	ctx := context.Background()
	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// The job works the whole table, so start from an empty one (see the file comment).
	// CASCADE takes the key_default rows that reference these keys.
	if _, err := pool.Exec(ctx, `TRUNCATE coach.api_key_config CASCADE`); err != nil {
		t.Fatalf("truncate api_key_config: %v", err)
	}

	k0 := randomKey(t)
	cipher, err := secrets.NewCipher(k0)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	return &rewrapEnv{ctx: ctx, pool: pool, st: store.New(pool), cipher: cipher, k0: k0, t: t}
}

func randomKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, secrets.MasterKeySize)
	if _, err := rand.Read(k); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return k
}

// putLegacyOnly writes a key with ONLY the legacy sealed pair — exactly the row shape
// v1.6.0 leaves behind, and what the backfill has to find.
func (e *rewrapEnv) putLegacyOnly(account, provider, rawKey string) {
	e.t.Helper()
	encKey, encDataKey, err := e.cipher.Seal([]byte(rawKey))
	if err != nil {
		e.t.Fatalf("seal: %v", err)
	}
	if _, err := e.pool.Exec(e.ctx, `
		INSERT INTO coach.api_key_config (account_id, provider, enc_key, enc_data_key, masked_key, default_model, name)
		VALUES ($1, $2, $3, $4, $5, 'm', 'n')
		ON CONFLICT (account_id, provider) DO UPDATE
		SET enc_key = EXCLUDED.enc_key, enc_data_key = EXCLUDED.enc_data_key,
		    masked_key = EXCLUDED.masked_key`,
		account, provider, encKey, encDataKey, secrets.Mask(rawKey)); err != nil {
		e.t.Fatalf("insert legacy-only key: %v", err)
	}
}

// row reads the stored pairs for (account, provider).
func (e *rewrapEnv) row(account, provider string) store.KeyConfig {
	e.t.Helper()
	k, err := e.st.GetKey(e.ctx, account, provider)
	if err != nil {
		e.t.Fatalf("get key %s/%s: %v", account, provider, err)
	}
	return k
}

// job builds the real re-wrap job over this test's store, legacy cipher and keyring.
func (e *rewrapEnv) job(kr *secrets.Keyring) *coach.Rewrapper {
	return coach.NewRewrapper(e.st, e.cipher, kr, testLogger())
}

// resealerFor is the real job's reseal step as a store.Resealer, so RewrapBatch-level
// tests (the advisory lock, the batch bound, the PUT race) exercise production logic.
func (e *rewrapEnv) resealerFor(kr *secrets.Keyring) store.Resealer {
	return e.job(kr).Reseal
}

// TestRewrapBackfillsLegacyRowsUnderK0 is the backfill: migration 00006 adds the columns
// as NULL and the job fills them, because a backfill inside a migration would need the
// master key, which lives in the pod rather than in goose.
func TestRewrapBackfillsLegacyRowsUnderK0(t *testing.T) {
	e := newRewrapEnv(t)
	kr := secrets.KeyringOf("k0", e.cipher)

	acct := newTestUUID()
	const raw = "sk-ant-legacy-only-key-1234"
	e.putLegacyOnly(acct, store.ProviderAnthropic, raw)

	before := e.row(acct, store.ProviderAnthropic)
	if len(before.EncKeyAD) != 0 || before.ADPairCurrent() {
		t.Fatal("the seeded row should have no AD pair")
	}

	stats, err := e.st.RewrapBatch(e.ctx, "k0", 50, e.resealerFor(kr))
	if err != nil {
		t.Fatalf("rewrap: %v", err)
	}
	if !stats.Locked || stats.Rewrapped == 0 || stats.Skipped != 0 {
		t.Fatalf("stats = %+v, want locked with at least one rewrapped and none skipped", stats)
	}

	after := e.row(acct, store.ProviderAnthropic)
	if !after.ADPairCurrent() {
		t.Fatal("the AD pair is not current after a pass")
	}
	if after.KEKID != "k0" {
		t.Fatalf("kek_id = %q, want k0", after.KEKID)
	}
	// The digest must match the legacy pair it was sealed beside.
	want := sha256.Sum256(after.EncDataKey)
	if !bytes.Equal(after.ADSrcDigest, want[:]) {
		t.Fatal("ad_src_digest does not describe the legacy pair beside it")
	}
	// The AD pair decrypts to the original secret, under the right associated data.
	got, err := kr.OpenAD(after.EncKeyAD, after.EncDataKeyAD, after.KEKID,
		secrets.CoachKeyAD(acct, store.ProviderAnthropic))
	if err != nil || string(got) != raw {
		t.Fatalf("AD open = (%q, %v), want %q", got, err, raw)
	}
	// And is genuinely bound: another account's ad must not open it.
	if _, err := kr.OpenAD(after.EncKeyAD, after.EncDataKeyAD, after.KEKID,
		secrets.CoachKeyAD(newTestUUID(), store.ProviderAnthropic)); err == nil {
		t.Fatal("the re-wrapped pair is not bound to its account")
	}
	// The LEGACY pair is untouched — that is what keeps v1.6.0 readable (ADR-0034 §3).
	if !bytes.Equal(before.EncKey, after.EncKey) || !bytes.Equal(before.EncDataKey, after.EncDataKey) {
		t.Fatal("the re-wrap modified the legacy pair; the rollback floor would be broken")
	}

	// A second pass finds nothing: the job must settle, not churn.
	stats2, err := e.st.RewrapBatch(e.ctx, "k0", 50, e.resealerFor(kr))
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if stats2.Rewrapped != 0 {
		t.Fatalf("second pass re-wrapped %d rows; the selector never settles", stats2.Rewrapped)
	}
}

// TestRewrapRotatesK0ToK1: adding a new active entry makes every row pending, and the job
// moves them across while the retiring key can still open what it sealed — which is what
// makes a rotation zero-downtime.
func TestRewrapRotatesK0ToK1(t *testing.T) {
	e := newRewrapEnv(t)
	acct := newTestUUID()
	const raw = "sk-ant-rotate-me-5678"
	e.putLegacyOnly(acct, store.ProviderAnthropic, raw)

	// Backfill under k0.
	k0ring := secrets.KeyringOf("k0", e.cipher)
	if _, err := e.st.RewrapBatch(e.ctx, "k0", 50, e.resealerFor(k0ring)); err != nil {
		t.Fatalf("k0 pass: %v", err)
	}
	onK0 := e.row(acct, store.ProviderAnthropic)
	if onK0.KEKID != "k0" {
		t.Fatalf("kek_id = %q, want k0", onK0.KEKID)
	}

	// Rotate: k1 becomes active, k0 stays available to open what it sealed.
	k1 := randomKey(t)
	k1ring, err := secrets.NewKeyring("k1", map[string][]byte{"k1": k1, "k0": e.k0})
	if err != nil {
		t.Fatalf("keyring: %v", err)
	}
	// Under the new active key the row is pending again.
	pending, err := e.st.RewrapPending(e.ctx, "k1")
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if pending == 0 {
		t.Fatal("a rotation left nothing pending; rows would never move to the new KEK")
	}

	if _, err := e.st.RewrapBatch(e.ctx, "k1", 50, e.resealerFor(k1ring)); err != nil {
		t.Fatalf("k1 pass: %v", err)
	}
	onK1 := e.row(acct, store.ProviderAnthropic)
	if onK1.KEKID != "k1" {
		t.Fatalf("kek_id = %q, want k1 after the rotation", onK1.KEKID)
	}
	if !onK1.ADPairCurrent() {
		t.Fatal("the rotated AD pair is not current")
	}
	got, err := k1ring.OpenAD(onK1.EncKeyAD, onK1.EncDataKeyAD, onK1.KEKID,
		secrets.CoachKeyAD(acct, store.ProviderAnthropic))
	if err != nil || string(got) != raw {
		t.Fatalf("open under k1 = (%q, %v), want %q", got, err, raw)
	}
	// The pair really is under the NEW key: k0 alone must not open it.
	k0only := secrets.KeyringOf("k0", e.cipher)
	if _, err := k0only.OpenAD(onK1.EncKeyAD, onK1.EncDataKeyAD, "k0",
		secrets.CoachKeyAD(acct, store.ProviderAnthropic)); err == nil {
		t.Fatal("the rotated pair still opens under the retired key")
	}
}

// TestRewrapSkipsACorruptedRow: a row it cannot decrypt is reported and left EXACTLY as it
// was. Deleting or disabling it would turn an operator mistake (a KEK retired too early,
// say) into learner data loss — the key may be perfectly valid.
func TestRewrapSkipsACorruptedRow(t *testing.T) {
	e := newRewrapEnv(t)
	kr := secrets.KeyringOf("k0", e.cipher)

	good, bad := newTestUUID(), newTestUUID()
	e.putLegacyOnly(good, store.ProviderAnthropic, "sk-ant-good-key-1111")
	e.putLegacyOnly(bad, store.ProviderAnthropic, "sk-ant-bad-key-2222")

	// Corrupt the bad row's legacy pair so nothing can open it.
	if _, err := e.pool.Exec(e.ctx,
		`UPDATE coach.api_key_config SET enc_key = $2 WHERE account_id = $1`,
		bad, []byte("this-is-not-a-valid-nonce-prefixed-ciphertext")); err != nil {
		t.Fatalf("corrupt row: %v", err)
	}
	before := e.row(bad, store.ProviderAnthropic)

	stats, err := e.st.RewrapBatch(e.ctx, "k0", 50, e.resealerFor(kr))
	if err != nil {
		t.Fatalf("rewrap: %v", err)
	}
	if stats.Skipped == 0 {
		t.Fatalf("stats = %+v, want at least one skipped row", stats)
	}

	// The good row went through — one bad row must not stall the batch.
	if !e.row(good, store.ProviderAnthropic).ADPairCurrent() {
		t.Fatal("a corrupted row blocked its neighbours")
	}

	// The bad row still EXISTS, is still ENABLED, and is byte-identical.
	after := e.row(bad, store.ProviderAnthropic)
	if !after.Enabled {
		t.Fatal("the re-wrap disabled a key it could not decrypt")
	}
	if !bytes.Equal(before.EncKey, after.EncKey) || !bytes.Equal(before.EncDataKey, after.EncDataKey) ||
		len(after.EncKeyAD) != 0 || len(after.EncDataKeyAD) != 0 {
		t.Fatalf("the corrupted row was modified: %+v", after)
	}
	// It stays pending, so the owner can see it in the summary and re-paste the key.
	pending, err := e.st.RewrapPending(e.ctx, "k0")
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if pending == 0 {
		t.Fatal("a skipped row stopped being pending; it would never be reported again")
	}
}

// TestRewrapLosesToARacingPut: the update is optimistic on the legacy pair, so a PUT that
// replaced the key mid-pass wins. Without that, the job would overwrite the fresh AD pair
// with one bound to the key the learner just replaced.
func TestRewrapLosesToARacingPut(t *testing.T) {
	e := newRewrapEnv(t)
	kr := secrets.KeyringOf("k0", e.cipher)

	acct := newTestUUID()
	e.putLegacyOnly(acct, store.ProviderAnthropic, "sk-ant-old-key-3333")

	// A resealer that simulates the race: it decrypts the row the job observed, then —
	// before the UPDATE lands — a PUT replaces the key underneath it.
	const newRaw = "sk-ant-new-key-4444"
	raced := false
	reseal := func(row store.RewrapRow) ([]byte, []byte, string, error) {
		if !raced {
			raced = true
			e.putLegacyOnly(acct, store.ProviderAnthropic, newRaw)
		}
		return e.resealerFor(kr)(row)
	}

	stats, err := e.st.RewrapBatch(e.ctx, "k0", 50, reseal)
	if err != nil {
		t.Fatalf("rewrap: %v", err)
	}
	if stats.Raced == 0 {
		t.Fatalf("stats = %+v, want the racing row counted as raced (not rewrapped)", stats)
	}

	// The row now holds the NEW legacy pair, and no AD pair bound to the old key.
	after := e.row(acct, store.ProviderAnthropic)
	legacy, err := e.cipher.Open(after.EncKey, after.EncDataKey)
	if err != nil || string(legacy) != newRaw {
		t.Fatalf("legacy pair = (%q, %v), want the new key %q", legacy, err, newRaw)
	}
	if after.ADPairCurrent() {
		// If an AD pair were installed here it would have to be the OLD key's — the job
		// decrypted before the PUT landed.
		got, _ := kr.OpenAD(after.EncKeyAD, after.EncDataKeyAD, after.KEKID,
			secrets.CoachKeyAD(acct, store.ProviderAnthropic))
		t.Fatalf("a current AD pair was installed over a racing PUT (it holds %q)", got)
	}

	// The next pass repairs it from the new legacy pair.
	if _, err := e.st.RewrapBatch(e.ctx, "k0", 50, e.resealerFor(kr)); err != nil {
		t.Fatalf("repair pass: %v", err)
	}
	repaired := e.row(acct, store.ProviderAnthropic)
	if !repaired.ADPairCurrent() {
		t.Fatal("the row was not repaired on the next pass")
	}
	got, err := kr.OpenAD(repaired.EncKeyAD, repaired.EncDataKeyAD, repaired.KEKID,
		secrets.CoachKeyAD(acct, store.ProviderAnthropic))
	if err != nil || string(got) != newRaw {
		t.Fatalf("repaired AD pair = (%q, %v), want the new key", got, err)
	}
}

// TestRewrapRepairsAStalePair is the rollback path, and the reason ad_src_digest exists.
//
// A v1.6.0 image replacing a key rewrites ONLY the legacy columns (its UpsertApiKeyConfig
// knows nothing about the AD ones), so after rolling forward the AD pair still holds the
// PREVIOUS key. Chat must read the newer legacy pair meanwhile — if it answered from the
// stale AD pair and the owner had rotated because the old key was revoked, the resulting
// 401 would disable the key they just pasted.
func TestRewrapRepairsAStalePair(t *testing.T) {
	e := newRewrapEnv(t)
	kr := secrets.KeyringOf("k0", e.cipher)

	acct := newTestUUID()
	const oldRaw = "sk-ant-key-before-rollback-5555"
	const newRaw = "sk-ant-key-pasted-on-v160-6666"

	// Normal state: both pairs, current.
	e.putLegacyOnly(acct, store.ProviderAnthropic, oldRaw)
	if _, err := e.st.RewrapBatch(e.ctx, "k0", 50, e.resealerFor(kr)); err != nil {
		t.Fatalf("initial pass: %v", err)
	}
	if !e.row(acct, store.ProviderAnthropic).ADPairCurrent() {
		t.Fatal("setup: the AD pair should be current")
	}

	// Now v1.6.0 replaces the key: legacy pair only.
	e.putLegacyOnly(acct, store.ProviderAnthropic, newRaw)

	stale := e.row(acct, store.ProviderAnthropic)
	if stale.ADPairCurrent() {
		t.Fatal("a legacy-only rewrite was not detected as stale; chat would use the OLD key")
	}
	// The AD pair is still there, still decryptable — it just holds the wrong key, which
	// is precisely why "present and decryptable" is not the test for usability.
	if orphan, err := kr.OpenAD(stale.EncKeyAD, stale.EncDataKeyAD, stale.KEKID,
		secrets.CoachKeyAD(acct, store.ProviderAnthropic)); err != nil || string(orphan) != oldRaw {
		t.Fatalf("the stale AD pair holds %q (err %v), want the old key — the premise of this test", orphan, err)
	}
	// And the legacy pair holds the new key, which is what chat must fall back to.
	if legacy, err := e.cipher.Open(stale.EncKey, stale.EncDataKey); err != nil || string(legacy) != newRaw {
		t.Fatalf("legacy pair = (%q, %v), want the new key", legacy, err)
	}

	// The next pass re-derives the AD pair FROM THE LEGACY PAIR (never from the stale AD
	// pair, which would make the staleness permanent).
	if _, err := e.st.RewrapBatch(e.ctx, "k0", 50, e.resealerFor(kr)); err != nil {
		t.Fatalf("repair pass: %v", err)
	}
	repaired := e.row(acct, store.ProviderAnthropic)
	if !repaired.ADPairCurrent() {
		t.Fatal("the stale row was not repaired")
	}
	got, err := kr.OpenAD(repaired.EncKeyAD, repaired.EncDataKeyAD, repaired.KEKID,
		secrets.CoachKeyAD(acct, store.ProviderAnthropic))
	if err != nil || string(got) != newRaw {
		t.Fatalf("repaired AD pair = (%q, %v), want the NEW key %q", got, err, newRaw)
	}
}

// TestGoAndPostgresSha256Agree: the selector compares ad_src_digest (written by Go on the
// PUT path, by PG on the re-wrap path) against PG's sha256(enc_data_key). If the two ever
// disagreed, every row would read as stale forever and the job would spin re-wrapping the
// same keys every ten minutes.
func TestGoAndPostgresSha256Agree(t *testing.T) {
	e := newRewrapEnv(t)
	acct := newTestUUID()
	e.putLegacyOnly(acct, store.ProviderAnthropic, "sk-ant-digest-7777")

	k := e.row(acct, store.ProviderAnthropic)
	// Find the row id (GetKey does not expose it as a DB id for this purpose).
	var id string
	if err := e.pool.QueryRow(e.ctx,
		`SELECT id::text FROM coach.api_key_config WHERE account_id = $1 AND provider = $2`,
		acct, store.ProviderAnthropic).Scan(&id); err != nil {
		t.Fatalf("read id: %v", err)
	}
	pg, err := e.st.LegacyPairDigest(e.ctx, id)
	if err != nil {
		t.Fatalf("pg digest: %v", err)
	}
	goDigest := sha256.Sum256(k.EncDataKey)
	if !bytes.Equal(pg, goDigest[:]) {
		t.Fatalf("PG sha256 = %x, Go sha256 = %x; the re-wrap selector would never settle", pg, goDigest)
	}
	if len(pg) != sha256.Size {
		t.Fatalf("PG digest is %d bytes, want %d", len(pg), sha256.Size)
	}
}

// TestRewrapRunsOneAtATime covers the lock. Two concurrent passes must not both work the
// same rows, and — the part that actually bites in production — no advisory lock may be
// left held on a pooled connection afterwards. A session-level pg_try_advisory_lock taken
// through pgxpool can have its unlock issued on a DIFFERENT connection, leaking the lock
// for the life of the pod and silently stopping every future pass; the transaction-scoped
// lock cannot, because COMMIT/ROLLBACK releases it on the connection that took it.
func TestRewrapRunsOneAtATime(t *testing.T) {
	e := newRewrapEnv(t)
	kr := secrets.KeyringOf("k0", e.cipher)

	// Enough rows that a pass has real work to hold the lock through.
	for i := 0; i < 6; i++ {
		e.putLegacyOnly(newTestUUID(), store.ProviderAnthropic, "sk-ant-concurrent-key-"+string(rune('a'+i)))
	}

	// Gate the first pass inside its transaction (so it holds the lock) until the second
	// has tried and been turned away.
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once

	gated := func(row store.RewrapRow) ([]byte, []byte, string, error) {
		once.Do(func() {
			close(entered)
			<-release
		})
		return e.resealerFor(kr)(row)
	}

	var wg sync.WaitGroup
	var firstStats, secondStats store.RewrapStats
	var firstErr, secondErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		firstStats, firstErr = e.st.RewrapBatch(e.ctx, "k0", 50, gated)
	}()

	<-entered // the first pass is inside the transaction, holding the lock
	secondStats, secondErr = e.st.RewrapBatch(e.ctx, "k0", 50, e.resealerFor(kr))
	close(release)
	wg.Wait()

	if firstErr != nil || secondErr != nil {
		t.Fatalf("errors: first=%v second=%v", firstErr, secondErr)
	}
	if !firstStats.Locked {
		t.Fatal("the first pass did not take the lock")
	}
	if secondStats.Locked {
		t.Fatalf("both passes took the lock (second = %+v); the runs are not serialised", secondStats)
	}
	if secondStats.Rewrapped != 0 {
		t.Fatalf("the locked-out pass re-wrapped %d rows", secondStats.Rewrapped)
	}
	if firstStats.Rewrapped == 0 {
		t.Fatal("the first pass re-wrapped nothing")
	}

	// NO advisory lock left anywhere once both passes have finished. hashtext() is what
	// the query uses, so compare on the same value.
	var held int
	if err := e.pool.QueryRow(e.ctx, `
		SELECT count(*) FROM pg_locks
		WHERE locktype = 'advisory'
		  AND ((classid::bigint << 32) | objid::bigint) = hashtext('coach.rewrap')::bigint`).Scan(&held); err != nil {
		// Some deployments compute the key differently; fall back to "any advisory lock".
		if err2 := e.pool.QueryRow(e.ctx,
			`SELECT count(*) FROM pg_locks WHERE locktype = 'advisory'`).Scan(&held); err2 != nil {
			t.Fatalf("pg_locks: %v / %v", err, err2)
		}
	}
	if held != 0 {
		t.Fatalf("%d advisory lock(s) still held after the passes; a session-scoped lock leaked on the pool", held)
	}
}

// TestRewrapBatchSizeIsHonoured keeps the pass inside its memory budget (ADR-0035 §5: the
// goroutine lives in coach's existing 128 Mi, with no new pod).
func TestRewrapBatchSizeIsHonoured(t *testing.T) {
	e := newRewrapEnv(t)
	kr := secrets.KeyringOf("k0", e.cipher)

	for i := 0; i < 5; i++ {
		e.putLegacyOnly(newTestUUID(), store.ProviderAnthropic, "sk-ant-batch-key-"+string(rune('a'+i)))
	}
	stats, err := e.st.RewrapBatch(e.ctx, "k0", 2, e.resealerFor(kr))
	if err != nil {
		t.Fatalf("rewrap: %v", err)
	}
	if stats.Rewrapped > 2 {
		t.Fatalf("batch of 2 re-wrapped %d rows", stats.Rewrapped)
	}
	if stats.Pending == 0 {
		t.Fatal("a partial batch reported nothing pending; the job would stop early")
	}
}

// TestRewrapUnknownKEKIsNotCorruption: a pair sealed under a retired entry reports
// ErrUnknownKEK, which the job can tell apart from a corrupt row — "restore the key
// entry", not "every learner re-pastes".
func TestRewrapUnknownKEKIsNotCorruption(t *testing.T) {
	e := newRewrapEnv(t)
	acct := newTestUUID()
	e.putLegacyOnly(acct, store.ProviderAnthropic, "sk-ant-retired-kek-8888")

	// Seal the AD pair under k9, then present a keyring that does not hold k9.
	k9 := randomKey(t)
	k9cipher, err := secrets.NewCipher(k9)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	k9ring, err := secrets.NewKeyring("k9", map[string][]byte{"k9": k9})
	if err != nil {
		t.Fatalf("keyring: %v", err)
	}
	if _, err := e.st.RewrapBatch(e.ctx, "k9", 50, e.resealerFor(k9ring)); err != nil {
		t.Fatalf("k9 pass: %v", err)
	}
	_ = k9cipher

	// A keyring without k9 cannot open that pair.
	k0ring := secrets.KeyringOf("k0", e.cipher)
	row := e.row(acct, store.ProviderAnthropic)
	_, err = k0ring.OpenAD(row.EncKeyAD, row.EncDataKeyAD, row.KEKID,
		secrets.CoachKeyAD(acct, store.ProviderAnthropic))
	if !errors.Is(err, secrets.ErrUnknownKEK) {
		t.Fatalf("open under a missing entry = %v, want ErrUnknownKEK", err)
	}
	// But the legacy pair still opens, so the job repairs the row rather than skipping it
	// — which is why the legacy pair is kept until the floor passes 1.7.0.
	if _, err := e.st.RewrapBatch(e.ctx, "k0", 50, e.resealerFor(k0ring)); err != nil {
		t.Fatalf("k0 repair pass: %v", err)
	}
	repaired := e.row(acct, store.ProviderAnthropic)
	if repaired.KEKID != "k0" || !repaired.ADPairCurrent() {
		t.Fatalf("row not recovered onto k0: kek=%q current=%v", repaired.KEKID, repaired.ADPairCurrent())
	}
}
