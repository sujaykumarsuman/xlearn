package cpp

// execAllow is spk-02's amd64 exec allowlist for `g++ -std=gnu++20 -O2 -static` programs,
// copied verbatim from t3 §16.2 (19 names, sorted; 0 unexpected SIGSYS under KILL), plus the
// justified additions marked below. There is no clone: a static single-threaded program
// never spawns.
//
//   - getpid, gettid, rt_sigaction, rt_sigprocmask, tgkill: + m3-04. abort() (a failed assert,
//     std::terminate) resets the SIGABRT handler, unblocks it, reads its pid and tid and
//     tgkill()s itself; as its pid namespace's init the case ignores that signal, so abort ends
//     in its trap instruction (a signal → RE). Without these a failed assert dies of SIGSYS,
//     which the runner counts as a security event and rotates on. spk-02's references never
//     aborted. All five touch only the process itself (the jail is its own pid namespace); the
//     go list has the same five.
var execAllow = []string{
	"arch_prctl", "brk", "execve", "exit_group", "fstat", "futex", "getpid", "getrandom", "gettid",
	"lseek", "mmap", "mprotect", "munmap", "prlimit64", "read", "readlinkat", "rseq",
	"rt_sigaction", "rt_sigprocmask", "set_robust_list", "set_tid_address", "tgkill", "write", "writev",
}

// compileAllow is spk-02's logged amd64 `g++ -static` compile set, copied verbatim from t3
// §16.2 (38 names). The profile unions it with the go build set for the runner's compile init.
var compileAllow = []string{
	"access", "arch_prctl", "brk", "chmod", "clone", "close", "dup", "execve", "exit_group",
	"faccessat2", "fcntl", "fstat", "futex", "getcwd", "getrandom", "getrusage", "ioctl", "lseek",
	"mmap", "mprotect", "mremap", "munmap", "newfstatat", "openat", "pread64", "prlimit64", "read",
	"readlink", "rseq", "rt_sigaction", "rt_sigprocmask", "set_robust_list", "set_tid_address",
	"sysinfo", "umask", "unlink", "wait4", "write",
}

// spk02Exec is t3 §16.2's cpp exec list exactly as published.
var spk02Exec = []string{
	"arch_prctl", "brk", "execve", "exit_group", "fstat", "futex", "getrandom", "lseek", "mmap",
	"mprotect", "munmap", "prlimit64", "read", "readlinkat", "rseq", "set_robust_list",
	"set_tid_address", "write", "writev",
}

// execAdditions are the justified additions to spk02Exec (decisions log, m3-04).
var execAdditions = []string{"getpid", "gettid", "rt_sigaction", "rt_sigprocmask", "tgkill"}
