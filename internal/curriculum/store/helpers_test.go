package store_test

import (
	"io"
	"log/slog"
)

// testLogger discards output; the integration test only cares about return values.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
