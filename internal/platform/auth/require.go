package auth

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"
)

// JWT roles (ADR-0033 §7, §12 row 5). Every user route requires RoleLearner, and the
// gateway mints exactly ["learner"] for every account. RolePublicRead is the public
// profile's token (M2b, m2-03): it is refused on every user route. The owner/tester
// cohort is NEVER a JWT role: roles and status live in identity's database, and the
// gateway reads them from session-validate.
const (
	RoleLearner    = "learner"
	RolePublicRead = "public-read"
)

// claimsKey is the one context key every service stores verified claims under.
type claimsKey struct{}

// WithClaims returns ctx carrying claims (RequireRole's context write; tests inject
// claims with it the way the middleware would).
func WithClaims(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, c)
}

// ClaimsFrom returns the verified claims RequireRole stored in ctx (zero Claims when the
// route is not behind RequireRole).
func ClaimsFrom(ctx context.Context) Claims {
	c, _ := ctx.Value(claimsKey{}).(Claims)
	return c
}

// HasRole reports whether the claims carry role.
func (c Claims) HasRole(role string) bool { return slices.Contains(c.Roles, role) }

// RequireOption configures RequireRole.
type RequireOption func(*requireConfig)

type requireConfig struct{ log *slog.Logger }

// WithLogger logs verification failures (at WARN) and role refusals (at INFO) to l.
func WithLogger(l *slog.Logger) RequireOption {
	return func(c *requireConfig) { c.log = l }
}

// RequireRole is the shared user-route middleware (ADR-0033 §12 row 5): it reads the
// gateway-minted bearer JWT, verifies it with v, and requires role in its roles claim.
// No or bad token → 401 {"error":{"code":"unauthenticated"}} (the api.md envelope the
// SPA redirects on); a valid token without role → 403 forbidden. The verified claims
// are stored for the handler (ClaimsFrom).
func RequireRole(v Verifier, role string, opts ...RequireOption) func(http.Handler) http.Handler {
	var cfg requireConfig
	for _, o := range opts {
		o(&cfg)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := BearerToken(r)
			if token == "" {
				writeUnauthenticated(w)
				return
			}
			claims, err := v.Verify(r.Context(), token)
			if err != nil {
				if cfg.log != nil {
					cfg.log.Warn("jwt verify failed", "err", err)
				}
				writeUnauthenticated(w)
				return
			}
			if !claims.HasRole(role) {
				if cfg.log != nil {
					cfg.log.Info("jwt lacks the route's role", "want", role, "roles", claims.Roles)
				}
				writeForbidden(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

// BearerToken returns the Authorization: Bearer token, or "".
func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func writeUnauthenticated(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
}

func writeForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":{"code":"forbidden","message":"this token cannot call this route"}}`))
}
