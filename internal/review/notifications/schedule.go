// Package notifications is the in-app reminder worker (v1). It is a cleanly separable
// package inside the review binary (ADR-0003/0016): its own consumer on
// xlearn.review.revision_due, its own store + account seams, and no dependency on the
// rest of review — so a later split into a standalone notifications service (once a
// second channel like email/push justifies it) is a lift-and-shift, not a rewrite.
package notifications

import (
	"encoding/json"
	"time"
)

// ReminderKind is the v1 in-app reminder kind (one channel).
const ReminderKind = "revision_due"

// studyBudget is the provisional study-budget shape (identity account.study_budget_json).
// v1 accounts have an empty budget ({}); the schema is finalised when the onboarding
// budget step lands (S10). `windows` are daily local-time windows the learner is
// available to study — reminders are clamped into the next one.
type studyBudget struct {
	Windows []budgetWindow `json:"windows"`
}

type budgetWindow struct {
	Start string `json:"start"` // "HH:MM" local time
	End   string `json:"end"`   // "HH:MM" local time
}

// computeReminderTime returns when a due-review reminder should surface, given the
// account's timezone and study-budget window. With no budget (v1 default) the reminder
// is immediate (`now`) — an in-app nudge the Dashboard shows right away. With windows,
// it is `now` when the learner is currently inside a window, else the start of the next
// upcoming window (today or a following day). A malformed budget degrades to immediate.
func computeReminderTime(now time.Time, tzName string, budgetJSON []byte) time.Time {
	loc, err := time.LoadLocation(tzName)
	if err != nil || loc == nil {
		loc = time.UTC
	}
	windows := parseWindows(budgetJSON)
	if len(windows) == 0 {
		return now
	}
	local := now.In(loc)
	minsNow := local.Hour()*60 + local.Minute()

	// Inside a window right now → immediate.
	for _, w := range windows {
		if w.startMin <= minsNow && minsNow < w.endMin {
			return now
		}
	}
	// Earliest window start still ahead today.
	best := -1
	for _, w := range windows {
		if w.startMin > minsNow && (best == -1 || w.startMin < best) {
			best = w.startMin
		}
	}
	if best >= 0 {
		return atLocalMinutes(local, best, 0)
	}
	// All of today's windows have passed → the earliest window start tomorrow.
	earliest := windows[0].startMin
	for _, w := range windows[1:] {
		if w.startMin < earliest {
			earliest = w.startMin
		}
	}
	return atLocalMinutes(local.AddDate(0, 0, 1), earliest, 0)
}

// parsedWindow is a validated window in minutes-since-midnight.
type parsedWindow struct{ startMin, endMin int }

// parseWindows extracts valid [start,end) windows from the budget JSON. Invalid or
// empty input yields no windows (→ immediate reminders).
func parseWindows(budgetJSON []byte) []parsedWindow {
	if len(budgetJSON) == 0 {
		return nil
	}
	var b studyBudget
	if err := json.Unmarshal(budgetJSON, &b); err != nil {
		return nil
	}
	out := make([]parsedWindow, 0, len(b.Windows))
	for _, w := range b.Windows {
		s, sok := parseHHMM(w.Start)
		e, eok := parseHHMM(w.End)
		if sok && eok && e > s {
			out = append(out, parsedWindow{startMin: s, endMin: e})
		}
	}
	return out
}

// parseHHMM parses "HH:MM" to minutes-since-midnight (0..1439).
func parseHHMM(s string) (int, bool) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

// atLocalMinutes returns the instant at `minutes`/`secs` past midnight on ref's local
// calendar day, in ref's location.
func atLocalMinutes(ref time.Time, minutes, secs int) time.Time {
	return time.Date(ref.Year(), ref.Month(), ref.Day(), minutes/60, minutes%60, secs, 0, ref.Location())
}
