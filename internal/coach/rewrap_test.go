package coach

import (
	"context"
	"errors"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// fakeRewrapStore replays a fixed set of rows the way Postgres would: rows that have been
// successfully re-wrapped stop being pending, and rows that were SKIPPED stay pending and
// are offered again on the next batch (ordered by id). That last behaviour is the one
// worth faking — it is what made a single corrupt row get WARNed and counted once per
// batch before Pass learned to remember what it had already given up on.
type fakeRewrapStore struct {
	rows      []store.RewrapRow
	done      map[string]bool // ids with a current AD pair
	batchSize int32
	// calls records the row ids handed to the resealer, in order, across every batch.
	calls   []string
	batches int
	locked  bool
}

func newFakeRewrapStore(rows ...store.RewrapRow) *fakeRewrapStore {
	return &fakeRewrapStore{rows: rows, done: map[string]bool{}, locked: true}
}

func (f *fakeRewrapStore) pending() []store.RewrapRow {
	var out []store.RewrapRow
	for _, r := range f.rows {
		if !f.done[r.ID] {
			out = append(out, r)
		}
	}
	return out
}

func (f *fakeRewrapStore) RewrapBatch(_ context.Context, _ string, batchSize int32, reseal store.Resealer) (store.RewrapStats, error) {
	f.batches++
	f.batchSize = batchSize
	if !f.locked {
		return store.RewrapStats{Locked: false}, nil
	}
	stats := store.RewrapStats{Locked: true}
	batch := f.pending()
	if int32(len(batch)) > batchSize {
		batch = batch[:batchSize]
	}
	for _, r := range batch {
		f.calls = append(f.calls, r.ID)
		if _, _, _, err := reseal(r); err != nil {
			stats.Skipped++
			continue
		}
		f.done[r.ID] = true
		stats.Rewrapped++
	}
	stats.Pending = int64(len(f.pending()))
	return stats, nil
}

func (f *fakeRewrapStore) RewrapPending(context.Context, string) (int64, error) {
	return int64(len(f.pending())), nil
}

// rewrapRow builds a row whose legacy pair really decrypts under cipher (so Reseal
// succeeds) or is garbage (so it skips).
func rewrapRow(t *testing.T, cipher *secrets.Cipher, id string, decryptable bool) store.RewrapRow {
	t.Helper()
	row := store.RewrapRow{ID: id, AccountID: "11111111-1111-4111-8111-111111111111", Provider: store.ProviderAnthropic}
	if decryptable {
		encKey, encDataKey, err := cipher.Seal([]byte("sk-ant-" + id))
		if err != nil {
			t.Fatalf("seal: %v", err)
		}
		row.EncKey, row.EncDataKey = encKey, encDataKey
		return row
	}
	row.EncKey = []byte("not-a-valid-nonce-prefixed-ciphertext")
	row.EncDataKey = []byte("not-a-valid-nonce-prefixed-ciphertext")
	return row
}

// TestPassReportsEachUnfixableRowOnce pins the fix for a real wart the m1-10 compose
// rehearsal surfaced: with one good row and one corrupt row, the pass logged
// `rewrapped=1 skipped=2 pending=1` — the corrupt row counted twice, and a summary that
// contradicts itself (2 skipped but only 1 pending) is worse than useless to whoever is
// reading it on demand (D34).
func TestPassReportsEachUnfixableRowOnce(t *testing.T) {
	cipher := testCipher()
	good := rewrapRow(t, cipher, "aaaaaaaa-0000-4000-8000-000000000001", true)
	bad := rewrapRow(t, cipher, "bbbbbbbb-0000-4000-8000-000000000002", false)
	fake := newFakeRewrapStore(good, bad)

	w := NewRewrapper(fake, cipher, secrets.KeyringOf(DefaultKEKID, cipher), discardLogger())
	pending, err := w.Pass(context.Background())
	if err != nil {
		t.Fatalf("pass: %v", err)
	}

	// The corrupt row stays pending — that is how the owner learns about it.
	if pending != 1 {
		t.Fatalf("pending = %d, want 1 (the corrupt row)", pending)
	}
	// The good row went through despite its neighbour.
	if !fake.done[good.ID] {
		t.Fatal("the good row was not re-wrapped")
	}
	if fake.done[bad.ID] {
		t.Fatal("the corrupt row was marked done")
	}
	// And the pass stopped instead of spinning: it must not keep asking for a batch that
	// holds only rows it has already given up on.
	if fake.batches > 2 {
		t.Fatalf("the pass ran %d batches over 2 rows; it is spinning on the skipped row", fake.batches)
	}
	// The corrupt row was offered at most once per batch, and the pass attempted it only
	// once for real (the second offer short-circuits without re-logging).
	badCalls := 0
	for _, id := range fake.calls {
		if id == bad.ID {
			badCalls++
		}
	}
	if badCalls > 2 {
		t.Fatalf("the corrupt row was handed to the resealer %d times", badCalls)
	}
}

// TestPassStopsWhenEverythingIsCurrent: the steady state must cost one batch, not twenty.
func TestPassStopsWhenEverythingIsCurrent(t *testing.T) {
	cipher := testCipher()
	fake := newFakeRewrapStore(rewrapRow(t, cipher, "aaaaaaaa-0000-4000-8000-000000000001", true))

	w := NewRewrapper(fake, cipher, nil, discardLogger())
	pending, err := w.Pass(context.Background())
	if err != nil {
		t.Fatalf("pass: %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending = %d, want 0", pending)
	}
	if fake.batches != 1 {
		t.Fatalf("ran %d batches, want 1", fake.batches)
	}

	// A second pass over an all-current table does one batch and changes nothing.
	fake.batches = 0
	if pending, err = w.Pass(context.Background()); err != nil || pending != 0 {
		t.Fatalf("second pass = (%d, %v)", pending, err)
	}
	if fake.batches != 1 {
		t.Fatalf("steady state ran %d batches, want 1", fake.batches)
	}
}

// TestPassYieldsToAnotherRunner: when the lock is held the pass ends immediately and
// reports nothing pending, so the caller waits out the idle interval rather than hammering.
func TestPassYieldsToAnotherRunner(t *testing.T) {
	cipher := testCipher()
	fake := newFakeRewrapStore(rewrapRow(t, cipher, "aaaaaaaa-0000-4000-8000-000000000001", false))
	fake.locked = false

	w := NewRewrapper(fake, cipher, nil, discardLogger())
	pending, err := w.Pass(context.Background())
	if err != nil {
		t.Fatalf("pass: %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending = %d, want 0 when another runner holds the lock", pending)
	}
	if fake.batches != 1 {
		t.Fatalf("ran %d batches after being locked out, want 1", fake.batches)
	}
	if len(fake.calls) != 0 {
		t.Fatal("rows were resealed despite the lock being held elsewhere")
	}
}

// TestPassUsesTheBoundedBatchSize keeps the memory-sum promise visible (ADR-0035 §5: the
// goroutine lives inside coach's existing 128 Mi, with no new pod).
func TestPassUsesTheBoundedBatchSize(t *testing.T) {
	cipher := testCipher()
	fake := newFakeRewrapStore(rewrapRow(t, cipher, "aaaaaaaa-0000-4000-8000-000000000001", true))
	w := NewRewrapper(fake, cipher, nil, discardLogger())
	if _, err := w.Pass(context.Background()); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if fake.batchSize != rewrapBatchSize {
		t.Fatalf("batch size = %d, want %d", fake.batchSize, rewrapBatchSize)
	}
	if rewrapBatchSize > 50 {
		t.Fatalf("rewrapBatchSize = %d; the plan caps a batch at 50 rows", rewrapBatchSize)
	}
}

// TestResealReadsTheRightPair covers the decision tree directly, including the arm that
// cannot be produced in Postgres until l-01 drops NOT NULL on the legacy pair.
func TestResealReadsTheRightPair(t *testing.T) {
	cipher := testCipher()
	kr := secrets.KeyringOf(DefaultKEKID, cipher)
	w := NewRewrapper(newFakeRewrapStore(), cipher, kr, discardLogger())

	const account = "11111111-1111-4111-8111-111111111111"
	ad := secrets.CoachKeyAD(account, store.ProviderAnthropic)

	t.Run("legacy-only row reads the legacy pair", func(t *testing.T) {
		encKey, encDataKey, _ := cipher.Seal([]byte("sk-legacy-only"))
		row := store.RewrapRow{
			ID: "r1", AccountID: account, Provider: store.ProviderAnthropic,
			EncKey: encKey, EncDataKey: encDataKey,
		}
		adKey, adData, kekID, err := w.Reseal(row)
		if err != nil {
			t.Fatalf("reseal: %v", err)
		}
		got, err := kr.OpenAD(adKey, adData, kekID, ad)
		if err != nil || string(got) != "sk-legacy-only" {
			t.Fatalf("resealed pair = (%q, %v)", got, err)
		}
	})

	t.Run("stale row reads the LEGACY pair, not the stale AD pair", func(t *testing.T) {
		// The AD pair holds the OLD key; the legacy pair holds the NEW one (what a v1.6.0
		// image leaves behind). Reading the AD pair here would make the staleness
		// permanent and keep billing the old key.
		oldAD, oldADData, oldKEK, _ := kr.SealAD([]byte("sk-OLD"), ad)
		newKey, newData, _ := cipher.Seal([]byte("sk-NEW"))
		row := store.RewrapRow{
			ID: "r2", AccountID: account, Provider: store.ProviderAnthropic,
			EncKey: newKey, EncDataKey: newData,
			EncKeyAD: oldAD, EncDataKeyAD: oldADData, KEKID: oldKEK,
			ADSrcDigest: []byte("a digest of some other legacy pair"),
		}
		if row.ADPairCurrent() {
			t.Fatal("the row should read as stale")
		}
		adKey, adData, kekID, err := w.Reseal(row)
		if err != nil {
			t.Fatalf("reseal: %v", err)
		}
		got, err := kr.OpenAD(adKey, adData, kekID, ad)
		if err != nil || string(got) != "sk-NEW" {
			t.Fatalf("resealed pair = (%q, %v), want sk-NEW", got, err)
		}
	})

	t.Run("a row with no readable pair is skipped", func(t *testing.T) {
		row := store.RewrapRow{ID: "r3", AccountID: account, Provider: store.ProviderAnthropic}
		if _, _, _, err := w.Reseal(row); err == nil {
			t.Fatal("a row with neither pair resealed successfully")
		} else if !errors.Is(err, errNoLegacyPair) {
			t.Fatalf("err = %v, want errNoLegacyPair", err)
		}
	})

	t.Run("a current AD pair under a retired KEK falls back to the legacy pair", func(t *testing.T) {
		// The entry that sealed it is gone. The legacy pair is the only way back in, and
		// taking it is what turns "every learner re-pastes" into "the row is repaired".
		retired := testCipher()
		retiredRing := secrets.KeyringOf("k9", retired)
		adKey, adData, _, _ := retiredRing.SealAD([]byte("sk-under-k9"), ad)
		encKey, encDataKey, _ := cipher.Seal([]byte("sk-under-k9"))

		row := store.RewrapRow{
			ID: "r4", AccountID: account, Provider: store.ProviderAnthropic,
			EncKey: encKey, EncDataKey: encDataKey,
			EncKeyAD: adKey, EncDataKeyAD: adData, KEKID: "k9",
		}
		// Make the AD pair read as CURRENT so Reseal tries it first.
		row.ADSrcDigest = sha256Of(encDataKey)
		if !row.ADPairCurrent() {
			t.Fatal("setup: the row should read as current")
		}
		newAD, newADData, kekID, err := w.Reseal(row)
		if err != nil {
			t.Fatalf("reseal: %v", err)
		}
		if kekID != DefaultKEKID {
			t.Fatalf("resealed under %q, want the active entry", kekID)
		}
		got, err := kr.OpenAD(newAD, newADData, kekID, ad)
		if err != nil || string(got) != "sk-under-k9" {
			t.Fatalf("resealed pair = (%q, %v)", got, err)
		}
	})
}
