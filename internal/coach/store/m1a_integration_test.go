package store_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
)

// coach.key_default is the ONLY default source (m1-10, M1b).
//
// m1-02 (M1a) created key_default beside api_key_config.is_default and dual-wrote both,
// with readers preferring key_default and FALLING BACK to is_default. This test pinned
// that. m1-10 completes the expand, so the assertions here have deliberately flipped:
//
//	m1-02 (was)                             m1-10 (now)
//	────────────────────────────────────    ────────────────────────────────────
//	every writer dual-writes is_default     coach never writes is_default at all
//	readers fall back to is_default         there is no fallback; no row = no default
//	one feature ('coach')                   per-feature ('coach' implicit, 'interview'
//	                                        explicit only, never auto-promoted)
//
// Together with internal/coach/contract_test.go's grep (no is_default reader or writer in
// the package) this is m1-08's precondition for dropping the column in M1c.
func TestKeyDefaultIsTheOnlyDefaultSource(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the coach store integration test")
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
	st := store.New(pool)
	acct := newTestUUID()

	// keyDefault reads one feature's default row as (provider, model); "" when none.
	keyDefault := func(feature string) (string, string) {
		t.Helper()
		var provider, model string
		err := pool.QueryRow(ctx, `
			SELECT k.provider, d.model FROM coach.key_default d
			JOIN coach.api_key_config k ON k.id = d.key_id
			WHERE d.account_id = $1 AND d.feature = $2`, acct, feature).Scan(&provider, &model)
		if err != nil {
			return "", ""
		}
		return provider, model
	}
	// anyIsDefault reports whether ANY of the account's rows has is_default set. From
	// m1-10 on coach never writes the column, so this must stay false for the whole test
	// — the column keeps its DEFAULT false for the benefit of a v1.6.0 rollback target.
	anyIsDefault := func() bool {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM coach.api_key_config WHERE account_id = $1 AND is_default`, acct).Scan(&n); err != nil {
			t.Fatalf("count is_default: %v", err)
		}
		return n > 0
	}
	put := func(provider, model string) {
		t.Helper()
		if _, err := st.PutKey(ctx, store.KeyConfig{
			AccountID: acct, Provider: provider, EncKey: []byte("k"), EncDataKey: []byte("d"),
			Masked: "sk-…1234", DefaultModel: model, Name: provider,
		}); err != nil {
			t.Fatalf("put %s: %v", provider, err)
		}
	}
	// coachDefault returns the provider backing the coach default, cross-checking that
	// GetDefaultKey and ListKeys's IsDefault flag agree (both read key_default, so an
	// open v1.6.0 tab and the coach itself can never disagree about who answers).
	coachDefault := func() string {
		t.Helper()
		k, err := st.GetDefaultKey(ctx, acct, store.FeatureCoach)
		if errors.Is(err, store.ErrNotFound) {
			return ""
		}
		if err != nil {
			t.Fatalf("get default: %v", err)
		}
		keys, err := st.ListKeys(ctx, acct)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		n := 0
		for _, kc := range keys {
			if kc.IsDefault {
				n++
				if kc.Provider != k.Provider {
					t.Fatalf("ListKeys marks %s default, GetDefaultKey returns %s", kc.Provider, k.Provider)
				}
			}
		}
		if n != 1 || !k.IsDefault {
			t.Fatalf("%d keys marked default (GetDefaultKey IsDefault=%v), want exactly 1", n, k.IsDefault)
		}
		return k.Provider
	}

	// --- the coach feature: implicitly maintained ---

	// First key → the coach default. is_default is NOT written.
	put(store.ProviderOpenAI, "gpt-a")
	if p, m := keyDefault(store.FeatureCoach); p != store.ProviderOpenAI || m != "gpt-a" {
		t.Fatalf("first key: key_default %s/%s, want openai/gpt-a", p, m)
	}
	if anyIsDefault() {
		t.Fatal("PutKey wrote is_default; m1-08 cannot drop a column coach still writes")
	}
	// A second key leaves the default alone.
	put(store.ProviderAnthropic, "claude-a")
	if p, _ := keyDefault(store.FeatureCoach); p != store.ProviderOpenAI || coachDefault() != store.ProviderOpenAI {
		t.Fatalf("second key moved the default to %s", p)
	}
	// Replacing / re-modelling the default key carries its model into key_default.
	put(store.ProviderOpenAI, "gpt-b")
	if _, m := keyDefault(store.FeatureCoach); m != "gpt-b" {
		t.Fatalf("replace: key_default model %s, want gpt-b", m)
	}
	if _, err := st.UpdateKeyMeta(ctx, acct, store.ProviderOpenAI, "gpt-c", "renamed"); err != nil {
		t.Fatalf("update meta: %v", err)
	}
	if _, m := keyDefault(store.FeatureCoach); m != "gpt-c" {
		t.Fatalf("update meta: key_default model %s, want gpt-c", m)
	}
	// Re-modelling a NON-default key never touches key_default.
	if _, err := st.UpdateKeyMeta(ctx, acct, store.ProviderAnthropic, "claude-b", "x"); err != nil {
		t.Fatalf("update meta: %v", err)
	}
	if p, m := keyDefault(store.FeatureCoach); p != store.ProviderOpenAI || m != "gpt-c" {
		t.Fatalf("non-default meta: key_default %s/%s", p, m)
	}
	// Set-default moves it (and still writes no is_default).
	if _, err := st.SetDefault(ctx, acct, store.ProviderAnthropic, store.FeatureCoach, ""); err != nil {
		t.Fatalf("set default: %v", err)
	}
	if p, m := keyDefault(store.FeatureCoach); p != store.ProviderAnthropic || m != "claude-b" {
		t.Fatalf("set default: key_default %s/%s, want anthropic/claude-b", p, m)
	}
	if coachDefault() != store.ProviderAnthropic {
		t.Fatal("reader disagrees after set-default")
	}
	if anyIsDefault() {
		t.Fatal("SetDefault wrote is_default")
	}

	// --- no is_default fallback any more ---

	// A stale is_default (left by a v1.6.0 writer during a rollback) is IGNORED: it does
	// not move the default…
	if _, err := pool.Exec(ctx,
		`UPDATE coach.api_key_config SET is_default = (provider = 'openai') WHERE account_id = $1`, acct); err != nil {
		t.Fatalf("stale is_default: %v", err)
	}
	if got := coachDefault(); got != store.ProviderAnthropic {
		t.Fatalf("stale is_default moved the default to %s", got)
	}
	// … and with the key_default row gone there is NO default at all. m1-02 fell back to
	// is_default here; m1-10 must not, or the column would still be load-bearing.
	if _, err := pool.Exec(ctx, `DELETE FROM coach.key_default WHERE account_id = $1`, acct); err != nil {
		t.Fatalf("drop key_default: %v", err)
	}
	if _, err := st.GetDefaultKey(ctx, acct, store.FeatureCoach); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetDefaultKey fell back to is_default (err = %v, want ErrNotFound)", err)
	}
	// Clear the stale column and restore a real default for the paths below.
	if _, err := pool.Exec(ctx,
		`UPDATE coach.api_key_config SET is_default = false WHERE account_id = $1`, acct); err != nil {
		t.Fatalf("clear is_default: %v", err)
	}
	if _, err := st.SetDefault(ctx, acct, store.ProviderAnthropic, store.FeatureCoach, ""); err != nil {
		t.Fatalf("set default: %v", err)
	}

	// --- the interview feature: explicit only ---

	// Never auto-assigned: two keys in, and still nothing.
	if _, err := st.GetDefaultKey(ctx, acct, store.FeatureInterview); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("interview default auto-assigned (err = %v, want ErrNotFound)", err)
	}
	// Set explicitly, with its OWN model — which may differ from the key's default_model.
	if _, err := st.SetDefault(ctx, acct, store.ProviderOpenAI, store.FeatureInterview, "gpt-brain"); err != nil {
		t.Fatalf("set interview default: %v", err)
	}
	if p, m := keyDefault(store.FeatureInterview); p != store.ProviderOpenAI || m != "gpt-brain" {
		t.Fatalf("interview default %s/%s, want openai/gpt-brain", p, m)
	}
	iv, err := st.GetDefaultKey(ctx, acct, store.FeatureInterview)
	if err != nil {
		t.Fatalf("get interview default: %v", err)
	}
	if iv.FeatureModel != "gpt-brain" {
		t.Fatalf("interview FeatureModel = %q, want gpt-brain", iv.FeatureModel)
	}
	if iv.IsDefault {
		t.Fatal("the interview default must not report IsDefault (that flag means the COACH default)")
	}
	// A coach-side model switch on the SAME key leaves the interview brain alone.
	if _, err := st.UpdateKeyMeta(ctx, acct, store.ProviderOpenAI, "gpt-d", "renamed"); err != nil {
		t.Fatalf("update meta: %v", err)
	}
	if _, m := keyDefault(store.FeatureInterview); m != "gpt-brain" {
		t.Fatalf("coach model switch clobbered the interview brain: %s", m)
	}

	// --- delete: cascade, promote coach only ---

	// Deleting the coach default's key cascades its row and promotes the survivor.
	if err := st.DeleteKey(ctx, acct, store.ProviderAnthropic); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if p, m := keyDefault(store.FeatureCoach); p != store.ProviderOpenAI || m != "gpt-d" {
		t.Fatalf("after delete: key_default %s/%s, want openai/gpt-d", p, m)
	}
	// Deleting the interview default's key leaves interview UNSET rather than moving it
	// to a model the learner never chose (which might not even be interview-capable).
	if err := st.DeleteKey(ctx, acct, store.ProviderOpenAI); err != nil {
		t.Fatalf("delete last: %v", err)
	}
	if p, _ := keyDefault(store.FeatureInterview); p != "" {
		t.Fatalf("interview default survived its key: %s", p)
	}
	if p, _ := keyDefault(store.FeatureCoach); p != "" || coachDefault() != "" {
		t.Fatalf("no keys but a coach default remains: %s", p)
	}

	// The feature CHECK holds.
	if _, err := pool.Exec(ctx,
		`INSERT INTO coach.key_default (account_id, feature, key_id) VALUES ($1, 'arena', gen_random_uuid())`, acct); err == nil {
		t.Fatal("key_default accepted feature 'arena'")
	}
}
