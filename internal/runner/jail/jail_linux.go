//go:build linux

// Package jail starts one per-case (or compile) jail with go-sandbox's low-level
// forkexec.Runner (ADR-0030 §1, t3 §16.1): fresh mnt/pid/net/ipc/uts/cgroup namespaces and
// never a user namespace; a fresh tmpfs root and pivot_root; read-only nosuid,nodev binds; a
// per-case /w tmpfs; no /proc unless asked; the job's UID; 0 capabilities, an empty bounding
// set and NO_NEW_PRIVS; RLIMIT_CORE=0 and the profile's rlimits; the seccomp filter last.
//
// Only this package imports go-sandbox, and only pkg/forkexec, pkg/mount and pkg/rlimit;
// never go-sandbox/container, which forces a user namespace and keeps ambient SYS_ADMIN
// (t3 §4; guarded by internal/runner/imports_test.go).
package jail

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/criyle/go-sandbox/pkg/forkexec"
	"github.com/criyle/go-sandbox/pkg/mount"
	"github.com/criyle/go-sandbox/pkg/rlimit"
	"golang.org/x/sys/unix"
)

// Bind is a read-only bind of Source at Target (absolute, inside the jail).
type Bind struct {
	Source string
	Target string
}

// Spec is one jail.
type Spec struct {
	Argv    []string
	Env     []string
	WorkDir string
	UID     int
	// RootDir is an existing empty directory under the jail dir; the child mounts a fresh
	// tmpfs on it inside its own mount namespace and pivots into it.
	RootDir string
	Binds   []Bind
	// WTmpfs is the /w tmpfs's options, e.g. "size=64m,nr_inodes=4096,mode=0700".
	WTmpfs string
	// WExec leaves /w executable (never for learner code; the compile tools don't need it).
	WExec bool
	// Proc mounts procfs with hidepid=invisible,subset=pid.
	Proc bool
	// Rlimits in addition to RLIMIT_CORE=0 (always set).
	RLimits []rlimit.RLimit
	Seccomp *syscall.SockFprog
	// Files become the child's fds 0..len-1.
	Files []uintptr
	// CgroupFd (CLONE_INTO_CGROUP) or CgroupProcs (cgroup.procs + sync) places the child in its
	// case cgroup before it execs. Exactly one is set.
	CgroupFd    int
	CgroupProcs string
}

// flags every jail gets. Never CLONE_NEWUSER (t3 §2.3 B0; ADR-0030).
const cloneFlags = unix.CLONE_NEWNS | unix.CLONE_NEWPID | unix.CLONE_NEWNET | unix.CLONE_NEWIPC | unix.CLONE_NEWUTS

// roBind is the read-only bind shape AppArmor's remount rule matches (t3 §16.2 block 3): no
// MS_PRIVATE (the jail's mount namespace root is already rprivate), never go-sandbox's
// WithBind(…, true).
const roBind = unix.MS_BIND | unix.MS_REC | unix.MS_NOSUID | unix.MS_NODEV | unix.MS_RDONLY

// Start clones the jail and returns the child's pid once it has exec'd.
func Start(s Spec) (int, error) {
	if len(s.Argv) == 0 || s.RootDir == "" || s.UID <= 0 {
		return 0, fmt.Errorf("jail: incomplete spec")
	}
	if (s.CgroupFd > 0) == (s.CgroupProcs != "") {
		return 0, fmt.Errorf("jail: set exactly one of CgroupFd and CgroupProcs")
	}
	mounts, err := buildMounts(s)
	if err != nil {
		return 0, err
	}
	r := &forkexec.Runner{
		Args:       s.Argv,
		Env:        s.Env,
		Files:      s.Files,
		WorkDir:    s.WorkDir,
		RLimits:    append([]rlimit.RLimit{{Res: unix.RLIMIT_CORE, Rlim: syscall.Rlimit{}}}, s.RLimits...),
		Seccomp:    s.Seccomp,
		CloneFlags: cloneFlags,
		Mounts:     mounts,
		PivotRoot:  s.RootDir,
		HostName:   "xl",
		DomainName: "xl",
		Credential: &syscall.Credential{Uid: uint32(s.UID), Gid: uint32(s.UID), Groups: []uint32{}},
		DropCaps:   true,
		NoNewPrivs: true,
	}
	if s.CgroupFd > 0 {
		r.CgroupFd = uintptr(s.CgroupFd)
		r.CloneFlags |= unix.CLONE_NEWCGROUP
	} else {
		procs := s.CgroupProcs
		r.SyncFunc = func(pid int) error {
			return os.WriteFile(procs, []byte(fmt.Sprint(pid)), 0)
		}
		// Enter the cgroup first, then unshare the cgroup namespace, then drop caps and load
		// seccomp (go-sandbox orders those after the sync when this is set).
		r.UnshareCgroupAfterSync = true
	}
	return StartEmptyBounding(r)
}

func buildMounts(s Spec) ([]mount.SyscallParams, error) {
	b := mount.NewBuilder()
	for _, bd := range s.Binds {
		if !filepath.IsAbs(bd.Source) || !filepath.IsAbs(bd.Target) || strings.Contains(bd.Target, "..") {
			return nil, fmt.Errorf("jail: bind %q → %q must be absolute", bd.Source, bd.Target)
		}
		b.WithMount(mount.Mount{Source: bd.Source, Target: strings.TrimPrefix(bd.Target, "/"), Flags: roBind})
	}
	// /dev/null is the one device a jail gets (the go command and os/exec open it). It is a
	// read-write bind shaped like go-sandbox's WithBind(…, false), which AppArmor's
	// `mount options=(rw, rbind, nosuid, rprivate) -> /jail/**` admits; nodev would make it
	// unopenable. The jail can't create device nodes (no CAP_MKNOD, and seccomp).
	b.WithMount(mount.Mount{Source: "/dev/null", Target: "dev/null", Flags: unix.MS_BIND | unix.MS_REC | unix.MS_NOSUID | unix.MS_PRIVATE})
	wflags := uintptr(unix.MS_NOSUID | unix.MS_NODEV)
	if !s.WExec {
		wflags |= unix.MS_NOEXEC
	}
	b.WithMount(mount.Mount{Source: "tmpfs", Target: "w", FsType: "tmpfs", Flags: wflags, Data: s.WTmpfs})
	if s.Proc {
		b.WithMount(mount.Mount{Source: "proc", Target: "proc", FsType: "proc",
			Flags: unix.MS_NOSUID | unix.MS_NODEV | unix.MS_NOEXEC, Data: "hidepid=invisible,subset=pid"})
	}
	return b.Build()
}

// StartEmptyBounding starts r from a locked OS thread whose capability bounding set was
// emptied first, so the child (and everything it execs) inherits an empty bounding set. The
// thread is never unlocked: it exits with its goroutine, so no other goroutine ever runs on a
// thread with a changed bounding set. Emptying the bounding set needs CAP_SETPCAP, as does
// go-sandbox's PR_SET_SECUREBITS (keep-caps across setuid, then NOROOT before capset).
func StartEmptyBounding(r *forkexec.Runner) (int, error) {
	type result struct {
		pid int
		err error
	}
	ch := make(chan result, 1)
	go func() {
		runtime.LockOSThread()
		// No UnlockOSThread: see above.
		if err := DropBoundingSet(); err != nil {
			ch <- result{0, err}
			return
		}
		pid, err := r.Start()
		ch <- result{pid, err}
	}()
	res := <-ch
	return res.pid, res.err
}

// DropBoundingSet empties the calling thread's capability bounding set.
func DropBoundingSet() error {
	for c := 0; c <= 63; c++ {
		err := unix.Prctl(unix.PR_CAPBSET_DROP, uintptr(c), 0, 0, 0)
		if errors.Is(err, unix.EINVAL) {
			return nil // past the kernel's last capability
		}
		if err != nil {
			return fmt.Errorf("jail: drop cap %d from the bounding set (needs CAP_SETPCAP): %w", c, err)
		}
	}
	return nil
}

// ChildErrorLocation extracts go-sandbox's failure location ("clone", "pivot_root", …) for
// logs and for the readiness probe that expects clone(CLONE_NEWUSER) to be refused.
func ChildErrorLocation(err error) (string, bool) {
	var ce forkexec.ChildError
	if errors.As(err, &ce) {
		return ce.Location.String(), true
	}
	return "", false
}

// ProbeUserNS tries clone(CLONE_NEWUSER) from this (privileged) process: the readiness canary
// wants it refused (AppArmor's userns restriction; t3 §5.3). It returns true when the clone was
// refused. A child that did get created is killed and reaped.
func ProbeUserNS(exe string, devnull uintptr) bool {
	r := &forkexec.Runner{
		Args:       []string{exe, "-version"},
		Files:      []uintptr{devnull, devnull, devnull},
		CloneFlags: unix.CLONE_NEWUSER,
		DropCaps:   true,
		NoNewPrivs: true,
	}
	pid, err := r.Start()
	if err != nil {
		loc, ok := ChildErrorLocation(err)
		return ok && loc == forkexec.LocClone.String()
	}
	_ = unix.Kill(pid, unix.SIGKILL)
	var ws unix.WaitStatus
	_, _ = unix.Wait4(pid, &ws, 0, nil)
	return false
}
