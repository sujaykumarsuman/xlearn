package runnerapi

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden fixtures in testdata/")

// exampleJob is the golden job.json (testdata/job.json): every field set, so the fixture pins
// every JSON name m3-06 relies on.
func exampleJob() *Job {
	return &Job{
		ID:      "job-0001",
		Profile: "go@1.26",
		Harness: "func-json@1",
		Mode:    ModeSubmit,
		Files: []File{
			{Path: "solution.go", Data: []byte("package main\n\nfunc twoSum(nums []int, target int) []int { return nil }\n")},
			{Path: "zz_xl_harness.go", Data: []byte("package main\n\nfunc main() {}\n")},
		},
		HiddenFiles: []File{{Path: "hidden_test.go", Data: []byte("package main\n")}},
		Cases: []CaseInput{
			{OpaqueID: "c1", Group: GroupSample, Size: 3},
			{OpaqueID: "c2", Group: GroupEdge, Size: 0},
			{OpaqueID: "c3", Group: GroupRandom, Size: 5},
			{OpaqueID: "c4", Group: GroupPerf, Size: 4},
		},
		Tests:       []TestSpec{{Name: "TestRace"}},
		Limits:      Limits{Case: CaseLimits{CPUms: 1000, MemMB: 256, OutputKB: 64}, Compile: CompileLimits{CPUms: 15000, MemMB: 768}},
		StopGroupOn: map[Group]StopRule{GroupSample: StopAnyFail, GroupPerf: StopTLE},
		Count:       1,
		OutputMode:  OutputBytes,
	}
}

func exampleInputs() [][]byte {
	return [][]byte{[]byte("[1]"), {}, []byte("[2,3]"), []byte("[99]")}
}

// exampleResult is the golden result.json (testdata/result.json) for exampleJob.
func exampleResult() *Result {
	return &Result{
		Compile: CompileResult{OK: true, Diags: []Diag{{File: "solution.go", Line: 3, Col: 6, Msg: "declared and not used: x"}}},
		Cases: []CaseResult{
			{OpaqueID: "c1", Term: TermOK, CPUms: 3, WallMs: 5, PeakKB: 812, Output: []byte("[0,1]"),
				OutputSHA256: "b2bfa5d4d6f2f6e7d8d2ac6a0a4b3d9f0b1e2a3c4d5e6f708192a3b4c5d6e7f8", OutputBytes: 5},
			{OpaqueID: "c2", Term: TermSignal, Signal: "SIGSYS", CPUms: 1, WallMs: 2, PeakKB: 640,
				OutputSHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
			{OpaqueID: "c3", Term: TermTLE, CPUms: 1500, WallMs: 1510, PeakKB: 900, IdleKill: false, Quiet: true,
				OutputSHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
			{OpaqueID: "c4", Term: TermNotRun},
		},
		Tests:     []TestEvent{{Test: "TestRace", Action: "pass", Elapsed: 0.01}},
		Race:      false,
		Throttled: false,
		Versions: Versions{
			Runner: "runner-v1.0.0", ImageDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			Profile: "go@1.26", Toolchain: "go1.26.8", ProfileSHA: "1111111111111111111111111111111111111111111111111111111111111111",
			CPUModel: "AMD EPYC 9355P 32-Core Processor", CanaryMedian: 152000,
			BootEpoch: "xlearn-runner-7d9f/1790000000000000000/0123456789abcdef",
		},
		Telemetry: Telemetry{StealPct: 2.5, CanaryRatio: 1.04, BusyMs: 4200, OtherSlotBusy: true, QuietReruns: 1, SlotMs: 3100, TestsCapHit: false},
	}
}

func golden(t *testing.T, name string, v any) {
	t.Helper()
	got, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s (run with -update to create): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s drifted from the golden fixture (a contract change is append-only; run -update only for additions):\n%s", path, got)
	}
}

func TestGoldenFixtures(t *testing.T) {
	job, res := exampleJob(), exampleResult()
	if err := ValidateJob(job); err != nil {
		t.Fatalf("example job invalid: %v", err)
	}
	if err := ValidateResult(job, res); err != nil {
		t.Fatalf("example result invalid: %v", err)
	}
	golden(t, "job.json", job)
	golden(t, "result.json", res)

	// The fixtures decode strictly back into the same values (round trip).
	raw, err := os.ReadFile(filepath.Join("testdata", "job.json"))
	if err != nil {
		t.Fatal(err)
	}
	gotJob, err := decodeJobJSON(raw)
	if err != nil {
		t.Fatalf("strict decode of the golden job: %v", err)
	}
	if !reflect.DeepEqual(gotJob, job) {
		t.Errorf("job round trip differs:\n got %+v\nwant %+v", gotJob, job)
	}
	raw, err = os.ReadFile(filepath.Join("testdata", "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var gotRes Result
	if err := dec.Decode(&gotRes); err != nil {
		t.Fatalf("strict decode of the golden result: %v", err)
	}
	if !reflect.DeepEqual(&gotRes, res) {
		t.Errorf("result round trip differs:\n got %+v\nwant %+v", gotRes, res)
	}
}

func TestStreamRoundTrip(t *testing.T) {
	job, inputs := exampleJob(), exampleInputs()
	var buf bytes.Buffer
	if err := EncodeJob(&buf, job, inputs); err != nil {
		t.Fatal(err)
	}
	gotJob, gotInputs, err := DecodeJob(&buf, DefaultCaps())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotJob, job) {
		t.Errorf("job differs after the stream round trip")
	}
	if len(gotInputs) != len(inputs) {
		t.Fatalf("got %d inputs, want %d", len(gotInputs), len(inputs))
	}
	for i := range inputs {
		if !bytes.Equal(gotInputs[i], inputs[i]) {
			t.Errorf("input %d differs", i)
		}
	}
}

// frame builds one length-prefixed frame.
func frame(b []byte) []byte {
	out := make([]byte, 4+len(b))
	binary.BigEndian.PutUint32(out, uint32(len(b)))
	copy(out[4:], b)
	return out
}

func jobFrame(t *testing.T, job *Job) []byte {
	t.Helper()
	b, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	return frame(b)
}

// tripwire fails the test if more than limit bytes are read from it: it proves a cap is
// enforced from the length prefix, before the payload is read.
type tripwire struct {
	t     *testing.T
	r     io.Reader
	read  int
	limit int
}

func (w *tripwire) Read(p []byte) (int, error) {
	n, err := w.r.Read(p)
	w.read += n
	if w.read > w.limit {
		w.t.Fatalf("read %d bytes, past the %d-byte tripwire: the cap was not enforced mid-stream", w.read, w.limit)
	}
	return n, err
}

// endless yields zeros forever: the decoder must stop on a cap, never read to the end.
type endless struct{}

func (endless) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

func TestCapsEnforcedMidStream(t *testing.T) {
	caps := DefaultCaps()

	t.Run("9 MiB case refused before it is read", func(t *testing.T) {
		job := exampleJob()
		job.Cases = []CaseInput{{OpaqueID: "big", Group: GroupPerf, Size: 9 << 20}}
		head := jobFrame(t, job)
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], 9<<20)
		body := io.MultiReader(bytes.NewReader(head), bytes.NewReader(hdr[:]), endless{})
		_, _, err := DecodeJob(&tripwire{t: t, r: body, limit: len(head) + 4 + 64<<10}, caps)
		if err == nil || !strings.Contains(err.Error(), "cap") {
			t.Fatalf("want a cap error, got %v", err)
		}
	})

	t.Run("job.json over 1 MiB refused from its prefix", func(t *testing.T) {
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], MaxJobJSONBytes+1)
		body := io.MultiReader(bytes.NewReader(hdr[:]), endless{})
		_, _, err := DecodeJob(&tripwire{t: t, r: body, limit: 4 + 64<<10}, caps)
		if err == nil || !strings.Contains(err.Error(), "job.json") {
			t.Fatalf("want a job.json cap error, got %v", err)
		}
	})

	t.Run("inputs over 16 MiB refused at the frame that crosses it", func(t *testing.T) {
		job := exampleJob()
		job.Cases = []CaseInput{
			{OpaqueID: "a", Group: GroupPerf, Size: 8 << 20},
			{OpaqueID: "b", Group: GroupPerf, Size: 8 << 20},
			{OpaqueID: "c", Group: GroupPerf, Size: 1},
		}
		head := jobFrame(t, job)
		var parts []io.Reader
		parts = append(parts, bytes.NewReader(head))
		for _, n := range []int{8 << 20, 8 << 20} {
			parts = append(parts, bytes.NewReader(frame(make([]byte, n))))
		}
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], 1)
		parts = append(parts, bytes.NewReader(hdr[:]), endless{})
		limit := len(head) + 2*(4+8<<20) + 4 + 64<<10
		_, _, err := DecodeJob(&tripwire{t: t, r: io.MultiReader(parts...), limit: limit}, caps)
		if err == nil || !strings.Contains(err.Error(), "inputs exceed") {
			t.Fatalf("want an inputs cap error, got %v", err)
		}
	})

	t.Run("more than 512 cases", func(t *testing.T) {
		job := exampleJob()
		job.Cases = nil
		for i := 0; i < MaxCases+1; i++ {
			job.Cases = append(job.Cases, CaseInput{OpaqueID: "c" + strings.Repeat("x", i%10) + string(rune('a'+i%26)), Group: GroupPerf})
		}
		body := io.MultiReader(bytes.NewReader(jobFrame(t, job)), endless{})
		_, _, err := DecodeJob(body, caps)
		if err == nil || !strings.Contains(err.Error(), "cases") {
			t.Fatalf("want a case-count error, got %v", err)
		}
	})
}

func TestDecodeStrictness(t *testing.T) {
	caps := DefaultCaps()
	job, inputs := exampleJob(), exampleInputs()
	var good bytes.Buffer
	if err := EncodeJob(&good, job, inputs); err != nil {
		t.Fatal(err)
	}

	cases := map[string][]byte{
		"empty body":          nil,
		"truncated header":    good.Bytes()[:2],
		"trailing bytes":      append(append([]byte{}, good.Bytes()...), 0),
		"missing case frames": good.Bytes()[:len(jobFrame(t, job))],
		"unknown field":       frame([]byte(`{"id":"x","cost":5}`)),
		"trailing json":       frame([]byte(`{"id":"x"} {}`)),
	}
	// A frame whose length disagrees with its case's Size.
	mismatch := jobFrame(t, job)
	mismatch = append(mismatch, frame([]byte("[1,2]"))...) // Size 3, frame 5
	cases["size mismatch"] = mismatch

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := DecodeJob(bytes.NewReader(body), caps)
			if err == nil {
				t.Fatal("want an error")
			}
			if !IsStreamError(err) {
				t.Fatalf("want a StreamError, got %T %v", err, err)
			}
		})
	}
}

func TestEncodeRefusesMismatch(t *testing.T) {
	job := exampleJob()
	if err := EncodeJob(io.Discard, job, exampleInputs()[:2]); err == nil {
		t.Error("want an error for a missing input")
	}
	in := exampleInputs()
	in[0] = []byte("[1,2,3]")
	if err := EncodeJob(io.Discard, job, in); err == nil {
		t.Error("want an error for an input whose length disagrees with Size")
	}
}

func TestCheckContentType(t *testing.T) {
	for ct, ok := range map[string]bool{
		ContentTypeJob:                                true,
		"application/vnd.xlearn.runner-job;v=1":       true,
		"application/vnd.xlearn.runner-job; v=2":      false,
		"application/vnd.xlearn.runner-job":           false,
		"application/json":                            false,
		"application/vnd.xlearn.runner-job; v=1; x=y": false,
		"": false,
	} {
		if err := CheckContentType(ct); (err == nil) != ok {
			t.Errorf("CheckContentType(%q) = %v, want ok=%v", ct, err, ok)
		}
	}
}

func TestValidFileName(t *testing.T) {
	for name, ok := range map[string]bool{
		"main.go": true, "solution.cpp": true, "zz_xl_harness.go": true, "A1_b.py": true,
		"":                              false,
		"main":                          false,
		".hidden":                       false,
		"a..go":                         false,
		"../x.go":                       false,
		"dir/x.go":                      false,
		`dir\x.go`:                      false,
		"x.go.bak":                      false,
		"x.":                            false,
		"x y.go":                        false,
		"x-y.go":                        false,
		"é.go":                          false,
		strings.Repeat("a", 61) + ".go": true,
		strings.Repeat("a", 62) + ".go": false,
	} {
		if err := ValidFileName(name); (err == nil) != ok {
			t.Errorf("ValidFileName(%q) = %v, want ok=%v", name, err, ok)
		}
	}
}

func TestValidateFiles(t *testing.T) {
	var many []File
	for i := 0; i < MaxFiles+1; i++ {
		many = append(many, File{Path: "f" + strings.Repeat("a", i) + ".go"})
	}
	if err := ValidateFiles(many, nil); err == nil {
		t.Error("want an error past 32 files")
	}
	if err := ValidateFiles([]File{{Path: "a.go"}}, []File{{Path: "a.go"}}); err == nil {
		t.Error("want an error for a name shared by a file and a hidden file")
	}
	big := []File{{Path: "a.go", Data: make([]byte, MaxFilesBytes)}, {Path: "b.go", Data: []byte("x")}}
	if err := ValidateFiles(big, nil); err == nil {
		t.Error("want an error past 1 MiB of file data")
	}
}

func TestValidateJob(t *testing.T) {
	mut := map[string]func(j *Job){
		"bad id":            func(j *Job) { j.ID = "has space" },
		"bad profile":       func(j *Job) { j.Profile = "go" },
		"bad harness":       func(j *Job) { j.Harness = "func-json" },
		"bad mode":          func(j *Job) { j.Mode = "debug" },
		"bad output mode":   func(j *Job) { j.OutputMode = "" },
		"no files":          func(j *Job) { j.Files = nil },
		"bad file name":     func(j *Job) { j.Files[0].Path = "../evil.go" },
		"cpu too low":       func(j *Job) { j.Limits.Case.CPUms = 10 },
		"cpu too high":      func(j *Job) { j.Limits.Case.CPUms = 60_000 },
		"mem too high":      func(j *Job) { j.Limits.Case.MemMB = 4096 },
		"output zero":       func(j *Job) { j.Limits.Case.OutputKB = 0 },
		"compile over 15 s": func(j *Job) { j.Limits.Compile.CPUms = 20_000 },
		"duplicate case id": func(j *Job) { j.Cases[1].OpaqueID = j.Cases[0].OpaqueID },
		"unknown group":     func(j *Job) { j.Cases[0].Group = "bonus" },
		"groups out of order": func(j *Job) {
			j.Cases[0], j.Cases[3] = j.Cases[3], j.Cases[0]
		},
		"unknown stop rule":  func(j *Job) { j.StopGroupOn[GroupPerf] = "never" },
		"unknown stop group": func(j *Job) { j.StopGroupOn["bonus"] = StopTLE },
		"negative count":     func(j *Job) { j.Count = -1 },
		"bad test name":      func(j *Job) { j.Tests[0].Name = "Test Race" },
	}
	for name, f := range mut {
		t.Run(name, func(t *testing.T) {
			j := exampleJob()
			f(j)
			err := ValidateJob(j)
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("want a ValidationError, got %v", err)
			}
		})
	}
	if err := ValidateJob(exampleJob()); err != nil {
		t.Fatalf("the example job is valid: %v", err)
	}
}

func TestValidateResult(t *testing.T) {
	mut := map[string]func(r *Result, j *Job){
		"no boot epoch":        func(r *Result, j *Job) { r.Versions.BootEpoch = "" },
		"wrong profile":        func(r *Result, j *Job) { r.Versions.Profile = "cpp@g++14" },
		"missing case":         func(r *Result, j *Job) { r.Cases = r.Cases[:3] },
		"reordered cases":      func(r *Result, j *Job) { r.Cases[0], r.Cases[1] = r.Cases[1], r.Cases[0] },
		"unknown term":         func(r *Result, j *Job) { r.Cases[0].Term = "wa" },
		"signal without name":  func(r *Result, j *Job) { r.Cases[1].Signal = "" },
		"signal on ok":         func(r *Result, j *Job) { r.Cases[0].Signal = "SIGSYS" },
		"output over cap":      func(r *Result, j *Job) { r.Cases[0].Output = make([]byte, 65<<10) },
		"bytes in sha256 mode": func(r *Result, j *Job) { j.OutputMode = OutputSHA256 },
		"stdout in submit":     func(r *Result, j *Job) { r.Cases[0].Stdout = []byte("x") },
		"stderr over cap": func(r *Result, j *Job) {
			j.Mode = ModeRun
			r.Cases[0].Stderr = make([]byte, KeptStderrBytes+1)
		},
		"ran after CE":          func(r *Result, j *Job) { r.Compile.OK = false },
		"bad sha":               func(r *Result, j *Job) { r.Cases[0].OutputSHA256 = "xyz" },
		"idle kill without tle": func(r *Result, j *Job) { r.Cases[0].IdleKill = true },
		"unknown compile limit": func(r *Result, j *Job) { r.Compile.OK = false; r.Compile.Limit = "disk" },
		"unknown infra kind":    func(r *Result, j *Job) { r.Infra = &InfraError{Kind: "boom"} },
	}
	for name, f := range mut {
		t.Run(name, func(t *testing.T) {
			j, r := exampleJob(), exampleResult()
			f(r, j)
			if err := ValidateResult(j, r); err == nil {
				t.Fatal("want an error")
			}
		})
	}
	// An infra-only result is valid with just a kind and a boot epoch.
	infra := &Result{Infra: NewInfra(InfraSetup, "teardown"), Versions: Versions{BootEpoch: "h/1/00"}}
	if err := ValidateResult(exampleJob(), infra); err != nil {
		t.Errorf("infra-only result: %v", err)
	}
}

func TestInfraKinds(t *testing.T) {
	for _, k := range []InfraKind{InfraSetup, InfraRunnerOOM, InfraKilled, InfraJobTimeout, InfraSaturated} {
		if !k.Valid() {
			t.Errorf("%s should be valid", k)
		}
	}
	if InfraKind("oom").Valid() {
		t.Error("the set is closed")
	}
	b, _ := json.Marshal(&Result{Infra: NewInfra(InfraSaturated, "")})
	if !bytes.Contains(b, []byte(`"infra":{"kind":"saturated"}`)) {
		t.Errorf("saturated body shape changed: %s", b)
	}
}
