package identity

import (
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// startEnrollment drives POST /paths/{slug}/start for account acct against svc.
func startEnrollment(t *testing.T, svc *Service, acct, slug string) (int, string) {
	t.Helper()
	claims := auth.Claims{Subject: acct, Audience: "identity"}
	rec := doJSON(t, svc.handleStartEnrollment, http.MethodPost, "/paths/"+slug+"/start", nil, &claims, map[string]string{"slug": slug})
	return rec.Code, rec.Body.String()
}

// Enrollment accepts only an `active` course (ADR-0033 §12 row 8, sprint m1-03): unknown
// and `preview` share one 404 body (no hint a preview course exists), `coming_soon` and
// `retired` are 409, and an active course enrolls, idempotently.
func TestStartEnrollmentValidatesCourse(t *testing.T) {
	st := newFakeStore()
	svc := NewService(testConfig(), st, nil, coursetest.Registry(t), slog.New(slog.NewJSONHandler(io.Discard, nil)))
	const acct = "6a1b2c3d-4e5f-4a6b-8c7d-000000000001"

	notFound := ""
	for _, slug := range []string{"no-such-course", coursetest.FixturePreview} {
		code, body := startEnrollment(t, svc, acct, slug)
		if code != http.StatusNotFound {
			t.Fatalf("%s: status %d, want 404", slug, code)
		}
		if notFound == "" {
			notFound = body
		} else if body != notFound {
			t.Fatalf("preview's 404 %s differs from an unknown course's %s", body, notFound)
		}
	}
	var e struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal([]byte(notFound), &e); err != nil || e.Error.Code != "course_not_found" || e.Error.Message != "no such course" {
		t.Fatalf("404 body %s", notFound)
	}

	for _, slug := range []string{coursetest.FixtureComingSoon, coursetest.FixtureRetired, "behavioral"} {
		code, body := startEnrollment(t, svc, acct, slug)
		if code != http.StatusConflict {
			t.Fatalf("%s: status %d, want 409", slug, code)
		}
		if err := json.Unmarshal([]byte(body), &e); err != nil || e.Error.Code != "course_not_available" {
			t.Fatalf("%s: 409 body %s", slug, body)
		}
	}
	if n := len(st.enrollments[acct]); n != 0 {
		t.Fatalf("a refused course enrolled: %+v", st.enrollments[acct])
	}

	// Active courses enroll, and a repeat start is idempotent.
	for _, slug := range []string{course.DefaultSlug, coursetest.FixtureActive, course.DefaultSlug} {
		code, body := startEnrollment(t, svc, acct, slug)
		if code != http.StatusOK {
			t.Fatalf("%s: status %d %s, want 200", slug, code, body)
		}
		var out struct {
			Enrollment struct {
				PathSlug string `json:"path_slug"`
				Status   string `json:"status"`
			} `json:"enrollment"`
		}
		if err := json.Unmarshal([]byte(body), &out); err != nil || out.Enrollment.PathSlug != slug || out.Enrollment.Status != "active" {
			t.Fatalf("%s: body %s", slug, body)
		}
	}
	if n := len(st.enrollments[acct]); n != 2 {
		t.Fatalf("%d enrollments, want 2 (the repeat start is idempotent)", n)
	}
}

// A new enrollment's public_visible is its manifest's public_stats.default_visible (D7):
// every embedded course's default (and each fixture's) flows through the handler to the
// store. Only an active course enrolls, so the test registry marks every course active.
func TestEnrollmentPublicVisibleFromManifest(t *testing.T) {
	all := maps.Clone(coursetest.All(t))
	maps.Copy(all, coursetest.Fixtures(t))
	active := map[string]*course.Manifest{}
	sawPrivate := false
	for slug, m := range all {
		c := *m // a shallow copy: only Status changes, nothing shared is mutated
		c.Status = course.StatusActive
		active[slug] = &c
		sawPrivate = sawPrivate || !m.PublicStats.Visible()
	}
	if !sawPrivate {
		t.Fatal("no course defaults to private: the test can't tell false from a missing value")
	}
	st := newFakeStore()
	svc := NewService(testConfig(), st, nil, course.NewRegistry(active), slog.New(slog.NewJSONHandler(io.Discard, nil)))
	const acct = "6a1b2c3d-4e5f-4a6b-8c7d-000000000002"
	for slug, m := range all {
		if code, body := startEnrollment(t, svc, acct, slug); code != http.StatusOK {
			t.Fatalf("%s: status %d %s", slug, code, body)
		}
		if got, want := st.publicVisible[acct+"|"+slug], m.PublicStats.Visible(); got != want {
			t.Fatalf("%s: public_visible %v, manifest default_visible %v", slug, got, want)
		}
	}
}
