# Prompt — Sprint m3-03 · Runner core: supervisor, jail, cgroups, API

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-03.md`](../sprints/sprint-m3-03.md)   ·   **Milestone:** M3 (runner track; code for rollout step MI-12)   ·   **Prereqs:** [spk-01](../sprints/sprint-spk-01.md), [spk-02](../sprints/sprint-spk-02.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync.
- The plan: [`../sprints/sprint-m3-03.md`](../sprints/sprint-m3-03.md). The API table, cgroup layout, kill policy and corpus are spelled out there.
- **The spike results:** [`../research/t3-sandbox.md`](../research/t3-sandbox.md) **§16.1–§16.4** (mechanism, SETPCAP, spawn path, allowlists, final host files, the MI-10 verdict and the "proposed ADR-0030 deltas" list). This sprint builds on what they chose, not on the draft.
- [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md) (all of it; you accept it in step 2) and [ADR-0035 §4 (L14), §5, §6](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) (the amendments you fold).
- [t3](../research/t3-sandbox.md) §2.3–§2.4 (trust boundaries, rule R-FS, attack table), **§5.2–§5.11** (process model, API, lifecycle, measurement, `throttled`, typed errors, cleanup invariants, the T4 §11.1 checklist), §7.1–§7.2 (cgroup layout, TL policy), §9 (P2 corpus), §12 (D20–D23 override the body), §15 (S0: CPU time includes steal).
- [t4 §11.1](../research/t4-judge-contract.md#111-t3-the-runner-contract-not-the-technology) (the contract) and [§2.6](../research/t4-judge-contract.md#26-evaluation) (case order, hidden-case execution).
- [ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams) (runner stream: a major is a contract break), [rollout §2.2, §5, §12](../rollout-plan.md).
- Neighbours: [m3-04](../sprints/sprint-m3-04.md) (fills the profiles), [m3-06](../sprints/sprint-m3-06.md) (judge's client), [m3-15](../sprints/sprint-m3-15.md) (image, acceptance), [mi-10](../sprints/sprint-mi-10.md) (what the HelmRelease expects: port, probes, token path).
- Code: `cmd/review/main.go` (service main pattern, `-version`), `internal/platform/{config,slogx,httpx,health}`, `deploy/version_test.go`, `.github/workflows/ci.yml`, `Makefile`, `go.mod`, `docs/architecture/services.md`, `docs/adr/README.md`.

## Context

v2 runs untrusted learner code (Go, C++, Python; D20) on the single production node. T3/ADR-0030 chose a thin Go runner:
a user-namespaced runc pod (`hostUsers:false`) in `xlearn-runner`, a capability-holding **spawner** that never parses a learner
byte, a capless **front** that parses all of them, and a **fresh jail per test case without a user namespace**, each in its own
cgroup v2 leaf, with all verdict evidence read outside the learner's process. The spike week (spk-01/spk-02) answered the
go/no-go questions and recorded the mechanism in t3 §16. ADR-0030 stayed **Proposed** until now (BP2); **you accept it as step 2**.
The host files ship in the Oct 24 window (mi-09), the guard objects in mi-14, the deployment in mi-10; none of that blocks you,
and you touch none of it. Production has one user (the owner, D35); nothing here reaches production until mi-10.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] t3 §16.1–§16.4 are on `main` and §16.4 reads **"Spike P0–P3 GO"** with a named mechanism (go-sandbox `forkexec.Runner`, or nsjail R1-N).
- [ ] No open owner decision from the spike (R1-U / R1b would be one; `docs/v2/status.md` decisions log). If one is open → **stop**.
- [ ] ADR-0030 is still `Proposed` and no peer PR edits it: `gh pr list --state open --search "0030"`, `git worktree list`, ListAgents.
- [ ] t3 §16.2 holds the amd64 `go` allowlist as a sorted syscall-name list.
- [ ] No open peer PR touches `cmd/runner`, `internal/runner`, `internal/platform/runnerapi` or the go-sandbox line in `go.mod`.

## Do this (in order)

1. **[X] Branch** `feat/runner-core` from an up-to-date `main`.
2. **[X] Accept ADR-0030** (plan task 1):
   - status → `Accepted (<date>, spike P0–P3 GO; ADR-0035 §6 amendments folded)`;
   - add "Spike results (MI-10, spk-01/spk-02)": the compact P0–P3 + image-volume table from t3 §16, and one bullet each for the
     mechanism (update §1's "Jail primitive" line), SETPCAP, the spawn path, where the allowlists live, the go-race ASLR policy,
     host-file diffs;
   - fold ADR-0035 §6 **and everything in today's §7** into §5, row by row (the plan's task 1 lists the exact text): A3 (live chart
     0.2.2; the knobs ship as 0.3.0 in one PR); A4 (forward-declared identity/coach/judge NATS callers, judge as a PG caller, `:8222`
     admits no pod); A5 (server-first nkeys with fine ACLs from `topology.go`, N0–N4 with the `legacy` bridge and the compose test;
     no SOPS decryption in `messaging`, seeds in `apps/secrets`, ops seed offline; N3 + A4 gate M3 and erase); A6 + L23 and pid limits
     in the same restart; A8 after MI-11a; Track B without "fine NATS ACLs" or L23. **Only then** shrink §7 to a one-line history
     note, and put the old-§7 → new-§5 mapping in the PR description (nothing may be lost);
   - settle "vendored and pinned" → "pinned in `go.mod`, import-restricted" if that's what you do in step 5;
   - update the `docs/adr/README.md` row for 0030.
   A NO-GO row or an unrecorded R1-U/R1b → don't accept; mark the sprint ⛔ and report.
3. **[X] Contract package** `internal/platform/runnerapi` (plan task 2): `types.go`, `errors.go`, `stream.go` (length-prefixed
   `job.json` + case frames; caps enforced as bytes arrive: job.json ≤ 1 MiB, case ≤ 8 MiB, inputs ≤ 16 MiB, ≤ 512 cases),
   `validate.go` (`ValidateJob`, `ValidateResult`), `names.go` (generic file-name rule). Stdlib only. Golden `job.json`/`result.json`
   fixtures under `testdata/`. `Job` carries `Files` and a separate `HiddenFiles` (compile jail only), plus `Tests[]`,
   `OutputMode`; `CaseResult` carries `OutputSHA256`; `Versions` has all eight fields; `Telemetry` includes `TestsCapHit`. **No
   `Job.Cost`.** Append-only from here on.
4. **[X] Config + skeleton**: `cmd/runner/main.go` (`runner`, `runner front`, `runner canary`, `runner -version` stamping
   `main.version`), `internal/runner/config.go` (`RUNNER_ADDR=:8090`, `RUNNER_TOKEN_FILE=/var/run/secrets/runner-auth/token`,
   `RUNNER_MODE=prod|dev`, `RUNNER_IMAGE_DIGEST`, `RUNNER_MAX_JOBS=500`, `RUNNER_MAX_AGE=6h`, `LOG_LEVEL`). Linux-only files carry
   `//go:build linux` with `*_other.go` stubs, so macOS builds and tests stay green.
5. **[X] Dependency**: `go get github.com/criyle/go-sandbox@<the version t3 §16.1 used>`; add `internal/runner/imports_test.go`
   (`go list -deps`): nothing imports `go-sandbox/container`; nothing outside `internal/runner/...` imports go-sandbox. (R1-N: no
   go-sandbox; the jail wrapper spawns the pinned `nsjail --disable_clone_newuser` with its cgroup/rlimit features off.)
6. **[X] Supervisor** (plan task 3): spawner (PID 1, namespaced caps, no listener) and front (re-exec'd as a fixed non-zero UID with
   an **empty bounding set, 0 caps and NNP set in the child before `execve`**). `internal/runner/ipc`: `SOCK_SEQPACKET`, fixed-size
   versioned messages with no learner content, `SCM_RIGHTS` for pipe fds; unknown message → kill the pair. Per-job `src` tmpfs owned
   by the front's UID (mode 0700; the front writes files **0444** through a passed dir fd, then sets the dir to **0555** before the
   compile so the compile UID can read it; the spawner never reads or chmods it). Compile jail → in-jail exporter (`O_NOFOLLOW`,
   ≤ 64 MiB over a pipe) → spawner writes the per-job artifact tmpfs (dir mode `0111`; the file mode is the profile's
   `ArtifactMode`: `0111` static ELF, `0444` interpreted artifact) before learner code runs.
   Per-case jail: `NEWNS|NEWPID|NEWNET|NEWIPC|NEWUTS|NEWCGROUP`, **never `NEWUSER`**, loopback down, tmpfs root + `pivot_root`,
   read-only `nosuid,nodev` binds, `/w` tmpfs, no `/proc` by default, per-job UID from a pool not reused within 1,000 jobs, caps 0 +
   bounding drop + NNP, `RLIMIT_CORE=0` and the profile's rlimits, seccomp last (`clone3` → ENOSYS), fds 0 `/dev/null`, 1/2 capped
   pipes, 3 input, 4 harness. Startup: wipe `slots/`, move into `runner/`, pre-read and hash toolchains, measure each profile's
   memory baseline, then serve.
7. **[X] cgroups + measurement** (plan task 4): the `runner/` + `slots/s{0,1}/job/{compile,case-K}` layout (cpu.max 1 CPU,
   memory.max 1.25 GiB, swap 0, pids 512 per slot; `oom.group 1` per case); spawn with `CLONE_INTO_CGROUP` or `cgroup.procs` + sync
   pipe (whichever §16.1 validated; the other behind a flag). `internal/runner/measure` as pure, table-tested functions: CPU from
   `cpu.stat`, wall via pidfd, `PeakKB` = `memory.peak` − baseline, MLE from `memory.events`, fork cap from `pids.events`, OLE from
   byte counters, SIGSYS → `signal`. Kill policy: CPU > TL + min(0.5 s, TL/2); wall 1.5·TL + 0.5 s; idle < 5% for ≥ 1 s → TLE
   (idle); `RLIMIT_CPU` at TL + 2 s; stdout 8 KiB / stderr 2 KiB kept in Run only; 1 MiB hard cap → OLE; fd 4 capped or hashed
   (`OutputMode`); compile ≤ 15 s / profile memory / pids 256, a limit hit is CE with its CPU in `Telemetry.SlotMs`; **tests ≤ 45 s
   wall (L14)** for the whole test phase (compile and the quiet re-run excluded): when it fires, the in-flight case is `tle`, later
   cases `not_run`, `Telemetry.TestsCapHit=true` (judge's term mapping gives TLE, counted; m3-02's pack gate keeps Σ TL ≤ 40 CPU-s),
   or `Throttled=true` if the job was suspect; the quiet re-run gets its own 45 s cap; `job_timeout` = 15 s + 45 s + (60 s + 45 s) +
   5 s = 170 s, a runner-fault backstop only. Record the choice in the decisions log. The case order and `StopGroupOn` of t4 §2.6;
   teardown assertions (baseline ±5 MiB, `pids.current` 0) or fail the slot closed.
8. **[X] API** (plan task 5): `POST /v1/jobs` (200 / 400 / 401 / 503 + `Retry-After` + `{"infra":{"kind":"saturated"}}`; disconnect →
   `cgroup.kill`, no Result), `GET /v1/profiles`, `GET /v1/stats`, `GET /readyz` (the eight canaries, every 60 s), `GET /healthz`.
   Constant-time bearer compare; never log the token. `RUNNER_MODE=dev` only downgrades failing security canaries to warnings;
   `prod` also refuses jobs with `infra: setup`. Drain on SIGTERM (60 s, then `killed`). Rotation after 500 jobs / 6 h / any
   SIGSYS; `BootEpoch` = `<hostname>/<start ns>/<8 random bytes hex>`. ERROR logs and counters only — **no alert of any kind (D34)**.
9. **[X] `throttled`** (plan task 6): s1 steal from `/proc/stat`; s2 `runner canary` (~150 ms ALU + a > 16 MiB memory stream; every
   5 min idle + on demand; rolling median → `CanaryMedian`); the three suspect rules; the quiet re-run (503, ≤ 60 s drain, steal < 5%
   over 10 s, pre-canary ≤ 1.1 × median, re-run the case and the rest of its group, final verdict); else `Throttled=true`. Thresholds
   in one file, with a pointer to the S0 data.
10. **[X] Corpus + tests + CI** (plan task 7): test profile `testgo@0` (build tag `runner_it`; spk-02's amd64 `go` list) with a guard
    test that release builds don't register it; corpus programs under `internal/runner/testdata/corpus/<name>/main.go`; integration
    tests under `internal/runner/it/` (`//go:build linux && runner_it`) for every row of the plan's task 7 list, including balloon
    × 100 (MLE 100/100, container `oom_kill` delta 0), 1,000 case cgroups (cleanup invariants) and the 45 s tests cap (record the
    per-case jail overhead). Add the `runner-it` CI job (`container: debian:trixie-slim@sha256:…` with `--privileged
    --cgroupns=private --memory=3g --cpus=2`, pinned Go tarball, `go test -tags runner_it -count=1 ./internal/runner/...`; fall back
    to `sudo systemd-run --scope -p Delegate=yes …` on the VM if the container can't delegate, and record which one works) and
    `make runner-it`. This CI run stands in for a local privileged VM run. The arm64 dev VM's LOG default is a
    dev-only switch (`runner_it` tag + `RUNNER_MODE=dev`), never an image default.
11. **[X] Docs** (plan task 8): `docs/architecture/runner.md` (new), the `services.md` entry; hand-off lines for m3-04, m3-06
    (including "judge's `/v1/profiles` check refuses `mode=dev` in production"), mi-10, m3-15 in `docs/v2/status.md`.
12. **[X] Verify**: `gofmt`, `go vet ./...`, `go test -race ./...` (macOS and Linux), `sqlc diff` (unchanged), the `runner-it` job
    green on the PR. Don't mark a jail task ✅ from a macOS run.
13. **[X] PR** → conventional commits (`docs(adr): accept 0030 …`, `feat(runner): …`) with the attribution lines → CI green →
    squash-merge.

## Constraints

- **Service boundaries** ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)): the runner has **no database, no NATS, no
  egress, no secrets but its bearer**; judge is its only caller. `runnerapi` is a shared, stdlib-only platform package; nothing in
  `internal/runner` imports judge or any other service.
- **goose + sqlc:** no migration here; `sqlc diff` must still pass unchanged. **Outbox/inbox:** n/a (no events).
- **Contract discipline:** `runnerapi` is append-only from `runner-v1.0.0`; a breaking change is a runner **major** (ADR-0034 §1.5).
- **R-FS is absolute:** no capability-holding process opens, chowns, copies or traverses a learner-writable path after learner
  code ran; only the capless front reads learner bytes, and only from pipes.
- **Never import `go-sandbox/container`**; never create a user namespace anywhere in the pod.
- **GitOps / production:** no infra PR, no `kubectl`, no host change. Read-only `ssh vps` for facts only. Local VMs are throwaway
  and never touch production credentials.
- **D34:** no alerting, no timer, no CronJob, no push channel. Failures are ERROR logs and counters read on demand.
- **Memory-sum rule** (ADR-0035 §5): no new always-on pod here (the runner's 3 GiB is counted in mi-10); keep `runner/` ≤ ~300 MiB
  so INV-14 holds.
- **Consumers-before-producers / NATS ACL PR before a consuming tag:** n/a (the runner has no NATS).
- **theme.css:** n/a (no UI).
- **Parallel sessions:** check peers' PRs, tags, worktrees and ListAgents before editing ADR-0030 or claiming any ADR number (you
  shouldn't need a new one; if you do, take the next free number after checking).

## Deliverables

- ADR-0030 Accepted (spike table, mechanism, ADR-0035 §6 folded) + the ADR index row.
- `internal/platform/runnerapi` (types, stream, validators, names, golden fixtures).
- `cmd/runner`, `internal/runner/{spawner,front,ipc,jail,seccomp,cgroup,measure,canary,profile,it}`, config, `testgo@0` (test-only).
- The hostile corpus under `internal/runner/testdata/corpus/` and the integration tests.
- CI job `runner-it`; `make runner-it`; go-sandbox pinned with the import guard.
- `docs/architecture/runner.md`; the `services.md` entry.

## Update status

- [`../sprints/sprint-m3-03.md`](../sprints/sprint-m3-03.md): each task ⬜ → 🔄 → ✅ (or ⛔ with a reason); _Overall_ ✅ at the end.
- [`../status.md`](../status.md): the Sprint board row (m3-03 ✅); **Milestones** (M3 🔄); the MI track row MI-12 "runner code: core ✅
  (m3-03)"; the ADR-0030 line → Accepted; **Decisions log**: the mechanism, SETPCAP, the spawn path, go-sandbox pinned (not vendored)
  if so, `RUNNER_MODE`, `BootEpoch` format, the 45 s tests-cap outcome and the measured per-case jail overhead, the CI jail
  environment that worked; **hand-offs** to m3-04, m3-06, mi-10 (port, probes,
  token path/key, `RUNNER_IMAGE_DIGEST` marker, SETPCAP, `emptyDir` needs) and m3-15.
- No flag, tag-floor or content-status change.

## Done when (acceptance)

- [ ] ADR-0030 is **Accepted** with the spike results and ADR-0035 §6 folded (A3–A8, Track B); nothing from the old §7 is lost;
      the index row is updated.
- [ ] L14 caps: compile 15 s → CE; tests 45 s → in-flight `tle` + `not_run` + `TestsCapHit` (or `Throttled`); `job_timeout` only
      past 170 s.
- [ ] `runnerapi` round-trips the golden fixtures and enforces every cap mid-stream; stdlib-only.
- [ ] The P2 corpus is classified correctly in the Linux CI jail run: **MLE 100/100, 0 container OOMs**, TLE / TLE (idle) / OLE /
      RE as specified, SIGSYS → `signal` + rotation, every network probe fails.
- [ ] Cleanup invariants green: 0 survivors, `runner/` baseline ±5 MiB, `nr_dying_descendants` → ~0 in 60 s after 1,000 case
      cgroups, cross-job markers invisible.
- [ ] 401 / 400 / 503 / disconnect / drain behaviour tested; no release build contains `testgo@0`; no `go-sandbox/container` import.
- [ ] CI green on macOS-safe and Linux lanes; `sqlc diff` unchanged; docs and hand-offs recorded.

Ship per AGENT.md land-and-sync with **this sprint's release action: merge only** (the runner ships in `runner-v1.0.0`, cut in
[m3-15](../sprints/sprint-m3-15.md)) — no tag, no infra PR; then `git checkout main && git pull`.
