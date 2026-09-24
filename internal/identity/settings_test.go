package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// seedAccount creates an account+onboarding and a service whose verifier accepts
// "good-token" as that account's owner.
func seedAccount(t *testing.T) (*fakeStore, *Service, store.Account) {
	t.Helper()
	st := newFakeStore()
	acct, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{
		Provider: "github", ProviderUserID: "acct", DisplayName: "Ada", Email: "ada@example.com",
	})
	v := fakeVerifier{token: "good-token", claims: auth.Claims{Subject: acct.ID, Audience: "identity", Roles: []string{"learner"}}}
	return st, newTestService(st, v), acct
}

func patchReq(t *testing.T, path, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer good-token")
	return req
}

func TestPatchAccountUpdatesFields(t *testing.T) {
	st, svc, acct := seedAccount(t)
	handler := svc.Handler()

	body := `{"display_name":"  Sujay Kumar  ","timezone":"Asia/Kolkata",` +
		`"study_budget":{"weekday_minutes":120,"weekend_band":"5"},` +
		`"reminders":{"daily_reminder_on":true,"daily_reminder_time":"20:00","revision_due_alerts_on":false}}`
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, patchReq(t, "/accounts/"+acct.ID, body))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}

	got := st.accounts[acct.ID]
	if got.DisplayName != "Sujay Kumar" {
		t.Fatalf("display_name = %q, want trimmed", got.DisplayName)
	}
	if got.Timezone != "Asia/Kolkata" {
		t.Fatalf("timezone = %q", got.Timezone)
	}
	var budget struct {
		WeekdayMinutes int    `json:"weekday_minutes"`
		WeekendBand    string `json:"weekend_band"`
	}
	if err := json.Unmarshal(got.StudyBudget, &budget); err != nil {
		t.Fatalf("budget json: %v (%s)", err, got.StudyBudget)
	}
	if budget.WeekdayMinutes != 120 || budget.WeekendBand != "5" {
		t.Fatalf("budget = %+v", budget)
	}
	// Response includes the account with study_budget + reminders so the form round-trips.
	if !strings.Contains(rec.Body.String(), `"study_budget"`) || !strings.Contains(rec.Body.String(), `"reminders"`) {
		t.Fatalf("patch response missing budget/reminders: %s", rec.Body.String())
	}
}

func TestPatchAccountPartial(t *testing.T) {
	st, svc, acct := seedAccount(t)
	handler := svc.Handler()

	// Only the display name — timezone must stay unchanged (starts "UTC").
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, patchReq(t, "/accounts/"+acct.ID, `{"display_name":"Renamed"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	if got := st.accounts[acct.ID]; got.DisplayName != "Renamed" || got.Timezone != "UTC" {
		t.Fatalf("partial update changed too much: %+v", got)
	}
}

func TestPatchAccountRejectsInvalid(t *testing.T) {
	_, svc, acct := seedAccount(t)
	handler := svc.Handler()

	cases := map[string]string{
		"empty display name":    `{"display_name":"   "}`,
		"bad timezone":          `{"timezone":"Mars/Phobos"}`,
		"budget out of range":   `{"study_budget":{"weekday_minutes":5,"weekend_band":"2"}}`,
		"bad weekend band":      `{"study_budget":{"weekday_minutes":90,"weekend_band":"9"}}`,
		"bad reminder time":     `{"reminders":{"daily_reminder_on":true,"daily_reminder_time":"25:00","revision_due_alerts_on":true}}`,
		"missing reminder flag": `{"reminders":{"daily_reminder_time":"20:00"}}`,
	}
	for name, body := range cases {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, patchReq(t, "/accounts/"+acct.ID, body))
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s: got %d, want 422 (%s)", name, rec.Code, rec.Body.String())
		}
	}
}

func TestPatchAccountAcceptsMultibyteNameAndEchoBack(t *testing.T) {
	st, svc, acct := seedAccount(t)
	handler := svc.Handler()

	// A 45-rune CJK name is 135 bytes — well within the 120-CHARACTER limit. The body
	// also echoes back the server-owned fields (id/email/created_at), which must be
	// accepted-and-ignored rather than rejected by DisallowUnknownFields.
	name := strings.Repeat("学", 45)
	body := `{"id":"` + acct.ID + `","email":"ada@example.com","created_at":"2026-09-20T00:00:00Z","display_name":"` + name + `","timezone":"UTC"}`
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, patchReq(t, "/accounts/"+acct.ID, body))
	if rec.Code != http.StatusOK {
		t.Fatalf("multibyte name / echo-back: %d %s", rec.Code, rec.Body.String())
	}
	if got := st.accounts[acct.ID].DisplayName; got != name {
		t.Fatalf("display_name = %q, want the 45-rune name", got)
	}
	if st.accounts[acct.ID].Email != "ada@example.com" {
		t.Fatalf("email must remain read-only (unchanged)")
	}
}

func TestPatchAccountWrongSubjectForbidden(t *testing.T) {
	st := newFakeStore()
	acct, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{Provider: "github", ProviderUserID: "acct", DisplayName: "Ada"})
	v := fakeVerifier{token: "good-token", claims: auth.Claims{Subject: "someone-else", Audience: "identity"}}
	handler := newTestService(st, v).Handler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, patchReq(t, "/accounts/"+acct.ID, `{"display_name":"Mallory"}`))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("wrong subject: %d", rec.Code)
	}
	if st.accounts[acct.ID].DisplayName != "Ada" {
		t.Fatalf("account mutated by non-owner")
	}
}

func TestOnboardingBudgetStep(t *testing.T) {
	st, svc, acct := seedAccount(t)
	handler := svc.Handler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, patchToPost(t, "/onboarding/step",
		`{"step":"budget","study_budget":{"weekday_minutes":90,"weekend_band":"3-4"}}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("budget step: %d %s", rec.Code, rec.Body.String())
	}
	if !st.onboarding[acct.ID].BudgetSet {
		t.Fatalf("budget_set not persisted")
	}
	var budget struct {
		WeekdayMinutes int    `json:"weekday_minutes"`
		WeekendBand    string `json:"weekend_band"`
	}
	_ = json.Unmarshal(st.accounts[acct.ID].StudyBudget, &budget)
	if budget.WeekdayMinutes != 90 || budget.WeekendBand != "3-4" {
		t.Fatalf("study budget not persisted to account: %+v", budget)
	}
}

func TestOnboardingBudgetStepRejectsInvalid(t *testing.T) {
	_, svc, _ := seedAccount(t)
	handler := svc.Handler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, patchToPost(t, "/onboarding/step", `{"step":"budget"}`))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("budget step with no budget: %d, want 422", rec.Code)
	}
}

func TestOnboardingFinishStepIdempotent(t *testing.T) {
	st, svc, acct := seedAccount(t)
	handler := svc.Handler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, patchToPost(t, "/onboarding/step", `{"step":"finish"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("finish step: %d %s", rec.Code, rec.Body.String())
	}
	first := st.onboarding[acct.ID].CompletedAt
	if first.IsZero() {
		t.Fatalf("completed_at not set")
	}

	// A second Finish must not move completed_at.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, patchToPost(t, "/onboarding/step", `{"step":"finish"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("re-finish: %d", rec.Code)
	}
	if !st.onboarding[acct.ID].CompletedAt.Equal(first) {
		t.Fatalf("completed_at moved on re-finish")
	}
}

// patchToPost builds a POST /onboarding/step request as the owner.
func patchToPost(t *testing.T, path, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer good-token")
	return req
}
