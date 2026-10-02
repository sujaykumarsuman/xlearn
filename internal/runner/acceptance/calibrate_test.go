//go:build runner_acceptance

package acceptance

// The calibration (CALIBRATE=1; t3 §7.3, ADR-0030 §3): 5 kernels × 30 runs × Go/C++/Python on
// one slot, sequentially (10 runs per job keeps each job far inside the L14 45 s tests cap). It
// reports each kernel's median CPU ms and CV per profile, the node's steal during the window
// (Telemetry.StealPct), baseline@1 keyed to each profile_sha256, speed_index (the geometric mean
// of the go@1.26 medians: TLs computed on another host scale by speed_index(prod) /
// speed_index(there)) and the kernel-derived language ratios. It sets nothing: the TL-baselines
// doc (docs/architecture/runner-tl-baselines.md) records the CI column; mi-10 records production.

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

const (
	calRuns     = 30
	calPerJob   = 10
	calBaseline = "baseline@1"
)

// kernels are t3 §7.3's five, with their sizes (the same n in every language, so the ratios
// compare the same work).
var kernels = []struct {
	name string
	n    int
}{
	{"intloop", 20_000_000},
	{"sort", 1_000_000},
	{"maphash", 1_000_000},
	{"alloc", 1_000_000},
	{"bfs", 200_000},
}

var calTL = map[string]int64{"go": 5000, "cpp": 5000, "python": 10000}

type calibration struct {
	Version      string           `json:"version"`
	Runs         int              `json:"runs"`
	Profiles     []calProfile     `json:"profiles"`
	SpeedIndexMs float64          `json:"speed_index_ms"`
	Ratios       []langRatio      `json:"ratios"`
	StealMeanPct float64          `json:"steal_mean_pct"`
	StealMaxPct  float64          `json:"steal_max_pct"`
	Checksums    map[string]int64 `json:"checksums"`
}

type calProfile struct {
	Profile       string               `json:"profile"`
	ProfileSHA256 string               `json:"profile_sha256"`
	Toolchain     string               `json:"toolchain"`
	MemBaselineKB int64                `json:"mem_baseline_kb"`
	Kernels       map[string]calKernel `json:"kernels"`
}

type calKernel struct {
	N        int     `json:"n"`
	MedianMs float64 `json:"median_ms"`
	CV       float64 `json:"cv"`
	MinMs    float64 `json:"min_ms"`
	MaxMs    float64 `json:"max_ms"`
}

type langRatio struct {
	Profile   string             `json:"profile"`
	PerKernel map[string]float64 `json:"per_kernel"`
	// GeoMean and Max are over the five kernels. intloop is a pure interpreter loop, so for
	// Python it bounds the ratio from above; m3-04's served multipliers come from the reference
	// perf cases instead. Both are informational: mi-10 calibrates the multipliers.
	GeoMean float64 `json:"geomean"`
	Max     float64 `json:"max"`
	// Suggested is max(1, the max ratio rounded up to the next 0.5): m3-04's rounding rule.
	Suggested float64 `json:"suggested_tl_multiplier"`
}

func (l langRatio) perKernel() string {
	var parts []string
	for _, k := range kernels {
		parts = append(parts, fmt.Sprintf("%s %.2f", k.name, l.PerKernel[k.name]))
	}
	return strings.Join(parts, ", ")
}

func (s *suite) calibrate(t *testing.T, r *section) {
	cal := &calibration{Version: calBaseline, Runs: calRuns, Checksums: map[string]int64{}}
	var steal []float64
	medians := map[string]map[string]float64{}
	for _, lang := range langs {
		p := s.profiles[lang]
		cp := calProfile{Profile: p.Profile, ProfileSHA256: p.ProfileSHA256, Toolchain: p.Toolchain,
			MemBaselineKB: p.MemBaselineKB, Kernels: map[string]calKernel{}}
		medians[lang] = map[string]float64{}
		for _, k := range kernels {
			src := source(t, "kernels", lang, k.name)
			var ms []float64
			for done := 0; done < calRuns; done += calPerJob {
				raws := make([][]byte, calPerJob)
				for i := range raws {
					raws[i] = argsJSON(k.n)
				}
				j := s.newJob(t, "CAL", lang, k.name, kernelSig, src, raws, runnerapi.ModeSubmit, limits{calTL[lang], 512}, nil)
				res := s.run(t, j)
				if !r.expect(t, res.Compile.OK, fmt.Sprintf("%s %s compiles", lang, k.name), "%+v", res.Compile) {
					break
				}
				steal = append(steal, res.Telemetry.StealPct)
				for _, c := range res.Cases {
					sum, err := kernelOut(c)
					if !r.expect(t, c.Term == runnerapi.TermOK && err == nil, fmt.Sprintf("%s %s runs", lang, k.name), "%s %v", describe(c), err) {
						continue
					}
					if want, ok := cal.Checksums[k.name]; !ok {
						cal.Checksums[k.name] = sum
					} else if sum != want {
						r.expect(t, false, fmt.Sprintf("%s %s checksum equals the other languages'", lang, k.name), "%d vs %d", sum, want)
					}
					ms = append(ms, float64(c.CPUms))
				}
			}
			med, cv, lo, hi := stats(ms)
			cp.Kernels[k.name] = calKernel{N: k.n, MedianMs: med, CV: round(cv, 4), MinMs: lo, MaxMs: hi}
			medians[lang][k.name] = med
			r.metric(fmt.Sprintf("%s_%s_median_ms", lang, k.name), med)
		}
		cal.Profiles = append(cal.Profiles, cp)
	}
	logSum, n := 0.0, 0
	for _, k := range kernels {
		if m := medians["go"][k.name]; m > 0 {
			logSum += math.Log(m)
			n++
		}
	}
	if n > 0 {
		cal.SpeedIndexMs = round(math.Exp(logSum/float64(n)), 2)
	}
	for _, lang := range []string{"cpp", "python"} {
		lr := langRatio{Profile: s.profiles[lang].Profile, PerKernel: map[string]float64{}}
		logs := 0.0
		for _, k := range kernels {
			g := math.Max(medians["go"][k.name], 1)
			ratio := medians[lang][k.name] / g
			lr.PerKernel[k.name] = round(ratio, 2)
			lr.Max = math.Max(lr.Max, round(ratio, 2))
			logs += math.Log(math.Max(ratio, 1e-3))
		}
		lr.GeoMean = round(math.Exp(logs/float64(len(kernels))), 2)
		lr.Suggested = math.Max(1, math.Ceil(lr.Max*2)/2)
		cal.Ratios = append(cal.Ratios, lr)
	}
	sort.Float64s(steal)
	if len(steal) > 0 {
		var sum float64
		for _, x := range steal {
			sum += x
		}
		cal.StealMeanPct, cal.StealMaxPct = round(sum/float64(len(steal)), 3), steal[len(steal)-1]
	}
	s.report.Calibration = cal
	r.add("calibration complete", len(cal.Profiles) == len(langs) && cal.SpeedIndexMs > 0,
		fmt.Sprintf("speed_index %.2f ms, steal mean %.2f%%", cal.SpeedIndexMs, cal.StealMeanPct))
}

func kernelOut(c runnerapi.CaseResult) (int64, error) {
	in, err := harness.DecodeInput(kernelSig, argsJSON(1))
	if err != nil {
		return 0, err
	}
	v, err := harness.DecodeOutputFor(kernelSig, in, c.Output)
	if err != nil {
		return 0, err
	}
	return v.Int, nil
}
