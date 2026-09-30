//go:build linux

package jail

import (
	"syscall"

	"github.com/criyle/go-sandbox/pkg/forkexec"
)

// Process is a plain (non-jailed) child the spawner starts: the front and the canary. It gets
// a fixed non-zero UID, 0 capabilities, an empty bounding set and NO_NEW_PRIVS, all set in
// the child before execve (never dropped from a running Go process: capability changes are
// per thread).
type Process struct {
	Argv  []string
	Env   []string
	Files []uintptr
	UID   int
	// CgroupFd, if set, places the child in that cgroup (CLONE_INTO_CGROUP).
	CgroupFd int
}

// StartProcess starts p and returns its pid.
func StartProcess(p Process) (int, error) {
	r := &forkexec.Runner{
		Args:       p.Argv,
		Env:        p.Env,
		Files:      p.Files,
		Credential: &syscall.Credential{Uid: uint32(p.UID), Gid: uint32(p.UID), Groups: []uint32{}},
		DropCaps:   true,
		NoNewPrivs: true,
	}
	if p.CgroupFd > 0 {
		r.CgroupFd = uintptr(p.CgroupFd)
	}
	return StartEmptyBounding(r)
}
