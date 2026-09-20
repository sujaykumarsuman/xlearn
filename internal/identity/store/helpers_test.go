package store_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func newTestID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// containsAccount reports whether the JSON outbox payload references accountID.
func containsAccount(payload []byte, accountID string) bool {
	return bytes.Contains(payload, []byte(accountID))
}
