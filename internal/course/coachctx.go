package course

import "strings"

// Coach page contexts (sprint m1-03; t0 §7). The coach thread key is
// coach_thread UNIQUE (account_id, page_context), so every COURSE-SCOPED context carries
// its course: `<course>:concept:<slug>`, `<course>:week:<n>`, `<course>:roadmap`, … . Item
// contexts keep `problem:<id>` (item ids are global) and the account-wide contexts
// (catalog, settings, general) stay bare.
//
// v1.6.0 tabs send the unprefixed v1 forms (concept:<slug>, week:<n>, the six bare
// words); the dual parser maps them to DefaultSlug, the course every v1 context meant.
// The coach service and the gateway both run the SAME parser (NormalizeCoachContext), and
// normalizing a normalized key returns it unchanged, so either side may run it first.

// Coach context kinds.
const (
	CoachKindProblem = "problem"
	CoachKindConcept = "concept"
	CoachKindWeek    = "week"
)

// CoachAccountContexts are the account-wide contexts: never prefixed, no course.
var CoachAccountContexts = []string{"catalog", "settings", "general"}

// CoachCourseScreens are the course-scoped contexts that take no argument.
var CoachCourseScreens = []string{"roadmap", "dashboard", "revision", "mistakes", "mock", "progress"}

// CoachContext is a normalized coach page context.
type CoachContext struct {
	// Key is the thread key: `<course>:<ctx>` for a course-scoped context, else unchanged.
	Key string
	// PathSlug is the course of a course-scoped context; "" for problem, account-wide and
	// unrecognised contexts (a problem's course is the item's, which only curriculum knows).
	PathSlug string
	// Kind is problem | concept | week | one of CoachCourseScreens | one of
	// CoachAccountContexts | "" (unrecognised: passed through as-is).
	Kind string
}

// CourseScoped reports whether the context belongs to one course (PathSlug is set).
func (c CoachContext) CourseScoped() bool { return c.PathSlug != "" }

// NormalizeCoachContext parses raw with the compiled-in registry
// (Embedded().NormalizeCoachContext).
func NormalizeCoachContext(raw string) CoachContext { return Embedded().NormalizeCoachContext(raw) }

// NormalizeCoachContext is the dual parser. A context is new-form only when its first
// segment resolves to a course in r and the rest is a known course-scoped kind
// (`<course>:concept:<slug>`, `<course>:week:<n>`, `<course>:<screen>`). Otherwise a
// legacy course-scoped form maps to DefaultSlug + ":" + raw; `problem:<id>`, the
// account-wide contexts and anything unrecognised pass through unchanged.
func (r *Registry) NormalizeCoachContext(raw string) CoachContext {
	if head, rest, ok := strings.Cut(raw, ":"); ok {
		if _, known := r.Lookup(head); known {
			if kind, ok := courseKind(rest); ok {
				return CoachContext{Key: raw, PathSlug: head, Kind: kind}
			}
		}
	}
	if kind, ok := courseKind(raw); ok {
		return CoachContext{Key: DefaultSlug + ":" + raw, PathSlug: DefaultSlug, Kind: kind}
	}
	if id, ok := strings.CutPrefix(raw, CoachKindProblem+":"); ok && id != "" {
		return CoachContext{Key: raw, Kind: CoachKindProblem}
	}
	for _, a := range CoachAccountContexts {
		if raw == a {
			return CoachContext{Key: raw, Kind: a}
		}
	}
	return CoachContext{Key: raw}
}

// courseKind classifies an unprefixed course-scoped context: concept:<slug>, week:<n> or
// one of CoachCourseScreens.
func courseKind(ctx string) (string, bool) {
	if kind, arg, ok := strings.Cut(ctx, ":"); ok {
		if (kind == CoachKindConcept || kind == CoachKindWeek) && arg != "" && !strings.Contains(arg, ":") {
			return kind, true
		}
		return "", false
	}
	for _, s := range CoachCourseScreens {
		if ctx == s {
			return s, true
		}
	}
	return "", false
}
