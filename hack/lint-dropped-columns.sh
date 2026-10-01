#!/bin/sh
# lint-dropped-columns.sh — the M1c drop-list query gate (sprint m1-03, M1b; ADR-0034 §3).
#
# m1-08's contract migration drops the v1 columns M1b moved every reader and writer
# off, and the v1 weak-area unique. Its rollback floor is the release that stopped
# touching them (v1.7.0), so from m1-03 on no query may read or write one; m1-08 reruns
# this script against the v1.7.0 tree to prove it (keep it self-contained). It scans
#
#   internal/*/store/queries/*.sql     the hand-written queries, and
#   internal/*/store/gen/*.sql.go      the SQL sqlc generated from them (only the text
#                                      inside Go raw strings): sqlc expands `SELECT *` /
#                                      `RETURNING *` to every column, so a star query on
#                                      a table that still has a drop-list column reads it
#                                      without naming it — only the generated SQL shows it
#
# with `--` comments stripped, and fails on
#
#   * a drop-list column as a whole word, case-insensitive:
#       total_35  is_reinforcement  leetcode_url  neetcode_url  code_template  is_default
#   * the retired conflict target ON CONFLICT (account_id, week_of), case- and
#     whitespace-tolerant (newlines between the tokens included).
#
# No allowlist. m1-03 exempted is_default in internal/coach/store/{queries,gen} because
# coach still dual-wrote it then; m1-10 moved coach entirely onto coach.key_default, so
# the gate now covers every service uniformly and a reintroduced reader fails CI wherever
# it appears. (internal/coach/contract_test.go greps the Go source for the same thing, so
# both the SQL and the code paths are covered.) Migrations are not scanned (history:
# never edited).
#
# Usage:
#   hack/lint-dropped-columns.sh              scan the tree
#   hack/lint-dropped-columns.sh FILE...      scan exactly these files (a .sql, or a
#                                             sqlc .sql.go; the allowlist applies)
#   hack/lint-dropped-columns.sh --self-test  run hack/testdata/dropped-columns/*: each
#                                             declares its expected exit code on a
#                                             `-- lint-expect: N` (or `// lint-expect: N`)
#                                             line, and may set the path the allowlist and
#                                             the scan mode see with `lint-as: <path>`
# Exit: 0 clean, 1 a hit, 2 usage. POSIX sh + awk only.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)

# lint_file FILE [AS] — print every hit as FILE:LINE: HIT (and allowed hits as notes);
# return 1 on any hit. AS is the repo path the allowlist and the scan mode (a .go path
# scans only Go raw strings) use; it defaults to FILE.
lint_file() {
	awk -v file="$1" -v as="${2:-$1}" '
	BEGIN {
		ncols = split("total_35 is_reinforcement leetcode_url neetcode_url code_template is_default", cols, " ")
		gomode = (as ~ /\.go$/)
		inraw = 0; buf = ""; bad = 0
	}
	{
		line = $0
		if (gomode) {
			# Keep only the text inside Go raw strings (sqlc emits each query as a
			# backtick const); a raw string cannot contain a backtick, so each one
			# toggles in or out.
			code = ""; rest = line
			while ((i = index(rest, "`")) > 0) {
				if (inraw) code = code substr(rest, 1, i - 1)
				code = code " "
				inraw = !inraw
				rest = substr(rest, i + 1)
			}
			if (inraw) code = code rest
			line = code
		}
		sub(/--.*/, "", line)
		line = tolower(line)
		for (c = 1; c <= ncols; c++) {
			if (line ~ ("(^|[^a-z0-9_])" cols[c] "([^a-z0-9_]|$)")) {
				printf "%s:%d: %s\n", file, FNR, cols[c]
				bad = 1
			}
		}
		# Join the lines with a space so the conflict target is found across them;
		# lineAt[n] is where line n starts in buf.
		lineAt[FNR] = length(buf) + 1
		buf = buf line " "
	}
	END {
		target = "(^|[^a-z0-9_])on[[:space:]]+conflict[[:space:]]*\\([[:space:]]*\"?account_id\"?[[:space:]]*,[[:space:]]*\"?week_of\"?[[:space:]]*\\)"
		s = buf; off = 0
		while (match(s, target)) {
			# Report the line of "on" itself, not of the separator matched before it.
			pos = off + RSTART + index(substr(s, RSTART, RLENGTH), "on") - 1
			n = 1
			while ((n + 1) in lineAt && lineAt[n + 1] <= pos) n++
			printf "%s:%d: ON CONFLICT (account_id, week_of)\n", file, n
			bad = 1
			off += RSTART + RLENGTH - 1
			s = substr(s, RSTART + RLENGTH)
		}
		exit bad
	}' "$1"
}

self_test() {
	rc=0
	n=0
	for f in "$root"/hack/testdata/dropped-columns/*; do
		[ -f "$f" ] || continue
		want=$(sed -n 's/^[-\/]*[[:space:]]*lint-expect:[[:space:]]*\([0-9]\).*/\1/p' "$f" | head -n 1)
		as=$(sed -n 's/^[-\/]*[[:space:]]*lint-as:[[:space:]]*\([^[:space:]]*\).*/\1/p' "$f" | head -n 1)
		if [ -z "$want" ]; then
			echo "self-test: $f has no lint-expect: N line" >&2
			rc=1
			continue
		fi
		set +e
		lint_file "$f" "${as:-$f}" >/dev/null
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
		echo "self-test: no cases under hack/testdata/dropped-columns" >&2
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
	sed -n '2,36p' "$0"
	exit 0
	;;
-*)
	echo "lint-dropped-columns: unknown flag $1 (see --help)" >&2
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
for f in internal/*/store/queries/*.sql internal/*/store/gen/*.sql.go; do
	[ -e "$f" ] || continue
	checked=$((checked + 1))
	lint_file "$f" || rc=1
done
if [ "$rc" -eq 0 ]; then
	echo "lint-dropped-columns: $checked query file(s) clean of the M1c drop list"
else
	echo "lint-dropped-columns: a query reads or writes an M1c drop-list column or the v1 weak-area conflict target (see above)" >&2
fi
exit $rc
