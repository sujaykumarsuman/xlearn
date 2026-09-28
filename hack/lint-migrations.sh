#!/bin/sh
# lint-migrations.sh — the contract-header migration lint (sprint m1-02; ADR-0034 §3).
#
# Every goose migration under internal/*/store/migrations/ that is NOT present at v1.5.2
# (the baseline list hack/migrations-baseline.txt, so CI needs no tag history) is
# classified from its `-- +goose Up` section (the Down section is ignored: it never runs
# in production). Pre-v2 files are exempt.
#
# A file is CONTRACT when a statement line in its Up section (comments stripped, case
# folded) contains any of:
#
#   * DROP <anything> except DROP NOT NULL — DROP COLUMN / TABLE / CONSTRAINT / DEFAULT,
#     a bare `ALTER TABLE t DROP col`, and EVERY DROP INDEX (a grep can't tell whether
#     the index was unique, i.e. whether an old writer relies on it);
#   * SET NOT NULL;
#   * ALTER [COLUMN] <col> [SET DATA] TYPE — EVERY type change (a grep can't tell
#     whether it narrows);
#   * RENAME (an old image still reads the old name);
#
# unless that same line carries `-- xlearn:relax <reason>`: a reviewed relaxation (e.g.
# m1-02's mock CHECK swap, which drops a CHECK only after adding a weaker one). The
# classifier is deliberately conservative; a reviewer marks the safe cases.
#
# Marker grammar (one definition for m1-08, p-02 and ga-01; also in
# docs/architecture/data-model.md, Ownership rules):
#
#   -- xlearn:contract floor=vX.Y.Z   the file's rollback floor (semver; ADR-0034 §3). A
#                                     contract file MUST carry it; a file with no
#                                     unmarked contract statement MUST NOT.
#   -- xlearn:relax <reason>          on a contract statement's own line: reviewed as a
#                                     relaxation, so it doesn't make the file contract.
#                                     The reason is required, and the marker is an error
#                                     on a line with no contract statement.
#
# Usage:
#   hack/lint-migrations.sh              lint every post-baseline migration
#   hack/lint-migrations.sh FILE...      lint exactly these files (baseline ignored)
#   hack/lint-migrations.sh --self-test  run hack/testdata/migrations/*.sql, each
#                                        declaring its expected exit code on a
#                                        `-- lint-expect: N` line
# Exit: 0 clean, 1 a lint failure, 2 usage / missing baseline. POSIX sh + grep/awk only.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
baseline="$root/hack/migrations-baseline.txt"

# lint_file FILE — print findings, return 0 (clean) or 1 (failure).
lint_file() {
	awk -v file="$1" '
	function trim(s) { sub(/^[[:space:]]+/, "", s); sub(/[[:space:]]+$/, "", s); return s }
	function fail(msg) { printf "%s:%d: %s\n", file, NR, msg; bad = 1 }
	BEGIN { section = "head"; contract = 0; marker = 0; bad = 0 }
	/^--[[:space:]]*\+goose[[:space:]]+Up/   { section = "up"; next }
	/^--[[:space:]]*\+goose[[:space:]]+Down/ { section = "down"; next }
	{
		raw = $0
		# The contract marker may sit anywhere outside the Down section (the header).
		if (section != "down" && raw ~ /--[[:space:]]*xlearn:contract/) {
			if (raw ~ /--[[:space:]]*xlearn:contract[[:space:]]+floor=v[0-9]+\.[0-9]+\.[0-9]+([[:space:]]|$)/) {
				marker++
				markerLine = NR
			} else {
				fail("malformed xlearn:contract marker (want: -- xlearn:contract floor=vX.Y.Z)")
			}
		}
		if (section != "up") next

		relax = 0
		if (raw ~ /--[[:space:]]*xlearn:relax/) {
			reason = raw
			sub(/.*--[[:space:]]*xlearn:relax/, "", reason)
			if (trim(reason) == "") fail("xlearn:relax needs a reason (-- xlearn:relax <reason>)")
			relax = 1
		}

		code = raw
		sub(/--.*/, "", code)          # strip the comment (markers included)
		code = " " toupper(code) " "
		gsub(/[[:space:]]+/, " ", code)

		hit = ""
		tmp = code
		gsub(/[^A-Z0-9_]DROP NOT NULL[^A-Z0-9_]/, " ", tmp)
		if (tmp ~ /[^A-Z0-9_]DROP[^A-Z0-9_]/) hit = "DROP"
		else if (code ~ /[^A-Z0-9_]SET NOT NULL[^A-Z0-9_]/) hit = "SET NOT NULL"
		else if (code ~ /[^A-Z0-9_]ALTER (COLUMN )?[A-Z0-9_"]+ (SET DATA )?TYPE[^A-Z0-9_]/) hit = "ALTER COLUMN ... TYPE"
		else if (code ~ /[^A-Z0-9_]RENAME[^A-Z0-9_]/) hit = "RENAME"

		if (hit != "" && !relax) {
			contract++
			if (contract == 1) firstHit = sprintf("line %d (%s)", NR, hit)
		}
		if (hit == "" && relax) fail("xlearn:relax on a line with no contract statement")
	}
	END {
		if (contract > 0 && marker == 0)
			printf "%s: contract statement at %s without a -- xlearn:contract floor=vX.Y.Z marker (or -- xlearn:relax <reason> on that line)\n", file, firstHit
		if (contract == 0 && marker > 0)
			printf "%s:%d: -- xlearn:contract marker on a file with no contract statement (expand only)\n", file, markerLine
		if (marker > 1)
			printf "%s: more than one xlearn:contract marker\n", file
		exit (bad || (contract > 0 && marker == 0) || (contract == 0 && marker > 0) || marker > 1) ? 1 : 0
	}' "$1"
}

self_test() {
	rc=0
	n=0
	for f in "$root"/hack/testdata/migrations/*.sql; do
		want=$(sed -n 's/^--[[:space:]]*lint-expect:[[:space:]]*\([0-9]\).*/\1/p' "$f" | head -n 1)
		if [ -z "$want" ]; then
			echo "self-test: $f has no -- lint-expect: N line" >&2
			rc=1
			continue
		fi
		set +e
		lint_file "$f" >/dev/null
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
		echo "self-test: no cases under hack/testdata/migrations" >&2
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
	sed -n '2,45p' "$0"
	exit 0
	;;
esac

rc=0
if [ "$#" -gt 0 ]; then
	for f in "$@"; do
		lint_file "$f" || rc=1
	done
	exit $rc
fi

if [ ! -f "$baseline" ]; then
	echo "lint-migrations: missing $baseline" >&2
	exit 2
fi
cd "$root"
checked=0
for f in internal/*/store/migrations/*.sql; do
	[ -e "$f" ] || continue
	if grep -qxF "$f" "$baseline"; then
		continue
	fi
	checked=$((checked + 1))
	lint_file "$f" || rc=1
done
if [ "$rc" -eq 0 ]; then
	echo "lint-migrations: $checked post-v1.5.2 migration(s) clean"
fi
exit $rc
