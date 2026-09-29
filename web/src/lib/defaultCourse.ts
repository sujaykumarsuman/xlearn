// The SPA's ONE default course (sprint m1-03). Course routes never need it: a v1 URL
// already names its course in the first segment, and every course link is built from the
// current course with coursePath(). The single exception is the NotFound screen, whose
// "Back to Today" button leaves no course context behind: AB02-F6 (frozen, v1 parity)
// keeps v1's target, the default course's dashboard.
//
// It mirrors course.DefaultSlug (internal/course/default.go), the course a v1 caller
// means. hack/lint-course-literals.sh allowlists this file for that reason alone; nothing
// else in web/src may name a default course.
export const DEFAULT_COURSE = "dsa";
