package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

const unauthenticatedBody = `{"error":{"code":"unauthenticated"}}`

// TestRequireRole drives RequireRole(learner) with real gateway-minted tokens verified
// against a JWKS (ADR-0033 §12 row 5): no or bad token → 401 with the exact api.md
// envelope; a valid token without the role → 403 forbidden; a token carrying the role
// (alone or with others) → the next handler, with the verified claims in its context.
func TestRequireRole(t *testing.T) {
	signer := NewSigner(mustKey(t), "xlearn-gateway", DefaultTTL)
	stranger := NewSigner(mustKey(t), "xlearn-gateway", DefaultTTL) // not in the JWKS
	js := newJWKSServer(t, signer)
	v := NewJWKSVerifier(js.srv.URL, "practice", "xlearn-gateway")
	mint := func(s *Signer, aud string, roles ...string) string {
		t.Helper()
		tok, err := s.Mint(context.Background(), "acct-1", aud, roles)
		if err != nil {
			t.Fatalf("mint: %v", err)
		}
		return tok
	}
	learner := mint(signer, "practice", RoleLearner)

	cases := []struct {
		name          string
		authorization string
		wantStatus    int
		wantBody      string // exact body, when set
		wantCode      string // error.code, when set
		wantRoles     []string
	}{
		{name: "no header", wantStatus: http.StatusUnauthorized, wantBody: unauthenticatedBody},
		{name: "not a bearer scheme", authorization: "Basic YWxhZGRpbjpvcGVu", wantStatus: http.StatusUnauthorized, wantBody: unauthenticatedBody},
		{name: "empty bearer", authorization: "Bearer   ", wantStatus: http.StatusUnauthorized, wantBody: unauthenticatedBody},
		{name: "malformed token", authorization: "Bearer not-a-jwt", wantStatus: http.StatusUnauthorized, wantBody: unauthenticatedBody},
		{name: "unknown signing key", authorization: "Bearer " + mint(stranger, "practice", RoleLearner), wantStatus: http.StatusUnauthorized, wantBody: unauthenticatedBody},
		{name: "wrong audience", authorization: "Bearer " + mint(signer, "review", RoleLearner), wantStatus: http.StatusUnauthorized, wantBody: unauthenticatedBody},
		{name: "public-read lacks learner", authorization: "Bearer " + mint(signer, "practice", RolePublicRead), wantStatus: http.StatusForbidden, wantCode: "forbidden"},
		{name: "no roles", authorization: "Bearer " + mint(signer, "practice"), wantStatus: http.StatusForbidden, wantCode: "forbidden"},
		{name: "learner", authorization: "Bearer " + learner, wantStatus: http.StatusNoContent, wantRoles: []string{RoleLearner}},
		{name: "learner and public-read", authorization: "Bearer " + mint(signer, "practice", RolePublicRead, RoleLearner), wantStatus: http.StatusNoContent, wantRoles: []string{RolePublicRead, RoleLearner}},
		{name: "case-insensitive scheme", authorization: "bearer " + learner, wantStatus: http.StatusNoContent, wantRoles: []string{RoleLearner}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var called bool
			var got Claims
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				got = ClaimsFrom(r.Context())
				w.WriteHeader(http.StatusNoContent)
			})
			h := RequireRole(v, RoleLearner)(next)

			r := httptest.NewRequest(http.MethodGet, "/state", nil)
			if c.authorization != "" {
				r.Header.Set("Authorization", c.authorization)
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, r)

			if rr.Code != c.wantStatus {
				t.Fatalf("status = %d (%s), want %d", rr.Code, rr.Body.String(), c.wantStatus)
			}
			if c.wantBody != "" && rr.Body.String() != c.wantBody {
				t.Errorf("body = %s, want %s", rr.Body.String(), c.wantBody)
			}
			if c.wantCode != "" {
				var env struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil || env.Error.Code != c.wantCode {
					t.Errorf("body = %s (%v), want error.code %q", rr.Body.String(), err, c.wantCode)
				}
			}
			if c.wantStatus != http.StatusNoContent {
				if called {
					t.Error("next handler ran for a refused request")
				}
				if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
				return
			}
			if !called {
				t.Fatal("next handler did not run")
			}
			if got.Subject != "acct-1" || got.Audience != "practice" || !slices.Equal(got.Roles, c.wantRoles) {
				t.Errorf("ClaimsFrom = %+v, want acct-1/practice/%v", got, c.wantRoles)
			}
		})
	}
}

// TestRequireRoleLogs: WithLogger records verification failures and role refusals (and
// never the token itself).
func TestRequireRoleLogs(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	const badTok, publicTok = "tok-unverifiable-7f3a", "tok-public-read-9c1e"
	v := stubVerifier{publicTok: {Subject: "acct-1", Roles: []string{RolePublicRead}}}
	h := RequireRole(v, RoleLearner, WithLogger(log))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("next handler ran")
	}))

	for _, tok := range []string{badTok, publicTok} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(httptest.NewRecorder(), r)
	}
	out := buf.String()
	if !strings.Contains(out, "jwt verify failed") || !strings.Contains(out, "jwt lacks the route's role") {
		t.Errorf("log = %q, want the verify failure and the role refusal", out)
	}
	if strings.Contains(out, badTok) || strings.Contains(out, publicTok) {
		t.Errorf("log leaked a token: %q", out)
	}
}

func TestClaimsFrom(t *testing.T) {
	if c := ClaimsFrom(context.Background()); c.Subject != "" || c.Roles != nil || c.Audience != "" {
		t.Errorf("ClaimsFrom(bare ctx) = %+v, want zero Claims", c)
	}
	want := Claims{Subject: "acct-1", Roles: []string{RoleLearner}, Audience: "coach"}
	got := ClaimsFrom(WithClaims(context.Background(), want))
	if got.Subject != want.Subject || got.Audience != want.Audience || !slices.Equal(got.Roles, want.Roles) {
		t.Errorf("ClaimsFrom(WithClaims) = %+v, want %+v", got, want)
	}
}

func TestClaimsHasRole(t *testing.T) {
	c := Claims{Roles: []string{RolePublicRead, RoleLearner}}
	if !c.HasRole(RoleLearner) || !c.HasRole(RolePublicRead) || c.HasRole("owner") {
		t.Errorf("HasRole on %v is wrong", c.Roles)
	}
	if (Claims{}).HasRole(RoleLearner) {
		t.Error("zero Claims has a role")
	}
}

// stubVerifier maps a token to its claims; any other token fails verification.
type stubVerifier map[string]Claims

func (s stubVerifier) Verify(_ context.Context, token string) (Claims, error) {
	c, ok := s[token]
	if !ok {
		return Claims{}, ErrUnauthenticated
	}
	return c, nil
}
