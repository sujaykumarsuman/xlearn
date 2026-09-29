#!/bin/sh
# lint-course-literals.sh — the course-literal gate (sprint m1-03, M1b; ADR-0026 §5).
#
# DSA stopped being special in M1b: every course-scoped code path resolves its course
# (the gateway against the compiled-in manifests, the SPA through useCourse()). A
# forgotten DSA literal would quietly keep a DSA-only path, so this gate fails CI when
# non-test code under internal/ or web/src/ spells the DSA slug as a string or a route:
#
#   "dsa"  'dsa'  `dsa…     a quoted slug (also "/dsa", `/dsa${…}`, '/dsa' …): a quote or
#                           backtick, an optional /, then dsa not followed by [a-z0-9_-]
#   /dsa/                   a route segment anywhere (code or comment)
#
# (dsa-mock@1, the DSA rubric id, is not the slug and doesn't match.)
#
# Allowlist — each entry with its reason:
#   internal/course/default.go            course.DefaultSlug / DSASlug: THE one spelling
#                                         every mixed-version default and alias uses
#   internal/platform/events/envelope.go  events.V1PathSlug, the frozen v1-envelope meaning
#                                         (a platform package; it doesn't import course;
#                                         a test pins it to course.DSASlug)
#   web/src/lib/defaultCourse.ts          the SPA's one default: v1's NotFound "Back to
#                                         Today" target, frozen by AB02-F6
#   internal/*/store/migrations/*.sql     goose migrations (history: never edited)
#   */testdata/*                          fixtures
#   *_test.go, *.test.ts, *.test.tsx,     tests
#   web/src/test/*
# Course content lives outside the scanned trees (curriculum/courses/dsa/**).
#
# The alias rows and the internal ?path= defaults are NOT allowlisted: they use
# course.DefaultSlug. Extend the allowlist only with a reason, here.
#
# Usage:
#   hack/lint-course-literals.sh              scan internal/ and web/src/
#   hack/lint-course-literals.sh FILE...      scan exactly these files (allowlist applies)
#   hack/lint-course-literals.sh --self-test  prove the pattern and the allowlist
# Exit: 0 clean, 1 a literal found, 2 usage. POSIX sh + grep.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)

# The literal pattern (ERE), as described above.
pattern="([\"'\`]/?dsa([^a-z0-9_-]|\$))|(/dsa/)"

# allowed REL — 0 when the repo-relative path is on the allowlist.
allowed() {
	case "$1" in
	internal/course/default.go) return 0 ;;
	internal/platform/events/envelope.go) return 0 ;;
	web/src/lib/defaultCourse.ts) return 0 ;;
	internal/*/store/migrations/*.sql) return 0 ;;
	*/testdata/*) return 0 ;;
	*_test.go | *.test.ts | *.test.tsx) return 0 ;;
	web/src/test/*) return 0 ;;
	esac
	return 1
}

# scan REL... — print every hit as file:line: text; return 1 when any.
scan() {
	bad=0
	for rel in "$@"; do
		if allowed "$rel"; then
			continue
		fi
		if hits=$(grep -nE "$pattern" "$root/$rel" 2>/dev/null); then
			printf '%s\n' "$hits" | sed "s|^|$rel:|"
			bad=1
		fi
	done
	return "$bad"
}

# files — every candidate source file under the scanned trees, repo-relative.
files() {
	(cd "$root" && find internal web/src -type f \
		\( -name '*.go' -o -name '*.sql' -o -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.json' \) \
		-not -path '*/node_modules/*' | LC_ALL=C sort)
}

self_test() {
	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT
	mkdir -p "$tmp/internal/x" "$tmp/internal/course" "$tmp/internal/x/testdata" "$tmp/web/src/lib"
	fail=0
	expect() { # expect WANT FILE CONTENT
		printf '%s\n' "$3" >"$tmp/$2"
		if (root=$tmp scan "$2" >/dev/null); then got=0; else got=1; fi
		if [ "$got" != "$1" ]; then
			echo "self-test: $2 containing [$3]: exit $got, want $1" >&2
			fail=1
		fi
	}
	# Hits.
	expect 1 internal/x/a.go 'x := "dsa"'
	expect 1 internal/x/b.go "p := '/dsa/week'"
	expect 1 internal/x/c.ts 'to={`/dsa/week/${n}`}'
	expect 1 internal/x/d.ts 'to="/dsa"'
	expect 1 internal/x/e.ts 'const r = `/dsa${suffix}`'
	expect 1 internal/x/f.go '// see /xlearn/dsa/week/2'
	expect 1 internal/x/g.sql "WHERE path_slug = 'dsa'"
	expect 1 web/src/lib/h.tsx '<Link to="/dsa/dashboard">'
	# Not the slug.
	expect 0 internal/x/i.go 'RubricID = "dsa-mock@1"'
	expect 0 internal/x/j.go '// the DSA course (course.DefaultSlug)'
	expect 0 internal/x/k.ts 'const ICON = { dsa: "code" }'
	expect 0 internal/x/l.go 'x := "dsab"'
	# Allowlisted.
	expect 0 internal/course/default.go 'const DefaultSlug = "dsa"'
	expect 0 internal/x/m_test.go 'x := "dsa"'
	expect 0 internal/x/testdata/n.json '{"slug":"dsa"}'
	expect 0 web/src/lib/defaultCourse.ts 'export const DEFAULT_COURSE = "dsa";'
	if [ "$fail" -ne 0 ]; then
		exit 1
	fi
	echo "lint-course-literals: self-test ok"
}

case "${1:-}" in
--self-test)
	self_test
	exit 0
	;;
-*)
	echo "usage: $0 [--self-test | FILE...]" >&2
	exit 2
	;;
esac

if [ "$#" -eq 0 ]; then
	# shellcheck disable=SC2046 # repo paths have no spaces
	set -- $(files)
fi
if scan "$@"; then
	echo "lint-course-literals: ok ($# files; allowlist in the script header)"
	exit 0
fi
echo "lint-course-literals: the DSA slug is spelled out above; use course.DefaultSlug (Go) or the course from useCourse() (SPA), or allowlist the file with a reason in hack/lint-course-literals.sh" >&2
exit 1
