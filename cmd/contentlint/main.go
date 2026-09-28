// Command contentlint runs the public content checks over curriculum/ (sprint m1-09;
// t1 §7.2). It is a dev and CI tool: it is never linked into a service binary (its
// Markdown parser and JSON Schema validator stay out of every image; a test checks it).
//
//	go run ./cmd/contentlint                    # all checks (make contentlint)
//	go run ./cmd/contentlint -write-allowlist   # refresh curriculum/embed.allowlist
//
// Checks:
//  1. schema + strict decode: every JSON file under curriculum/ validates against its
//     _schema/ file, and the whole tree loads with the service's own glob loader
//     (strict decoding into the Go types, item validation, sidecars);
//  2. id / slug guards: every item directory is in ids.lock.json with its course and
//     status, every lock id has an item directory, course slugs pass the guard, and
//     against the previous release tag no id disappears or changes course and no
//     asset id@v changes bytes (skipped until a tag carries the lock);
//  3. the Markdown profile for every .md under curriculum/courses/ (goldmark + GFM): no
//     raw HTML, images only as asset: refs, links https:// only; an SVG lint;
//  4. filename rules: the private-looking denylist, and every *.go under curriculum/ is
//     a complete, gofmt-clean Go file (fragments are *.snip);
//  5. the embedded-file allowlist: `go list -json ./curriculum` EmbedFiles must equal
//     curriculum/embed.allowlist.
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
	writeAllowlist := flag.Bool("write-allowlist", false, "rewrite curriculum/embed.allowlist from the current embed and exit")
	flag.Parse()

	l := &linter{root: *root, prevTag: *prevTag, out: os.Stdout}
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
