package assessment

import (
	"net/http"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// resolveCourse reads the internal `?path=<slug>` param (sprint m1-03, M1b): the course
// the gateway resolved for a mock start or a course-scoped read (trend, progress). It
// travels as a query param, never a body field: a v1.6.0 assessment decodes request
// bodies with DisallowUnknownFields. It is optional and defaults to course.DefaultSlug,
// the course a v1 caller means, because a v1.6.0 gateway calls without it during a
// rolling update. A slug the compiled-in manifests don't know writes the uniform 404
// course_not_found and returns ok=false (visibility — preview, coming_soon, retired — is
// the gateway's call; this only guards a stale or hand-made call).
func (s *Service) resolveCourse(w http.ResponseWriter, r *http.Request) (string, *course.Manifest, bool) {
	slug := r.URL.Query().Get("path")
	if slug == "" {
		slug = course.DefaultSlug
	}
	m, ok := s.courses.Lookup(slug)
	if !ok {
		writeError(w, http.StatusNotFound, "course_not_found", "no such course")
		return "", nil, false
	}
	return slug, m, true
}
