package store_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// Integration test against a real Postgres, gated on XLEARN_TEST_DATABASE_URL so CI
// (which has no database) skips it. Run locally, e.g.:
//
//	XLEARN_TEST_DATABASE_URL=postgres://xlearn_coach:pw@localhost:5599/xlearndb?sslmode=disable \
//	  go test ./internal/coach/store/ -run TestStore -count=1
//
// The DSN must be for the xlearn_coach role (search_path coach, owning only schema
// coach) so the test also exercises the least-privilege model.
func TestStoreIntegration(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the coach store integration test")
	}
	ctx := context.Background()

	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Idempotent re-run (advisory lock + goose versioning).
	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate (rerun): %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	st := store.New(pool)

	if err := st.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	cipher := newTestCipher(t)

	t.Run("multi-provider keys, default, and delete-promotes", func(t *testing.T) {
		acct := newTestUUID()
		raw := "sk-openai-integration-secret-cdef"
		encKey, encDataKey, err := cipher.Seal([]byte(raw))
		if err != nil {
			t.Fatalf("seal: %v", err)
		}
		// First key (openai) → becomes the account default.
		stored, err := st.PutKey(ctx, store.KeyConfig{
			AccountID: acct, Provider: store.ProviderOpenAI,
			EncKey: encKey, EncDataKey: encDataKey, Masked: secrets.Mask(raw), DefaultModel: "gpt-5.6-sol",
		})
		if err != nil {
			t.Fatalf("put: %v", err)
		}
		if !stored.Enabled || !stored.IsDefault || stored.Masked != secrets.Mask(raw) {
			t.Fatalf("first key = %+v (want enabled + default)", stored)
		}

		got, err := st.GetKey(ctx, acct, store.ProviderOpenAI)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		dec, err := cipher.Open(got.EncKey, got.EncDataKey)
		if err != nil || string(dec) != raw {
			t.Fatalf("round trip: dec=%q err=%v", dec, err)
		}

		// Second key (anthropic) → connected, NOT default.
		raw2 := "sk-ant-integration-secret-9999"
		e2, d2, _ := cipher.Seal([]byte(raw2))
		ant, err := st.PutKey(ctx, store.KeyConfig{AccountID: acct, Provider: store.ProviderAnthropic, EncKey: e2, EncDataKey: d2, Masked: secrets.Mask(raw2), DefaultModel: "claude-sonnet-5"})
		if err != nil {
			t.Fatalf("second put: %v", err)
		}
		if ant.IsDefault {
			t.Fatal("second key must not be default")
		}
		if keys, _ := st.ListKeys(ctx, acct); len(keys) != 2 {
			t.Fatalf("want 2 keys, got %d", len(keys))
		}
		if def, _ := st.GetDefaultKey(ctx, acct); def.Provider != store.ProviderOpenAI {
			t.Fatalf("default = %q, want openai", def.Provider)
		}

		// Move the default to anthropic.
		moved, err := st.SetDefault(ctx, acct, store.ProviderAnthropic)
		if err != nil || !moved.IsDefault || moved.Provider != store.ProviderAnthropic {
			t.Fatalf("set default: %+v err=%v", moved, err)
		}
		if def, _ := st.GetDefaultKey(ctx, acct); def.Provider != store.ProviderAnthropic {
			t.Fatalf("default did not move: %q", def.Provider)
		}

		// Disable anthropic, confirm per-provider.
		if err := st.SetKeyEnabled(ctx, acct, store.ProviderAnthropic, false); err != nil {
			t.Fatalf("disable: %v", err)
		}
		if g, _ := st.GetKey(ctx, acct, store.ProviderAnthropic); g.Enabled {
			t.Fatal("still enabled after disable")
		}

		// Delete the default (anthropic) → openai is promoted back to default.
		if err := st.DeleteKey(ctx, acct, store.ProviderAnthropic); err != nil {
			t.Fatalf("delete: %v", err)
		}
		if def, _ := st.GetDefaultKey(ctx, acct); def.Provider != store.ProviderOpenAI {
			t.Fatalf("delete-default did not promote openai: %q", def.Provider)
		}

		// Delete the last key → no default remains, and a re-delete is a no-op.
		if err := st.DeleteKey(ctx, acct, store.ProviderOpenAI); err != nil {
			t.Fatalf("delete last: %v", err)
		}
		if _, err := st.GetDefaultKey(ctx, acct); err != store.ErrNotFound {
			t.Fatalf("default after all deleted = %v, want ErrNotFound", err)
		}
		if err := st.DeleteKey(ctx, acct, store.ProviderOpenAI); err != store.ErrNotFound {
			t.Fatalf("delete no-op = %v, want ErrNotFound", err)
		}
	})

	t.Run("cross-account isolation", func(t *testing.T) {
		a, b := newTestUUID(), newTestUUID()
		e, d, _ := cipher.Seal([]byte("sk-openai-abcdef-cdef"))
		if _, err := st.PutKey(ctx, store.KeyConfig{AccountID: a, Provider: store.ProviderOpenAI, EncKey: e, EncDataKey: d, Masked: "sk-...cdef"}); err != nil {
			t.Fatalf("put: %v", err)
		}
		if _, err := st.GetKey(ctx, b, store.ProviderOpenAI); err != store.ErrNotFound {
			t.Fatalf("other account get = %v, want ErrNotFound", err)
		}
		if keys, _ := st.ListKeys(ctx, b); len(keys) != 0 {
			t.Fatalf("other account list = %+v, want empty", keys)
		}
	})

	t.Run("thread get-or-create + message ordering", func(t *testing.T) {
		acct := newTestUUID()
		id1, err := st.EnsureThread(ctx, acct, "problem:16")
		if err != nil {
			t.Fatalf("ensure: %v", err)
		}
		id2, err := st.EnsureThread(ctx, acct, "problem:16")
		if err != nil {
			t.Fatalf("ensure again: %v", err)
		}
		if id1 != id2 {
			t.Fatalf("ensure not idempotent: %s vs %s", id1, id2)
		}
		// A different context is a distinct thread.
		id3, _ := st.EnsureThread(ctx, acct, "concept:sliding-window")
		if id3 == id1 {
			t.Fatal("distinct contexts collided into one thread")
		}

		for _, m := range []struct{ role, content string }{
			{store.RoleUser, "first"}, {store.RoleAssistant, "second"}, {store.RoleUser, "third"},
		} {
			if err := st.AppendMessage(ctx, id1, m.role, m.content); err != nil {
				t.Fatalf("append: %v", err)
			}
		}
		hist, err := st.ThreadHistory(ctx, acct, "problem:16")
		if err != nil {
			t.Fatalf("history: %v", err)
		}
		if len(hist) != 3 || hist[0].Content != "first" || hist[1].Content != "second" || hist[2].Content != "third" {
			t.Fatalf("history order wrong: %+v", hist)
		}
		// An unseen context has empty history (not an error).
		if h, err := st.ThreadHistory(ctx, acct, "never:chatted"); err != nil || len(h) != 0 {
			t.Fatalf("empty history = %+v err=%v", h, err)
		}
	})
}

func newTestCipher(t *testing.T) *secrets.Cipher {
	t.Helper()
	m := make([]byte, secrets.MasterKeySize)
	if _, err := rand.Read(m); err != nil {
		t.Fatalf("rand: %v", err)
	}
	c, err := secrets.NewCipher(m)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	return c
}

func newTestUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf[:])
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
