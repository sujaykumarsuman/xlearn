// Command runner is xlearn-runner, the sandbox that runs learner code for judge (ADR-0030,
// t3 §5; docs/architecture/runner.md). One binary, several roles:
//
//	runner                 the spawner: PID 1 in the pod, holds the namespaced capabilities,
//	                       builds cgroups, mounts and per-case jails; never parses a learner byte
//	runner front           the capless front (re-exec'd by the spawner): HTTP on :8090, bearer
//	                       auth, all decoding, Result assembly
//	runner canary          the throttling canary (s2), run capless by the spawner
//	runner compile-init    the compile jail's init and in-jail artifact exporter
//	runner seed-gocache    build the go@1.26 GOCACHE seed (m3-15's image build): -out <dir>
//	                       (required, absent or empty) [-goroot <dir>]; prints its tree hash
//	runner -version        print the build version
//
// The language profiles (go@1.26, cpp@g++14, python@3.13) register from their packages.
// Judge is the only caller. The runner has no database, no NATS, no egress and no secret but
// its bearer token.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sujaykumarsuman/xlearn/internal/runner/canary"
	"github.com/sujaykumarsuman/xlearn/internal/runner/front"
	"github.com/sujaykumarsuman/xlearn/internal/runner/jail"
	_ "github.com/sujaykumarsuman/xlearn/internal/runner/profile/cpp"
	goprofile "github.com/sujaykumarsuman/xlearn/internal/runner/profile/go"
	_ "github.com/sujaykumarsuman/xlearn/internal/runner/profile/python"
	"github.com/sujaykumarsuman/xlearn/internal/runner/spawner"
)

// version is stamped with -ldflags "-X main.version=runner-vX.Y.Z" (m3-15's Dockerfile, guarded
// by deploy/version_test.go once it exists). It must be the main-package path: -X on a missing
// symbol is a silent no-op.
var version = "dev"

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "-version", "--version":
			fmt.Println(version)
			return
		case "front":
			os.Exit(front.Main())
		case "canary":
			if err := canary.Run(os.Stdout); err != nil {
				os.Exit(1)
			}
			return
		case "compile-init":
			os.Exit(jail.CompileInit(os.Args[2:]))
		case "seed-gocache":
			os.Exit(seedGocache(os.Args[2:], os.Stdout, os.Stderr))
		default:
			fmt.Fprintf(os.Stderr, "runner: unknown subcommand %q\n", os.Args[1])
			os.Exit(2)
		}
	}
	os.Exit(spawner.Main(version))
}

// seedGocache is `runner seed-gocache -out <dir> [-goroot <dir>]`: the go@1.26 GOCACHE seed
// recipe (goprofile.BuildSeed: `go build std` with the profile's exact toolchain, flags and env,
// fixed future mtimes). It prints the seed's tree hash and exits non-zero, leaving no partial
// seed, on any failure.
func seedGocache(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("seed-gocache", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "the seed directory (required; must be absent or empty)")
	goroot := fs.String("goroot", goprofile.GOROOT(), "the Go toolchain (default: the go@1.26 profile's)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *out == "" || fs.NArg() > 0 {
		fmt.Fprintln(stderr, "usage: runner seed-gocache -out <dir> [-goroot <dir>]")
		return 2
	}
	dir, err := filepath.Abs(*out)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(dir), 0o755)
	}
	if err != nil {
		fmt.Fprintln(stderr, "seed-gocache:", err)
		return 1
	}
	hash, err := goprofile.BuildSeed(*goroot, dir)
	if err != nil {
		fmt.Fprintln(stderr, "seed-gocache:", err)
		return 1
	}
	fmt.Fprintln(stdout, hash)
	return 0
}
