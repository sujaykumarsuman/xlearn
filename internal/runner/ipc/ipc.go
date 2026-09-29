// Package ipc is the fixed-schema spawner⇄front protocol (t3 §5.2): versioned, fixed-size
// binary messages over a socketpair(AF_UNIX, SOCK_SEQPACKET). A message carries enums, sizes,
// limits and opaque indices — never learner content — and the per-case pipe fds travel beside
// it with SCM_RIGHTS. Decoding is strict: a wrong magic, version, length, type, direction,
// fd count, enum value or a non-zero padding byte is an error, and either side kills the pair
// on any error (the container then restarts).
//
// The codec is portable and tested everywhere; the transport (conn_linux.go) is Linux-only.
package ipc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
)

// Wire constants.
const (
	Magic      uint32 = 0x584C524E // "XLRN"
	Version    uint16 = 1
	MsgSize           = 128
	headerSize        = 16
	bodySize          = MsgSize - headerSize
	// MaxFds is the most fds one message carries (CaseStarted's four pipe ends).
	MaxFds = 4
	// Slots is the number of job slots (t3 §7.1).
	Slots = 2
)

// Type is a message type.
type Type uint16

// Message types. F→S are requests from the front; S→F are the spawner's replies and events.
const (
	TPing           Type = iota + 1 // F→S
	TPong                           // S→F
	TJobBegin                       // F→S: set up a job in a slot → JobBegun (+ the src dir fd)
	TJobBegun                       // S→F, 1 fd
	TCompile                        // F→S: run the compile jail → CompileStarted, then CompileDone
	TCompileStarted                 // S→F, 2 fds: stdout and stderr read ends
	TCompileDone                    // S→F
	TCaseRun                        // F→S: run one case → CaseStarted, then CaseDone
	TCaseStarted                    // S→F, 4 fds: fd 3 write end; fds 1, 2, 4 read ends
	TCaseDone                       // S→F
	TCaseKill                       // F→S: kill the running case (OLE); no reply
	TJobEnd                         // F→S: tear the job down → JobEnded
	TJobEnded                       // S→F
	TQuietCheck                     // F→S: measure quiet conditions → QuietStatus
	TQuietStatus                    // S→F
	TStats                          // F→S → StatsReply
	TStatsReply                     // S→F
	TProbe                          // F→S: run the privileged readiness canaries → ProbeReply
	TProbeReply                     // S→F
	TDrain                          // S→F (ReqID 0): SIGTERM arrived; drain and exit
	TFail                           // S→F: the request failed
	typeEnd
)

var typeNames = [...]string{"", "Ping", "Pong", "JobBegin", "JobBegun", "Compile", "CompileStarted", "CompileDone",
	"CaseRun", "CaseStarted", "CaseDone", "CaseKill", "JobEnd", "JobEnded", "QuietCheck", "QuietStatus",
	"Stats", "StatsReply", "Probe", "ProbeReply", "Drain", "Fail"}

func (t Type) String() string {
	if t > 0 && t < typeEnd {
		return typeNames[t]
	}
	return fmt.Sprintf("Type(%d)", uint16(t))
}

// FromFront reports whether t is a front→spawner request.
func (t Type) FromFront() bool {
	switch t {
	case TPing, TJobBegin, TCompile, TCaseRun, TCaseKill, TJobEnd, TQuietCheck, TStats, TProbe:
		return true
	}
	return false
}

// spec is a type's body and fd count.
type spec struct {
	body reflect.Type
	fds  int
}

var specs = map[Type]spec{
	TPing:           {reflect.TypeOf(Ping{}), 0},
	TPong:           {reflect.TypeOf(Pong{}), 0},
	TJobBegin:       {reflect.TypeOf(JobBegin{}), 0},
	TJobBegun:       {reflect.TypeOf(JobBegun{}), 1},
	TCompile:        {reflect.TypeOf(Compile{}), 0},
	TCompileStarted: {reflect.TypeOf(CompileStarted{}), 2},
	TCompileDone:    {reflect.TypeOf(CompileDone{}), 0},
	TCaseRun:        {reflect.TypeOf(CaseRun{}), 0},
	TCaseStarted:    {reflect.TypeOf(CaseStarted{}), 4},
	TCaseDone:       {reflect.TypeOf(CaseDone{}), 0},
	TCaseKill:       {reflect.TypeOf(CaseKill{}), 0},
	TJobEnd:         {reflect.TypeOf(JobEnd{}), 0},
	TJobEnded:       {reflect.TypeOf(JobEnded{}), 0},
	TQuietCheck:     {reflect.TypeOf(QuietCheck{}), 0},
	TQuietStatus:    {reflect.TypeOf(QuietStatus{}), 0},
	TStats:          {reflect.TypeOf(Stats{}), 0},
	TStatsReply:     {reflect.TypeOf(StatsReply{}), 0},
	TProbe:          {reflect.TypeOf(Probe{}), 0},
	TProbeReply:     {reflect.TypeOf(ProbeReply{}), 0},
	TDrain:          {reflect.TypeOf(Drain{}), 0},
	TFail:           {reflect.TypeOf(Fail{}), 0},
}

// Msg is one decoded message. Body is a pointer to the type's body struct.
type Msg struct {
	Type  Type
	ReqID uint32
	Slot  uint8
	Body  any
	Fds   []int
}

// Encode renders m into a MsgSize buffer. It checks the body type and fd count.
func Encode(m *Msg) ([]byte, error) {
	sp, ok := specs[m.Type]
	if !ok {
		return nil, fmt.Errorf("ipc: encode unknown type %v", m.Type)
	}
	if m.Slot >= Slots {
		return nil, fmt.Errorf("ipc: slot %d", m.Slot)
	}
	if len(m.Fds) != sp.fds {
		return nil, fmt.Errorf("ipc: %v carries %d fds, not %d", m.Type, sp.fds, len(m.Fds))
	}
	bv := reflect.ValueOf(m.Body)
	if bv.Kind() != reflect.Pointer || bv.Elem().Type() != sp.body {
		return nil, fmt.Errorf("ipc: %v body is %T, want *%v", m.Type, m.Body, sp.body)
	}
	if err := validate(m.Body); err != nil {
		return nil, fmt.Errorf("ipc: encode %v: %w", m.Type, err)
	}
	buf := make([]byte, 0, MsgSize)
	w := bytes.NewBuffer(buf)
	hdr := header{Magic: Magic, Version: Version, Type: uint16(m.Type), ReqID: m.ReqID, Slot: m.Slot, NFds: uint8(len(m.Fds))}
	if err := binary.Write(w, binary.BigEndian, &hdr); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.BigEndian, m.Body); err != nil {
		return nil, err
	}
	if w.Len() > MsgSize {
		return nil, fmt.Errorf("ipc: %v body too large", m.Type)
	}
	out := make([]byte, MsgSize)
	copy(out, w.Bytes())
	return out, nil
}

type header struct {
	Magic    uint32
	Version  uint16
	Type     uint16
	ReqID    uint32
	Slot     uint8
	NFds     uint8
	Reserved uint16
}

// Decode parses one message. fds are the SCM_RIGHTS fds that arrived with it; on any error the
// caller closes them (and the pair).
func Decode(b []byte, fds []int) (*Msg, error) {
	if len(b) != MsgSize {
		return nil, fmt.Errorf("ipc: message is %d bytes, want %d", len(b), MsgSize)
	}
	var hdr header
	if err := binary.Read(bytes.NewReader(b[:headerSize]), binary.BigEndian, &hdr); err != nil {
		return nil, err
	}
	if hdr.Magic != Magic {
		return nil, fmt.Errorf("ipc: bad magic %#x", hdr.Magic)
	}
	if hdr.Version != Version {
		return nil, fmt.Errorf("ipc: version %d, want %d", hdr.Version, Version)
	}
	if hdr.Reserved != 0 {
		return nil, fmt.Errorf("ipc: reserved header bits set")
	}
	t := Type(hdr.Type)
	sp, ok := specs[t]
	if !ok {
		return nil, fmt.Errorf("ipc: unknown message type %d", hdr.Type)
	}
	if hdr.Slot >= Slots {
		return nil, fmt.Errorf("ipc: slot %d", hdr.Slot)
	}
	if int(hdr.NFds) != sp.fds || len(fds) != sp.fds {
		return nil, fmt.Errorf("ipc: %v with %d/%d fds, want %d", t, hdr.NFds, len(fds), sp.fds)
	}
	body := reflect.New(sp.body)
	n := binary.Size(body.Interface())
	if n < 0 || n > bodySize {
		return nil, fmt.Errorf("ipc: %v body size %d", t, n)
	}
	if err := binary.Read(bytes.NewReader(b[headerSize:headerSize+n]), binary.BigEndian, body.Interface()); err != nil {
		return nil, err
	}
	for _, c := range b[headerSize+n:] {
		if c != 0 {
			return nil, fmt.Errorf("ipc: %v: non-zero padding", t)
		}
	}
	if err := validate(body.Interface()); err != nil {
		return nil, fmt.Errorf("ipc: %v: %w", t, err)
	}
	return &Msg{Type: t, ReqID: hdr.ReqID, Slot: hdr.Slot, Body: body.Interface(), Fds: fds}, nil
}

// validator is implemented by bodies with enum or range fields.
type validator interface{ validate() error }

func validate(body any) error {
	if v, ok := body.(validator); ok {
		return v.validate()
	}
	return nil
}

func boolByte(name string, b uint8) error {
	if b > 1 {
		return fmt.Errorf("%s = %d, want 0 or 1", name, b)
	}
	return nil
}
