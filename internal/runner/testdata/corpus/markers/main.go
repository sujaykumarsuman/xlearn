// markers leaves (input "write <token>") or looks for (input "read <token>") cross-job
// markers: files in /w, /tmp and /dev/shm, a SysV shm segment, a POSIX message queue, an
// abstract unix socket and a user-keyring key (t3 §5.10). On read it prints every marker it can
// see and exits with their count; job N's markers must be invisible to job N+1.
package main

import (
	"fmt"
	"hash/fnv"
	"io"
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

func main() {
	in, _ := io.ReadAll(os.NewFile(3, "input"))
	mode, token, _ := strings.Cut(strings.TrimSpace(string(in)), " ")
	h := fnv.New32a()
	h.Write([]byte(token))
	shmKey := uintptr(h.Sum32() & 0x7fffffff)
	name := "xl-" + token
	files := []string{"/w/" + name, "/tmp/" + name, "/dev/shm/" + name}

	if mode == "write" {
		for _, f := range files {
			err := os.WriteFile(f, []byte(token), 0o666)
			fmt.Println("file", f, err)
		}
		id, _, e := syscall.Syscall(syscall.SYS_SHMGET, shmKey, 4096, 0o1000|0o666) // IPC_CREAT
		fmt.Println("shm", int(id), e)
		fd, _, e := syscall.Syscall6(syscall.SYS_MQ_OPEN, cstr(name), uintptr(os.O_CREATE|os.O_RDWR), 0o666, 0, 0, 0)
		fmt.Println("mq", int(fd), e)
		if s, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0); err == nil {
			err = syscall.Bind(s, &syscall.SockaddrUnix{Name: "@" + name})
			if err == nil {
				err = syscall.Listen(s, 1)
			}
			fmt.Println("abstract", err)
		}
		k, _, e := syscall.Syscall6(syscall.SYS_ADD_KEY, cstr("user"), cstr(name), cstr(token), uintptr(len(token)), keySpecUserKeyring, 0)
		fmt.Println("key", int(k), e)
		return
	}

	var seen []string
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			seen = append(seen, "file:"+f)
		}
	}
	if _, _, e := syscall.Syscall(syscall.SYS_SHMGET, shmKey, 0, 0); e == 0 {
		seen = append(seen, "sysv-shm")
	}
	if fd, _, e := syscall.Syscall6(syscall.SYS_MQ_OPEN, cstr(name), uintptr(os.O_RDONLY), 0, 0, 0, 0); e == 0 {
		syscall.Close(int(fd))
		seen = append(seen, "posix-mq")
	}
	if s, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0); err == nil {
		if syscall.Connect(s, &syscall.SockaddrUnix{Name: "@" + name}) == nil {
			seen = append(seen, "abstract-socket")
		}
	}
	const keyctlSearch = 10
	if _, _, e := syscall.Syscall6(syscall.SYS_KEYCTL, keyctlSearch, keySpecUserKeyring, cstr("user"), cstr(name), 0, 0); e == 0 {
		seen = append(seen, "keyring")
	}
	fmt.Println("visible:", seen)
	os.NewFile(4, "result").Write([]byte(strings.Join(seen, ",")))
	os.Exit(len(seen))
}
