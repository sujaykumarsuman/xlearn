// Command contentlint runs the public content checks over curriculum/ (sprints m1-09 and
// m3-01; t1 §7.2). It is a dev and CI tool: it is never linked into a service binary (its
// Markdown parser and JSON Schema validator stay out of every image; a test checks it).
//
//	go run ./cmd/contentlint                     # all checks (make contentlint)
//	go run ./cmd/contentlint -report-unstamped   # also list grandfathered unstamped sections
//	go run ./cmd/contentlint -write-allowlist    # refresh curriculum/embed.allowlist
//
// Checks:
//  1. schema + strict decode: every JSON file under curriculum/ validates against its
//     _schema/ file, and the whole tree loads with the service's own glob loader
//     (strict decoding into the Go types, item validation, sidecars);
//  2. id / slug guards: every item directory is in ids.lock.json with its course and
//     status, every lock id has an item directory, course slugs pass the guard, and
//     against the previous release tag no id disappears or changes course and no
//     asset id@v changes bytes (skipped until a tag carries the lock);
//  3. the t4 §5.6 structure lints #1–#8, #12–#14 (internal/course/lint);
//  4. against the PR base (-base, CONTENTLINT_BASE, else the merge base with origin/main):
//     the stamp gate (an added or changed hint or editorial file needs the item's
//     review.hints / review.editorial stamp; unchanged v1 sections are grandfathered) and
//     the label-edit flag (a changed option/field label under the same id on a
//     key-graded part or probe fails unless the PR body, CONTENTLINT_PR_BODY, carries
//     `label-edit-ok: <item>/<part>/<id>`);
//  5. the Markdown profile for every .md under curriculum/courses/ (goldmark + GFM): no
//     raw HTML, images only as asset: refs, links https:// only; an SVG lint;
//  6. filename rules: the private-looking denylist (pack file names included), and every
//     *.go under curriculum/ is a complete, gofmt-clean Go file (fragments are *.snip);
//  7. the repo-wide pack-artefact pass over git ls-files: tests.lock, cases.jsonl* and
//     *.jsonl.zst fail anywhere except inside an internal/**/testdata/ tree marked by a
//     SYNTHETIC.md;
//  8. the embedded-file allowlists: every package with a //go:embed has an
//     embed.allowlist and embeds nothing it does not list; curriculum/embed.allowlist is
//     the exact list.
//
// The seeded-row snapshot test (TestSeedMatchesV1Snapshot) runs in the CI content job
// beside it, against Postgres.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	root := flag.String("root", "curriculum", "the curriculum/ directory (the embed package)")
	prevTag := flag.String("prev-tag", "", "release tag to compare ids.lock.json against (default: the latest v* tag reachable from HEAD)")
	base := flag.String("base", os.Getenv("CONTENTLINT_BASE"), "PR base commit for the stamp gate and the label-edit flag (default $CONTENTLINT_BASE, else the merge base with origin/main)")
	reportUnstamped := flag.Bool("report-unstamped", false, "list the items whose existing hint/editorial sections are unstamped (grandfathered)")
	writeAllowlist := flag.Bool("write-allowlist", false, "rewrite curriculum/embed.allowlist from the current embed and exit")
	flag.Parse()

	l := &linter{
		root: *root, prevTag: *prevTag, out: os.Stdout,
		base: *base, prBody: os.Getenv("CONTENTLINT_PR_BODY"), reportUnstamped: *reportUnstamped,
	}
	if *writeAllowlist {
		n, err := l.writeAllowlist()
		if err != nil {
			fmt.Fprintln(os.Stderr, "contentlint:", err)
			os.Exit(1)
		}
		fmt.Printf("contentlint: wrote %d embedded files to %s\n", n, l.allowlistPath())
		return
	}
	if !l.run() {
		os.Exit(1)
	}
}
