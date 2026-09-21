// Package notifications is the in-app reminder worker (v1). It is a cleanly separable
// package inside the review binary (ADR-0003/0016): its own consumer on
// xlearn.review.revision_due, its own store + account seams, and no dependency on the
// rest of review — so a later split into a standalone notifications service (once a
// second channel like email/push justifies it) is a lift-and-shift, not a rewrite.
package notifications

import (
	"encoding/json"
	"strings"
	"time"
)

// ReminderKind is the v1 in-app reminder kind (one channel).
const ReminderKind = "revision_due"

// Defaults for an account that has not configured its budget/reminders yet (empty
// {} blobs). The daily nudge defaults ON at 20:00 local and alerts default ON, matching
// the Settings artboard's initial state — so "disabled prefs" is an explicit opt-out.
const (
	defaultReminderMin = 20 * 60 // 20:00 local
	defaultWeekdayMins = 90
)

// reminderPrefs is the reminders_json shape (S10). Pointer bools distinguish an
// explicit false from an absent field, so an empty {} defaults to ON.
type reminderPrefs struct {
	DailyOn   *bool  `json:"daily_reminder_on"`
	DailyTime string `json:"daily_reminder_time"`
	AlertsOn  *bool  `json:"revision_due_alerts_on"`
}

// studyBudget is the study_budget_json shape (S10). weekend_band is a coarse hours band.
type studyBudget struct {
	WeekdayMinutes int    `json:"weekday_minutes"`
	WeekendBand    string `json:"weekend_band"`
}

// computeReminder decides whether a due-review reminder is surfaced, and when, in the
// account's LOCAL time. Two gated behaviours (S10):
//
//   - Within today's study window (before daily_reminder_time + the day's budget has
//     elapsed): the DAILY reminder — surfaced at daily_reminder_time local, gated by
//     daily_reminder_on. If the daily time has already passed but the budget window is
//     still open, it surfaces immediately (a late in-day nudge).
//   - Past the day's budget window (reviews piled up beyond the daily budget): a
//     REVISION-DUE ALERT — surfaced immediately, gated by revision_due_alerts_on.
//
// A false `write` means the reminder is suppressed by the user's prefs. All returned
// instants are absolute (a UTC comparison is exact regardless of `loc`).
func computeReminder(now time.Time, tzName string, budgetJSON, remindersJSON []byte) (write bool, dueAt time.Time) {
	loc, err := time.LoadLocation(tzName)
	if err != nil || loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)

	dailyOn, alertsOn, dailyMin := parsePrefs(remindersJSON)
	budgetMins := todaysBudgetMinutes(local, budgetJSON)

	windowStart := atLocalMinutes(local, dailyMin)
	windowEnd := windowStart.Add(time.Duration(budgetMins) * time.Minute)

	if local.Before(windowEnd) {
		// The review still fits in today's budget → the daily reminder channel.
		if !dailyOn {
			return false, time.Time{}
		}
		if local.Before(windowStart) {
			return true, windowStart // the nudge fires at the daily time (still ahead)
		}
		return true, now // daily time passed but budget window still open → surface now
	}
	// Reviews have piled up past the day's budget window → the revision-due alert channel.
	if !alertsOn {
		return false, time.Time{}
	}
	return true, now
}

// parsePrefs resolves reminder prefs, defaulting a missing/malformed blob or field to
// the ON defaults (daily at 20:00, alerts on).
func parsePrefs(remindersJSON []byte) (dailyOn, alertsOn bool, dailyMin int) {
	dailyOn, alertsOn, dailyMin = true, true, defaultReminderMin
	if len(remindersJSON) == 0 {
		return
	}
	var p reminderPrefs
	if err := json.Unmarshal(remindersJSON, &p); err != nil {
		return
	}
	if p.DailyOn != nil {
		dailyOn = *p.DailyOn
	}
	if p.AlertsOn != nil {
		alertsOn = *p.AlertsOn
	}
	if m, ok := parseHHMM(strings.TrimSpace(p.DailyTime)); ok {
		dailyMin = m
	}
	return dailyOn, alertsOn, dailyMin
}

// todaysBudgetMinutes returns the study-window length for the account's local calendar
// day: the weekend band on Sat/Sun, else weekday minutes. An empty/malformed budget
// falls back to the weekday default so the window is consistent all week.
func todaysBudgetMinutes(local time.Time, budgetJSON []byte) int {
	weekday := defaultWeekdayMins
	band := ""
	if len(budgetJSON) > 0 {
		var b studyBudget
		if err := json.Unmarshal(budgetJSON, &b); err == nil {
			if b.WeekdayMinutes > 0 {
				weekday = b.WeekdayMinutes
			}
			band = b.WeekendBand
		}
	}
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
		if m := weekendBandMinutes(band); m > 0 {
			return m
		}
	}
	return weekday
}

// weekendBandMinutes maps the Settings weekend band ("2"/"3-4"/"5" hours) to a
// study-window length in minutes ("3-4" → its 3.5h midpoint). 0 for an unknown band.
// These bound the "past the budget window" alert, not exact study time.
func weekendBandMinutes(band string) int {
	switch band {
	case "2":
		return 120
	case "3-4":
		return 210
	case "5":
		return 300
	default:
		return 0
	}
}

// parseHHMM parses "HH:MM" to minutes-since-midnight (0..1439).
func parseHHMM(s string) (int, bool) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

// atLocalMinutes returns the instant `minutes` past midnight on ref's local calendar
// day, in ref's location.
func atLocalMinutes(ref time.Time, minutes int) time.Time {
	return time.Date(ref.Year(), ref.Month(), ref.Day(), minutes/60, minutes%60, 0, 0, ref.Location())
}
