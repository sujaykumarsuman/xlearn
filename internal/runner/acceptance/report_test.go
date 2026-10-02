//go:build runner_acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// Report is the JSON report (schema runner-acceptance@1), written to bin/ (never committed).
type Report struct {
	Schema      string    `json:"schema"`
	Started     time.Time `json:"started"`
	Finished    time.Time `json:"finished"`
	Subset      string    `json:"subset"`
	RequireProd bool      `json:"require_prod"`
	Calibrate   bool      `json:"calibrate"`
	// Target is the runner's host:port (never the token).
	Target      string                  `json:"target"`
	Runner      string                  `json:"runner_version"`
	ImageDigest string                  `json:"image_digest"`
	Mode        string                  `json:"mode"`
	CPUModel    string                  `json:"cpu_model,omitempty"`
	Profiles    []runnerapi.ProfileInfo `json:"profiles"`
	Sections    []*section              `json:"sections"`
	Rotations   rotations               `json:"rotations"`
	OOM         oomReport               `json:"oom"`
	Canary      canaryReport            `json:"canary"`
	Calibration *calibration            `json:"calibration,omitempty"`
	Jobs        int                     `json:"jobs"`
	Cases       int                     `json:"cases"`
	Aborted     string                  `json:"aborted,omitempty"`
	// Only is set for a debugging run of some sections (ONLY=…): never a release gate.
	Only string `json:"only,omitempty"`
	Pass bool   `json:"pass"`
}

type section struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Pass      bool           `json:"pass"`
	Skipped   string         `json:"skipped,omitempty"`
	DurationS float64        `json:"duration_s"`
	Checks    []check        `json:"checks"`
	Metrics   map[string]any `json:"metrics,omitempty"`
}

type check struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail,omitempty"`
}

type rotations struct {
	// Planned is the number of jobs the plan expects to end in SIGSYS (each rotates the runner).
	Planned int `json:"planned"`
	// Expected is the number of rotations the suite can explain: jobs that came back with a
	// SIGSYS case, plus RUNNER_MAX_JOBS rotations. It must equal Observed.
	Expected int             `json:"expected"`
	Observed int             `json:"observed"`
	Events   []rotationEvent `json:"events"`
}

type rotationEvent struct {
	Section string  `json:"section"`
	Job     string  `json:"job"`
	Reason  string  `json:"reason"`
	From    string  `json:"from"`
	To      string  `json:"to"`
	WaitS   float64 `json:"wait_s"`
}

type oomReport struct {
	// ContainerLocalDelta is the container cgroup's memory.events.local oom_kill delta summed
	// over every epoch: INV-14 keeps it at 0, and any rise aborts the run.
	ContainerLocalDelta int64 `json:"container_oom_kill_local_delta"`
	// ContainerHierDelta counts every case OOM (hierarchical memory.events), for reference.
	ContainerHierDelta int64 `json:"container_oom_kill_hierarchical_delta"`
}

type canaryReport struct {
	Samples   int     `json:"samples"`
	MedianUs  float64 `json:"median_us"`
	MinUs     int64   `json:"min_us"`
	MaxUs     int64   `json:"max_us"`
	CV        float64 `json:"cv"`
	Threshold string  `json:"thresholds"`
}

func (r *Report) newSection(id, name string) *section {
	s := &section{ID: id, Name: name, Pass: true, Metrics: map[string]any{}}
	r.Sections = append(r.Sections, s)
	return s
}

func (s *section) add(name string, pass bool, detail string) {
	s.Checks = append(s.Checks, check{Name: name, Pass: pass, Detail: detail})
	if !pass {
		s.Pass = false
	}
}

func (s *section) done() {
	if s.Skipped != "" {
		s.Pass = true
	}
}

// expect records a check and fails the test when it does not hold.
func (s *section) expect(t *testing.T, ok bool, name, format string, args ...any) bool {
	t.Helper()
	detail := fmt.Sprintf(format, args...)
	s.add(name, ok, detail)
	if !ok {
		t.Errorf("%s: %s", name, detail)
	}
	return ok
}

func (s *section) metric(name string, v any) { s.Metrics[name] = v }

// stats returns the median, the coefficient of variation, the min and the max.
func stats(xs []float64) (median, cv, lo, hi float64) {
	if len(xs) == 0 {
		return 0, 0, 0, 0
	}
	ys := append([]float64{}, xs...)
	sort.Float64s(ys)
	n := len(ys)
	if n%2 == 1 {
		median = ys[n/2]
	} else {
		median = (ys[n/2-1] + ys[n/2]) / 2
	}
	var sum, sq float64
	for _, y := range ys {
		sum += y
	}
	mean := sum / float64(n)
	for _, y := range ys {
		sq += (y - mean) * (y - mean)
	}
	if mean > 0 && n > 1 {
		cv = math.Sqrt(sq/float64(n-1)) / mean
	}
	return median, cv, ys[0], ys[n-1]
}

func round(x float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(x*p) / p
}

// write saves the JSON report and returns the Markdown summary (also saved beside it).
func (r *Report) write() (string, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.report), 0o755); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(cfg.report, append(b, '\n'), 0o644); err != nil {
		return "", err
	}
	md := r.markdown()
	return md, os.WriteFile(strings.TrimSuffix(cfg.report, ".json")+".md", []byte(md), 0o644)
}

func targetOf(raw string) string {
	if u, err := url.Parse(raw); err == nil {
		return u.Host
	}
	return "?"
}

func (r *Report) markdown() string {
	var b strings.Builder
	verdict := "PASS"
	if !r.Pass {
		verdict = "FAIL"
	}
	fmt.Fprintf(&b, "### Runner acceptance: %s (SUBSET=%s", verdict, r.Subset)
	if r.RequireProd {
		b.WriteString(" REQUIRE_PROD=1")
	}
	if r.Calibrate {
		b.WriteString(" CALIBRATE=1")
	}
	if r.Only != "" {
		b.WriteString(" ONLY=" + r.Only + ": a debugging run, not a release gate")
	}
	fmt.Fprintf(&b, ")\n\n- Runner `%s`, mode `%s`, image `%s`, %s\n", r.Runner, r.Mode, r.ImageDigest, r.CPUModel)
	fmt.Fprintf(&b, "- %d jobs, %d cases, %.0f s\n", r.Jobs, r.Cases, r.Finished.Sub(r.Started).Seconds())
	fmt.Fprintf(&b, "- Rotations: expected %d (planned SIGSYS jobs %d), observed %d\n", r.Rotations.Expected, r.Rotations.Planned, r.Rotations.Observed)
	fmt.Fprintf(&b, "- Container `oom_kill` (local) delta %d; case OOMs (hierarchical) %d\n", r.OOM.ContainerLocalDelta, r.OOM.ContainerHierDelta)
	fmt.Fprintf(&b, "- Canary: median %.0f µs, CV %.3f over %d samples (%s)\n", r.Canary.MedianUs, r.Canary.CV, r.Canary.Samples, r.Canary.Threshold)
	if r.Aborted != "" {
		fmt.Fprintf(&b, "- **Aborted:** %s\n", r.Aborted)
	}
	b.WriteString("\n| Profile | Toolchain | profile_sha256 | mem baseline | tl_multiplier |\n|---|---|---|---|---|\n")
	for _, p := range r.Profiles {
		fmt.Fprintf(&b, "| `%s` | %s | `%s` | %d KiB | %v (calibrated %v) |\n", p.Profile, p.Toolchain, short(p.ProfileSHA256), p.MemBaselineKB, p.TLMultiplier, p.Calibrated)
	}
	b.WriteString("\n| Section | Result | Checks | Time |\n|---|---|---|---|\n")
	for _, s := range r.Sections {
		res := "✅"
		switch {
		case s.Skipped != "":
			res = "— " + s.Skipped
		case !s.Pass:
			res = "❌"
		}
		failed := 0
		for _, c := range s.Checks {
			if !c.Pass {
				failed++
			}
		}
		fmt.Fprintf(&b, "| %s · %s | %s | %d (%d failed) | %.0f s |\n", s.ID, s.Name, res, len(s.Checks), failed, s.DurationS)
	}
	for _, s := range r.Sections {
		for _, c := range s.Checks {
			if !c.Pass {
				fmt.Fprintf(&b, "\n- ❌ %s · %s: %s", s.ID, c.Name, c.Detail)
			}
		}
	}
	if c := r.Calibration; c != nil {
		fmt.Fprintf(&b, "\n\n#### Calibration (%d runs per kernel, steal mean %.2f%% max %.2f%%)\n\n", c.Runs, c.StealMeanPct, c.StealMaxPct)
		b.WriteString("| Kernel | n |")
		for _, p := range c.Profiles {
			fmt.Fprintf(&b, " %s median ms (CV) |", p.Profile)
		}
		b.WriteString("\n|---|---|")
		for range c.Profiles {
			b.WriteString("---|")
		}
		b.WriteString("\n")
		for _, k := range kernels {
			fmt.Fprintf(&b, "| %s | %d |", k.name, k.n)
			for _, p := range c.Profiles {
				kr := p.Kernels[k.name]
				fmt.Fprintf(&b, " %.1f (%.3f) |", kr.MedianMs, kr.CV)
			}
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "\n- speed_index (geometric mean of the go@1.26 medians): **%.2f ms**\n", c.SpeedIndexMs)
		for _, lr := range c.Ratios {
			fmt.Fprintf(&b, "- %s / go: geomean %.2f, max %.2f (max rule → %.1f; informational) — per kernel: %s\n",
				lr.Profile, lr.GeoMean, lr.Max, lr.Suggested, lr.perKernel())
		}
	}
	b.WriteString("\n")
	return b.String()
}

func short(h string) string {
	if len(h) > 12 {
		return h[:12] + "…"
	}
	return h
}
