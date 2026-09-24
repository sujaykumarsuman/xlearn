# Sprint m3-03 — Runner core: supervisor, jail, cgroups, API

> **Milestone:** M3 — judge plus code grader (runner track; code for rollout step **MI-12**) · **Track:** product · **Order:** 30
> **Prereqs:** [spk-01](sprint-spk-01.md) (P0–P2: mechanism, host files) · [spk-02](sprint-spk-02.md) (P3: amd64 allowlists, TSAN, image volume; the MI-10 verdict)
> **Unblocks:** [m3-04](sprint-m3-04.md) (language profiles and harness codecs) · [m3-06](sprint-m3-06.md) (judge's runner lane builds on the contract types) · through m3-04 → [m3-15](sprint-m3-15.md) (`runner-v1.0.0`)
> **Release action:** **merge only** (ships in `runner-v1.0.0`, cut in [m3-15](sprint-m3-15.md)). The `internal/platform/runnerapi` package is also compiled into judge from `v1.13.0` ([m3-07](sprint-m3-07.md)). No tag, no infra PR.
> **Calendar:** late October (after the spike week, Mon 2026-10-12 → Fri 10-16). Independent of the Sat 2026-10-24 host window: runner code never waits on [mi-09](sprint-mi-09.md).
> **Execute with:** [`../prompts/prompt-m3-03.md`](../prompts/prompt-m3-03.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Accept ADR-0030 from the spike results (BP2), folding ADR-0035 §6 | X | ⬜ |
| 2 | Contract types + wire stream: `internal/platform/runnerapi` | X | ⬜ |
| 3 | Supervisor: spawner/front privilege split + per-case jail | X | ⬜ |
| 4 | cgroups, measurement and hard caps (L14) | X | ⬜ |
| 5 | HTTP API, typed infra errors, drain and front rotation | X | ⬜ |
| 6 | `throttled`: steal, runner-owned canary, in-runner quiet re-run | X | ⬜ |
| 7 | Hostile corpus, cleanup invariants, Linux CI job `runner-it` | X | ⬜ |
| 8 | Docs: `docs/architecture/runner.md`, `services.md`, hand-offs | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the M3 milestone row, the MI-12 row's "runner code" part, and the ADR-0030 line).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **spk-01 and spk-02 report GO** and a mechanism is chosen: [t3 §16.1–§16.4](../research/t3-sandbox.md) exist on `main`, and §16.4 reads "Spike P0–P3 GO" with the jail mechanism named (go-sandbox `forkexec.Runner`, or the nsjail `--disable_clone_newuser` fallback, R1-N).
- [ ] **No open owner decision from the spike.** If P1 fell through to R1-U or R1b, those are owner decisions (R1b is a D21 trigger that moves the runner to R2). They must be recorded in `docs/v2/status.md` before this sprint starts.
- [ ] **ADR-0030 is still Proposed** and no peer PR edits it (`gh pr list --search 0030`, `git worktree list`, ListAgents). Task 1 accepts it; [mi-09](sprint-mi-09.md) and the spikes deliberately left it alone.
- [ ] **The amd64 per-profile allowlists are in t3 §16.2** as sorted syscall-name lists (at least `go`). Task 7's test profile uses the `go` list; the `cpp` and `python` lists go to [m3-04](sprint-m3-04.md).
- [ ] **No open peer PR touches** `cmd/runner`, `internal/runner`, `internal/platform/runnerapi` or `go.mod`'s go-sandbox line.

The [M3 hard entry checklist](../rollout-plan.md#5-m3-hard-entry-checklist) gates the M3 UI sprint ([m3-11](sprint-m3-11.md)) and is copied verbatim there and into [m3-13](sprint-m3-13.md). It does **not** gate this sprint: runner code runs nowhere near production until [mi-10](sprint-mi-10.md). This sprint consumes that checklist's first line (spike GO) and produces the code half of MI-12.

## Goal

Build the runner's **trusted core** on the mechanism the spike chose, and accept ADR-0030 with the spike's numbers:
- a **privilege-split supervisor**: a capability-holding *spawner* (PID 1) that never parses a learner byte, and a capability-less *front* that parses all of them;
- a **per-case jail without a user namespace** (fresh mnt/pid/net/ipc/uts/cgroup namespaces, a fresh tmpfs root, a per-job UID, zero capabilities, `NO_NEW_PRIVS`, a KILL-by-default seccomp filter);
- **per-case cgroup v2 leaves** with every piece of verdict evidence **measured outside the learner's process**;
- the **judge-only job API** (bearer token, 2 slots, typed infra errors, `throttled`);
- **cleanup invariants** proven over a hostile corpus in a Linux jail-capable CI job.

The language profiles and harness codecs are [m3-04](sprint-m3-04.md); the image, release stream and acceptance suite are [m3-15](sprint-m3-15.md).

## Scope

**In**
- ADR-0030 → **Accepted** (task 1; BP2). The spike's results table and the chosen mechanism go into the ADR; ADR-0035 §6's amendments are folded into ADR-0030 §5.
- `cmd/runner` (spawner, front, canary) and `internal/runner/...`.
- The jail per the spike: go-sandbox `forkexec.Runner` without `CLONE_NEWUSER`, or nsjail `--disable_clone_newuser` if §16.4 chose R1-N.
- Per-case cgroups, outside measurement and the L14 caps ([ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory)).
- The job API: `POST /v1/jobs`, `GET /v1/profiles`, `GET /v1/stats`, `/readyz`, `/healthz`; bearer auth; 2 slots; typed `InfraError`; `Throttled`.
- The shared contract package `internal/platform/runnerapi` (types, wire stream, validators) that judge imports in [m3-06](sprint-m3-06.md).
- Cleanup invariants and the hostile corpus as committed tests; a Linux CI job that runs them.
- A test-only profile `testgo@0` (build tag `runner_it`) so this sprint can exercise compile → artifact → cases end to end.

**Out**
- The real language profiles (`go@1.26`, `cpp`, `python`), `func-json@1` / `class-ops@1`, source lints and per-profile allowlists → [m3-04](sprint-m3-04.md).
- `deploy/runner.Dockerfile`, `runner-release.yml`, the acceptance suite, the compose service, TL baselines, the `runner-v1.0.0` tag → [m3-15](sprint-m3-15.md).
- Deployment: the `runner` Kustomization, HelmRelease, bearer secret, 2nd IUA → [mi-10](sprint-mi-10.md). Host files (drop-in, AppArmor, pod seccomp, subuid, sysctls) → [mi-09](sprint-mi-09.md). Guard objects and the VAP → [mi-14](sprint-mi-14.md).
- judge's runner client, retry classification, poison-pill quarantine, budgets and the duty-cycle breaker → [m3-06](sprint-m3-06.md) / [m3-14](sprint-m3-14.md).
- `go-race`, `gotest@1`, the goroutine dump → `runner-v1.1.0`, [p-01](sprint-p-01.md). `sql-pg` → not in the pilot plan.

## Tasks

### 1 · Accept ADR-0030 from the spike results (BP2) [X]

Sources: [t3 §16.1–§16.4](../research/t3-sandbox.md) (spk-01/spk-02 results), [ADR-0035 §6](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#6-amendments), [rollout §12](../rollout-plan.md#12-downstream-constraints-for-the-build-plan-session) ("fold the text into the amended ADRs").

Edit [`docs/adr/0030-runner-technology-and-host-hardening.md`](../../adr/0030-runner-technology-and-host-hardening.md) in place:
- **Status line:** `Accepted (<date>, spike P0–P3 GO; ADR-0035 §6 amendments folded)`.
- **New section "Spike results (MI-10, spk-01/spk-02)"**, a compact table copied from t3 §16: one row per P0, P1 (positive and negative probes), P1b, P2 (MLE count, container OOMs, survivors, baseline drift, `nr_dying_descendants`), P3 (TSAN under `mmap_rnd_bits=32`, the ASLR policy, allowlist sizes per profile, unexpected SIGSYS under KILL) and the image-volume verdict. Plus one bullet each for:
  - **the chosen jail mechanism** (R1 `forkexec.Runner` or R1-N nsjail), and the §1 "Jail primitive" line updated to match;
  - **`SETPCAP`**: needed or not (the VAP's cap list and the HelmRelease values follow it; [mi-10](sprint-mi-10.md) reads it);
  - **the spawn path**: `CLONE_INTO_CGROUP` or `cgroup.procs` + sync pipe on 6.8;
  - **where the allowlists live**: pod seccomp on the host ([mi-09](sprint-mi-09.md)), per-case exec/compile filters inside the runner image ([m3-04](sprint-m3-04.md));
  - **the go-race ASLR policy** (for [p-01](sprint-p-01.md));
  - **host-file diffs** against t3 §8.7, if any.
- **Fold ADR-0035 §6 (and everything in today's ADR-0030 §7) into §5**, row by row:
  - **A3** gains "the live chart is **0.2.2**; A3's knobs (the union of every topic's asks) ship as **0.3.0 in one PR**".
  - **A4** gains "forward-declares identity, coach and judge as NATS callers and judge as a PG caller (MI-5); `:8222` admits no pod (opscheck dropped, D34)".
  - **A5** reads "NATS nkeys, **server-first with fine ACLs** rendered from `internal/platform/events/topology.go`: nkey users in `$G` bridged by `no_auth_user: legacy`; N0 client code dark → N1 users + bridge (the one NATS restart) → N2 seeds per service → N3 `legacy` denied → N4 bridge removed (a reload); a compose test runs NATS with the rendered block before N2. **No SOPS decryption in `messaging`**: public keys in plaintext in its values, seeds as SOPS secrets in `apps/secrets`, the ops seed offline (break-glass over an `ssh` port-forward only). **N3 plus A4 gate M3 and erase.** Why server-first and fine: ADR-0035 §2."
  - **A6** gains "plus L23 (`system-reserved` 1 GiB, `eviction-hard memory.available<500Mi`) and the pod pid limit, in the same k3s restart (October host window); `host-bootstrap.sh` sets them, `host-verify` asserts them".
  - **A8** gains "after **MI-11a** (Flux controllers 1 GiB → 512 Mi; limits for Traefik, cert-manager, metrics-server), because the runner's 3 GiB of limits would otherwise break the node's memory sum".
  - **Track B** drops "fine NATS ACLs" (absorbed into A5) and the L23 kubelet args (moved into A6).
  - **Only then** shrink §7 to a one-line history note ("§5 amended by ADR-0035 §6 (T7, 2026-09-24), folded at acceptance") so nothing is stated twice. **Check before committing:** every bullet of the old §7 (the A5 rollout, the secrets rule, the N3 + A4 gate, L23 + MI-11a, the chart line) is now in §5 or behind an explicit ADR-0035 pointer; list the old-§7 → new-§5 mapping in the PR description.
- **Wording to settle:** §1 says go-sandbox is "vendored and pinned". If you pin it in `go.mod` (recommended, so the Renovate `gomod` rule in [mi-11](sprint-mi-11.md) can track it), say "pinned in `go.mod` (`go.sum`-verified), import-restricted to `pkg/forkexec`, `pkg/seccomp`, `pkg/mount`" instead.
- `docs/adr/README.md` index row 0030 → `Accepted (<date>; spike GO; §5 amended by 0035, folded)`. Leave ADR-0035's own back-pointer as it is.
- **Stop conditions:** a NO-GO row, or a fallback beyond R1-N (R1-U, R1b) without a recorded owner decision → don't accept; mark the sprint ⛔ and report. Don't reopen a decision the spike didn't test.

### 2 · Contract types + wire stream: `internal/platform/runnerapi` [X]

Sources: [t4 §11.1](../research/t4-judge-contract.md#111-t3-the-runner-contract-not-the-technology) with the T3 amendments ([t3 §10](../research/t3-sandbox.md#10-what-t3-constrains-downstream) "T4 §11.1"), [t3 §5.3](../research/t3-sandbox.md#53-api-judge-is-the-only-caller), [§5.5](../research/t3-sandbox.md#55-measurement-all-outside-the-learners-process), [§5.9](../research/t3-sandbox.md#59-typed-infra-errors-and-poison-pills), [§5.11](../research/t3-sandbox.md#511-t4-111-checklist).

A dependency-free package (stdlib only), imported by the runner and by judge ([m3-06](sprint-m3-06.md)). **This is the judge↔runner contract**: a breaking change to it is a runner **major** ([ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)), so every type is append-only from `runner-v1.0.0`.

| File | Content |
|---|---|
| `types.go` | `Job{ID, Profile, Harness, Mode (run\|submit), Files []File{Path, Data []byte}, HiddenFiles []File, Cases []CaseInput{OpaqueID, Group (sample\|edge\|random\|perf), Size int64}, Tests []TestSpec, Limits, StopGroupOn map[string]string, Count int, OutputMode (bytes\|sha256)}` — `HiddenFiles` exist **only in the compile jail** (honor profiles' hidden tests, later) and are never echoed back beyond a file name in a diagnostic; **no `Job.Cost`** (dropped, t3 §14). `Limits{Case{CPUms, MemMB, OutputKB}, Compile{CPUms, MemMB}}`; wall, idle and job deadlines are **derived by the runner**, never sent. `Result{Compile{OK, Diags []Diag{File, Line, Col, Msg}}, Cases []CaseResult, Tests []TestEvent, Race bool, Dump *GoroutineDump, Throttled bool, Infra *InfraError, Versions, Telemetry}`. `CaseResult{OpaqueID, Term (ok\|tle\|mle\|ole\|signal\|exit_nonzero\|not_run), Signal, ExitCode, CPUms, WallMs, PeakKB, PanicClass, Output []byte, OutputSHA256, OutputBytes, Stdout, Stderr (run mode only), Quiet bool, IdleKill bool}`. `Versions{Runner, ImageDigest, Profile, Toolchain, ProfileSHA, CPUModel, CanaryMedian, BootEpoch}`. `Telemetry{StealPct, CanaryRatio, BusyMs, OtherSlotBusy, QuietReruns, SlotMs, TestsCapHit}` (internal; judge persists it per job, t3 §10; `TestsCapHit` = the L14 45 s tests cap fired, task 4). `Tests`, `Race`, `Dump`, `Count` exist now and stay empty until [p-01](sprint-p-01.md). |
| `errors.go` | `InfraError{Kind}` with the closed set `setup \| runner_oom \| killed \| job_timeout \| saturated`, and a doc comment per kind (who emits it: the runner emits `setup`, `killed`, `job_timeout`, `saturated`; **judge infers `runner_oom`** from a dropped connection plus a changed `BootEpoch`). |
| `stream.go` | The `POST /v1/jobs` body: `Content-Type: application/vnd.xlearn.runner-job; v=1`; a length-prefixed stream (`u32` big-endian length + bytes) of `job.json`, then one frame per case input in `Cases` order. `EncodeJob(w, job, inputs)` for judge; `DecodeJob(r, caps)` for the front, which **enforces the caps as bytes arrive**: `job.json` ≤ 1 MiB, each case ≤ 8 MiB, all inputs ≤ 16 MiB, ≤ 512 cases; strict JSON (`DisallowUnknownFields`), exact frame count. |
| `validate.go` | `ValidateJob(job)` (enums, limits in range, `Files` paths pass `names.go`, case ids unique) → 400; `ValidateResult(job, res)` for judge (case ids and counts match, enums closed, size caps, `Versions.Profile` = `job.Profile`, `BootEpoch` present) → a mismatch is `infra: setup` in judge. |
| `names.go` | The generic file-name rule the front re-checks (t3 §2.4 A5): a flat name, `[A-Za-z0-9_]` plus one extension, no separators, no `..`, no leading dot, ≤ 64 bytes, ≤ 32 files, ≤ 1 MiB total. Per-profile extension and directive rules are [m3-04](sprint-m3-04.md)'s. |

Tests: codec round-trip; every cap enforced mid-stream (a 9 MiB case is refused before it is fully read); unknown field → error; golden `job.json` / `result.json` fixtures in `internal/platform/runnerapi/testdata/` (they become m3-06's contract fixtures).

### 3 · Supervisor: spawner/front privilege split + per-case jail [X]

Sources: [t3 §2.3](../research/t3-sandbox.md#23-trust-boundaries) (rule **R-FS**), [§5.2](../research/t3-sandbox.md#52-process-model-privilege-split), [§5.4](../research/t3-sandbox.md#54-lifecycle-of-one-job), [ADR-0030 §1](../../adr/0030-runner-technology-and-host-hardening.md#1-build-a-thin-go-runner-xlearn-runner--r1).

**Layout.**
- `cmd/runner/main.go`: `runner` (spawner, PID 1), `runner front` (re-exec'd by the spawner), `runner canary` (task 6), `runner -version` (stamped `main.version`, guarded by `deploy/version_test.go` once m3-15 adds the Dockerfile).
- `internal/runner/spawner/` (capability-holding: cgroups, mounts, jails, evidence), `internal/runner/front/` (capless: HTTP, spool, all decoding, Result assembly), `internal/runner/ipc/` (the fixed-schema spawner⇄front protocol), `internal/runner/jail/` (the forkexec or nsjail wrapper), `internal/runner/seccomp/` (filters from syscall-name lists), `internal/runner/profile/` (the `Profile` type and registry that m3-04 fills), `internal/runner/config.go`.
- **Linux-only code** carries `//go:build linux`, with `*_other.go` stubs returning `ErrUnsupported`, so `go build ./...`, `go vet ./...` and `go test ./...` stay green on macOS.

**The dependency.** Add `github.com/criyle/go-sandbox` at the exact version the spike used (t3 §16.1), import-restricted: a test (`internal/runner/imports_test.go`, `go list -deps`) fails if anything imports `github.com/criyle/go-sandbox/container` (it forces a user namespace and keeps ambient `SYS_ADMIN`, t3 §4), or if any package outside `internal/runner/...` imports go-sandbox at all. If §16.4 chose nsjail, the jail wrapper spawns the pinned `nsjail` binary (m3-15 puts it in the image) with `--disable_clone_newuser`, its own cgroup and rlimit features **off**, spawned into our case cgroup; accounting stays ours.

**Process model.**

| Process | Identity | Does | Never does |
|---|---|---|---|
| **spawner** (PID 1) | UID 0 in the pod's user namespace; the namespaced caps the ADR lists (`SYS_ADMIN SETUID SETGID KILL`, `SETPCAP` per the spike) | creates cgroups, mounts, per-job tmpfs and per-case jails; reads cgroup evidence and wait status; `cgroup.kill` | parse learner bytes; open, chown, copy or traverse a learner-writable path after learner code ran (R-FS); listen on a socket |
| **front** | a fixed non-zero UID (e.g. 65532), **empty bounding set, 0 caps, `NO_NEW_PRIVS`**, set before its first instruction | HTTP on `:8090`, bearer auth, spooling inputs into memory, writing source files, streaming inputs to fd 3, decoding fds 1/2/4 and compile diagnostics, assembling the Result | hold a capability; touch cgroupfs |

- **IPC:** a `socketpair(AF_UNIX, SOCK_SEQPACKET)`; fixed-size, versioned binary messages (enums, sizes, limits, opaque ids — **no learner content**), strict decode, any unknown message kills the pair. Per-case pipe fds pass to the front with `SCM_RIGHTS`.
- **Sources without breaking R-FS:** the spawner creates the per-job `src` tmpfs (mode 0700, owned by the front's UID) and passes a directory fd to the front, which writes the files **mode 0444** and then sets the directory to **0555** before the compile starts (so the separate compile UID can read, and nobody can write). The compile jail binds it read-only. The spawner never reads, chmods or traverses it.
- **Artifacts:** an in-jail exporter running as the compile UID opens the build output with `O_NOFOLLOW` and streams it over a pipe (≤ 64 MiB); the spawner writes those raw bytes into a new per-job **artifact tmpfs** (directory mode `0111`, owned by pod root) before any learner code runs. **The file mode is the profile's** (`Profile.ArtifactMode`): `0111` for a static ELF (exec needs no read bit), `0444` for an interpreted artifact (m3-04's `app.pyz`, which `python3` must read). `testgo@0` uses `0111`; m3-04 adds a test per language.

**Per-case jail** ([t3 §5.4](../research/t3-sandbox.md#54-lifecycle-of-one-job) steps 4–5):
- clone flags `NEWNS|NEWPID|NEWNET|NEWIPC|NEWUTS|NEWCGROUP`, **never `NEWUSER`**; loopback left down;
- a fresh tmpfs root, `pivot_root`; read-only `nosuid,nodev` binds of the profile's toolchain/runtime paths and the artifact; a per-case writable `/w` tmpfs (64 MiB, 4k inodes by default); one overlay where a profile asks for it (the Go compile cache, m3-04); **no `/proc`** unless the profile asks (go-race later: `hidepid=invisible,subset=pid`);
- a per-job UID from a pool inside the pod's userns, **not reused within 1,000 jobs** (t3 §2.4 A6); a separate compile UID;
- caps → 0 and the bounding set dropped, `NO_NEW_PRIVS`, `RLIMIT_CORE=0`, `RLIMIT_FSIZE`, `RLIMIT_NOFILE`, `RLIMIT_STACK` per profile, `RLIMIT_CPU` backstop at TL + 2 s;
- the profile's seccomp filter installed last (KILL-by-default for exec; ENOSYS-by-default with KILL for dangerous calls for compile; `clone3` → ENOSYS);
- fds: 3 = input pipe, 4 = harness result pipe, 1/2 = capped pipes, 0 = `/dev/null`.

**Startup** ([t3 §5.4](../research/t3-sandbox.md#54-lifecycle-of-one-job) step 7): wipe `slots/`; move the spawner and front into `runner/`; pre-read the profiles' toolchain files so their page cache is charged to `runner/` (and hash them for `ProfileSHA`); measure each profile's memory baseline (median `memory.peak` of its no-op program); start serving only when all of that succeeded.

### 4 · cgroups, measurement and hard caps (L14) [X]

Sources: [t3 §5.5](../research/t3-sandbox.md#55-measurement-all-outside-the-learners-process), [§7.1](../research/t3-sandbox.md#71-cgroup-layout), [§7.2](../research/t3-sandbox.md#72-time-limit-policy), [§5.11](../research/t3-sandbox.md#511-t4-111-checklist); L14 in [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory).

**Layout** (`internal/runner/cgroup/`), inside the container's writable cgroup namespace:
```
/sys/fs/cgroup            (container: cpu.max 2 · memory.max 3 GiB — set by the pod)
├─ runner/                spawner + front
└─ slots/                 subtree_control +cpu +memory +pids
   ├─ s0/  cpu.max "100000 100000" · memory.max 1.25 GiB · memory.swap.max 0 · pids.max 512
   │   └─ job/ ├─ compile/   └─ case-K/  (memory.max = mem_mb + baseline · memory.oom.group 1 · swap.max 0 · pids.max)
   └─ s1/  same
```
- Respect the no-internal-process rule (move PID 1 into `runner/` before enabling `subtree_control`). No `memory.high`, no `io.max` (every writable path is tmpfs).
- **Spawn into the case cgroup** with `CLONE_INTO_CGROUP` (a cgroup fd), or the `cgroup.procs` write gated on a sync pipe if §16.1 says `clone3` isn't usable on 6.8.
- **Invariant (INV-14):** 2 × 1.25 GiB + `runner/` (≤ ~300 MiB) < 3 GiB, so a learner OOM always lands in a `case-K` or `compile` cgroup and **never** in the container.

**Measurement** (`internal/runner/measure/`, pure functions, table-tested on every platform):

| Field | Source |
|---|---|
| `CPUms` (the verdict clock) | Δ `cpu.stat usage_usec` of the case cgroup, cross-checked with `wait4` rusage |
| `WallMs` | monotonic clock around pidfd readiness |
| `PeakKB` | case `memory.peak` − the profile's calibrated baseline |
| MLE | `memory.events` `oom_kill > 0` |
| fork cap | `pids.events` `max > 0` → RE (fork limit) |
| OLE | front byte counters on fds 1, 2 and 4 |
| exit/signal | pidfd + `waitid`; **SIGSYS → `Term=signal`, `Signal=SIGSYS`** (judge maps it to RE, counted, t3 §2.4 A18) |

**Kill policy** (poll every 10 ms):
- CPU > TL + min(0.5 s, TL/2) → kill, TLE; `RLIMIT_CPU` at TL + 2 s is the backstop.
- Wall = 1.5·TL + 0.5 s. A wall kill with CPU < TL is **suspect** (task 6) if the CPU rate over the last second was > 5%; if it was < 5%, it's **TLE (idle)** (`IdleKill=true`).
- Idle kill: CPU rate < 5% for ≥ 1 s → TLE (idle). Inputs are fully spooled before a case starts, so a stalled caller can never make a learner look idle.
- Output: keep ≤ 8 KiB stdout and ≤ 2 KiB stderr (Run mode only; dropped in Submit); a hard cap of 1 MiB per fd → OLE; fd 4 capped at `OutputKB` (bytes mode) or hashed while streaming (`OutputMode: sha256` → `OutputSHA256` + `OutputBytes` only).
- Compile: ≤ 15 s CPU and wall, profile memory (e.g. 768 MiB), pids 256; hitting a limit is **CE**, and its CPU counts in `Telemetry.SlotMs` (judge charges it to the learner's budget).
- **Tests: ≤ 45 s wall (L14 hard cap)** for the whole test phase (every case of the job, from the first case start; compile and the quiet re-run excluded). When it fires: the in-flight case is killed and reported `tle`, every later case is `not_run`, and `Telemetry.TestsCapHit=true`. Judge's existing term mapping then gives **TLE, counted** (learner-caused: m3-02's pack `tl` gate keeps **Σ TL ≤ 40 CPU-s**, so a within-TL solution fits). **Exception:** if the job was suspect under task 6's rules while the cap ran (other slot busy, s1 ≥ 0.10 or s2 ≥ 1.2 × median), set `Throttled=true` instead of a verdict. It is never `job_timeout`. Record this choice in the decisions log.
- The quiet re-run (task 6) gets its own ≤ 45 s cap with the same rule.
- Job deadline (`job_timeout`, the runner's own backstop, never a learner verdict): **15 s compile + 45 s tests + the quiet re-run allowance (≤ 60 s wait + ≤ 45 s re-run) + 5 s slack = 170 s**. Anything past it is a runner fault.
- Case order ([t4 §2.6](../research/t4-judge-contract.md#26-evaluation)): samples first (stop at the first failure; the rest `not_run`), then **every** correctness case, then the perf block (stop at the first TLE). `StopGroupOn` drives it.

**Teardown** after each case and job: `cgroup.kill`, wait for `cgroup.events populated 0`, `rmdir`; unmount the artifact and `src` tmpfs; assert the slot's `memory.current` is back to baseline ±5 MiB and `pids.current` is 0; else fail the slot closed (`infra: setup`) and log at ERROR.

### 5 · HTTP API, typed infra errors, drain and front rotation [X]

Sources: [t3 §5.3](../research/t3-sandbox.md#53-api-judge-is-the-only-caller), [§5.9](../research/t3-sandbox.md#59-typed-infra-errors-and-poison-pills), [§5.2](../research/t3-sandbox.md#52-process-model-privilege-split) (restart policy), [t4 §11.1](../research/t4-judge-contract.md#111-t3-the-runner-contract-not-the-technology).

| Endpoint | Auth | Behaviour |
|---|---|---|
| `POST /v1/jobs` | bearer | Synchronous inside judge's lease. `200` + `Result`; `400` contract error (`ValidateJob`, unknown profile or harness, `go-race`/`gotest@1` before p-01); `401` bad or missing token; `503` + `Retry-After` + `{"infra":{"kind":"saturated"}}` when both slots are busy, while draining, or while a quiet re-run holds exclusivity. **A 503 is never a verdict.** If the client disconnects, the job subtree is `cgroup.kill`ed and no Result is sent. |
| `GET /v1/profiles` | bearer | `[{profile@v, toolchain, profile_sha256, image_digest, harness majors, baseline@v, tl_multiplier, calibrated, mem_baseline_kb}]` + `boot_epoch`, `canary_median`, `mode`. m3-04 fills the profile fields. |
| `GET /v1/stats` | bearer | Read-only, no learner data: per-slot `memory.current`, `pids.current`, `nr_dying_descendants`, `runner/` `memory.current`, the container's `memory.events` `oom_kill`, jobs served, SIGSYS count, `boot_epoch`. Read by the acceptance suite ([m3-15](sprint-m3-15.md)) and later by `judge admin`. |
| `GET /readyz` | none | 200 only when every canary passed in the last 60 s: egress to kube-dns (`10.43.0.10:53`), the apiserver VIP (`10.43.0.1:443`) and `1.1.1.1:443` **fails**; the AppArmor label is `xlearn-runner`; the UID map is **not** the identity map; cgroupfs is writable; `unshare(CLONE_NEWUSER)`, `fsopen("tmpfs")` and `socket(AF_INET, SOCK_STREAM, IPPROTO_SCTP)` are denied; `/proc/sys/kernel/core_pattern` doesn't start with `\|`. |
| `GET /healthz` | none | Liveness: the front is up and the IPC pair answers. The path [mi-10](sprint-mi-10.md)'s HelmRelease probes. |

- **Config** (`internal/runner/config.go`, env): `RUNNER_ADDR` (`:8090`), `RUNNER_TOKEN_FILE` (default `/var/run/secrets/runner-auth/token`; the Secret `runner-auth`, key `token`, that [mi-10](sprint-mi-10.md) mounts), `RUNNER_MODE` (`prod` default \| `dev`), `RUNNER_IMAGE_DIGEST` (reported in `Versions.ImageDigest`; empty → `unknown`), `RUNNER_MAX_JOBS` (500), `RUNNER_MAX_AGE` (6h), `LOG_LEVEL`. The token is read once, compared in constant time, never logged.
- **`RUNNER_MODE=dev`** (compose, CI, a dev VM): failing *security* canaries only warn. In `prod` a failing security canary also makes `POST /v1/jobs` answer `infra: setup`. `GET /v1/profiles` reports the mode, the acceptance suite's `-require-prod` fails on `dev`, and judge refuses a `dev` runner in production (hand-off to [m3-06](sprint-m3-06.md)).
- **Typed errors** emitted here: `setup` (a spawner syscall failed before learner code ran, or a failed teardown assertion), `killed` (SIGTERM/drain mid-job; judge re-queues it as `saturated`, spending no retry), `job_timeout`, `saturated`.
- **Drain:** SIGTERM → stop admitting (503), finish in-flight jobs within 60 s (the pod's grace is 70 s), answer `killed` for anything unfinished, exit 0.
- **Front rotation (t3 §5.2):** after `RUNNER_MAX_JOBS` jobs or `RUNNER_MAX_AGE`, and **at once after any SIGSYS**, drain and exit so kubelet restarts the container. `BootEpoch` = `<hostname>/<container start, unix ns>/<8 random bytes hex>`, stamped into every Result so a suspect epoch can be regraded.
- **No alerting (D34):** SIGSYS, quarantine-shaped events and teardown failures are ERROR log lines plus `Telemetry` / `/v1/stats` counters that judge persists and `judge admin` reads on demand.

### 6 · `throttled`: steal, runner-owned canary, in-runner quiet re-run [X]

Sources: [t3 §5.7](../research/t3-sandbox.md#57-throttled-and-the-quiet-re-run-learner-proof-bounded) (the T4 §11.1 `throttled` semantics are amended there: "no clean quiet re-run was possible"), [ADR-0030 §3](../../adr/0030-runner-technology-and-host-hardening.md#3-timing-on-a-noisy-4-vcpu-vm), [t3 §15](../research/t3-sandbox.md#15-s0-results-read-only-production-facts-2026-09-24-owner-approved) (cgroup CPU time **includes steal**: `PARAVIRT_TIME_ACCOUNTING` is off).

- **s1 — steal:** the host `cpu` line of `/proc/stat` over the case window.
- **s2 — canary:** `runner canary`, ~150 ms of mixed ALU work plus a memory stream larger than the LLC (> 16 MiB per L3 slice on the EPYC 9355P), run by the spawner in a slot cgroup with caps dropped. It runs **every 5 min when idle** (rolling median → `Versions.CanaryMedian`) and **on demand** after a failing time verdict, once no process of that job is alive. `nr_throttled` and PSI are **not** signals (dropped in t3 §5.7).
- **Suspect time verdict:** a CPU-TLE or wall kill is suspect if (a) the other slot was busy during the case, (b) s1 ≥ 0.10, or (c) s2 ≥ 1.2 × median.
- **Quiet re-run:** stop admitting (503); wait ≤ 60 s for the other slot to drain; require steal < 5% over 10 s and a pre-canary ≤ 1.1 × median; re-run the failing case and the rest of its group; **the quiet verdict is final** (a wall kill in quiet mode is TLE). If quiet conditions can't be met in 60 s, or s1/s2 fire during the re-run, set `Result.Throttled=true` (judge re-queues once after 5 min, then `inconclusive(runner_throttled)` — [m3-06](sprint-m3-06.md)).
- Thresholds are constants in one file with a comment pointing at the S0 steal data; [mi-10](sprint-mi-10.md) may retune them on prod (a runner patch).

### 7 · Hostile corpus, cleanup invariants, Linux CI job `runner-it` [X]

Sources: [t3 §5.10](../research/t3-sandbox.md#510-cleanup-invariants-release-gate-a8), [t3 §9](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead) (the P2 corpus), [t3 §2.4](../research/t3-sandbox.md#24-attacks-and-the-control-that-stops-each-one).

- **Test profile `testgo@0`** (`internal/runner/profile/testgo/`, build tag `runner_it`, never in a release build): compile = `go build` with the job's Go toolchain bound read-only; exec = the artifact; the per-case filter = spk-02's amd64 `go` allowlist from t3 §16.2 (m3-04 moves it into the real `go@1.26` profile). A `-tags runner_it` guard test asserts the release build's registry has no `testgo`.
- **Corpus** as committed Go programs under `internal/runner/testdata/corpus/<name>/main.go` (`testdata` keeps them out of `./...`): 1 GiB balloon, fork bomb, thread bomb, tmpfs fill, inode fill, stdout flood, orphan double-fork, sleep (idle), spin, a SIGSYS probe (`bpf`, `io_uring_setup`, `unshare(CLONE_NEWUSER)`, `mount`, `ptrace`, `keyctl`), a network probe (TCP to `1.1.1.1:443`, UDP to `10.43.0.10:53`) and cross-job markers (`/w`, `/tmp`, `/dev/shm`, SysV shm, POSIX mq, an abstract socket, a keyring). The SQL rows and the race fixture wait for their profiles (not M3).
- **Integration tests** (`//go:build linux && runner_it`, `internal/runner/it/`), run as root with a delegated cgroup:
  - balloon × 100 → **MLE 100/100**, the container's `memory.events oom_kill` delta **= 0**;
  - fork and thread bombs → RE (fork limit) or SIGSYS; stdout flood → OLE; spin → TLE; sleep → TLE (idle); tmpfs/inode fill → the program's own error (RE/WA), never infra;
  - SIGSYS probe → `Term=signal(SIGSYS)` and the front rotates (new `BootEpoch`);
  - network probe → every connect fails (empty netns, loopback down);
  - job N's markers are invisible to job N+1;
  - orphan double-fork → **0 surviving processes** after teardown;
  - after **1,000 case cgroups**: no leftover cgroups, mounts or processes; `runner/` memory back to baseline ±5 MiB; `nr_dying_descendants` → ~0 within 60 s;
  - **L14 tests cap:** a job of many cases that each spin just under TL crosses 45 s → the in-flight case `tle`, the rest `not_run`, `TestsCapHit=true`, no `Infra`; the same job with the other slot busy → `Throttled=true`. Record the measured per-case jail overhead (spawn + teardown) in the PR: the 5 s between Σ TL ≤ 40 CPU-s and the cap must absorb it for ≤ 512 cases; if it doesn't, hand the number to m3-02/m3-13 (a lower Σ TL or case-count lint), never raise the cap;
  - contract: 401, 400, 503 on a third concurrent job, client disconnect → no survivors, drain → `killed`.
- **CI job `runner-it`** in `.github/workflows/ci.yml`: `ubuntu-24.04`, a job `container:` of `debian:trixie-slim@sha256:…` (the image base m3-15 pins) with `options: --privileged --cgroupns=private --memory=3g --cpus=2` (mirrors the pod's limits, so "0 container OOMs" is meaningful); install the pinned Go tarball; `go test -tags runner_it -race=false -count=1 ./internal/runner/...`. Fallback if the container job can't delegate cgroups: `sudo systemd-run --scope -p Delegate=yes -p MemoryMax=3G -p CPUQuota=200% …` on the VM itself. Record which one works.
- `Makefile`: `runner-it` (the same command, for a Linux dev VM). **macOS can't run the jail**; the optional dev loop is an arm64 multipass VM (`multipass launch 24.04 --cpus 4 --memory 8G`). Until [m3-04](sprint-m3-04.md) adds the arm64 name lists, the arm64 per-case filter's default is LOG via a **dev-only switch** (honoured only under the `runner_it` build tag with `RUNNER_MODE=dev`; the allowlists are amd64, t3 §16.2). No release or image build ever has a LOG default: from m3-04 on, arm64 filters are KILL-default like amd64.
- **CI stands in for a local privileged VM run.** The P2 corpus classification is proven in CI `runner-it` (a privileged Linux container); a Linux VM run with `make runner-it` is optional and equivalent. What this CI job does **not** exercise (the pod's user namespace, Localhost AppArmor and pod seccomp, the VAP) is covered by m3-15's VM rehearsal and [mi-10](sprint-mi-10.md) on prod.

### 8 · Docs: `docs/architecture/runner.md`, `services.md`, hand-offs [X]

- New `docs/architecture/runner.md` (created here): placement, the process model and R-FS, the jail, the cgroup layout, measurement, the kill policy, the API table, config env, typed errors, `throttled`, rotation and `BootEpoch`. Link ADR-0030 and t3; no duplicated rationale.
- [`docs/architecture/services.md`](../../architecture/services.md): a `runner · sandbox (xlearn-runner ns)` entry: judge is its only caller, no DB, no NATS, no egress, own release stream.
- Hand-off lines in `docs/v2/status.md`: to [m3-04](sprint-m3-04.md) (the `Profile` interface incl. `ArtifactMode`, the seccomp builder, where the lists go, the dev-only LOG switch to retire), [m3-06](sprint-m3-06.md) (`runnerapi` fixtures, `ValidateResult`, judge infers `runner_oom`, the 45 s tests cap → TLE via the existing term mapping, and **refuse a `dev` runner in production**: its `GET /v1/profiles` check must read `mode` and turn the runner lane off, with an ERROR log, when `mode=dev` in prod), [mi-10](sprint-mi-10.md) (port 8090, `/readyz`, `/healthz`, token path and key, `RUNNER_IMAGE_DIGEST` set with the same `$imagepolicy …:digest` marker as `image.digest`, `SETPCAP` answer, `emptyDir` needs if any), [m3-15](sprint-m3-15.md) (`/v1/stats`, `RUNNER_MODE`, the SIGSYS rotation the acceptance suite must wait across).

## Acceptance criteria

- [ ] ADR-0030 is **Accepted** with the spike's results table, the chosen mechanism and ADR-0035 §6 folded into §5 (A3, A4, A5, A6, A8, Track B); **nothing from the old §7 is lost** (the mapping is in the PR description); the ADR index row is updated.
- [ ] `internal/platform/runnerapi` round-trips the golden `job`/`result` fixtures and enforces every stream cap mid-stream; it has no non-stdlib import.
- [ ] The **P2 corpus is classified correctly in a Linux jail-capable run** (CI `runner-it`, standing in for a local privileged VM run): **MLE 100/100, 0 container OOMs**, TLE/TLE (idle)/OLE/RE as specified, SIGSYS → `signal` + rotation, every network probe fails.
- [ ] The **L14 caps hold**: compile ≤ 15 s → CE; the 45 s tests cap → in-flight `tle`, rest `not_run`, `TestsCapHit` (or `Throttled` when suspect); `job_timeout` only past the 170 s backstop.
- [ ] **Cleanup invariants green**: 0 surviving processes, `runner/` memory baseline ±5 MiB, `nr_dying_descendants` → ~0 within 60 s after 1,000 case cgroups, cross-job markers invisible.
- [ ] Contract behaviour: 401 / 400 / 503-on-third-job / disconnect-kills / drain-`killed` tested.
- [ ] No release build contains `testgo@0`; no package imports `go-sandbox/container`; `go build ./...` and `go test ./...` are green on macOS and Linux; CI green (`sqlc diff` unchanged).
- [ ] `docs/architecture/runner.md` and the `services.md` entry exist; hand-offs are in status.md.

## Release

**Merge only.** Nothing is built or deployed from this sprint:
- the runner binary first ships in the `runner-v1.0.0` image ([m3-15](sprint-m3-15.md)), deployed dark by [mi-10](sprint-mi-10.md);
- `deploy.yml` doesn't build the runner (it has no runner job, and its `v*` filter never matches `runner-v*`), so the next fleet tag carries only `internal/platform/runnerapi` inside judge (from `v1.13.0`, [m3-07](sprint-m3-07.md)).

## Definition of Done

CI green, including the new `runner-it` job and an unchanged `sqlc diff` · ADR-0030 Accepted · acceptance criteria met · no host, cluster or infra change · statuses updated (this file + [`../status.md`](../status.md): board, M3 row, MI-12 "runner code" part, ADR-0030 line, hand-offs) · notable calls in the decisions log.

## Risks / watch-outs

- **macOS can't run the jail.** Everything Linux-only is build-tagged with stubs; the real proof is the CI `runner-it` job. Don't mark a task ✅ on a macOS run.
- **A k3d (or kind) rehearsal expects NotReady.** k3s-in-Docker lacks the host files (the Localhost AppArmor profile, the pod seccomp file, the kubelet subuid range), so the pod is refused or `/readyz`'s security canaries fail and it stays NotReady in `prod` mode. That's correct behaviour, not a bug to work around; the pod-shape rehearsal is m3-15's multipass VM.
- **The CI container is not the pod.** A privileged container with a private cgroupns exercises the runner's own layers (jail, cgroups, per-case seccomp, measurement) but not `hostUsers:false`, Localhost AppArmor, the pod seccomp or the VAP. m3-15's VM rehearsal and mi-10 on prod cover those.
- **Go and `fork`.** Never fork from a multi-threaded Go process by hand; go through `forkexec.Runner` (it clones on a locked OS thread and runs only async-signal-safe steps before `execve`), or the nsjail binary.
- **Capability drops are per thread in Linux.** The front must start capless (re-exec'd with credentials, bounding set and NNP set in the child before `execve`), never drop caps from a running Go process.
- **`CLONE_INTO_CGROUP` on 6.8.** Use whichever path §16.1 validated; keep the other behind a flag with a test.
- **Dying memcgs.** Page cache charged to a removed cgroup keeps it "dying". Every writable path is tmpfs and is unmounted before `rmdir`; if `nr_dying_descendants` still doesn't converge, pre-read toolchains into `runner/` (task 3) is the fix, not a longer timeout.
- **Container restarts and kubelet back-off.** SIGSYS-triggered rotations restart the container; repeated fast restarts hit CrashLoopBackOff (judge sees `saturated`). Judge's budgets and the poison-pill quarantine bound it ([m3-06](sprint-m3-06.md)). If it bites in the acceptance suite, the fallback is an in-process front restart with a new `BootEpoch`; record that as an ADR-0030 delta, don't improvise it silently.
- **Steal inflates CPU time** (S0). Never tune a TL on a noisy run; `throttled` and the quiet re-run exist for this.
- **Contract churn.** Every `runnerapi` field is append-only from `runner-v1.0.0`; removing or retyping one is a runner major (GA in miniature, ADR-0034 §1.5).
