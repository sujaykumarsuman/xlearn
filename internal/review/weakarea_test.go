package review

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
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
	accounts []string
	counts   map[string]map[string]int // accountID → category counts
	saved    []savedSnapshot
}

type savedSnapshot struct {
	accountID   string
	weekOf      time.Time
	topCategory string
	counts      map[string]int
}

func (f *fakeWeakStore) AccountsWithMistakes(context.Context) ([]string, error) {
	return f.accounts, nil
}
func (f *fakeWeakStore) CountOpenMistakesByCategory(_ context.Context, accountID string, _, _ time.Time) (map[string]int, error) {
	return f.counts[accountID], nil
}
func (f *fakeWeakStore) SaveWeakAreaSnapshot(_ context.Context, accountID string, weekOf time.Time, top string, counts map[string]int) error {
	f.saved = append(f.saved, savedSnapshot{accountID, weekOf, top, counts})
	return nil
}

type fakeResolver struct{ tz map[string]string }

func (f fakeResolver) ResolveAccount(_ context.Context, accountID string) (string, []byte, []byte, error) {
	return f.tz[accountID], nil, nil, nil
}

func TestWeakAreaRecompute(t *testing.T) {
	st := &fakeWeakStore{
		accounts: []string{"a", "b"},
		counts: map[string]map[string]int{
			"a": {"off_by_one": 3, "communication": 1},
			"b": {}, // no categorised open entries → top ""
		},
	}
	c := &weakAreaComputer{
		store:    st,
		accounts: fakeResolver{tz: map[string]string{"a": "UTC", "b": "America/Los_Angeles"}},
		log:      slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}
	n, err := c.Recompute(context.Background())
	if err != nil {
		t.Fatalf("recompute: %v", err)
	}
	if n != 2 {
		t.Fatalf("recomputed %d accounts, want 2", n)
	}
	if len(st.saved) != 2 {
		t.Fatalf("saved %d snapshots, want 2", len(st.saved))
	}
	byAcct := map[string]savedSnapshot{}
	for _, s := range st.saved {
		byAcct[s.accountID] = s
	}
	if byAcct["a"].topCategory != "off_by_one" {
		t.Errorf("account a top = %q, want off_by_one", byAcct["a"].topCategory)
	}
	if byAcct["b"].topCategory != "" {
		t.Errorf("account b top = %q, want \"\" (no categorised entries)", byAcct["b"].topCategory)
	}
}
