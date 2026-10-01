package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// The re-wrap must SETTLE on a row with no legacy pair — the state l-01 creates when it
// stops writing one.
//
// This is the write half of a forward-compat path the rest of the sprint already relies
// on: KeyConfig.ADPairCurrent() returns true for `enc_data_key IS NULL` (so chat reads the
// AD pair), and a unit test covers that read. The WRITE half did not work, and the m1-10
// adversarial review caught it:
//
//   - `sha256(NULL)` is NULL, and `<non-null digest> IS DISTINCT FROM NULL` is TRUE, so
//     the selector marked such a row pending FOREVER while the currency rule called it
//     current — the two disagreed;
//   - the optimistic UPDATE pinned `enc_data_key = $seen`, and `NULL = NULL` is NULL, so
//     the write could never land either.
//
// Net effect, had it shipped: every pass would re-seal the row and discard the result,
// `pending` would never reach 0, the job would sit on the 10-minute busy cadence forever,
// and **l-01's own precondition (`pending=0 skipped=0`) would be unsatisfiable** — the
// gate that authorises dropping the legacy pair.
//
// The columns are still NOT NULL in the schema, so the test drops the constraint for its
// own transaction-free scope and restores it, rather than waiting for l-01 to make the
// state reachable.
func TestRewrapSettlesOnARowWithNoLegacyPair(t *testing.T) {
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
	defer pool.Close()
	if _, err := pool.Exec(ctx, `TRUNCATE coach.api_key_config CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	// Simulate l-01's expand, and put it back afterwards so no other test sees a relaxed
	// schema.
	relax := []string{
		`ALTER TABLE coach.api_key_config ALTER COLUMN enc_key DROP NOT NULL`,
		`ALTER TABLE coach.api_key_config ALTER COLUMN enc_data_key DROP NOT NULL`,
	}
	for _, q := range relax {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatalf("relax NOT NULL: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM coach.api_key_config WHERE enc_key IS NULL OR enc_data_key IS NULL`)
		_, _ = pool.Exec(ctx, `ALTER TABLE coach.api_key_config ALTER COLUMN enc_key SET NOT NULL`)
		_, _ = pool.Exec(ctx, `ALTER TABLE coach.api_key_config ALTER COLUMN enc_data_key SET NOT NULL`)
	})

	master := make([]byte, secrets.MasterKeySize)
	for i := range master {
		master[i] = byte(i + 1)
	}
	cipher, err := secrets.NewCipher(master)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	kr := secrets.KeyringOf("k0", cipher)
	st := store.New(pool)

	// A post-l-01 row: AD pair only, no legacy pair, no digest.
	acct := newTestUUID()
	const raw = "sk-ant-no-legacy-pair-4242"
	ad := secrets.CoachKeyAD(acct, store.ProviderAnthropic)
	encKeyAD, encDataKeyAD, kekID, err := kr.SealAD([]byte(raw), ad)
	if err != nil {
		t.Fatalf("seal ad: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO coach.api_key_config
			(account_id, provider, enc_key, enc_data_key, enc_key_ad, enc_data_key_ad, kek_id,
			 masked_key, default_model, name)
		VALUES ($1, $2, NULL, NULL, $3, $4, $5, 'sk-...4242', 'claude-sonnet-5', 'n')`,
		acct, store.ProviderAnthropic, encKeyAD, encDataKeyAD, kekID); err != nil {
		t.Fatalf("insert NULL-legacy row: %v", err)
	}

	// The row is CURRENT, so it must not be pending at all.
	k, err := st.GetKey(ctx, acct, store.ProviderAnthropic)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if !k.ADPairCurrent() {
		t.Fatal("a row with no legacy pair and an AD pair must read as current")
	}
	pending, err := st.RewrapPending(ctx, "k0")
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending = %d, want 0 — the selector disagrees with ADPairCurrent, so l-01's precondition could never be met", pending)
	}

	// Chat still reads it: the AD pair opens under its own kek_id.
	got, err := kr.OpenAD(k.EncKeyAD, k.EncDataKeyAD, k.KEKID, ad)
	if err != nil || string(got) != raw {
		t.Fatalf("AD open = (%q, %v), want %q", got, err, raw)
	}

	// And a ROTATION of such a row lands: under a new active KEK it becomes pending, and
	// the optimistic update must match despite the NULL legacy pair.
	k1 := make([]byte, secrets.MasterKeySize)
	for i := range k1 {
		k1[i] = byte(200 - i)
	}
	k1ring, err := secrets.NewKeyring("k1", map[string][]byte{"k1": k1, "k0": master})
	if err != nil {
		t.Fatalf("keyring: %v", err)
	}
	pending, err = st.RewrapPending(ctx, "k1")
	if err != nil {
		t.Fatalf("pending under k1: %v", err)
	}
	if pending != 1 {
		t.Fatalf("pending under the new KEK = %d, want 1", pending)
	}

	stats, err := st.RewrapBatch(ctx, "k1", 50, func(row store.RewrapRow) ([]byte, []byte, string, error) {
		raw, err := k1ring.OpenAD(row.EncKeyAD, row.EncDataKeyAD, row.KEKID,
			secrets.CoachKeyAD(row.AccountID, row.Provider))
		if err != nil {
			return nil, nil, "", err
		}
		defer secrets.Zero(raw)
		return k1ring.SealAD(raw, secrets.CoachKeyAD(row.AccountID, row.Provider))
	})
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if stats.Rewrapped != 1 || stats.Raced != 0 {
		t.Fatalf("stats = %+v, want exactly one rewrapped and none raced — the optimistic pin could not match a NULL legacy pair", stats)
	}
	if stats.Pending != 0 {
		t.Fatalf("pending after the rotation = %d, want 0", stats.Pending)
	}

	rotated, err := st.GetKey(ctx, acct, store.ProviderAnthropic)
	if err != nil {
		t.Fatalf("get rotated key: %v", err)
	}
	if rotated.KEKID != "k1" || !rotated.ADPairCurrent() {
		t.Fatalf("after rotation: kek=%q current=%v", rotated.KEKID, rotated.ADPairCurrent())
	}
	got, err = k1ring.OpenAD(rotated.EncKeyAD, rotated.EncDataKeyAD, rotated.KEKID, ad)
	if err != nil || string(got) != raw {
		t.Fatalf("rotated AD open = (%q, %v), want %q", got, err, raw)
	}
}
