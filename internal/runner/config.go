// Package runner holds the xlearn-runner's configuration. The runner itself is split into a
// capability-holding spawner (internal/runner/spawner, PID 1) and a capability-less front
// (internal/runner/front); see docs/architecture/runner.md and ADR-0030.
package runner

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Mode is RUNNER_MODE. dev (compose, CI, a dev VM) only downgrades failing *security*
// canaries to warnings; prod also refuses jobs (infra: setup) while one fails. Judge refuses a
// dev runner in production (m3-06).
type Mode string

// Modes.
const (
	ModeProd Mode = "prod"
	ModeDev  Mode = "dev"
)

// SpawnPath selects how a jail enters its case cgroup (t3 §16.1: both work on 6.8).
type SpawnPath string

// Spawn paths. CloneIntoCgroup is the default (clone3 + a cgroup fd); CgroupProcs writes the
// child's pid into cgroup.procs, gated on a sync pipe, before it execs.
const (
	SpawnCloneIntoCgroup SpawnPath = "clone_into_cgroup"
	SpawnCgroupProcs     SpawnPath = "cgroup_procs"
)

// Defaults (docs/architecture/runner.md "Config").
const (
	DefaultAddr         = ":8090"
	DefaultTokenFile    = "/var/run/secrets/runner-auth/token"
	DefaultMaxJobs      = 500
	DefaultMaxAge       = 6 * time.Hour
	DefaultLogLevel     = "info"
	DefaultCgroupRoot   = "/sys/fs/cgroup"
	DefaultJailDir      = "/jail"
	DefaultDrainTimeout = 60 * time.Second
	DefaultFrontUID     = 65532
	DefaultCanaryEvery  = 5 * time.Minute
	UnknownImageDigest  = "unknown"
)

// Config is the resolved runner configuration (env, 12-factor like every xLearn service).
type Config struct {
	// Addr is the front's listen address (RUNNER_ADDR, default :8090; mi-10's Service port).
	Addr string
	// TokenFile is the bearer token file (RUNNER_TOKEN_FILE; the Secret runner-auth, key token,
	// that mi-10 mounts). The spawner reads it once and hands it to the front over a pipe; the
	// token is compared in constant time and never logged.
	TokenFile string
	// Mode is RUNNER_MODE (prod default | dev).
	Mode Mode
	// ImageDigest is RUNNER_IMAGE_DIGEST, reported in Versions.ImageDigest ("unknown" if unset).
	ImageDigest string
	// MaxJobs is RUNNER_MAX_JOBS: the front drains and exits after this many jobs (rotation).
	MaxJobs int
	// MaxAge is RUNNER_MAX_AGE: the front drains and exits after this long (rotation).
	MaxAge time.Duration
	// LogLevel is LOG_LEVEL.
	LogLevel string

	// Internal knobs (tests and dev VMs; the pod keeps the defaults).

	// CgroupRoot is the container's cgroup v2 mount (RUNNER_CGROUP_ROOT).
	CgroupRoot string
	// JailDir holds the per-slot mount points (RUNNER_JAIL_DIR; the pod's /jail emptyDir, the
	// only place the AppArmor profile admits mounts).
	JailDir string
	// DrainTimeout bounds a drain before unfinished jobs answer killed (RUNNER_DRAIN_TIMEOUT).
	DrainTimeout time.Duration
	// FrontUID is the fixed non-zero UID the front runs as (RUNNER_FRONT_UID).
	FrontUID int
	// SpawnPath is RUNNER_SPAWN_PATH (clone_into_cgroup | cgroup_procs).
	SpawnPath SpawnPath
	// CanaryEvery is the idle canary period (RUNNER_CANARY_EVERY, default 5m).
	CanaryEvery time.Duration
}

// Load reads the environment and validates it.
func Load() (Config, error) {
	c := Config{
		Addr:         env("RUNNER_ADDR", DefaultAddr),
		TokenFile:    env("RUNNER_TOKEN_FILE", DefaultTokenFile),
		Mode:         Mode(env("RUNNER_MODE", string(ModeProd))),
		ImageDigest:  env("RUNNER_IMAGE_DIGEST", UnknownImageDigest),
		LogLevel:     env("LOG_LEVEL", DefaultLogLevel),
		CgroupRoot:   env("RUNNER_CGROUP_ROOT", DefaultCgroupRoot),
		JailDir:      env("RUNNER_JAIL_DIR", DefaultJailDir),
		SpawnPath:    SpawnPath(env("RUNNER_SPAWN_PATH", string(SpawnCloneIntoCgroup))),
		MaxJobs:      DefaultMaxJobs,
		MaxAge:       DefaultMaxAge,
		DrainTimeout: DefaultDrainTimeout,
		FrontUID:     DefaultFrontUID,
		CanaryEvery:  DefaultCanaryEvery,
	}
	var err error
	if c.MaxJobs, err = envInt("RUNNER_MAX_JOBS", DefaultMaxJobs); err != nil {
		return c, err
	}
	if c.FrontUID, err = envInt("RUNNER_FRONT_UID", DefaultFrontUID); err != nil {
		return c, err
	}
	if c.MaxAge, err = envDuration("RUNNER_MAX_AGE", DefaultMaxAge); err != nil {
		return c, err
	}
	if c.DrainTimeout, err = envDuration("RUNNER_DRAIN_TIMEOUT", DefaultDrainTimeout); err != nil {
		return c, err
	}
	if c.CanaryEvery, err = envDuration("RUNNER_CANARY_EVERY", DefaultCanaryEvery); err != nil {
		return c, err
	}
	return c, c.Validate()
}

// Validate checks ranges and enums.
func (c Config) Validate() error {
	switch c.Mode {
	case ModeProd, ModeDev:
	default:
		return fmt.Errorf("RUNNER_MODE=%q: want prod or dev", c.Mode)
	}
	switch c.SpawnPath {
	case SpawnCloneIntoCgroup, SpawnCgroupProcs:
	default:
		return fmt.Errorf("RUNNER_SPAWN_PATH=%q: want clone_into_cgroup or cgroup_procs", c.SpawnPath)
	}
	if c.MaxJobs < 1 {
		return fmt.Errorf("RUNNER_MAX_JOBS must be ≥ 1")
	}
	if c.MaxAge < time.Minute {
		return fmt.Errorf("RUNNER_MAX_AGE must be ≥ 1m")
	}
	if c.DrainTimeout < time.Second || c.DrainTimeout > 65*time.Second {
		// The pod's grace period is 70 s (ADR-0035 §5): a drain must end inside it.
		return fmt.Errorf("RUNNER_DRAIN_TIMEOUT must be within 1s..65s")
	}
	if c.FrontUID <= 0 || c.FrontUID >= 65536 {
		return fmt.Errorf("RUNNER_FRONT_UID must be a non-zero UID inside the pod's 65536-UID map")
	}
	if c.CanaryEvery < 10*time.Second {
		return fmt.Errorf("RUNNER_CANARY_EVERY must be ≥ 10s")
	}
	if !strings.HasPrefix(c.JailDir, "/") || !strings.HasPrefix(c.CgroupRoot, "/") {
		return fmt.Errorf("RUNNER_JAIL_DIR and RUNNER_CGROUP_ROOT must be absolute")
	}
	return nil
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %w", key, v, err)
	}
	return n, nil
}

func envDuration(key string, def time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %w", key, v, err)
	}
	return d, nil
}
