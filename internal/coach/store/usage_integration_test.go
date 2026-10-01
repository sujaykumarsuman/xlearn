package store_test

import (
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
)

// TestUsageMonthOnAnEmptyAccount pins the ZERO-ROW case of the month-to-date summary
// (m1-10 task 4).
//
// It exists because this exact query shipped broken until the compose rehearsal ran it:
// over zero rows — a learner who has connected a key but not chatted yet, i.e. EVERY new
// account — `sum()` and `bool_or()` both return NULL, while sqlc types those columns as
// non-null. `GET /keys` therefore failed with "cannot scan NULL into *bool" and the whole
// Settings key panel 500'd. Every handler unit test runs against the in-memory fake and
// never executes this SQL, which is precisely why it got through.
//
// The lesson generalises: an aggregate the fake store computes in Go needs a real-database
// test for its empty case, because "no rows" is a shape Go's zero values hide.
func TestUsageMonthOnAnEmptyAccount(t *testing.T) {
	e := newRewrapEnv(t)
	acct := newTestUUID()

	// No keys, no threads, no messages at all.
	u, err := e.st.UsageMonth(e.ctx, acct)
	if err != nil {
		t.Fatalf("UsageMonth on an empty account: %v", err)
	}
	if u.Messages != 0 || u.InputTokens != 0 || u.OutputTokens != 0 || u.EstCostMicros != 0 {
		t.Fatalf("usage = %+v, want all zero", u)
	}
	// Nothing unknown, because nothing was spent — not NULL, and not true.
	if u.HasUnknownCost {
		t.Fatal("has_unknown_cost is true for an account with no messages")
	}

	// A connected key but still no chats: the state right after onboarding, which is what
	// actually broke.
	e.putLegacyOnly(acct, store.ProviderAnthropic, "sk-ant-no-chats-yet-9999")
	u, err = e.st.UsageMonth(e.ctx, acct)
	if err != nil {
		t.Fatalf("UsageMonth after connecting a key: %v", err)
	}
	if u.Messages != 0 || u.HasUnknownCost {
		t.Fatalf("usage = %+v, want zeroed with has_unknown_cost=false", u)
	}
}
