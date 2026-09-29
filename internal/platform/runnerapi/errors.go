package runnerapi

import "fmt"

// InfraKind is the closed set of typed infra errors (t3 §5.9). None of them is a learner
// verdict: judge maps each to a retry, a re-queue or inconclusive, never to a class.
type InfraKind string

const (
	// InfraSetup: a spawner syscall failed before learner code ran, or a teardown assertion
	// failed (the slot failed closed). Emitted by the runner (200 + Result). Judge also uses
	// it when ValidateResult rejects a Result. ≤ 1 retry.
	InfraSetup InfraKind = "setup"
	// InfraRunnerOOM: the runner container died of OOM or restarted unexplained. Never emitted
	// by the runner: judge infers it from a dropped connection plus a changed BootEpoch. Counts
	// toward the poison pill.
	InfraRunnerOOM InfraKind = "runner_oom"
	// InfraKilled: the runner was draining (SIGTERM, rotation, rollout) and could not finish
	// the job inside the grace period. Emitted by the runner (503 + Retry-After). Judge
	// re-queues it as saturated: no retry spent, no poison count.
	InfraKilled InfraKind = "killed"
	// InfraJobTimeout: the runner's own job deadline (170 s: compile 15 s + tests 45 s + the
	// quiet re-run allowance 60 s + 45 s + 5 s slack) passed. A runner-fault backstop, never a
	// learner verdict. Emitted by the runner (200 + Result). Counts toward the poison pill.
	InfraJobTimeout InfraKind = "job_timeout"
	// InfraSaturated: both slots busy, draining, or a quiet re-run holds exclusivity. Emitted
	// by the runner (503 + Retry-After). Judge re-queues with run_after. A 503 is never a verdict.
	InfraSaturated InfraKind = "saturated"
)

// infraKinds is the closed set.
var infraKinds = map[InfraKind]bool{
	InfraSetup: true, InfraRunnerOOM: true, InfraKilled: true, InfraJobTimeout: true, InfraSaturated: true,
}

// Valid reports whether k is in the closed set.
func (k InfraKind) Valid() bool { return infraKinds[k] }

// InfraError is Result.Infra. Detail is a short runner-generated reason for ops (never learner
// bytes); judge logs it and never shows it to a learner.
type InfraError struct {
	Kind   InfraKind `json:"kind"`
	Detail string    `json:"detail,omitempty"`
}

func (e *InfraError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("runner infra error: %s", e.Kind)
	}
	return fmt.Sprintf("runner infra error: %s (%s)", e.Kind, e.Detail)
}

// NewInfra is a small constructor.
func NewInfra(kind InfraKind, detail string) *InfraError {
	return &InfraError{Kind: kind, Detail: detail}
}
