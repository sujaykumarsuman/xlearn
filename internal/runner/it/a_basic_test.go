//go:build linux && runner_it

package it

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

func TestReadyzProfilesHealthz(t *testing.T) {
	ensure(t)
	if code, _ := get(t, "/healthz", false); code != http.StatusOK {
		t.Fatalf("/healthz = %d", code)
	}
	code, body := get(t, "/readyz", false)
	if code != http.StatusOK {
		t.Fatalf("/readyz = %d %s", code, body)
	}
	t.Logf("/readyz (dev mode; security canaries only warn outside the pod): %s", body)
	if !bytes.Contains(body, []byte(`"cgroupfs_writable":"pass"`)) {
		t.Errorf("cgroupfs canary: %s", body)
	}
	p := getJSON[runnerapi.ProfilesResponse](t, "/v1/profiles")
	if p.Mode != "dev" || p.BootEpoch == "" || p.CanaryMedian <= 0 {
		t.Errorf("profiles: %+v", p)
	}
	found := false
	for _, pi := range p.Profiles {
		if pi.Profile == "testgo@0" {
			found = true
			if !strings.HasPrefix(pi.Toolchain, "go1.") || len(pi.ProfileSHA256) != 64 || pi.MemBaselineKB <= 0 {
				t.Errorf("testgo@0: %+v", pi)
			}
			metric(t, "testgo_mem_baseline_kb", pi.MemBaselineKB)
		}
	}
	if !found {
		t.Errorf("testgo@0 not listed: %+v", p.Profiles)
	}
	metric(t, "canary_median_us", p.CanaryMedian)
}

func TestEchoModes(t *testing.T) {
	ensure(t)
	job, in := newJob(t, "echo", "testgo@0", "hello", "", strings.Repeat("x", 5000))
	job.Mode = runnerapi.ModeRun
	res := run(t, job, in)
	if !res.Compile.OK {
		t.Fatalf("compile: %+v", res.Compile)
	}
	for i, c := range res.Cases {
		if c.Term != runnerapi.TermOK || !bytes.Equal(c.Output, in[i]) || c.OutputBytes != int64(len(in[i])) {
			t.Errorf("case %d: %+v", i, c)
		}
		if string(c.Stdout) != "ok\n" || string(c.Stderr) != "note\n" {
			t.Errorf("case %d: run mode keeps stdout/stderr: %q %q", i, c.Stdout, c.Stderr)
		}
		sum := sha256.Sum256(in[i])
		if c.OutputSHA256 != hex.EncodeToString(sum[:]) {
			t.Errorf("case %d: sha256 %s", i, c.OutputSHA256)
		}
	}
	if res.Versions.Profile != "testgo@0" || res.Versions.ProfileSHA == "" || res.Versions.BootEpoch == "" {
		t.Errorf("versions: %+v", res.Versions)
	}
	if res.Telemetry.SlotMs <= 0 || res.Telemetry.BusyMs <= 0 {
		t.Errorf("telemetry: %+v", res.Telemetry)
	}

	// Submit + sha256 mode: no stdout/stderr, no output bytes, only the hash and length.
	job, in = newJob(t, "echo", "testgo@0", "abc")
	job.OutputMode = runnerapi.OutputSHA256
	res = run(t, job, in)
	c := res.Cases[0]
	sum := sha256.Sum256([]byte("abc"))
	if c.Term != runnerapi.TermOK || len(c.Output) != 0 || c.Stdout != nil || c.OutputSHA256 != hex.EncodeToString(sum[:]) || c.OutputBytes != 3 {
		t.Errorf("sha256 mode: %+v", c)
	}
	assertClean(t)
}

func TestCompileErrorIsCE(t *testing.T) {
	ensure(t)
	job, in := newJob(t, "badcompile", "testgo@0", "x")
	res := run(t, job, in)
	if res.Compile.OK || res.Compile.Limit != "" {
		t.Fatalf("want a plain CE: %+v", res.Compile)
	}
	found := false
	for _, d := range res.Compile.Diags {
		if d.File == "main.go" && d.Line == 5 && strings.Contains(d.Msg, "x") {
			found = true
		}
	}
	if !found {
		t.Errorf("no positioned diagnostic: %+v", res.Compile.Diags)
	}
	if res.Cases[0].Term != runnerapi.TermNotRun {
		t.Errorf("cases after a CE are not_run: %+v", res.Cases[0])
	}
	assertClean(t)
}

// TestCompileCapIsCE: a compile over its CPU limit is a CE with limit "cpu" (L14), and its CPU
// is still charged in SlotMs. A 100 ms limit stands in for the 15 s cap (same code path).
func TestCompileCapIsCE(t *testing.T) {
	ensure(t)
	job, in := newJob(t, "echo", "testgo@0", "x")
	job.Limits.Compile.CPUms = 100
	res := run(t, job, in)
	if res.Compile.OK || res.Compile.Limit != runnerapi.CompileLimitCPU {
		t.Fatalf("want CE with limit cpu: %+v", res.Compile)
	}
	if res.Telemetry.SlotMs < 100 {
		t.Errorf("the compile's CPU counts in SlotMs: %+v", res.Telemetry)
	}
	assertClean(t)
}

// TestJailIdentity checks the jail from inside: a pool UID, no capabilities in any set, an
// empty bounding set, NO_NEW_PRIVS, a root holding only /job and /w, and no /proc.
func TestJailIdentity(t *testing.T) {
	ensure(t)
	job, in := newJob(t, "whoami", "testgo-open@0", "/job "+env.goroot+" /w")
	res := run(t, job, in)
	c := res.Cases[0]
	if c.Term != runnerapi.TermOK {
		t.Fatalf("whoami: %+v", c)
	}
	var who struct {
		UID         int               `json:"uid"`
		EUID        int               `json:"euid"`
		CapEff      uint64            `json:"cap_eff"`
		CapPrm      uint64            `json:"cap_prm"`
		CapInh      uint64            `json:"cap_inh"`
		CapBnd      uint64            `json:"cap_bnd"`
		CapgetErrno int               `json:"capget_errno"`
		NoNewPrivs  int               `json:"no_new_privs"`
		Root        []string          `json:"root"`
		Dev         []string          `json:"dev"`
		Proc        bool              `json:"proc"`
		Hostname    string            `json:"hostname"`
		Probes      map[string]string `json:"write_probes"`
	}
	if err := json.Unmarshal(c.Output, &who); err != nil {
		t.Fatalf("%v: %s", err, c.Output)
	}
	t.Logf("jail identity: %s", c.Output)
	if who.UID < uidLo || who.UID >= uidHi || who.EUID != who.UID {
		t.Errorf("uid %d is not a pool UID", who.UID)
	}
	if who.CapgetErrno != 0 || who.CapEff != 0 || who.CapPrm != 0 || who.CapInh != 0 || who.CapBnd != 0 {
		t.Errorf("capabilities left in the jail: %s", c.Output)
	}
	if who.NoNewPrivs != 1 {
		t.Errorf("NO_NEW_PRIVS not set")
	}
	if who.Proc {
		t.Errorf("/proc is mounted in the jail")
	}
	// /dev holds only the /dev/null bind; /job is the artifact, /w the per-case tmpfs, and the
	// profile's toolchain bind keeps its own path.
	top := strings.Split(strings.TrimPrefix(env.goroot, "/"), "/")[0]
	want := []string{"dev", "job", top, "w"}
	sort.Strings(want)
	if root := strings.Join(who.Root, ","); root != strings.Join(want, ",") {
		t.Errorf("jail root holds %q, want %q", root, strings.Join(want, ","))
	}
	// The read-only bind rule (MS_BIND|MS_REC|MS_NOSUID|MS_NODEV|MS_RDONLY, no MS_PRIVATE; t3
	// §16.2 block 3): the artifact and the toolchain refuse writes, /w takes them.
	for _, dir := range []string{"/job", env.goroot} {
		if !strings.Contains(who.Probes[dir], "read-only file system") {
			t.Errorf("bind %s is not read-only: %q", dir, who.Probes[dir])
		}
	}
	if who.Probes["/w"] != "rw" {
		t.Errorf("/w is not writable: %q", who.Probes["/w"])
	}
	if strings.Join(who.Dev, ",") != "null" {
		t.Errorf("jail /dev holds %v, want only null", who.Dev)
	}
	if who.Hostname != "xl" {
		t.Errorf("hostname %q (a fresh UTS namespace)", who.Hostname)
	}
	assertClean(t)
}

// TestFrontIdentity checks the front from outside: UID 65532, no capabilities, an empty
// bounding set, NO_NEW_PRIVS.
func TestFrontIdentity(t *testing.T) {
	ensure(t)
	ents, _ := os.ReadDir("/proc")
	found := false
	for _, e := range ents {
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		cmdline, _ := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if !bytes.HasPrefix(cmdline, []byte(env.bin+"\x00front")) {
			continue
		}
		found = true
		st, _ := os.ReadFile(filepath.Join("/proc", e.Name(), "status"))
		s := string(st)
		for _, want := range []string{"CapInh:\t0000000000000000", "CapPrm:\t0000000000000000", "CapEff:\t0000000000000000",
			"CapBnd:\t0000000000000000", "CapAmb:\t0000000000000000", "NoNewPrivs:\t1", "Uid:\t65532\t65532\t65532\t65532"} {
			if !strings.Contains(s, want) {
				t.Errorf("front status lacks %q:\n%s", want, s)
			}
		}
	}
	if !found {
		t.Fatal("no front process found")
	}
}
