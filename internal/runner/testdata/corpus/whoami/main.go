// whoami reports the jail as seen from inside, as JSON on fd 4: UIDs, the capability sets
// (capget), the bounding set (PR_CAPBSET_READ), NO_NEW_PRIVS, the root directory's and /dev's
// entries, whether /proc exists, the hostname, and whether each directory named in its input
// is writable (read-only binds must refuse with EROFS).
package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

type capHeader struct {
	version uint32
	pid     int32
}

type capData struct {
	effective, permitted, inheritable uint32
}

func main() {
	out := map[string]any{"uid": os.Getuid(), "euid": os.Geteuid(), "gid": os.Getgid()}

	hdr := capHeader{version: 0x20080522} // _LINUX_CAPABILITY_VERSION_3
	var data [2]capData
	_, _, e := syscall.RawSyscall(syscall.SYS_CAPGET, uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Pointer(&data[0])), 0)
	out["capget_errno"] = int(e)
	out["cap_eff"] = uint64(data[0].effective) | uint64(data[1].effective)<<32
	out["cap_prm"] = uint64(data[0].permitted) | uint64(data[1].permitted)<<32
	out["cap_inh"] = uint64(data[0].inheritable) | uint64(data[1].inheritable)<<32

	const prCapbsetRead, prGetNoNewPrivs = 23, 39
	var bnd uint64
	for c := uintptr(0); c < 64; c++ {
		r, _, e := syscall.RawSyscall(syscall.SYS_PRCTL, prCapbsetRead, c, 0)
		if e != 0 {
			break
		}
		if r == 1 {
			bnd |= 1 << c
		}
	}
	out["cap_bnd"] = bnd
	nnp, _, _ := syscall.RawSyscall(syscall.SYS_PRCTL, prGetNoNewPrivs, 0, 0)
	out["no_new_privs"] = int(nnp)

	var names []string
	if ents, err := os.ReadDir("/"); err == nil {
		for _, e := range ents {
			names = append(names, e.Name())
		}
	}
	out["root"] = names
	var devs []string
	if ents, err := os.ReadDir("/dev"); err == nil {
		for _, e := range ents {
			devs = append(devs, e.Name())
		}
	}
	out["dev"] = devs
	_, err := os.Stat("/proc/self")
	out["proc"] = err == nil
	host, _ := os.Hostname()
	out["hostname"] = host

	// Writability probes: one directory per input line ("rw" or the error).
	in, _ := io.ReadAll(os.NewFile(3, "input"))
	probes := map[string]string{}
	for _, dir := range strings.Fields(string(in)) {
		err := os.WriteFile(dir+"/.xl-rw-probe", []byte("x"), 0o644)
		if err == nil {
			probes[dir] = "rw"
		} else {
			probes[dir] = err.Error()
		}
	}
	out["write_probes"] = probes
	b, _ := json.Marshal(out)
	os.NewFile(4, "result").Write(b)
}
