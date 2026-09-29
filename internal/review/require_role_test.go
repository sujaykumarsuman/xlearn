package review

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// routeParam matches a {name} wildcard in a ServeMux pattern.
var routeParam = regexp.MustCompile(`\{[^}]+\}`)

// TestUserRoutesRequireLearner walks the per-user route table through the full Handler
// (ADR-0033 §12 row 5): a public-read token is refused (403 forbidden), a learner token
// reaches the handler, and a request with no token is 401 unauthenticated. The 403 also
// proves each table entry is registered (an unregistered path would be the mux's 404).
func TestUserRoutesRequireLearner(t *testing.T) {
	st := &fakeStore{
		score: func(context.Context, string, string, store.ScoreInput) (store.ScoreResult, error) {
			return store.ScoreResult{}, store.ErrNotFound
		},
		dueQueue: func(context.Context, string, string, int) ([]store.DueItem, error) {
			return nil, nil
		},
		listMistakes: func(context.Context, string, string, string) ([]store.Mistake, error) {
			return nil, nil
		},
		getMistake: func(context.Context, string, string) (store.Mistake, error) {
			return store.Mistake{}, store.ErrNotFound
		},
		createMistake: func(context.Context, string, string, store.MistakeInput) (store.Mistake, error) {
			return store.Mistake{}, store.ErrNotFound
		},
		updateMistake: func(context.Context, string, string, store.MistakePatch) (store.Mistake, error) {
			return store.Mistake{}, store.ErrNotFound
		},
		weakAreaCurrent: func(context.Context, string, string) (store.WeakArea, bool, error) {
			return store.WeakArea{}, false, nil
		},
		listReminders: func(context.Context, string, int) ([]store.Reminder, error) {
			return nil, nil
		},
	}
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	learner := NewService(st, fakeVerifier{subject: "acct-1"}, log)
	publicRead := NewService(st, fakeVerifier{subject: "acct-1", roles: []string{auth.RolePublicRead}}, log).Handler()

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
