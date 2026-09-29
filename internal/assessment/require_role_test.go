package assessment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// routeParam matches a {name} wildcard in a ServeMux pattern.
var routeParam = regexp.MustCompile(`\{[^}]+\}`)

// TestUserRoutesRequireLearner walks the per-user route table through the full Handler
// (ADR-0033 §12 row 5): a public-read token is refused (403 forbidden), a learner token
// reaches the handler, and a request with no token is 401 unauthenticated. The 403 also
// proves each table entry is registered (an unregistered path would be the mux's 404).
func TestUserRoutesRequireLearner(t *testing.T) {
	fs := &fakeStore{
		createMock: func(context.Context, string, string, string, string, string, time.Time, time.Time) (store.MockSession, error) {
			return store.MockSession{}, store.ErrNotFound
		},
		getMock: func(context.Context, string, string) (store.MockSession, []store.RubricScore, error) {
			return store.MockSession{}, nil, store.ErrNotFound
		},
		scoreMock: func(context.Context, string, string, map[string]int, string) (store.MockSession, []store.RubricScore, error) {
			return store.MockSession{}, nil, store.ErrNotFound
		},
		trend:       func(context.Context, string, string) ([]store.TrendPoint, error) { return nil, nil },
		solvedCount: func(context.Context, string) (int, error) { return 0, nil },
		retention:   func(context.Context, string) (int, int, error) { return 0, 0, nil },
		heatmap: func(context.Context, string, time.Time) ([]store.HeatmapDay, error) {
			return nil, nil
		},
		mastery:    func(context.Context, string) ([]store.ProblemMastery, error) { return nil, nil },
		outcomeMix: func(context.Context, string) (map[string]int, error) { return map[string]int{}, nil },
		mockStats: func(context.Context, string, string) (store.MockStats, error) {
			return store.MockStats{}, nil
		},
	}
	learner := newTestService(fs)
	publicRead := NewService(fs, fakeVerifier{subject: testAccount, roles: []string{auth.RolePublicRead}}, testLogger()).Handler()

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
