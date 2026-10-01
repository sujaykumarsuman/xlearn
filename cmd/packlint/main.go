// Command packlint is the authoring linter that sees both halves of an item: the public
// item in this repo and its private eval pack in the sibling xlearn-evalpack checkout
// (sprint m3-01; t1 §3.3, §3.4, §7.2; ADR-0027 §2, §5). It is a dev tool: it is never
// linked into a service binary or copied into an image.
//
//	packlint check --public . --pack ../xlearn-evalpack [--item <id>]… [--since <ref>] [--json] [--strict]
//	packlint hash  --public . [--item <id>]…
//	packlint fingerprint --pack ../xlearn-evalpack (--pre-push | --diff <range> | --tree <dir>) [--repo .] [--min-len 24]
//	packlint lock      --public . --pack … [--item <id>]… (--write | --verify)                  (m3-02)
//	packlint validate  --public . --pack … [--item <id>]…
//	packlint exec      --public . --pack … [--item <id>]… --gate oracle|wrong|tl|syntax|all [--provisional]
//	packlint exec      --warm-cache [--cache-dir <dir>]
//	packlint build     --public . --pack … --out build/ --version <semver> --validated-against <sha>
//	packlint listing   <image ref | image archive .tar | build directory>
//
// check (the default) applies the nine pack rules (docs/v2/sprints/sprint-m3-01.md task 2):
// exit 0 clean, 1 on an ERROR (or a WARN with --strict), 2 on a usage error. hash prints
// each item's content_hash and contract_hash (the author pastes the contract hash into
// the pack's accepts_contract_hashes). fingerprint is the pre-push leak scan
// (hack/git-hooks/pre-push): exit 1 when outgoing commits carry pack data. The m3-02
// pipeline (pipeline.go) materializes cases (the expected outputs computed from the public
// Go reference), locks, validates, runs the oracle / wrong-solution / time-limit gates,
// builds the data image's root and checks an image against the allowlist; every program
// runs in a network-less container (internal/packspec/executor), never on the host.
//
// Output never carries pack payloads: only ids, paths and hashes. --public takes the repo
// root or any content root holding courses/ (with paths.json and ids.lock.json).
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Exit codes.
const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cmd := "check"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "check":
		return runCheck(args, stdout, stderr)
	case "hash":
		return runHash(args, stdout, stderr)
	case "fingerprint":
		return runFingerprint(args, stdin, stdout, stderr)
	case "lock":
		return runLock(args, stdout, stderr)
	case "validate":
		return runValidate(args, stdout, stderr)
	case "exec":
		return runExec(args, stdout, stderr)
	case "build":
		return runBuild(args, stdout, stderr)
	case "listing":
		return runListing(args, stdout, stderr)
	case "help":
		fmt.Fprint(stdout, usageText)
		return exitOK
	default:
		fmt.Fprintf(stderr, "packlint: unknown command %q\n%s", cmd, usageText)
		return exitUsage
	}
}

const usageText = `usage:
  packlint check --public . --pack ../xlearn-evalpack [--item <id>]... [--since <ref>] [--json] [--strict]
  packlint hash  --public . [--item <id>]...
  packlint fingerprint --pack ../xlearn-evalpack (--pre-push | --diff <range> | --tree <dir>) [--repo .] [--min-len 24]
  packlint lock      --public . --pack ../xlearn-evalpack [--item <id>]... (--write | --verify)
  packlint lock      --tools
  packlint validate  --public . --pack ../xlearn-evalpack [--item <id>]...
  packlint exec      --public . --pack ../xlearn-evalpack [--item <id>]... --gate oracle|wrong|tl|syntax|all [--provisional]
  packlint exec      --warm-cache [--cache-dir <dir>]
  packlint build     --public . --pack ../xlearn-evalpack --out build/ --version <semver> --validated-against <sha> [--dockerfile=false]
  packlint listing   <image ref | image archive .tar | build directory>
`

// multiFlag is a repeatable string flag (--item a --item b).
type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

// defaultPack is the sibling checkout, overridable by XLEARN_EVALPACK_DIR.
func defaultPack() string {
	if d := os.Getenv("XLEARN_EVALPACK_DIR"); d != "" {
		return d
	}
	return "../xlearn-evalpack"
}
