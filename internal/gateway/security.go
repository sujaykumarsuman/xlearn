package gateway

import (
	"mime"
	"net/http"
	"strings"
)

// Browser-facing security floor (m1-04 task 6; ADR-0033 §9 threat "cross-site writes from
// sibling *.sujaykumar.dev hosts"). SameSite=Lax admits same-SITE requests, so a sibling
// host could POST to xLearn with the learner's cookie; the gateway therefore requires every
// mutating /api call to be same-origin (Sec-Fetch-Site) and JSON (a cross-origin form or
// no-cors fetch can't send application/json without a CORS preflight, which we never grant).

// contentSecurityPolicy is the CSP on every SPA shell response (index.html). Notes:
//   - script-src is 'self' only — never 'unsafe-inline'. Vite emits external module scripts.
//   - style-src allows the Google Fonts stylesheet; React style={{…}} props go through the
//     CSSOM and are not blocked, but a <style> element or style="" attribute in index.html
//     would be.
//   - connect-src 'self' covers the BFF fetches and the coach SSE stream.
//   - form-action lists https://github.com: Chrome applies form-action to the redirect that
//     follows the OAuth start form POST (/api/auth/{provider}/start → github.com).
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; object-src 'none'; " +
	"base-uri 'self'; frame-ancestors 'none'; form-action 'self' https://github.com"

// setSecurityHeaders stamps the headers every gateway response carries (probes, API,
// assets and the shell alike).
func setSecurityHeaders(h http.Header) {
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "same-origin")
}

// setShellHeaders adds the SPA-shell-only headers (the CSP) on top of setSecurityHeaders.
func setShellHeaders(h http.Header) {
	h.Set("Content-Security-Policy", contentSecurityPolicy)
}

// isMutatingMethod reports whether method changes server state (the methods the
// cross-site write checks apply to).
func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// isOAuthStart reports whether appPath is POST /api/auth/{provider}/start — the top-level
// HTML form POST (sign-in, and Settings' "Connect GitHub" with ?link=1). A form can't send
// application/json, so this route keeps only the Sec-Fetch-Site check.
func isOAuthStart(method, appPath string) bool {
	if method != http.MethodPost {
		return false
	}
	rest, ok := strings.CutPrefix(appPath, "/api/auth/")
	if !ok {
		return false
	}
	provider, tail, ok := strings.Cut(rest, "/")
	return ok && provider != "" && tail == "start"
}

// allowWrite enforces the cross-site write checks on a mutating /api request. appPath is
// the request path after the base-path strip and the /api/v1 → /api rewrite. It writes the
// error envelope and returns false when the request is refused:
//   - Sec-Fetch-Site present and not same-origin/none → 403 cross_site_request. An absent
//     header (non-browser clients) is allowed by design; the JSON check still applies.
//   - Content-Type not application/json (parameters such as charset allowed) → 415
//     unsupported_media_type — except the OAuth start form POST.
func allowWrite(w http.ResponseWriter, r *http.Request, appPath string) bool {
	if !isMutatingMethod(r.Method) {
		return true
	}
	if site := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))); site != "" && site != "same-origin" && site != "none" {
		writeError(w, http.StatusForbidden, "cross_site_request", "Cross-site requests can't change xLearn data.")
		return false
	}
	if isOAuthStart(r.Method, appPath) {
		return true
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send this request as application/json.")
		return false
	}
	return true
}
