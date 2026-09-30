// orphan double-forks a sleeping daemon (setsid) and exits 0: after teardown no process of the
// job may survive (the jail's pid namespace and cgroup.kill both see to it).
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}
	switch arg {
	case "grandchild":
		time.Sleep(time.Hour)
	case "child":
		cmd := exec.Command("/job/bin", "grandchild")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			fmt.Println("grandchild:", err)
			os.Exit(3)
		}
		fmt.Println("daemon", cmd.Process.Pid)
	default:
		cmd := exec.Command("/job/bin", "child")
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			fmt.Println("child:", err)
			os.Exit(3)
		}
		fmt.Println("parent done")
	}
}
