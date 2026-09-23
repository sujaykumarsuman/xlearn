package identity

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"testing"

	seeddata "github.com/sujaykumarsuman/xlearn/curriculum"
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
	// Reserved words (that share the /xlearn route namespace) are rejected.
	for _, s := range []string{"settings", "dsa", "system-design", "api", "admin", "auth", "u", "judge", "courses"} {
		if err := validateUsername(s); err == nil {
			t.Errorf("validateUsername(%q) = nil, want reserved error", s)
		}
	}
}

// Every curriculum path slug becomes a /xlearn/<slug> route that would shadow a same-named
// profile (ADR-0024), so each one in the embedded seed must be reserved — this makes adding
// a path without reserving its slug a test failure rather than a latent collision.
func TestReservedCoversCurriculumPathSlugs(t *testing.T) {
	b, err := fs.ReadFile(seeddata.FS, "paths.json")
	if err != nil {
		t.Fatalf("read curriculum paths.json: %v", err)
	}
	var paths []struct {
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(b, &paths); err != nil {
		t.Fatalf("decode curriculum paths.json: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("curriculum paths.json has no paths")
	}
	for _, p := range paths {
		if p.Slug == "" {
			t.Errorf("curriculum path with an empty slug")
			continue
		}
		if !reservedUsernames[p.Slug] {
			t.Errorf("curriculum path slug %q is not in reservedUsernames (internal/identity/username.go)", p.Slug)
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
	if rec := doJSON(t, svc.handleSetUsername, http.MethodPost, "/x", map[string]string{"username": "settings"}, claimsA, pvA); rec.Code != http.StatusUnprocessableEntity {
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
	for _, want := range []string{"account_id", "username", "display_name", "created_at", "region"} {
		if _, ok := body[want]; !ok {
			t.Errorf("public payload missing %q", want)
		}
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
