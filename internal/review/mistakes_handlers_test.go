package review

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

func TestMistakesRequireJWT(t *testing.T) {
	svc := testService(&fakeStore{})
	for _, path := range []string{"/mistakes", "/weak-area/current", "/reminders"} {
		rec, body := do(t, svc.Handler(), http.MethodGet, path, "", "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("GET %s status = %d, want 401", path, rec.Code)
		}
		if errObj, _ := body["error"].(map[string]any); errObj["code"] != "unauthenticated" {
			t.Fatalf("GET %s body = %v, want unauthenticated", path, body)
		}
	}
}

func TestListMistakes(t *testing.T) {
	st := &fakeStore{
		listMistakes: func(_ context.Context, accountID, status string) ([]store.Mistake, error) {
			if accountID != "acct-1" {
				t.Fatalf("account = %q, want acct-1", accountID)
			}
			if status != "" {
				t.Fatalf("handler must fetch all (status=%q) and filter in-memory", status)
			}
			return []store.Mistake{
				{ID: "m1", ProblemID: "18", Pattern: "Sliding window", Category: "off_by_one", Status: "open", RevisitCount: 1, RevisitDate: time.Now().Add(48 * time.Hour)},
				{ID: "m2", ProblemID: "04", Pattern: "Canonical key", Category: "complexity_misjudged", Status: "closed", RevisitCount: 2},
			}, nil
		},
	}
	rec, body := do(t, testService(st).Handler(), http.MethodGet, "/mistakes", "tok", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if oc, _ := body["openCount"].(float64); oc != 1 {
		t.Errorf("openCount = %v, want 1", body["openCount"])
	}
	if cc, _ := body["closedCount"].(float64); cc != 1 {
		t.Errorf("closedCount = %v, want 1", body["closedCount"])
	}
	if ct, _ := body["closeThreshold"].(float64); ct != 2 {
		t.Errorf("closeThreshold = %v, want 2", body["closeThreshold"])
	}
	items, _ := body["mistakes"].([]any)
	if len(items) != 2 {
		t.Fatalf("mistakes = %d, want 2", len(items))
	}
}

func TestListMistakesFiltersByStatus(t *testing.T) {
	st := &fakeStore{
		listMistakes: func(context.Context, string, string) ([]store.Mistake, error) {
			return []store.Mistake{
				{ID: "m1", Status: "open"},
				{ID: "m2", Status: "closed"},
			}, nil
		},
	}
	rec, body := do(t, testService(st).Handler(), http.MethodGet, "/mistakes?status=open", "tok", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	items, _ := body["mistakes"].([]any)
	if len(items) != 1 {
		t.Fatalf("filtered mistakes = %d, want 1 (open only)", len(items))
	}
	// Counts stay over the full journal.
	if oc, _ := body["openCount"].(float64); oc != 1 {
		t.Errorf("openCount = %v, want 1", body["openCount"])
	}
	if cc, _ := body["closedCount"].(float64); cc != 1 {
		t.Errorf("closedCount = %v, want 1", body["closedCount"])
	}
}

func TestListMistakesRejectsBadStatus(t *testing.T) {
	rec, _ := do(t, testService(&fakeStore{}).Handler(), http.MethodGet, "/mistakes?status=weird", "tok", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateMistake(t *testing.T) {
	st := &fakeStore{
		createMistake: func(_ context.Context, accountID string, in store.MistakeInput) (store.Mistake, error) {
			if accountID != "acct-1" || in.ProblemID != "16" || in.Category != "off_by_one" {
				t.Fatalf("create input = %+v (acct %s)", in, accountID)
			}
			return store.Mistake{ID: "new", ProblemID: in.ProblemID, Category: in.Category, Status: "open"}, nil
		},
	}
	rec, body := do(t, testService(st).Handler(), http.MethodPost, "/mistakes", "tok",
		`{"problemId":"16","category":"off_by_one","rootCause":"inclusive bounds"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if body["id"] != "new" {
		t.Errorf("body = %v, want created entry", body)
	}
}

func TestCreateMistakeRequiresProblem(t *testing.T) {
	rec, _ := do(t, testService(&fakeStore{}).Handler(), http.MethodPost, "/mistakes", "tok", `{"category":"off_by_one"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestCreateMistakeConflict(t *testing.T) {
	st := &fakeStore{
		createMistake: func(context.Context, string, store.MistakeInput) (store.Mistake, error) {
			return store.Mistake{}, store.ErrConflict
		},
	}
	rec, body := do(t, testService(st).Handler(), http.MethodPost, "/mistakes", "tok", `{"problemId":"16"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if errObj, _ := body["error"].(map[string]any); errObj["code"] != "conflict" {
		t.Errorf("body = %v, want conflict", body)
	}
}

func TestCreateMistakeInvalidCategory(t *testing.T) {
	st := &fakeStore{
		createMistake: func(context.Context, string, store.MistakeInput) (store.Mistake, error) {
			return store.Mistake{}, store.ErrInvalidCategory
		},
	}
	rec, _ := do(t, testService(st).Handler(), http.MethodPost, "/mistakes", "tok", `{"problemId":"16","category":"nope"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestPatchMistake(t *testing.T) {
	var gotPatch store.MistakePatch
	st := &fakeStore{
		updateMistake: func(_ context.Context, accountID, id string, in store.MistakePatch) (store.Mistake, error) {
			if accountID != "acct-1" || id != "m1" {
				t.Fatalf("patch(%q,%q)", accountID, id)
			}
			gotPatch = in
			return store.Mistake{ID: id, Category: "off_by_one", Status: "open"}, nil
		},
	}
	rec, _ := do(t, testService(st).Handler(), http.MethodPatch, "/mistakes/m1", "tok",
		`{"category":"off_by_one","rootCause":"inclusive bounds need +1"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if gotPatch.Category == nil || *gotPatch.Category != "off_by_one" {
		t.Errorf("patch category = %v, want off_by_one", gotPatch.Category)
	}
	if gotPatch.RootCause == nil || *gotPatch.RootCause != "inclusive bounds need +1" {
		t.Errorf("patch rootCause = %v", gotPatch.RootCause)
	}
	// Untouched fields stay nil (partial patch).
	if gotPatch.Insight != nil || gotPatch.Status != nil {
		t.Errorf("unset fields should be nil: %+v", gotPatch)
	}
}

func TestPatchMistakeNotFound(t *testing.T) {
	st := &fakeStore{
		updateMistake: func(context.Context, string, string, store.MistakePatch) (store.Mistake, error) {
			return store.Mistake{}, store.ErrNotFound
		},
	}
	rec, _ := do(t, testService(st).Handler(), http.MethodPatch, "/mistakes/missing", "tok", `{"status":"closed"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestWeakAreaHandler(t *testing.T) {
	st := &fakeStore{
		weakAreaCurrent: func(_ context.Context, accountID string) (store.WeakArea, bool, error) {
			return store.WeakArea{
				WeekOf:      time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC),
				TopCategory: "off_by_one",
				TopCount:    3,
				Counts:      map[string]int{"off_by_one": 3, "communication": 1},
				Entries:     []store.Mistake{{ID: "m1", ProblemID: "18", Category: "off_by_one", Status: "open"}},
			}, true, nil
		},
	}
	rec, body := do(t, testService(st).Handler(), http.MethodGet, "/weak-area/current", "tok", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body["topCategory"] != "off_by_one" {
		t.Errorf("topCategory = %v, want off_by_one", body["topCategory"])
	}
	if tc, _ := body["topCount"].(float64); tc != 3 {
		t.Errorf("topCount = %v, want 3", body["topCount"])
	}
	if entries, _ := body["entries"].([]any); len(entries) != 1 {
		t.Errorf("entries = %v, want 1", body["entries"])
	}
}

func TestWeakAreaHandlerEmpty(t *testing.T) {
	st := &fakeStore{
		weakAreaCurrent: func(context.Context, string) (store.WeakArea, bool, error) {
			return store.WeakArea{}, false, nil
		},
	}
	rec, body := do(t, testService(st).Handler(), http.MethodGet, "/weak-area/current", "tok", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body["topCategory"] != "" {
		t.Errorf("topCategory = %v, want empty (no snapshot)", body["topCategory"])
	}
}

func TestRemindersHandler(t *testing.T) {
	st := &fakeStore{
		listReminders: func(_ context.Context, accountID string, _ int) ([]store.Reminder, error) {
			if accountID != "acct-1" {
				t.Fatalf("account = %q", accountID)
			}
			return []store.Reminder{{ID: "r1", Kind: "revision_due", DueAt: time.Now()}}, nil
		},
	}
	rec, body := do(t, testService(st).Handler(), http.MethodGet, "/reminders", "tok", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	rems, _ := body["reminders"].([]any)
	if len(rems) != 1 {
		t.Fatalf("reminders = %d, want 1", len(rems))
	}
}
