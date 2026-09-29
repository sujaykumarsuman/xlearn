//go:build linux && (amd64 || arm64)

package seccomp

import "syscall"

// Native is the running architecture's table.
func Native() Arch {
	return Arch{Name: nativeName, Audit: nativeAudit, X32Bit: nativeX32, Numbers: nativeNumbers}
}

// SockFprog converts an assembled program for seccomp(2) / forkexec.Runner.Seccomp. The
// returned value keeps the instruction slice alive.
func SockFprog(prog []Instr) *syscall.SockFprog {
	if len(prog) == 0 {
		return nil
	}
	f := make([]syscall.SockFilter, len(prog))
	for i, in := range prog {
		f[i] = syscall.SockFilter{Code: in.Code, Jt: in.Jt, Jf: in.Jf, K: in.K}
	}
	return &syscall.SockFprog{Len: uint16(len(f)), Filter: &f[0]}
}
