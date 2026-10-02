package main

// Section E (cross-job markers, t3 §5.10), Go: arg is "<write|read>:<channel>:<token>". A write
// answers "written" or "denied: ..." (or dies of SIGSYS: the call is denied by the filter); a read
// answers "visible" or "absent: ...". Job N's markers must be invisible to job N+1.

import (
	"hash/fnv"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const keySpecUserKeyring = ^uintptr(3) // KEY_SPEC_USER_KEYRING (-4)

func cstr(s string) uintptr {
	b := append([]byte(s), 0)
	return uintptr(unsafe.Pointer(&b[0]))
}

var dirs = map[string]string{"w": "/w/", "tmp": "/tmp/", "devshm": "/dev/shm/"}

func probe(arg string) string {
	parts := strings.SplitN(arg, ":", 3)
	if len(parts) != 3 {
		return "bad probe " + arg
	}
	op, ch, tok := parts[0], parts[1], parts[2]
	name := "xl-" + tok
	write := op == "write"
	switch ch {
	case "w", "tmp", "devshm":
		path := dirs[ch] + name
		if write {
			if err := os.WriteFile(path, []byte(tok), 0o666); err != nil {
				return "denied: " + err.Error()
			}
			return "written"
		}
		f, err := os.Open(path)
		if err != nil {
			return "absent: " + err.Error()
		}
		f.Close()
		return "visible"
	case "sysvshm":
		h := fnv.New32a()
		h.Write([]byte(tok))
		key := uintptr(h.Sum32() & 0x7fffffff)
		if write {
			if _, _, e := syscall.RawSyscall(syscall.SYS_SHMGET, key, 4096, 0o1000|0o666); e != 0 {
				return "denied: " + e.Error()
			}
			return "written"
		}
		if _, _, e := syscall.RawSyscall(syscall.SYS_SHMGET, key, 0, 0); e != 0 {
			return "absent: " + e.Error()
		}
		return "visible"
	case "posixmq":
		if write {
			fd, _, e := syscall.RawSyscall6(syscall.SYS_MQ_OPEN, cstr(name), uintptr(os.O_CREATE|os.O_RDWR), 0o666, 0, 0, 0)
			if e != 0 {
				return "denied: " + e.Error()
			}
			_ = fd
			return "written"
		}
		if _, _, e := syscall.RawSyscall6(syscall.SYS_MQ_OPEN, cstr(name), uintptr(os.O_RDONLY), 0, 0, 0, 0); e != 0 {
			return "absent: " + e.Error()
		}
		return "visible"
	case "abstract":
		s, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
		if err != nil {
			return "denied: " + err.Error()
		}
		if write {
			if err := syscall.Bind(s, &syscall.SockaddrUnix{Name: "@" + name}); err != nil {
				return "denied: " + err.Error()
			}
			if err := syscall.Listen(s, 1); err != nil {
				return "denied: " + err.Error()
			}
			return "written"
		}
		if err := syscall.Connect(s, &syscall.SockaddrUnix{Name: "@" + name}); err != nil {
			return "absent: " + err.Error()
		}
		return "visible"
	case "keyring":
		if write {
			if _, _, e := syscall.RawSyscall6(syscall.SYS_ADD_KEY, cstr("user"), cstr(name), cstr(tok), uintptr(len(tok)), keySpecUserKeyring, 0); e != 0 {
				return "denied: " + e.Error()
			}
			return "written"
		}
		const keyctlSearch = 10
		if _, _, e := syscall.RawSyscall6(syscall.SYS_KEYCTL, keyctlSearch, keySpecUserKeyring, cstr("user"), cstr(name), 0, 0); e != 0 {
			return "absent: " + e.Error()
		}
		return "visible"
	}
	return "bad probe " + arg
}
