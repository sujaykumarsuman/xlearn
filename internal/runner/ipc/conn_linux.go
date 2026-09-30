//go:build linux

package ipc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

// Conn is one end of the spawner⇄front pair.
type Conn struct {
	c    *net.UnixConn
	wmu  sync.Mutex
	rmu  sync.Mutex
	once sync.Once
}

// Pair creates the socketpair(AF_UNIX, SOCK_SEQPACKET). The spawner keeps the Conn; the
// *os.File is the front's end, passed to it at spawn time (then closed by the spawner).
func Pair() (*Conn, *os.File, error) {
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("socketpair: %w", err)
	}
	mine := os.NewFile(uintptr(fds[0]), "ipc-spawner")
	theirs := os.NewFile(uintptr(fds[1]), "ipc-front")
	c, err := FromFile(mine)
	if err != nil {
		theirs.Close()
		return nil, nil, err
	}
	return c, theirs, nil
}

// FromFile wraps an inherited socket (the front's fd 3). f is consumed.
func FromFile(f *os.File) (*Conn, error) {
	defer f.Close()
	fc, err := net.FileConn(f)
	if err != nil {
		return nil, fmt.Errorf("ipc: file conn: %w", err)
	}
	uc, ok := fc.(*net.UnixConn)
	if !ok {
		fc.Close()
		return nil, fmt.Errorf("ipc: %T is not a unix socket", fc)
	}
	return &Conn{c: uc}, nil
}

// Send encodes and sends m with its fds. The caller still owns (and closes) the fds.
func (c *Conn) Send(m *Msg) error {
	b, err := Encode(m)
	if err != nil {
		return err
	}
	var oob []byte
	if len(m.Fds) > 0 {
		oob = unix.UnixRights(m.Fds...)
	}
	c.wmu.Lock()
	defer c.wmu.Unlock()
	n, oobn, err := c.c.WriteMsgUnix(b, oob, nil)
	if err != nil {
		return fmt.Errorf("ipc: send %v: %w", m.Type, err)
	}
	if n != len(b) || oobn != len(oob) {
		return fmt.Errorf("ipc: short send of %v", m.Type)
	}
	return nil
}

// ErrClosed is returned by Recv after the peer closed the pair.
var ErrClosed = errors.New("ipc: pair closed")

// Recv receives and strictly decodes one message. Any error means the pair is unusable: the
// caller must Close it (killing the peer's side too). Received fds are close-on-exec.
func (c *Conn) Recv() (*Msg, error) {
	c.rmu.Lock()
	defer c.rmu.Unlock()
	buf := make([]byte, MsgSize+1) // one spare byte detects an oversize message
	oob := make([]byte, unix.CmsgSpace(MaxFds*4))
	n, oobn, flags, _, err := c.c.ReadMsgUnix(buf, oob)
	if err != nil {
		return nil, fmt.Errorf("ipc: recv: %w", err)
	}
	if n == 0 && oobn == 0 {
		return nil, ErrClosed
	}
	fds, ferr := parseRights(oob[:oobn])
	if flags&(unix.MSG_TRUNC|unix.MSG_CTRUNC) != 0 {
		closeAll(fds)
		return nil, fmt.Errorf("ipc: truncated message (flags %#x)", flags)
	}
	if ferr != nil {
		closeAll(fds)
		return nil, ferr
	}
	m, err := Decode(buf[:n], fds)
	if err != nil {
		closeAll(fds)
		return nil, err
	}
	return m, nil
}

func parseRights(oob []byte) ([]int, error) {
	if len(oob) == 0 {
		return nil, nil
	}
	cmsgs, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return nil, fmt.Errorf("ipc: control message: %w", err)
	}
	var fds []int
	for _, cm := range cmsgs {
		got, err := unix.ParseUnixRights(&cm)
		if err != nil {
			closeAll(fds)
			return nil, fmt.Errorf("ipc: unexpected control message: %w", err)
		}
		fds = append(fds, got...)
	}
	if len(fds) > MaxFds {
		closeAll(fds)
		return nil, fmt.Errorf("ipc: %d fds, cap %d", len(fds), MaxFds)
	}
	return fds, nil
}

func closeAll(fds []int) {
	for _, fd := range fds {
		unix.Close(fd)
	}
}

// Close shuts the pair; the peer's Recv then fails.
func (c *Conn) Close() error {
	var err error
	c.once.Do(func() { err = c.c.Close() })
	return err
}
