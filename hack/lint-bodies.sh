#!/bin/sh
# lint-bodies.sh — the L6 request-body gate (sprint m1-05, M1b; ADR-0035 §4).
#
# Every client request body the gateway reads must go through httpx.ReadBody (a typed
# 413 body_too_large past the cap), and a streaming proxy must wrap the body in
# http.MaxBytesReader. A silent io.LimitReader truncation, or an uncapped body handed to an
# upstream, is what L6 removed; this gate keeps it out. It scans the non-test Go files of
#
#   internal/gateway/   and   internal/platform/
#
# (comments stripped: text after a `//` that starts a line or follows a space/tab) and
# fails on
#
#   * LimitReader(r.Body  or  ReadAll(r.Body           (both trees)
#   * http.NewRequest(…r.Body…) / NewRequestWithContext  (internal/gateway only: pass a
#                                                        MaxBytesReader, never r.Body)
#   * json.NewDecoder(r.Body)                            (internal/gateway only)
#
# Upstream RESPONSE reads (resp.Body) keep their LimitReader: they bound replies from our
# own services, not client input. Test files (*_test.go) are not scanned.
#
# Usage:
#   hack/lint-bodies.sh              scan the trees
#   hack/lint-bodies.sh FILE...      scan exactly these files (the tree rules apply by path)
#   hack/lint-bodies.sh --self-test  run hack/testdata/bodies/*: each declares its expected
#                                    exit code on a `// lint-expect: N` line and the path the
#                                    rules see on a `// lint-as: <path>` line
# Exit: 0 clean, 1 a hit, 2 usage. POSIX sh + awk only.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)

# lint_file FILE [AS] — print every hit as FILE:LINE: WHAT; return 1 on any hit. AS is the
# repo path the scope rules use; it defaults to FILE.
lint_file() {
	as=${2:-$1}
	case "$as" in
	*_test.go) return 0 ;;
	internal/gateway/* | */internal/gateway/*) gw=1 ;;
	internal/platform/* | */internal/platform/*) gw=0 ;;
	*) return 0 ;;
	esac
	awk -v file="$1" -v gw="$gw" '
	BEGIN { bad = 0 }
	{
		code = $0
		sub(/(^|[ \t])\/\/.*/, "", code)
		if (code ~ /(LimitReader|ReadAll)\(r\.Body([^A-Za-z0-9_]|$)/) {
			printf "%s:%d: raw request-body read; use httpx.ReadBody (typed 413)\n", file, FNR
			bad = 1
		}
		if (gw && code ~ /http\.NewRequest(WithContext)?\(.*[(, \t]r\.Body([^A-Za-z0-9_]|$)/) {
			printf "%s:%d: raw r.Body passed upstream; wrap it in http.MaxBytesReader\n", file, FNR
			bad = 1
		}
		if (gw && code ~ /json\.NewDecoder\(r\.Body\)/) {
			printf "%s:%d: json.NewDecoder(r.Body) reads an uncapped body; use httpx.ReadBody\n", file, FNR
			bad = 1
		}
	}
	END { exit bad }' "$1"
}

self_test() {
	rc=0
	n=0
	for f in "$root"/hack/testdata/bodies/*; do
		[ -f "$f" ] || continue
		want=$(sed -n 's/^\/\/[[:space:]]*lint-expect:[[:space:]]*\([0-9]\).*/\1/p' "$f" | head -n 1)
		as=$(sed -n 's/^\/\/[[:space:]]*lint-as:[[:space:]]*\([^[:space:]]*\).*/\1/p' "$f" | head -n 1)
		if [ -z "$want" ] || [ -z "$as" ]; then
			echo "self-test: $f needs // lint-expect: N and // lint-as: <path> lines" >&2
			rc=1
			continue
		fi
		set +e
		lint_file "$f" "$as" >/dev/null
		got=$?
		set -e
		n=$((n + 1))
		if [ "$got" != "$want" ]; then
			echo "self-test FAIL: $(basename "$f"): exit $got, want $want" >&2
			rc=1
		else
			echo "self-test ok:   $(basename "$f") (exit $got)"
		fi
	done
	if [ "$n" -eq 0 ]; then
		echo "self-test: no cases under hack/testdata/bodies" >&2
		return 1
	fi
	return $rc
}

case "${1:-}" in
--self-test)
	self_test
	exit $?
	;;
-h | --help)
	sed -n '2,29p' "$0"
	exit 0
	;;
-*)
	echo "lint-bodies: unknown flag $1 (see --help)" >&2
	exit 2
	;;
esac

rc=0
if [ "$#" -gt 0 ]; then
	for f in "$@"; do
		lint_file "$f" || rc=1
	done
	exit $rc
fi

cd "$root"
checked=0
for f in $(find internal/gateway internal/platform -type f -name '*.go' ! -name '*_test.go' | sort); do
	checked=$((checked + 1))
	lint_file "$f" || rc=1
done
if [ "$rc" -eq 0 ]; then
	echo "lint-bodies: $checked file(s) read request bodies through httpx.ReadBody / MaxBytesReader"
else
	echo "lint-bodies: a raw request-body read, proxy or decoder (see above); L6 needs the typed 413" >&2
fi
exit $rc
