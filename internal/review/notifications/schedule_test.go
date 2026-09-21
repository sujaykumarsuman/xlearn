package notifications

import (
	"testing"
	"time"
)

func TestComputeReminderTimeNoBudget(t *testing.T) {
	now := time.Date(2026, 9, 21, 15, 0, 0, 0, time.UTC)
	for _, budget := range [][]byte{nil, []byte(""), []byte("{}"), []byte(`{"windows":[]}`), []byte("not json")} {
		if got := computeReminderTime(now, "UTC", budget); !got.Equal(now) {
			t.Errorf("budget %q → %v, want immediate (%v)", budget, got, now)
		}
	}
}

func TestComputeReminderTimeWindow(t *testing.T) {
	budget := []byte(`{"windows":[{"start":"18:00","end":"21:00"}]}`)

	// Before the window → today's window start.
	before := time.Date(2026, 9, 21, 17, 0, 0, 0, time.UTC)
	got := computeReminderTime(before, "UTC", budget)
	if got.UTC().Hour() != 18 || got.UTC().Day() != 21 {
		t.Errorf("before window → %v, want 2026-09-21 18:00Z", got.UTC())
	}

	// Inside the window → immediate.
	inside := time.Date(2026, 9, 21, 19, 0, 0, 0, time.UTC)
	if g := computeReminderTime(inside, "UTC", budget); !g.Equal(inside) {
		t.Errorf("inside window → %v, want immediate (%v)", g, inside)
	}

	// After the window → tomorrow's window start.
	after := time.Date(2026, 9, 21, 22, 0, 0, 0, time.UTC)
	g := computeReminderTime(after, "UTC", budget)
	if g.UTC().Hour() != 18 || g.UTC().Day() != 22 {
		t.Errorf("after window → %v, want 2026-09-22 18:00Z", g.UTC())
	}
}

func TestComputeReminderTimeWindowTimezone(t *testing.T) {
	// A window in the account's local time, not UTC. 10:00 UTC is 03:00 in LA (before
	// an 18:00 local window), so the reminder is LA-18:00 = 2026-09-22 01:00 UTC.
	budget := []byte(`{"windows":[{"start":"18:00","end":"21:00"}]}`)
	now := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC)
	got := computeReminderTime(now, "America/Los_Angeles", budget)
	loc, _ := time.LoadLocation("America/Los_Angeles")
	if got.In(loc).Hour() != 18 {
		t.Fatalf("windowed reminder local hour = %d, want 18 (LA)", got.In(loc).Hour())
	}
}
