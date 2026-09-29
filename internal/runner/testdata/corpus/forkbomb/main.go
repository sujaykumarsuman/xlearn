// forkbomb re-execs itself until the case's pids.max refuses (RE, fork limit). Under the go
// allowlist the first fork is already SIGSYS; this runs under testgo-open@0 to prove the pids
// cap on its own.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" {
		time.Sleep(time.Hour)
		return
	}
	failed := 0
	for i := 0; i < 1000 && failed < 20; i++ {
		if err := exec.Command("/job/bin", "child").Start(); err != nil {
			failed++
		}
	}
	fmt.Println("fork failures:", failed)
	os.Exit(3)
}
