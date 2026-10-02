//go:build runner_acceptance

package acceptance

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The Makefile passes every knob as a flag (make runner-acceptance; the token file may be a
// process substitution, which only the test binary itself can read, so the Makefile builds the
// test binary with `go test -c` and runs it directly).
var (
	flagURL         = flag.String("url", "", "the runner's base URL (RUNNER_URL), e.g. http://127.0.0.1:18090")
	flagTokenFile   = flag.String("token-file", "", "the bearer token file (RUNNER_TOKEN_FILE)")
	flagSubset      = flag.String("subset", "", "full | prod (SUBSET; required, no default)")
	flagRequireProd = flag.String("require-prod", "", "1: run section I; a dev-mode runner fails the run (REQUIRE_PROD)")
	flagCalibrate   = flag.String("calibrate", "", "1: run the calibration (CALIBRATE)")
	flagReport      = flag.String("report", "", "the JSON report path (default <module>/bin/runner-acceptance-<subset>-<utc>.json)")
	flagRepo        = flag.String("repo", "", "the module root (default: found upward from the working directory)")
	flagRotation    = flag.Duration("rotation-timeout", 6*time.Minute, "how long to wait for a new BootEpoch after a rotation")
	// flagOnly is a debugging aid (ONLY=A,B): run just these sections of the SUBSET. A run with
	// it is never a release gate; the report records it.
	flagOnly = flag.String("only", "", "comma-separated section ids to run (debugging only)")
)

const usage = "usage: make runner-acceptance RUNNER_URL=… RUNNER_TOKEN_FILE=… SUBSET=full|prod [REQUIRE_PROD=1] [CALIBRATE=1]"

// config is the validated run configuration.
type config struct {
	url         string
	token       string
	subset      string
	requireProd bool
	calibrate   bool
	report      string
	repo        string
}

var cfg config

func TestMain(m *testing.M) {
	flag.Parse()
	if err := configure(); err != nil {
		fmt.Fprintln(os.Stderr, "runner-acceptance:", err)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	os.Exit(m.Run())
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes":
		return true
	}
	return false
}

func configure() error {
	switch *flagSubset {
	case "full", "prod":
		cfg.subset = *flagSubset
	case "":
		return errors.New("SUBSET is required (full or prod): there is no default, so nobody runs the wrong set on production by omission")
	default:
		return fmt.Errorf("SUBSET=%q: want full or prod", *flagSubset)
	}
	u, err := url.Parse(strings.TrimRight(*flagURL, "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("RUNNER_URL=%q: want http(s)://host:port", *flagURL)
	}
	cfg.url = u.String()
	if *flagTokenFile == "" {
		return errors.New("RUNNER_TOKEN_FILE is required")
	}
	b, err := os.ReadFile(*flagTokenFile)
	if err != nil {
		return fmt.Errorf("read RUNNER_TOKEN_FILE: %w", err)
	}
	if cfg.token = strings.TrimSpace(string(b)); len(cfg.token) < 16 {
		return errors.New("RUNNER_TOKEN_FILE holds no token (≥ 16 bytes)")
	}
	cfg.requireProd, cfg.calibrate = truthy(*flagRequireProd), truthy(*flagCalibrate)
	if cfg.repo = *flagRepo; cfg.repo == "" {
		if cfg.repo, err = moduleRoot(); err != nil {
			return err
		}
	}
	cfg.report = *flagReport
	if cfg.report == "" {
		cfg.report = filepath.Join(cfg.repo, "bin", fmt.Sprintf("runner-acceptance-%s-%s.json", cfg.subset, time.Now().UTC().Format("20060102T150405Z")))
	}
	return nil
}

// moduleRoot walks up from the working directory to the go.mod of this module.
func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if b, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil && strings.Contains(string(b), "module github.com/sujaykumarsuman/xlearn\n") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("can't find the xlearn module root (pass -repo)")
		}
		dir = parent
	}
}

// TestAcceptance runs the sections in order. A stop condition aborts the run (every later
// section is skipped and the report says what ran); an ordinary failed check fails its section
// and the run carries on.
func TestAcceptance(t *testing.T) {
	s := newSuite(t)
	defer s.finish(t)
	if !s.start(t) {
		return
	}
	sections := []struct {
		id, name string
		subsets  string
		run      func(*testing.T, *section)
	}{
		{"I0", "prod-only pre-check (REQUIRE_PROD)", "full prod", s.sectionIPre},
		{"A", "contract", "full prod", s.sectionA},
		{"B", "network", "full prod", s.sectionB},
		{"C", "syscalls", "full prod", s.sectionC},
		{"D", "P2 corpus", "full prod", s.sectionD},
		{"E", "cross-job markers", "full prod", s.sectionE},
		{"F", "cleanup", "full prod", s.sectionF},
		{"G", "references", "full", s.sectionG},
		{"I", "prod-only (REQUIRE_PROD)", "full prod", s.sectionI},
		{"CAL", "calibration (CALIBRATE)", "full prod", s.calibrate},
		{"H", "canary", "full prod", s.sectionH},
	}
	for _, sec := range sections {
		r := s.report.newSection(sec.id, sec.name)
		switch {
		case !strings.Contains(sec.subsets, cfg.subset):
			r.Skipped = "not in SUBSET=" + cfg.subset
			continue
		case *flagOnly != "" && !contains(strings.Split(*flagOnly, ","), sec.id):
			r.Skipped = "not in ONLY=" + *flagOnly + " (a debugging run, never a release gate)"
			s.report.Only = *flagOnly
			continue
		case (sec.id == "I0" || sec.id == "I") && !cfg.requireProd:
			r.Skipped = "REQUIRE_PROD not set"
			continue
		case sec.id == "CAL" && !cfg.calibrate:
			r.Skipped = "CALIBRATE not set"
			continue
		case s.aborted != "":
			r.Skipped = "aborted: " + s.aborted
			continue
		}
		t0 := time.Now()
		t.Run(sec.id+"_"+strings.Fields(sec.name)[0], func(t *testing.T) {
			defer func() { r.DurationS = time.Since(t0).Seconds() }()
			sec.run(t, r)
		})
		r.done()
	}
}
