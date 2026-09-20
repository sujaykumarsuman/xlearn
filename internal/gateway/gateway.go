// Package gateway is the xLearn edge service: it serves the embedded React SPA
// and the (currently minimal) BFF API under a base path, and exposes the k8s
// liveness/readiness probes. It owns no schema (ADR-0002/0008); later sprints
// add session validation, JWT minting and screen aggregation.
package gateway

import (
	"bytes"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/health"
)

// Options configure the gateway handler.
type Options struct {
	// BasePath is the URL prefix the SPA + API are mounted under (e.g. /xlearn),
	// normalised: leading slash, no trailing slash, or "" for root.
	BasePath string
	// Version is stamped into the app health response.
	Version string
	// Dist is the built SPA filesystem, rooted at the directory that contains
	// index.html and assets/ (i.e. fs.Sub(xlearn.Dist, "web/dist")).
	Dist fs.FS
	// Logger is the structured logger.
	Logger *slog.Logger

	// Signer mints internal JWTs and provides the JWKS the gateway publishes
	// (ADR-0006). Nil disables the auth BFF (the auth routes 503) — used only in
	// tests that exercise SPA/probe serving.
	Signer *auth.Signer
	// IdentityBaseURL is the identity service's internal URL. Empty disables the
	// identity-backed BFF routes.
	IdentityBaseURL string
	// AudienceIdentity is the "aud" for JWTs forwarded to identity.
	AudienceIdentity string
	// CurriculumBaseURL is the curriculum service's internal URL. Empty disables the
	// curriculum content proxy routes (they 503).
	CurriculumBaseURL string
}

// Gateway serves the SPA, the app BFF API and the k8s probes.
type Gateway struct {
	basePath string
	version  string
	dist     fs.FS
	log      *slog.Logger
	health   *health.Handler

	signer      *auth.Signer
	identity    *identityClient
	curriculum  *curriculumClient
	audIdentity string
	api         *http.ServeMux
}

// New builds a Gateway. Readiness is trivially OK this sprint (stateless; no DB).
func New(opt Options) *Gateway {
	log := opt.Logger
	if log == nil {
		log = slog.New(slog.NewTextHandler(nopWriter{}, nil))
	}
	g := &Gateway{
		basePath:    opt.BasePath,
		version:     opt.Version,
		dist:        opt.Dist,
		log:         log,
		health:      health.New(),
		signer:      opt.Signer,
		audIdentity: opt.AudienceIdentity,
	}
	if opt.IdentityBaseURL != "" {
		g.identity = newIdentityClient(opt.IdentityBaseURL)
	}
	if opt.CurriculumBaseURL != "" {
		g.curriculum = newCurriculumClient(opt.CurriculumBaseURL)
	}
	g.api = g.newAPIMux()
	return g
}

// nopWriter drops logs (used when no logger is supplied, e.g. in tests).
type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

// Handler returns the composed HTTP handler.
//
// Routing model (ADR-0009 base-path decision): in production Traefik strips the
// base path (stripPrefix: true) so the pod sees "/", while k8s probes hit the
// pod directly at /healthz and /readyz. Run locally without Traefik the pod
// sees the full "/xlearn/..." path. The handler is therefore base-path tolerant
// — it strips BasePath when present (a no-op behind Traefik) — so the same
// binary serves correctly in both environments.
func (g *Gateway) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// k8s probes are always at the pod root, prefix-independent.
		switch r.URL.Path {
		case "/healthz":
			g.health.Live(w, r)
			return
		case "/readyz":
			g.health.Ready(w, r)
			return
		}

		p := g.stripBase(r.URL.Path)

		// The BFF API and the (non-secret) JWKS go through the API mux — which
		// handles method/pattern routing and 404s unknown /api/* paths (never
		// falling through to the SPA shell). JWKS is served at the pod root too so
		// internal services can fetch it directly on the ClusterIP.
		if p == "/.well-known/jwks.json" || p == "/api" || strings.HasPrefix(p, "/api/") {
			r2 := r.Clone(r.Context())
			r2.URL.Path = p
			r2.URL.RawPath = ""
			g.api.ServeHTTP(w, r2)
			return
		}

		g.serveStatic(w, r, p)
	})
}

// stripBase removes the configured base path from an incoming request path,
// leaving an absolute app path ("/", "/assets/x.js", "/api/healthz"). It is a
// no-op when the prefix is absent (behind Traefik stripPrefix, or at root).
func (g *Gateway) stripBase(p string) string {
	if g.basePath == "" {
		return p
	}
	if p == g.basePath {
		return "/"
	}
	if strings.HasPrefix(p, g.basePath+"/") {
		return p[len(g.basePath):]
	}
	return p
}

// appHealth is the BFF-level health check, reachable externally at
// <base>/api/healthz. Distinct from the k8s /healthz probe.
func (g *Gateway) appHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"gateway","version":"` + jsonEscape(g.version) + `"}`))
}

// serveStatic serves an embedded asset, or falls back to index.html for any
// path the SPA router should handle (SPA fallback). Hashed assets get an
// immutable long cache; index.html is never cached so a deploy is picked up.
func (g *Gateway) serveStatic(w http.ResponseWriter, r *http.Request, appPath string) {
	name := strings.TrimPrefix(path.Clean("/"+appPath), "/")
	if name == "" || name == "." {
		g.serveIndex(w, r)
		return
	}
	data, err := fs.ReadFile(g.dist, name)
	if err != nil {
		// Not a real asset: a client-side route (e.g. /dsa/week/2) or unknown
		// path — serve the shell and let the router decide.
		g.serveIndex(w, r)
		return
	}
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	if ctype := mime.TypeByExtension(path.Ext(name)); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}

// serveIndex writes index.html with a no-cache header. Returns 200 for SPA
// fallback routes; the client router renders its own not-found.
func (g *Gateway) serveIndex(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(g.dist, "index.html")
	if err != nil {
		// web/dist has not been built into the binary.
		http.Error(w, "web UI not built", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// jsonEscape escapes the small set of characters that could break the inlined
// version string in the app-health JSON literal.
func jsonEscape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`)
	return r.Replace(s)
}
