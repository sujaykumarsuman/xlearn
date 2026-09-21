package store_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
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

// weekdayMinutes parses weekday_minutes out of a study_budget_json blob (Postgres
// jsonb reorders keys + adds whitespace, so tests compare parsed values).
func weekdayMinutes(t *testing.T, budget []byte) int {
	t.Helper()
	var b struct {
		WeekdayMinutes int `json:"weekday_minutes"`
	}
	if err := json.Unmarshal(budget, &b); err != nil {
		t.Fatalf("parse study_budget %q: %v", budget, err)
	}
	return b.WeekdayMinutes
}
