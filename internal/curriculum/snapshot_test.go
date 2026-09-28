package curriculum

import (
	"context"
	"io"
	"log/slog"
	"testing"

	seeddata "github.com/sujaykumarsuman/xlearn/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// TestSeedMatchesV1Snapshot seeds the embedded curriculum into a fresh schema and
// compares every seeded row (v1 columns, uuids stripped) with the committed v1 row
// snapshot. The fixture was written by the v1 loader before the one-shot converter ran
// (m1-09 task 1a), so an equal snapshot proves the conversion changed no row. The only
// intended difference is the two rewritten Example-1 bodies (items 3 and 16), which the
// Example-1 commit updates in the fixture itself.
//
//	go test ./internal/curriculum -run TestSeedMatchesV1Snapshot -update   # rewrite (review the diff)
func TestSeedMatchesV1Snapshot(t *testing.T) {
	pool, dsn := testPool(t)
	freshSchema(t, pool, dsn)

	if err := Seed(context.Background(), store.New(pool), seeddata.FS, discardLogger()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	assertSnapshot(t, pool, snapshotFile)
}
