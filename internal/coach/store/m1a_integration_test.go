package store_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
)

// m1-02 (M1a expand): every writer of is_default dual-writes key_default(feature =
// 'coach') in the same transaction, and every reader prefers key_default, falling back
// to is_default.
func TestM1aKeyDefaultDualWrite(t *testing.T) {
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

	// keyDefault reads the coach default row as (provider, model); "" when none.
	keyDefault := func() (string, string) {
		t.Helper()
		var provider, model string
		err := pool.QueryRow(ctx, `
			SELECT k.provider, d.model FROM coach.key_default d
			JOIN coach.api_key_config k ON k.id = d.key_id
			WHERE d.account_id = $1 AND d.feature = 'coach'`, acct).Scan(&provider, &model)
		if err != nil {
			return "", ""
		}
		return provider, model
	}
	isDefault := func() string {
		t.Helper()
		var p string
		_ = pool.QueryRow(ctx, `SELECT provider FROM coach.api_key_config WHERE account_id = $1 AND is_default`, acct).Scan(&p)
		return p
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
	defaultProvider := func() string {
		t.Helper()
		k, err := st.GetDefaultKey(ctx, acct)
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

	// First key → the default in both places.
	put(store.ProviderOpenAI, "gpt-a")
	if p, m := keyDefault(); p != store.ProviderOpenAI || m != "gpt-a" || isDefault() != store.ProviderOpenAI {
		t.Fatalf("first key: key_default %s/%s, is_default %s", p, m, isDefault())
	}
	// A second key leaves the default alone.
	put(store.ProviderAnthropic, "claude-a")
	if p, _ := keyDefault(); p != store.ProviderOpenAI || defaultProvider() != store.ProviderOpenAI {
		t.Fatalf("second key moved the default to %s", p)
	}
	// Replacing / re-modelling the default key carries its model into key_default.
	put(store.ProviderOpenAI, "gpt-b")
	if _, m := keyDefault(); m != "gpt-b" {
		t.Fatalf("replace: key_default model %s, want gpt-b", m)
	}
	if _, err := st.UpdateKeyMeta(ctx, acct, store.ProviderOpenAI, "gpt-c", "renamed"); err != nil {
		t.Fatalf("update meta: %v", err)
	}
	if _, m := keyDefault(); m != "gpt-c" {
		t.Fatalf("update meta: key_default model %s, want gpt-c", m)
	}
	// Re-modelling a NON-default key never touches key_default.
	if _, err := st.UpdateKeyMeta(ctx, acct, store.ProviderAnthropic, "claude-b", "x"); err != nil {
		t.Fatalf("update meta: %v", err)
	}
	if p, m := keyDefault(); p != store.ProviderOpenAI || m != "gpt-c" {
		t.Fatalf("non-default meta: key_default %s/%s", p, m)
	}
	// Set-default moves both.
	if _, err := st.SetDefault(ctx, acct, store.ProviderAnthropic); err != nil {
		t.Fatalf("set default: %v", err)
	}
	if p, m := keyDefault(); p != store.ProviderAnthropic || m != "claude-b" || isDefault() != store.ProviderAnthropic {
		t.Fatalf("set default: key_default %s/%s, is_default %s", p, m, isDefault())
	}
	if defaultProvider() != store.ProviderAnthropic {
		t.Fatal("reader disagrees after set-default")
	}

	// Reader preference: key_default wins over a stale is_default (an R-b from v1.7.0,
	// which stops writing is_default) …
	if _, err := pool.Exec(ctx, `UPDATE coach.api_key_config SET is_default = (provider = 'openai') WHERE account_id = $1`, acct); err != nil {
		t.Fatalf("stale is_default: %v", err)
	}
	if got := defaultProvider(); got != store.ProviderAnthropic {
		t.Fatalf("key_default not preferred: default %s", got)
	}
	// … and is_default is the fallback when no key_default row exists (a default set by
	// v1.5.2 during an R-b).
	if _, err := pool.Exec(ctx, `DELETE FROM coach.key_default WHERE account_id = $1`, acct); err != nil {
		t.Fatalf("drop key_default: %v", err)
	}
	if got := defaultProvider(); got != store.ProviderOpenAI {
		t.Fatalf("is_default fallback: default %s, want openai", got)
	}
	// Restore agreement for the delete path.
	if _, err := st.SetDefault(ctx, acct, store.ProviderAnthropic); err != nil {
		t.Fatalf("set default: %v", err)
	}

	// Deleting the default cascades its key_default row and promotes the survivor in both.
	if err := st.DeleteKey(ctx, acct, store.ProviderAnthropic); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if p, m := keyDefault(); p != store.ProviderOpenAI || m != "gpt-c" || isDefault() != store.ProviderOpenAI {
		t.Fatalf("after delete: key_default %s/%s, is_default %s", p, m, isDefault())
	}
	// Deleting the last key leaves no default anywhere.
	if err := st.DeleteKey(ctx, acct, store.ProviderOpenAI); err != nil {
		t.Fatalf("delete last: %v", err)
	}
	if p, _ := keyDefault(); p != "" || defaultProvider() != "" {
		t.Fatalf("no keys but a default remains: %s", p)
	}

	// The feature CHECK holds.
	if _, err := pool.Exec(ctx, `INSERT INTO coach.key_default (account_id, feature, key_id) VALUES ($1, 'arena', gen_random_uuid())`, acct); err == nil {
		t.Fatal("key_default accepted feature 'arena'")
	}
}
