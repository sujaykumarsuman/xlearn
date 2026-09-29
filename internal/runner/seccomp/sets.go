package seccomp

// Dangerous is the set every runner filter kills (t3 §2.4 A1, m3-04 task 5): io_uring, bpf,
// perf, userfaultfd, the keyring, ptrace and process_vm_*, the whole mount family including the
// new mount API, unshare/setns, socket, and the pipe-splicing family. Namespace flags on clone
// are handled by Policy.CloneThreadOnly; clone3 returns ENOSYS.
var Dangerous = []string{
	"add_key", "bpf", "delete_module", "finit_module", "fsconfig", "fsmount", "fsopen", "fspick",
	"init_module", "io_uring_enter", "io_uring_register", "io_uring_setup", "kexec_file_load",
	"kexec_load", "keyctl", "mount", "mount_setattr", "move_mount", "open_by_handle_at",
	"open_tree", "perf_event_open", "pivot_root", "process_vm_readv", "process_vm_writev",
	"ptrace", "reboot", "request_key", "setns", "socket", "splice", "swapoff", "swapon", "tee",
	"umount2", "unshare", "userfaultfd", "vmsplice",
}

// GoExecAMD64 is spk-02's amd64 exec allowlist for the static Go profile (t3 §16.2 block 2,
// 27 names, 0 unexpected SIGSYS under KILL). clone and prctl are here as names; the filter
// carries their argument rules (CloneThreadOnly, PrctlSetVMAOnly), so Allow must omit them.
// m3-04 moves this list into the real go@1.26 profile.
var GoExecAMD64 = []string{
	"arch_prctl", "clone", "epoll_create1", "epoll_ctl", "epoll_pwait", "eventfd2", "execve",
	"exit_group", "fcntl", "futex", "getpid", "gettid", "madvise", "mmap", "nanosleep", "openat",
	"prctl", "prlimit64", "read", "rt_sigaction", "rt_sigprocmask", "rt_sigreturn",
	"sched_getaffinity", "sched_yield", "sigaltstack", "tgkill", "write",
}

// GoBuildAMD64 is spk-02's logged `go build` compile set (t3 §16.2 block 2, 54 names). Compile
// filters are ENOSYS-default, so this is an allowlist whose misses fail softly. The compile
// jail spawns processes (posix_spawn → clone with CLONE_VM|CLONE_VFORK), so it can't use the
// exec profiles' CLONE_THREAD-only rule.
var GoBuildAMD64 = []string{
	"arch_prctl", "chdir", "clone", "close", "copy_file_range", "dup3", "epoll_create1",
	"epoll_ctl", "epoll_pwait", "eventfd2", "execve", "exit_group", "faccessat2", "fallocate",
	"fchmodat", "fcntl", "flock", "fstat", "ftruncate", "futex", "getcwd", "getdents64", "getpid",
	"gettid", "lseek", "madvise", "mkdirat", "mmap", "munmap", "nanosleep", "newfstatat",
	"openat", "pidfd_open", "pidfd_send_signal", "pipe2", "prctl", "pread64", "prlimit64",
	"pwrite64", "read", "readlinkat", "renameat", "rt_sigaction", "rt_sigprocmask",
	"rt_sigreturn", "sched_getaffinity", "sched_yield", "sigaltstack", "tgkill", "uname",
	"unlinkat", "utimensat", "waitid", "write",
}

// CompileInitExtra are the calls the runner's own in-jail compile init (a Go program that runs
// the compiler as a child, then exports the artifact) makes beyond the go build set.
var CompileInitExtra = []string{"exit", "getrandom", "kill", "wait4"}

// Without returns names minus drop.
func Without(names []string, drop ...string) []string {
	skip := map[string]bool{}
	for _, d := range drop {
		skip[d] = true
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if !skip[n] {
			out = append(out, n)
		}
	}
	return out
}
