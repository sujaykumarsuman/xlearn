package coach

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// routeParam matches a {name} wildcard in a ServeMux pattern.
var routeParam = regexp.MustCompile(`\{[^}]+\}`)

// TestUserRoutesRequireLearner walks the per-user route table through the full Handler
// (ADR-0033 §12 row 5): a public-read token is refused (403 forbidden), a learner token
// reaches the handler, and a request with no token is 401 unauthenticated. The 403 also
// proves each table entry is registered (an unregistered path would be the mux's 404).
func TestUserRoutesRequireLearner(t *testing.T) {
	// A provider that always fails: no route under test should reach it with an empty
	// body and no stored key, and if one did it must not leave the machine.
	prov := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(prov.Close)
	openai := NewOpenAIProvider(prov.URL, prov.Client())
	anthropic := NewAnthropicProvider(prov.URL, prov.Client())
	const account = "11111111-1111-4111-8111-111111111111"
	newSvc := func(v fakeVerifier) *Service {
		// A nil keyring exercises the single-entry fallback over the legacy cipher, which
		// is what production runs while COACH_MASTER_KEYS is unset.
		return NewService(newMemStore(), v, testCipher(), nil, openai, anthropic, coursetest.Registry(t), discardLogger())
	}
	learner := newSvc(fakeVerifier{subject: account})
	publicRead := newSvc(fakeVerifier{subject: account, roles: []string{auth.RolePublicRead}}).Handler()

	routes := learner.userRoutes()
	if len(routes) == 0 {
		t.Fatal("userRoutes() is empty")
	}
	h := learner.Handler()
	for _, rt := range routes {
		target := routeParam.ReplaceAllString(rt.Pattern, "22222222-2222-4222-8222-222222222222")
		t.Run(rt.Method+" "+rt.Pattern, func(t *testing.T) {
			rr := serveRoute(publicRead, rt.Method, target, "Bearer public-token")
			if rr.Code != http.StatusForbidden || !strings.Contains(rr.Body.String(), `"code":"forbidden"`) {
				t.Errorf("public-read token: status %d (%s), want 403 forbidden", rr.Code, rr.Body.String())
			}

			rr = serveRoute(h, rt.Method, target, "Bearer good-token")
			if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
				t.Errorf("learner token: status %d (%s), want the handler's answer", rr.Code, rr.Body.String())
			}
			if rr.Code == http.StatusNotFound && rr.Body.String() == "404 page not found\n" {
				t.Errorf("learner token: %s %s is not registered", rt.Method, target)
			}

			rr = serveRoute(h, rt.Method, target, "")
			if rr.Code != http.StatusUnauthorized || rr.Body.String() != `{"error":{"code":"unauthenticated"}}` {
				t.Errorf("no token: status %d (%s), want 401 unauthenticated", rr.Code, rr.Body.String())
			}
		})
	}
}

// serveRoute sends one request (a JSON {} body on non-GET) through h.
func serveRoute(h http.Handler, method, target, authorization string) *httptest.ResponseRecorder {
	var r *http.Request
	if method == http.MethodGet {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(`{}`))
		r.Header.Set("Content-Type", "application/json")
	}
	if authorization != "" {
		r.Header.Set("Authorization", authorization)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr
}
