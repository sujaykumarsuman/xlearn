package goprofile

// execAllow is spk-02's amd64 exec allowlist for the static Go profile, copied verbatim from
// t3 §16.2 (27 names, sorted; 0 unexpected SIGSYS under KILL), plus the justified additions
// marked below. clone and prctl are listed as names; the filter carries their argument rules
// (clone only with CLONE_THREAD and no namespace flag, prctl only PR_SET_VMA).
//
//   - clone: † threads only (the Go runtime); fork dies.
//   - epoll_create1, epoll_ctl, epoll_pwait, eventfd2: ‡ the netpoller behind every timer.
//   - prctl: § PR_SET_VMA only (Go ≥ 1.25 names its anonymous mappings).
//   - close: + m3-04. The generated harness closes fd 4 after writing its frame (m3-02's
//     zz_xl_harness.go, `@1` bytes unchanged); close only drops one of the process's own fds.
//     spk-02's references were driven by a throwaway harness that never closed.
var execAllow = []string{
	"arch_prctl", "clone", "close", "epoll_create1", "epoll_ctl", "epoll_pwait", "eventfd2", "execve",
	"exit_group", "fcntl", "futex", "getpid", "gettid", "madvise", "mmap", "nanosleep", "openat",
	"prctl", "prlimit64", "read", "rt_sigaction", "rt_sigprocmask", "rt_sigreturn",
	"sched_getaffinity", "sched_yield", "sigaltstack", "tgkill", "write",
}

// compileAllow is spk-02's logged amd64 `go build` compile set, copied verbatim from t3 §16.2
// (54 names). Compile filters are ENOSYS-default (a miss fails softly) and KILL the dangerous
// set; the compile jail spawns the compiler and linker (posix_spawn → clone(CLONE_VM |
// CLONE_VFORK)), so clone has no argument rule here. It also covers the runner's own Go
// compile init, with seccomp.CompileInitExtra.
var compileAllow = []string{
	"arch_prctl", "chdir", "clone", "close", "copy_file_range", "dup3", "epoll_create1",
	"epoll_ctl", "epoll_pwait", "eventfd2", "execve", "exit_group", "faccessat2", "fallocate",
	"fchmodat", "fcntl", "flock", "fstat", "ftruncate", "futex", "getcwd", "getdents64", "getpid",
	"gettid", "lseek", "madvise", "mkdirat", "mmap", "munmap", "nanosleep", "newfstatat",
	"openat", "pidfd_open", "pidfd_send_signal", "pipe2", "prctl", "pread64", "prlimit64",
	"pwrite64", "read", "readlinkat", "renameat", "rt_sigaction", "rt_sigprocmask",
	"rt_sigreturn", "sched_getaffinity", "sched_yield", "sigaltstack", "tgkill", "uname",
	"unlinkat", "utimensat", "waitid", "write",
}

// spk02Exec is t3 §16.2's go exec list exactly as published; the invariant test checks that
// execAllow is it plus the recorded additions.
var spk02Exec = []string{
	"arch_prctl", "clone", "epoll_create1", "epoll_ctl", "epoll_pwait", "eventfd2", "execve",
	"exit_group", "fcntl", "futex", "getpid", "gettid", "madvise", "mmap", "nanosleep", "openat",
	"prctl", "prlimit64", "read", "rt_sigaction", "rt_sigprocmask", "rt_sigreturn",
	"sched_getaffinity", "sched_yield", "sigaltstack", "tgkill", "write",
}

// execAdditions are the justified additions to spk02Exec (decisions log, m3-04).
var execAdditions = []string{"close"}
