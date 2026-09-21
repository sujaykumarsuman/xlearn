package assessment

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
)

func TestComputeStreak(t *testing.T) {
	mk := func(dates ...string) []store.HeatmapDay {
		out := make([]store.HeatmapDay, 0, len(dates))
		for _, d := range dates {
			tm, _ := time.Parse("2006-01-02", d)
			out = append(out, store.HeatmapDay{Date: tm, Solves: 1})
		}
		return out
	}
	now, _ := time.Parse("2006-01-02", "2026-09-21")

	tests := []struct {
		name            string
		days            []store.HeatmapDay
		wantCur, wantLg int
	}{
		{"empty", nil, 0, 0},
		{"today active, 3-day run", mk("2026-09-19", "2026-09-20", "2026-09-21"), 3, 3},
		{"today idle but yesterday active", mk("2026-09-19", "2026-09-20"), 2, 2},
		{"broken >1 day ago", mk("2026-09-15", "2026-09-16"), 0, 2},
		{"gap splits longest", mk("2026-09-01", "2026-09-02", "2026-09-03", "2026-09-10", "2026-09-20", "2026-09-21"), 2, 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cur, lg := computeStreak(tc.days, now)
			if cur != tc.wantCur || lg != tc.wantLg {
				t.Fatalf("current/longest = %d/%d, want %d/%d", cur, lg, tc.wantCur, tc.wantLg)
			}
		})
	}
}

func TestRetentionPct(t *testing.T) {
	cases := []struct{ ladders, resets, want int }{
		{0, 0, 100}, {5, 0, 100}, {5, 1, 80}, {4, 2, 50}, {2, 5, 0},
	}
	for _, c := range cases {
		if got := retentionPct(c.ladders, c.resets); got != c.want {
			t.Fatalf("retentionPct(%d,%d) = %d, want %d", c.ladders, c.resets, got, c.want)
		}
	}
}

func TestProgressSummaryHandler(t *testing.T) {
	today := truncDay(time.Now())
	fs := &fakeStore{
		solvedCount: func(context.Context, string) (int, error) { return 28, nil },
		retention:   func(context.Context, string) (int, int, error) { return 5, 1, nil },
		outcomeMix: func(context.Context, string) (map[string]int, error) {
			return map[string]int{"clean": 15, "rough": 7, "assisted": 4, "miss": 2}, nil
		},
		mockStats: func(context.Context, string) (store.MockStats, error) {
			return store.MockStats{Count: 3, Average: 22, Best: 24}, nil
		},
		trend: func(context.Context, string) ([]store.TrendPoint, error) {
			return []store.TrendPoint{{Total35: 20}, {Total35: 22}, {Total35: 24}}, nil
		},
		heatmap: func(context.Context, string, time.Time) ([]store.HeatmapDay, error) {
			return []store.HeatmapDay{
				{Date: today.AddDate(0, 0, -1), Solves: 1},
				{Date: today, Reviews: 2},
			}, nil
		},
	}
	w := do(t, newTestService(fs).Handler(), "GET", "/progress/summary", "", true)
	if w.Code != 200 {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var got summaryJSON
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Solved != 28 || got.Total != 151 {
		t.Fatalf("solved/total = %d/%d", got.Solved, got.Total)
	}
	if got.Retention.Pct != 80 || got.Retention.Resets != 1 {
		t.Fatalf("retention = %+v", got.Retention)
	}
	if got.Streak.Current != 2 {
		t.Fatalf("current streak = %d, want 2", got.Streak.Current)
	}
	if got.Mock.Average != 22 || got.Mock.Best != 24 || got.Mock.Last != 24 || got.Mock.Delta != 2 {
		t.Fatalf("mock = %+v", got.Mock)
	}
	if got.OutcomeMix.Total != 28 || got.OutcomeMix.Clean != 15 {
		t.Fatalf("outcome mix = %+v", got.OutcomeMix)
	}
}

func TestProgressMasteryHandler(t *testing.T) {
	fs := &fakeStore{
		mastery: func(context.Context, string) ([]store.ProblemMastery, error) {
			return []store.ProblemMastery{
				{ProblemID: "16", BestOutcome: "clean", BestRank: 4, CleanSolves: 1, SolveCount: 1},
				{ProblemID: "17", BestOutcome: "rough", BestRank: 3, CleanSolves: 0, SolveCount: 2},
			}, nil
		},
	}
	w := do(t, newTestService(fs).Handler(), "GET", "/progress/mastery", "", true)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var got masteryJSON
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Problems) != 2 {
		t.Fatalf("problems = %d", len(got.Problems))
	}
	if got.Problems[0].Weight != 1.0 || got.Problems[1].Weight != 0.7 {
		t.Fatalf("weights = %v / %v", got.Problems[0].Weight, got.Problems[1].Weight)
	}
}

func TestProgressRoutesRequireAuth(t *testing.T) {
	fs := &fakeStore{}
	h := newTestService(fs).Handler()
	for _, p := range []string{"/progress/summary", "/progress/heatmap", "/progress/mastery"} {
		if w := do(t, h, "GET", p, "", false); w.Code != 401 {
			t.Fatalf("%s unauth status = %d, want 401", p, w.Code)
		}
	}
}
