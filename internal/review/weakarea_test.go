package review

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

func TestWeekWindowTimezone(t *testing.T) {
	// 02:00 UTC on Monday 2026-09-21. In UTC the week is 2026-09-21; but in
	// America/Los_Angeles (UTC-7) it is still Sunday 2026-09-20 19:00, whose week
	// started Monday 2026-09-14 — proving the boundary must be tz-aware (R-MJ3).
	now := time.Date(2026, 9, 21, 2, 0, 0, 0, time.UTC)

	cases := []struct {
		tz     string
		weekOf string // YYYY-MM-DD Monday
	}{
		{"UTC", "2026-09-21"},
		{"America/Los_Angeles", "2026-09-14"},
		{"", "2026-09-21"},          // empty → UTC fallback
		{"Not/AZone", "2026-09-21"}, // unknown → UTC fallback
	}
	for _, c := range cases {
		weekOf, start, end := weekWindow(now, c.tz)
		if got := weekOf.Format("2006-01-02"); got != c.weekOf {
			t.Errorf("weekWindow(%q).weekOf = %s, want %s", c.tz, got, c.weekOf)
		}
		if !start.Before(end) || end.Sub(start) != 7*24*time.Hour {
			t.Errorf("weekWindow(%q): window is not a 7-day span [%v,%v)", c.tz, start, end)
		}
		if !start.Before(now) || !now.Before(end) {
			// `now` must fall inside its own week window.
			t.Errorf("weekWindow(%q): now %v not inside [%v,%v)", c.tz, now, start, end)
		}
	}
}

func TestWeekWindowKiribati(t *testing.T) {
	// Sunday 2026-09-20 12:00 UTC is already Monday 2026-09-21 02:00 in Kiribati
	// (UTC+14) — so the account's week is 2026-09-21, not UTC's 2026-09-14.
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	weekOf, _, _ := weekWindow(now, "Pacific/Kiritimati")
	if got := weekOf.Format("2006-01-02"); got != "2026-09-21" {
		t.Fatalf("Kiribati weekOf = %s, want 2026-09-21", got)
	}
	weekOfUTC, _, _ := weekWindow(now, "UTC")
	if got := weekOfUTC.Format("2006-01-02"); got != "2026-09-14" {
		t.Fatalf("UTC weekOf = %s, want 2026-09-14", got)
	}
}

func TestTopCategory(t *testing.T) {
	if got := topCategory(map[string]int{}); got != "" {
		t.Errorf("empty counts → %q, want \"\"", got)
	}
	if got := topCategory(map[string]int{"communication": 1, "off_by_one": 3}); got != "off_by_one" {
		t.Errorf("max → %q, want off_by_one", got)
	}
	// Tie breaks by canonical order (off_by_one precedes communication).
	if got := topCategory(map[string]int{"communication": 2, "off_by_one": 2}); got != "off_by_one" {
		t.Errorf("tie → %q, want off_by_one (canonical order)", got)
	}
}

// --- Recompute with fakes ---

type fakeWeakStore struct {
	scopes []store.MistakeScope
	counts map[string]map[string]int // "account|course" → category counts
	saved  []savedSnapshot
}

type savedSnapshot struct {
	accountID   string
	pathSlug    string
	weekOf      time.Time
	topCategory string
	counts      map[string]int
}

func (f *fakeWeakStore) MistakeScopes(context.Context) ([]store.MistakeScope, error) {
	return f.scopes, nil
}
func (f *fakeWeakStore) CountOpenMistakesByCategory(_ context.Context, accountID, pathSlug string, _, _ time.Time) (map[string]int, error) {
	return f.counts[accountID+"|"+pathSlug], nil
}
func (f *fakeWeakStore) SaveWeakAreaSnapshot(_ context.Context, accountID, pathSlug string, weekOf time.Time, top string, counts map[string]int) error {
	f.saved = append(f.saved, savedSnapshot{accountID, pathSlug, weekOf, top, counts})
	return nil
}

type fakeResolver struct {
	tz    map[string]string
	calls map[string]int
}

func (f fakeResolver) ResolveAccount(_ context.Context, accountID string) (string, []byte, []byte, error) {
	if f.calls != nil {
		f.calls[accountID]++
	}
	return f.tz[accountID], nil, nil, nil
}

// m1-03: the recompute runs per (account, course) — one snapshot per pair, each over
// that course's entries — and resolves an account's timezone once.
func TestWeakAreaRecompute(t *testing.T) {
	st := &fakeWeakStore{
		scopes: []store.MistakeScope{
			{AccountID: "a", PathSlug: "dsa"},
			{AccountID: "a", PathSlug: "zz-fixture"},
			{AccountID: "b", PathSlug: "dsa"},
		},
		counts: map[string]map[string]int{
			"a|dsa":        {"off_by_one": 3, "communication": 1},
			"a|zz-fixture": {"communication": 2},
			"b|dsa":        {}, // no categorised open entries → top ""
		},
	}
	resolver := fakeResolver{tz: map[string]string{"a": "UTC", "b": "America/Los_Angeles"}, calls: map[string]int{}}
	c := &weakAreaComputer{
		store:    st,
		accounts: resolver,
		log:      slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}
	n, err := c.Recompute(context.Background())
	if err != nil {
		t.Fatalf("recompute: %v", err)
	}
	if n != 3 || len(st.saved) != 3 {
		t.Fatalf("recomputed %d, saved %d snapshots, want 3 (one per account and course)", n, len(st.saved))
	}
	byScope := map[string]savedSnapshot{}
	for _, s := range st.saved {
		byScope[s.accountID+"|"+s.pathSlug] = s
	}
	for scope, want := range map[string]string{"a|dsa": "off_by_one", "a|zz-fixture": "communication", "b|dsa": ""} {
		if got := byScope[scope].topCategory; got != want {
			t.Errorf("%s top = %q, want %q", scope, got, want)
		}
	}
	if resolver.calls["a"] != 1 || resolver.calls["b"] != 1 {
		t.Errorf("timezone resolves = %v, want one per account", resolver.calls)
	}
}
