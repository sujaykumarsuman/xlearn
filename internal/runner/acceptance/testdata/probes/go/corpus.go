package main

// Section D (t3 §9's P2 corpus), Go, plus the no-op and spin programs sections A and F use. arg
// names the program; the suite checks the runner's terminal state for each.

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func probe(arg string) string {
	name, param, _ := strings.Cut(arg, ":")
	switch name {
	case "noop":
		return ""
	case "spin": // spin:<ms> burns CPU for ms milliseconds (0: forever, a CPU TLE)
		ms, _ := strconv.Atoi(param)
		deadline := time.Now().Add(time.Duration(ms) * time.Millisecond)
		x := 0
		for ms == 0 || time.Now().Before(deadline) {
			for i := 0; i < 100000; i++ {
				x += i ^ x
			}
		}
		return strconv.Itoa(x & 1)
	case "sleep": // idle: TLE (idle)
		time.Sleep(time.Hour)
		return "woke"
	case "balloon": // 1 GiB in 16 MiB chunks, every page touched: MLE in the case cgroup
		var keep [][]byte
		for n := 0; n < 1<<30; n += 16 << 20 {
			b := make([]byte, 16<<20)
			for i := 0; i < len(b); i += 4096 {
				b[i] = 1
			}
			keep = append(keep, b)
		}
		return fmt.Sprint("survived ", len(keep))
	case "tmpfsfill": // 512 KiB files into /w until the tmpfs is full (RLIMIT_FSIZE is 1 MiB)
		buf := make([]byte, 512<<10)
		for i := 0; ; i++ {
			f, err := os.Create(fmt.Sprintf("/w/f%d", i))
			if err != nil {
				return "enospc: " + err.Error()
			}
			_, err = f.Write(buf)
			f.Close()
			if err != nil {
				return "enospc: " + err.Error()
			}
		}
	case "inodefill": // empty files into /w until the tmpfs runs out of inodes
		for i := 0; ; i++ {
			f, err := os.Create(fmt.Sprintf("/w/i%d", i))
			if err != nil {
				return fmt.Sprintf("enospc after %d: %v", i, err)
			}
			f.Close()
		}
	case "stdoutflood": // past the 1 MiB fd cap: OLE
		buf := []byte(strings.Repeat("x", 64<<10))
		for {
			if _, err := os.Stdout.Write(buf); err != nil {
				return "write: " + err.Error()
			}
		}
	case "threadbomb": // OS threads until pids.max: the runtime dies (RE, fork limit)
		for i := 0; i < 5000; i++ {
			go func() {
				runtime.LockOSThread()
				time.Sleep(time.Hour)
			}()
		}
		time.Sleep(time.Hour)
		return "survived"
	case "forkbomb": // fork until pids.max refuses (the exec filter kills the first fork: SIGSYS)
		failed := 0
		for i := 0; i < 1000 && failed < 20; i++ {
			if err := exec.Command("/job/bin").Start(); err != nil {
				failed++
			}
		}
		os.Exit(3)
	case "orphan": // a double-forked daemon (the exec filter kills the first fork: SIGSYS)
		cmd := exec.Command("/job/bin")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			return "fork: " + err.Error()
		}
		return "parent done"
	}
	return "bad probe " + arg
}
