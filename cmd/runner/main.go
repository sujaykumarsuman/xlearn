// Command runner is xlearn-runner, the sandbox that runs learner code for judge (ADR-0030,
// t3 §5; docs/architecture/runner.md). One binary, several roles:
//
//	runner                 the spawner: PID 1 in the pod, holds the namespaced capabilities,
//	                       builds cgroups, mounts and per-case jails; never parses a learner byte
//	runner front           the capless front (re-exec'd by the spawner): HTTP on :8090, bearer
//	                       auth, all decoding, Result assembly
//	runner canary          the throttling canary (s2), run capless by the spawner
//	runner compile-init    the compile jail's init and in-jail artifact exporter
//	runner -version        print the build version
//
// Judge is the only caller. The runner has no database, no NATS, no egress and no secret but
// its bearer token.
package main

import (
	"fmt"
	"os"

	"github.com/sujaykumarsuman/xlearn/internal/runner/canary"
	"github.com/sujaykumarsuman/xlearn/internal/runner/front"
	"github.com/sujaykumarsuman/xlearn/internal/runner/jail"
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
		default:
			fmt.Fprintf(os.Stderr, "runner: unknown subcommand %q\n", os.Args[1])
			os.Exit(2)
		}
	}
	os.Exit(spawner.Main(version))
}
