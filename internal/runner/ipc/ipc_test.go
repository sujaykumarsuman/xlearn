package ipc

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/runner/measure"
)

func TestEveryTypeRoundTrips(t *testing.T) {
	msgs := []*Msg{
		{Type: TPing, ReqID: 1, Body: &Ping{Nonce: 42}},
		{Type: TPong, ReqID: 1, Body: &Pong{Nonce: 42}},
		{Type: TJobBegin, ReqID: 2, Slot: 1, Body: &JobBegin{ProfileIdx: 3, JobSeq: 99, CompileCPUms: 15000, CompileMemMB: 768}},
		{Type: TJobBegun, ReqID: 2, Slot: 1, Body: &JobBegun{CanaryMedianUs: 150000}, Fds: []int{7}},
		{Type: TCompile, ReqID: 3, Body: &Compile{}},
		{Type: TCompileStarted, ReqID: 3, Body: &CompileStarted{}, Fds: []int{7, 8}},
		{Type: TCompileDone, ReqID: 3, Body: &CompileDone{Exited: 1, ExitCode: 2, CPUus: 1, WallUs: 2, PeakBytes: 3, ArtifactBytes: 4, Export: uint8(ExportSkipped)}},
		{Type: TCaseRun, ReqID: 4, Body: &CaseRun{CaseIdx: 511, CPUms: 1000, MemMB: 256, WallCapMs: 45000, Quiet: 1}},
		{Type: TCaseStarted, ReqID: 4, Body: &CaseStarted{}, Fds: []int{7, 8, 9, 10}},
		{Type: TCaseDone, ReqID: 4, Body: &CaseDone{Exited: 0, Signal: 31, Kill: uint8(measure.KillCPU), CPUus: 1, StealPermille: 1000, CanaryPermille: 1250, OtherBusy: 1}},
		{Type: TCaseKill, ReqID: 4, Body: &CaseKill{Reason: uint8(measure.KillOLE)}},
		{Type: TJobEnd, ReqID: 5, Body: &JobEnd{Abort: 1}},
		{Type: TJobEnded, ReqID: 5, Body: &JobEnded{Outcome: uint16(TeardownMemory)}},
		{Type: TQuietCheck, ReqID: 6, Body: &QuietCheck{}},
		{Type: TQuietStatus, ReqID: 6, Body: &QuietStatus{StealPermille: 12, CanaryPermille: 1010}},
		{Type: TStats, ReqID: 7, Body: &Stats{}},
		{Type: TStatsReply, ReqID: 7, Body: &StatsReply{MemoryCurrent: [2]uint64{1, 2}, PidsCurrent: [2]uint32{3, 4}, NrDying: [2]uint32{5, 6}, RunnerMemory: 7, OOMKills: 8, OOMKillsLocal: 9, CanaryMedianUs: 10}},
		{Type: TProbe, ReqID: 8, Body: &Probe{}},
		{Type: TProbeReply, ReqID: 8, Body: &ProbeReply{Ran: ProbeAll, Failed: ProbeUserNS}},
		{Type: TDrain, Body: &Drain{}},
		{Type: TFail, ReqID: 9, Body: &Fail{Code: uint16(FailSetup), Errno: 13}},
	}
	seen := map[Type]bool{}
	for _, m := range msgs {
		b, err := Encode(m)
		if err != nil {
			t.Fatalf("encode %v: %v", m.Type, err)
		}
		if len(b) != MsgSize {
			t.Fatalf("%v: %d bytes", m.Type, len(b))
		}
		got, err := Decode(b, m.Fds)
		if err != nil {
			t.Fatalf("decode %v: %v", m.Type, err)
		}
		if got.Type != m.Type || got.ReqID != m.ReqID || got.Slot != m.Slot || !reflect.DeepEqual(got.Body, m.Body) {
			t.Errorf("%v round trip: got %+v want %+v", m.Type, got, m)
		}
		seen[m.Type] = true
	}
	for tt := TPing; tt < typeEnd; tt++ {
		if !seen[tt] {
			t.Errorf("type %v has no round-trip case", tt)
		}
		if n := binary.Size(reflectNew(tt)); n > bodySize {
			t.Errorf("%v body is %d bytes, over %d", tt, n, bodySize)
		}
	}
}

func reflectNew(t Type) any { return reflect.New(specs[t].body).Interface() }

func mustEncode(t *testing.T, m *Msg) []byte {
	t.Helper()
	b, err := Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestStrictDecode(t *testing.T) {
	good := mustEncode(t, &Msg{Type: TCaseRun, ReqID: 4, Body: &CaseRun{CaseIdx: 1, CPUms: 1000, MemMB: 256, WallCapMs: 1000}})
	flip := func(i int, v byte) []byte {
		b := append([]byte(nil), good...)
		b[i] = v
		return b
	}
	cases := map[string]struct {
		b   []byte
		fds []int
	}{
		"short":             {good[:MsgSize-1], nil},
		"long":              {append(append([]byte(nil), good...), 0), nil},
		"bad magic":         {flip(0, 'Y'), nil},
		"bad version":       {flip(5, 2), nil},
		"unknown type":      {flip(7, 99), nil},
		"slot 2":            {flip(12, 2), nil},
		"fd count mismatch": {good, []int{3}},
		"reserved set":      {flip(15, 1), nil},
		"padding set":       {flip(MsgSize-1, 1), nil},
		"bool byte 2":       {flip(headerSize+16, 2), nil}, // Quiet
		"cpu out of range":  {flip(headerSize+4, 0xff), nil},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode(c.b, c.fds); err == nil {
				t.Fatal("want an error")
			}
		})
	}
	if _, err := Decode(good, nil); err != nil {
		t.Fatalf("the unmodified message decodes: %v", err)
	}
}

func TestEncodeChecks(t *testing.T) {
	if _, err := Encode(&Msg{Type: TCaseStarted, Body: &CaseStarted{}, Fds: []int{1}}); err == nil {
		t.Error("wrong fd count must fail")
	}
	if _, err := Encode(&Msg{Type: TPing, Body: &Pong{}}); err == nil {
		t.Error("wrong body type must fail")
	}
	if _, err := Encode(&Msg{Type: TCaseKill, Body: &CaseKill{Reason: uint8(measure.KillCPU)}}); err == nil {
		t.Error("the front may only ask for an OLE kill")
	}
	if _, err := Encode(&Msg{Type: Type(999), Body: &Ping{}}); err == nil {
		t.Error("unknown type must fail")
	}
}

func TestDirections(t *testing.T) {
	for _, tt := range []Type{TPing, TJobBegin, TCompile, TCaseRun, TCaseKill, TJobEnd, TQuietCheck, TStats, TProbe} {
		if !tt.FromFront() {
			t.Errorf("%v is a front request", tt)
		}
	}
	for _, tt := range []Type{TPong, TJobBegun, TCompileStarted, TCompileDone, TCaseStarted, TCaseDone, TJobEnded, TQuietStatus, TStatsReply, TProbeReply, TDrain, TFail} {
		if tt.FromFront() {
			t.Errorf("%v is a spawner message", tt)
		}
	}
}
