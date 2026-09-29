package identity

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

func TestValidateUsername(t *testing.T) {
	ok := []string{"ada", "ada-lovelace", "user123", "a1b", "sujay-k-2"}
	for _, s := range ok {
		if err := validateUsername(normalizeUsername(s)); err != nil {
			t.Errorf("validateUsername(%q) = %v, want nil", s, err)
		}
	}
	bad := []string{"ab", "-ada", "ada-", "ada--lovelace", "Ada Lovelace", "a.b", "user_1", "über", "this-name-is-way-too-long-to-be-valid"}
	for _, s := range bad {
		if err := validateUsername(normalizeUsername(s)); err == nil {
			t.Errorf("validateUsername(%q) = nil, want error", s)
		}
	}
	// Impersonation / system handles are reserved.
	for _, s := range []string{"admin", "support", "xlearn", "login", "system"} {
		if err := validateUsername(s); err == nil {
			t.Errorf("validateUsername(%q) = nil, want reserved error", s)
		}
	}
	// Course slugs and route words are claimable: profiles live under /xlearn/u/<username>, so
	// they can't shadow a route (owner direction, ADR-0025 2026-09-23 update).
	for _, s := range []string{"dsa", "system-design", "sql", "settings", "auth", "api", "dashboard", "judge", "courses"} {
		if err := validateUsername(s); err != nil {
			t.Errorf("validateUsername(%q) = %v, want nil (course slugs / route words are not reserved)", s, err)
		}
	}
}

func TestSetUsernameAndAvailability(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	a, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{Provider: "github", ProviderUserID: "1", DisplayName: "Ada", Email: "ada@example.com"})
	b, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{Provider: "github", ProviderUserID: "2", DisplayName: "Bo", Email: "bo@example.com"})
	claimsA := &auth.Claims{Subject: a.ID}
	pvA := map[string]string{"id": a.ID}

	// Reserved name → 422.
	if rec := doJSON(t, svc.handleSetUsername, http.MethodPost, "/x", map[string]string{"username": "admin"}, claimsA, pvA); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reserved username status %d, want 422", rec.Code)
	}
	// Claim "ada" (normalises from mixed case) → 200.
	if rec := doJSON(t, svc.handleSetUsername, http.MethodPost, "/x", map[string]string{"username": "Ada"}, claimsA, pvA); rec.Code != http.StatusOK {
		t.Fatalf("claim status %d, want 200", rec.Code)
	}
	// Availability now reports "ada" taken for account B, but available for its owner A.
	if rec := doJSON(t, svc.handleUsernameAvailable, http.MethodGet, "/username/available?u=ada", nil, &auth.Claims{Subject: b.ID}, nil); !availableIs(t, rec, false) {
		t.Fatalf("ada should be unavailable to account B")
	}
	if rec := doJSON(t, svc.handleUsernameAvailable, http.MethodGet, "/username/available?u=ada", nil, claimsA, nil); !availableIs(t, rec, true) {
		t.Fatalf("ada should read available to its owner A")
	}
	// Account B cannot claim "ada" → 409.
	if rec := doJSON(t, svc.handleSetUsername, http.MethodPost, "/x", map[string]string{"username": "ada"}, &auth.Claims{Subject: b.ID}, map[string]string{"id": b.ID}); rec.Code != http.StatusConflict {
		t.Fatalf("taken username status %d, want 409", rec.Code)
	}
	// Cross-account write is forbidden.
	if rec := doJSON(t, svc.handleSetUsername, http.MethodPost, "/x", map[string]string{"username": "ada"}, claimsA, map[string]string{"id": b.ID}); rec.Code != http.StatusForbidden {
		t.Fatalf("cross-account status %d, want 403", rec.Code)
	}
}

func TestLoginByUsername(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	// Email sign-up, then claim a username.
	if rec := doJSON(t, svc.handleSignup, http.MethodPost, "/auth/signup", map[string]string{"email": "ada@example.com", "password": "hunter2hunter"}, nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("signup status %d", rec.Code)
	}
	acct, _ := st.GetAccountByEmail(context.Background(), "ada@example.com")
	if _, err := st.SetUsername(context.Background(), acct.ID, "ada"); err != nil {
		t.Fatalf("set username: %v", err)
	}
	// Login with the username instead of the email.
	if rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "ada", "password": "hunter2hunter"}, nil, nil); rec.Code != http.StatusOK || !hasCookie(rec.Result().Cookies(), auth.SessionCookieName) {
		t.Fatalf("username login status %d (want 200 + cookie)", rec.Code)
	}
	// Wrong password via username → uniform 401.
	if rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "ada", "password": "wrongwrong"}, nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status %d, want 401", rec.Code)
	}
	// Unknown username → uniform 401 (no enumeration).
	if rec := doJSON(t, svc.handleLogin, http.MethodPost, "/auth/login", map[string]string{"email": "nobody", "password": "whatever12"}, nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unknown username status %d, want 401", rec.Code)
	}
}

func TestInternalByUsernameReturnsPublicFieldsOnly(t *testing.T) {
	st := newFakeStore()
	svc := newTestService(st, nil)
	acct, _, _ := st.FindOrCreateAccount(context.Background(), store.OAuthUpsert{Provider: "github", ProviderUserID: "1", DisplayName: "Ada", Email: "ada@example.com"})
	if _, err := st.SetUsername(context.Background(), acct.ID, "ada"); err != nil {
		t.Fatalf("set username: %v", err)
	}
	rec := doJSON(t, svc.handleInternalGetAccountByUsername, http.MethodGet, "/internal/accounts/by-username/ada", nil, nil, map[string]string{"username": "ada"})
	if rec.Code != http.StatusOK {
		t.Fatalf("by-username status %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, want := range []string{"account_id", "username", "display_name", "created_at", "region", "visible_courses"} {
		if _, ok := body[want]; !ok {
			t.Errorf("public payload missing %q", want)
		}
	}
	// No enrollments: an empty array, never null (the gateway iterates it).
	if vc, ok := body["visible_courses"].([]any); !ok || len(vc) != 0 {
		t.Errorf("visible_courses = %#v, want []", body["visible_courses"])
	}
	// region is a COARSE UTC-offset band, never the IANA zone. The fake account is UTC.
	if body["region"] != "UTC" {
		t.Errorf("region = %v, want UTC (coarse offset, not the zone name)", body["region"])
	}
	// Must NOT leak PII — including the raw timezone (only the coarse offset is public).
	for _, leak := range []string{"email", "password_hash", "timezone", "study_budget", "reminders", "linked_providers"} {
		if _, ok := body[leak]; ok {
			t.Errorf("public payload leaks %q", leak)
		}
	}
	// Unknown username → 404.
	if rec := doJSON(t, svc.handleInternalGetAccountByUsername, http.MethodGet, "/internal/accounts/by-username/ghost", nil, nil, map[string]string{"username": "ghost"}); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown username status %d, want 404", rec.Code)
	}
}

// resolveByUsername drives the internal public-profile resolver.
func resolveByUsername(t *testing.T, svc *Service, name string) (int, string, []string) {
	t.Helper()
	rec := doJSON(t, svc.handleInternalGetAccountByUsername, http.MethodGet, "/internal/accounts/by-username/"+name, nil, nil, map[string]string{"username": name})
	var body struct {
		VisibleCourses []string `json:"visible_courses"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, rec.Body.String(), body.VisibleCourses
}

// P2 (m1-05; ADR-0033 §13): visible_courses = enrolled ∩ public_visible ∩ `active`.
func TestInternalByUsernameVisibleCourses(t *testing.T) {
	ctx := context.Background()
	st := newFakeStore()
	svc := NewService(testConfig(), st, nil, coursetest.Registry(t), slog.New(slog.NewJSONHandler(io.Discard, nil)))
	acct, _, _ := st.FindOrCreateAccount(ctx, store.OAuthUpsert{Provider: "github", ProviderUserID: "1", DisplayName: "Ada", Email: "ada@example.com"})
	if _, err := st.SetUsername(ctx, acct.ID, "ada"); err != nil {
		t.Fatal(err)
	}
	// Enrolled and visible: the default course; plus a preview enrollment (the owner's
	// cohort can enroll in one) and, directly in the store, the non-enrollable statuses —
	// none of which may ever show. The fixture active course is not enrolled yet.
	for _, slug := range []string{course.DefaultSlug, coursetest.FixturePreview, coursetest.FixtureComingSoon, coursetest.FixtureRetired} {
		if _, err := st.StartEnrollment(ctx, acct.ID, slug, true); err != nil {
			t.Fatal(err)
		}
	}
	want := func(step string, slugs ...string) {
		t.Helper()
		code, body, got := resolveByUsername(t, svc, "ada")
		if code != http.StatusOK {
			t.Fatalf("%s: status %d %s", step, code, body)
		}
		if strings.Join(got, ",") != strings.Join(slugs, ",") {
			t.Fatalf("%s: visible_courses = %v, want %v", step, got, slugs)
		}
	}
	want("active-but-not-enrolled excluded; preview/coming_soon/retired excluded", course.DefaultSlug)

	// Enrolled but hidden: excluded.
	if _, err := st.StartEnrollment(ctx, acct.ID, coursetest.FixtureActive, false); err != nil {
		t.Fatal(err)
	}
	want("enrolled-but-hidden excluded", course.DefaultSlug)

	st.setPublicVisible(acct.ID, coursetest.FixtureActive, true)
	want("enrolled and visible shown", course.DefaultSlug, coursetest.FixtureActive)

	st.setPublicVisible(acct.ID, course.DefaultSlug, false)
	want("the default course hidden too", coursetest.FixtureActive)
}

// P11 (m1-05): the resolver answers only an ACTIVE account; a suspended one gets exactly
// the unknown username's 404, and a reactivated one resolves again.
func TestInternalByUsernameSuspendedIs404(t *testing.T) {
	ctx := context.Background()
	st := newFakeStore()
	svc := newTestService(st, nil)
	acct, _, _ := st.FindOrCreateAccount(ctx, store.OAuthUpsert{Provider: "github", ProviderUserID: "1", DisplayName: "Ada", Email: "ada@example.com"})
	if _, err := st.SetUsername(ctx, acct.ID, "ada"); err != nil {
		t.Fatal(err)
	}
	if code, body, _ := resolveByUsername(t, svc, "ada"); code != http.StatusOK {
		t.Fatalf("active: %d %s", code, body)
	}
	_, unknown, _ := resolveByUsername(t, svc, "ghost")

	st.setRoleStatus(acct.ID, "", store.StatusSuspended)
	code, body, _ := resolveByUsername(t, svc, "ada")
	if code != http.StatusNotFound || body != unknown {
		t.Fatalf("suspended: %d %s, want the unknown username's 404 %s", code, body, unknown)
	}

	st.setRoleStatus(acct.ID, "", store.StatusActive)
	if code, body, _ := resolveByUsername(t, svc, "ada"); code != http.StatusOK {
		t.Fatalf("reactivated: %d %s", code, body)
	}
}

// availableIs asserts the {available:bool} shape of the availability endpoint.
func availableIs(t *testing.T, rec interface{ Result() *http.Response }, want bool) bool {
	t.Helper()
	resp := rec.Result()
	var body struct {
		Available bool `json:"available"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return body.Available == want
}
