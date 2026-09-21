package notifications

import (
	"testing"
	"time"
)

// A Monday 15:00 UTC anchor keeps the "weekday" budget window in play.
var scheduleNow = time.Date(2026, 9, 21, 15, 0, 0, 0, time.UTC)

func TestComputeReminderDefaults(t *testing.T) {
	// Empty budget + empty reminders → defaults ON, daily at 20:00. 15:00 is before the
	// window → the daily nudge fires at today's 20:00 (UTC here).
	for _, budget := range [][]byte{nil, []byte(""), []byte("{}"), []byte("not json")} {
		write, at := computeReminder(scheduleNow, "UTC", budget, nil)
		if !write {
			t.Fatalf("budget %q: want write", budget)
		}
		if at.UTC().Hour() != 20 || at.UTC().Day() != 21 {
			t.Fatalf("budget %q → %v, want 2026-09-21 20:00Z", budget, at.UTC())
		}
	}
}

func TestComputeReminderDailyOff(t *testing.T) {
	// Daily off, alerts on, but we are still within today's budget window → nothing to
	// alert on yet → suppressed.
	prefs := []byte(`{"daily_reminder_on":false,"daily_reminder_time":"20:00","revision_due_alerts_on":true}`)
	if write, _ := computeReminder(scheduleNow, "UTC", nil, prefs); write {
		t.Fatalf("daily off within window → want suppressed")
	}
}

func TestComputeReminderAllOffSuppressed(t *testing.T) {
	prefs := []byte(`{"daily_reminder_on":false,"daily_reminder_time":"20:00","revision_due_alerts_on":false}`)
	// Even past the budget window, both channels off → suppressed.
	past := time.Date(2026, 9, 21, 23, 30, 0, 0, time.UTC)
	if write, _ := computeReminder(past, "UTC", nil, prefs); write {
		t.Fatalf("all prefs off → want suppressed")
	}
}

func TestComputeReminderPastBudgetAlerts(t *testing.T) {
	// 23:30 is past the 20:00 + 90min weekday window (ends 21:30) → a revision-due alert
	// surfaces immediately (now), gated by revision_due_alerts_on.
	prefs := []byte(`{"daily_reminder_on":true,"daily_reminder_time":"20:00","revision_due_alerts_on":true}`)
	past := time.Date(2026, 9, 21, 23, 30, 0, 0, time.UTC)
	write, at := computeReminder(past, "UTC", nil, prefs)
	if !write || !at.Equal(past) {
		t.Fatalf("past-window alert → write=%v at=%v, want immediate (%v)", write, at, past)
	}

	// Same instant, alerts off → suppressed (the daily nudge for today already passed).
	prefsNoAlert := []byte(`{"daily_reminder_on":true,"daily_reminder_time":"20:00","revision_due_alerts_on":false}`)
	if w, _ := computeReminder(past, "UTC", nil, prefsNoAlert); w {
		t.Fatalf("past-window with alerts off → want suppressed")
	}
}

func TestComputeReminderWithinWindowAfterDailyTime(t *testing.T) {
	// 20:30 is after the 20:00 daily time but still within the 90-min budget window
	// (ends 21:30) → surface now (a late in-day nudge).
	prefs := []byte(`{"daily_reminder_on":true,"daily_reminder_time":"20:00","revision_due_alerts_on":true}`)
	at2030 := time.Date(2026, 9, 21, 20, 30, 0, 0, time.UTC)
	write, at := computeReminder(at2030, "UTC", nil, prefs)
	if !write || !at.Equal(at2030) {
		t.Fatalf("within-window after daily time → write=%v at=%v, want now", write, at)
	}
}

func TestComputeReminderTimezone(t *testing.T) {
	// The daily time is the account's LOCAL time. At 10:00 UTC (03:00 in LA), a 20:00
	// LA daily reminder fires at LA-20:00 = 2026-09-21 20:00 local.
	prefs := []byte(`{"daily_reminder_on":true,"daily_reminder_time":"20:00","revision_due_alerts_on":true}`)
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	write, at := computeReminder(now, "America/Los_Angeles", nil, prefs)
	if !write {
		t.Fatalf("want write")
	}
	loc, _ := time.LoadLocation("America/Los_Angeles")
	if at.In(loc).Hour() != 20 || at.In(loc).Day() != 21 {
		t.Fatalf("local reminder = %v, want 2026-09-21 20:00 LA", at.In(loc))
	}
}

func TestComputeReminderWeekendBandWindow(t *testing.T) {
	// Sunday 2026-09-20, weekend band "5" → a 5h (300min) window from 20:00 → ends
	// Monday 01:00. At 23:30 Sunday we are still WITHIN the weekend window → the daily
	// channel surfaces now (not a past-budget alert).
	budget := []byte(`{"weekday_minutes":90,"weekend_band":"5"}`)
	prefs := []byte(`{"daily_reminder_on":true,"daily_reminder_time":"20:00","revision_due_alerts_on":false}`)
	sun := time.Date(2026, 9, 20, 23, 30, 0, 0, time.UTC)
	write, at := computeReminder(sun, "UTC", budget, prefs)
	if !write || !at.Equal(sun) {
		t.Fatalf("weekend within window → write=%v at=%v, want now (alerts off would suppress a past-window review)", write, at)
	}
}
