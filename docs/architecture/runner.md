# Runner (`xlearn-runner`)

The sandbox that runs learner code for judge, as built in [m3-03](../v2/sprints/sprint-m3-03.md). The decision
and its rationale are [ADR-0030](../adr/0030-runner-technology-and-host-hardening.md); the research, threat model
and spike results are [t3](../v2/research/t3-sandbox.md). This page is the map of the code.

## Placement

- One Deployment (`strategy: Recreate`, grace 70 s) in the `xlearn-runner` namespace, on the production node,
  under the `judge` runtime handler with `hostUsers:false`, the five namespaced capabilities
  (`SYS_ADMIN SETUID SETGID SETPCAP KILL`), Localhost AppArmor and seccomp, no SA token, no DNS, no egress
  ([mi-10](../v2/sprints/sprint-mi-10.md) deploys it; [mi-14](../v2/sprints/sprint-mi-14.md)'s VAP pins the shape).
- **Judge is the only caller** (ClusterIP `:8090`, a bearer token). The runner has no database, no NATS, no egress
  and no secret but that token, and never initiates a connection ([`services.md`](services.md)).
- Own release stream: `runner-v*` tags and the image `ghcr.io/sujaykumarsuman/xlearn-runner` ([m3-15](../v2/sprints/sprint-m3-15.md)).

## Process model and R-FS

One binary (`cmd/runner`), several roles:

| Process | Identity | Does | Never does |
|---|---|---|---|
| **spawner** (`runner`, PID 1; `internal/runner/spawner`) | UID 0 in the pod's user namespace, the five namespaced caps | cgroups, the per-job tmpfs mounts, jails; reads cgroup evidence and wait statuses; `cgroup.kill` | parse a learner byte; listen on a socket; open, chown, copy or traverse a learner-writable path after learner code ran |
| **front** (`runner front`; `internal/runner/front`) | UID 65532, **0 caps, an empty bounding set, `NO_NEW_PRIVS`**, all set in the child before `execve` | HTTP, bearer auth, spooling inputs, writing the source files, streaming inputs to fd 3, decoding fds 1/2/4 and diagnostics, assembling the Result | hold a capability; touch cgroupfs (it refuses to start if `/proc/self/status` shows any privilege) |
| canary (`runner canary`) | UID 65531, capless, in a slot cgroup | the fixed throttling workload (s2) | — |
| compile init (`runner compile-init`) | the job's compile UID, inside the compile jail | runs the build as a child, then exports the artifact over fd 3 | — |

- **IPC** (`internal/runner/ipc`): a `socketpair(AF_UNIX, SOCK_SEQPACKET)`; 128-byte, versioned, fixed-schema
  messages (enums, sizes, limits, indices — never learner content); strict decode; per-case pipe fds travel with
  `SCM_RIGHTS`. Any malformed, unknown or wrong-direction message breaks the pair: the front exits, the spawner
  kills it and exits, kubelet restarts the container.
- **The token** is read once by the spawner (it can read a 0400 Secret file) and handed to the front over a pipe
  with the startup manifest; it is compared in constant time and never logged.
- **Sources without breaking R-FS:** the spawner mounts a per-job `src` tmpfs owned by the front's UID (0700) and
  passes the front a directory fd; the front writes the files 0444 and sets the directory 0555 before the compile.
  The spawner never reads, chmods or traverses it.
- **Artifacts:** the compile jail's init opens the build output with `O_NOFOLLOW` and streams its raw bytes
  (≤ 64 MiB) over a pipe; the spawner writes them into a per-job artifact tmpfs (directory `0111`, owned by pod
  root; the file mode is the profile's `ArtifactMode`: `0111` for a static ELF, `0444` for an interpreted
  artifact) **before any learner code runs**, without parsing them.

## The jail

`internal/runner/jail` wraps go-sandbox's low-level `forkexec.Runner` (pinned `v0.13.7`; only `pkg/forkexec`,
`pkg/mount`, `pkg/rlimit`; never `go-sandbox/container` — `internal/runner/imports_test.go`):

- clone flags `NEWNS|NEWPID|NEWNET|NEWIPC|NEWUTS|NEWCGROUP`, **never `NEWUSER`**; loopback stays down;
- a fresh tmpfs root and `pivot_root` (the root is remounted read-only);
- read-only binds as `MS_BIND|MS_REC|MS_NOSUID|MS_NODEV|MS_RDONLY` **without `MS_PRIVATE`** (the AppArmor remount
  rule; t3 §16.2 block 3): the artifact directory at `/job`, the profile's toolchain paths at their own paths;
- `/dev/null`, the one device, as a read-write bind (go-sandbox's `WithBind(…, false)` shape);
- a per-case `/w` tmpfs (`nosuid,nodev,noexec`, 64 MiB / 4k inodes by default); **no overlay** (it EACCESes under
  `hostUsers:false`); **no `/proc`** unless a profile asks (`hidepid=invisible,subset=pid`, go-race later);
- a per-job UID from a 50,000-UID pool (two per job: compile, exec; not reused within 1,000 jobs);
- caps → 0 with `SECURE_NOROOT` locked, **an empty bounding set**, `NO_NEW_PRIVS`, `RLIMIT_CORE=0`, `RLIMIT_CPU`
  at TL + 2 s and the profile's `FSIZE`/`NOFILE`/`STACK`;
- the profile's seccomp filter loaded last; fds `0 /dev/null`, `1`/`2` capped pipes, `3` input, `4` harness result.

**Seccomp** (`internal/runner/seccomp`): classic BPF assembled from syscall-name lists, with the fixed rules — a
foreign arch or an x32 number → `KILL_PROCESS`; the dangerous set (io_uring, bpf, perf, userfaultfd, the keyring,
ptrace, `process_vm_*`, the mount family, unshare/setns, socket, splice/vmsplice/tee) → `KILL_PROCESS`; `clone3`
→ `ENOSYS`; optionally `clone` only with `CLONE_THREAD` and no namespace flag, `prctl` only with `PR_SET_VMA`.
Exec filters default to `KILL_PROCESS`; compile filters to `ENOSYS`. A dev-only switch gives an arm64 exec filter
a LOG default under the `runner_it` build tag with `RUNNER_MODE=dev` (the lists are amd64 until m3-04).

**Spawn path** (`RUNNER_SPAWN_PATH`): `clone_into_cgroup` (default, clone3 + a cgroup fd) or `cgroup_procs` (a
`cgroup.procs` write gated on a sync pipe). The bounding set is emptied on a locked OS thread that spawns and is
then discarded, which needs `CAP_SETPCAP` — as does go-sandbox's `PR_SET_SECUREBITS`.

## Profiles

`internal/runner/profile` holds the `Profile` type (compile and exec specs, binds, env, limits, rlimits, filters,
`ArtifactMode`, the toolchain paths to pre-read, a no-op baseline program) and the registry. m3-04 adds `go@1.26`,
`cpp` and `python`. The test-only `testgo@0` and `testgo-open@0` (build tag `runner_it`) exist for the jail suite
and are never linked into a release build.

## Cgroup layout (`internal/runner/cgroup`)

```
/sys/fs/cgroup            (container: cpu.max 2 · memory.max 3 GiB — set by the pod)
├─ runner/                spawner + front
└─ slots/                 +cpu +memory +pids
   ├─ s0/  cpu.max 1 CPU · memory.max 1.25 GiB · swap 0 · pids.max 512
   │   ├─ job/ ├─ compile/   (profile memory · oom.group 1 · pids 256)
   │   │       └─ case-K/    (memory.max = mem_mb + baseline · oom.group 1 · swap 0 · pids.max)
   │   └─ canary/
   └─ s1/  same
```

INV-14: 2 × 1.25 GiB + `runner/` (≤ ~300 MiB) < 3 GiB, so a learner OOM lands in a case or compile cgroup, never
in the container. No `memory.high`, no `io.max` (every writable path is tmpfs).

**Startup** (before serving): move into `runner/`, wipe and rebuild `slots/`, unmount leftovers, pre-read and hash
each profile's toolchain (its page cache is charged to `runner/`, and the hash goes into `ProfileSHA`), seed the
canary median, compile and run each profile's no-op program to measure its memory baseline (median `memory.peak`).

## Measurement (`internal/runner/measure`, pure functions)

| Field | Source |
|---|---|
| `CPUms` (the verdict clock) | Δ `cpu.stat usage_usec` of the case cgroup (cross-checked with `wait4` rusage) |
| `WallMs` | monotonic clock from the jail's exec to its reaping |
| `PeakKB` | case `memory.peak` − the profile's baseline |
| MLE | `memory.events` `oom_kill > 0` |
| fork limit | `pids.events` `max > 0` → `ForkLimit` (RE); it wins over an OLE its own crash dump caused |
| OLE | the front's byte counters: fds 1/2 past 1 MiB, fd 4 past `OutputKB` |
| exit / signal | wait status; SIGSYS → `Term=signal`, `Signal=SIGSYS` (judge: RE, counted) |

**Kill policy** (polled every 10 ms): CPU > TL + min(0.5 s, TL/2) → TLE; wall 1.5·TL + 0.5 s → TLE (idle if the
last second's CPU rate was < 5%, else a wall kill); CPU rate < 5% for ≥ 1 s → TLE (idle); `RLIMIT_CPU` at TL + 2 s
is the backstop. The compile gets ≤ 15 s of CPU and wall, the profile's memory and 256 pids; a limit hit is a CE
(`Compile.Limit`) and its CPU counts in `Telemetry.SlotMs`.

**L14 tests cap:** the whole test phase (every case from the first case start; the compile and the quiet re-run
excluded) gets 45 s of wall. When it fires, the in-flight case is `tle`, every later case `not_run`,
`Telemetry.TestsCapHit=true` — TLE, counted — or `Throttled=true` if the job was suspect while the cap ran. The
quiet re-run gets its own 45 s. `job_timeout` = 15 + 45 + (60 + 45) + 5 = 170 s is a runner-fault backstop only.

**Case order** (t4 §2.6): cases arrive sample → edge → random → perf. `StopGroupOn`: `any_fail` ends the test phase
at a group's first non-ok case (a sample failure skips the hidden cases); `tle` ends the group at its first TLE (the
perf block).

**Teardown** after each job: `cgroup.kill`, `populated 0`, `rmdir`; unmount the artifact and `src` tmpfs; the slot's
`memory.current` back to its baseline ±5 MiB and `pids.current` 0 — else the slot fails closed (`infra: setup`) and
the runner rotates.

## API (the front)

| Endpoint | Auth | Behaviour |
|---|---|---|
| `POST /v1/jobs` | bearer | `Content-Type: application/vnd.xlearn.runner-job; v=1`: `job.json` then one frame per case, `u32`-BE length-prefixed; caps enforced from each prefix (job.json ≤ 1 MiB, a case ≤ 8 MiB, inputs ≤ 16 MiB, ≤ 512 cases). `200` + Result; `400` contract error; `401`; `503` + `Retry-After` + `{"infra":{"kind":"saturated"}}` when both slots are busy, draining, or a quiet re-run holds exclusivity. A client disconnect kills the job subtree and sends no Result |
| `GET /v1/profiles` | bearer | the profiles (toolchain, `profile_sha256`, harnesses, `mem_baseline_kb`; calibration fields from m3-04/A8), `boot_epoch`, `canary_median_us`, `mode` |
| `GET /v1/stats` | bearer | per-slot `memory.current`, `pids.current`, `nr_dying_descendants`; `runner/` memory; the container's `oom_kill` (hierarchical and `.local`); jobs served; SIGSYS count |
| `GET /readyz` | none | 200 only when every canary passed in the last 60 s: egress to `10.43.0.10:53`, `10.43.0.1:443`, `1.1.1.1:443` fails; the AppArmor label is `xlearn-runner`; the UID map isn't the identity map; cgroupfs is writable; `clone(CLONE_NEWUSER)`, `fsopen("tmpfs")` and an SCTP socket are denied; `core_pattern` isn't a pipe. 503 while draining |
| `GET /healthz` | none | the front is up and the IPC pair answers |

The contract types are `internal/platform/runnerapi` (stdlib-only, append-only from `runner-v1.0.0`; golden
fixtures in its `testdata/`). Judge validates every Result with `runnerapi.ValidateResult`.

**Typed infra errors** (`Result.Infra`): `setup` (a spawner syscall failed before learner code ran, or a teardown
assertion failed; 200), `killed` (drain; 503, judge re-queues as saturated), `job_timeout` (the 170 s backstop;
200), `saturated` (503). Judge infers `runner_oom` from a dropped connection plus a changed `BootEpoch`.

## `throttled` and the quiet re-run

- **s1:** the node's steal fraction from `/proc/stat` over each case window.
- **s2:** the runner canary (~150 ms of ALU work plus a 48 MiB memory stream), run capless in a slot cgroup every
  5 min when idle (the rolling median of 12 is `CanaryMedian`) and on demand after a failing time verdict. Each
  measurement is the faster of two back-to-back runs (a CPU that just idled runs its first pass slow).
- **Suspect:** a CPU or wall kill while the other slot was busy, or with s1 ≥ 0.10, or s2 ≥ 1.2 × median; a wall
  kill under TL with a CPU rate ≥ 5% is always suspect.
- **Quiet re-run** (at most one per job): stop admitting (503), wait ≤ 60 s for the other slot to drain, require
  steal < 5% over 10 s and a pre-canary ≤ 1.1 × median, then re-run the case and the rest of its group; the quiet
  verdict is final (`Quiet=true`). Otherwise, or if s1/s2 fire during the re-run, `Throttled=true` (judge re-queues
  once after 5 min, then `inconclusive(runner_throttled)`).
- Every threshold is in `internal/runner/measure/thresholds.go`, with a pointer to the S0 steal data.

## Rotation, drain and `BootEpoch`

- **Drain:** SIGTERM to the spawner → the front stops admitting (503), finishes in-flight jobs within
  `RUNNER_DRAIN_TIMEOUT` (60 s; the pod's grace is 70 s), answers `killed` for the rest, and exits 0; so does the spawner.
- **Rotation:** after `RUNNER_MAX_JOBS` (500) jobs or `RUNNER_MAX_AGE` (6 h), at once after any SIGSYS, and when a
  slot fails closed: the same drain, then kubelet restarts the container.
- **`BootEpoch`** = `<hostname>/<container start, unix ns>/<8 random bytes hex>`, stamped into every Result so a
  suspect epoch's evaluations can be regraded.
- **No alerting (D34):** SIGSYS, teardown failures and other runner faults are ERROR log lines plus counters in
  `Telemetry` and `/v1/stats`, read on demand.

## Config (env)

| Var | Default | |
|---|---|---|
| `RUNNER_ADDR` | `:8090` | the front's listen address |
| `RUNNER_TOKEN_FILE` | `/var/run/secrets/runner-auth/token` | the Secret `runner-auth`, key `token` |
| `RUNNER_MODE` | `prod` | `dev` only downgrades failing *security* canaries to warnings; `prod` also answers `infra: setup` while one fails. Judge refuses a `dev` runner in production |
| `RUNNER_IMAGE_DIGEST` | `unknown` | reported in `Versions.ImageDigest` |
| `RUNNER_MAX_JOBS` / `RUNNER_MAX_AGE` | `500` / `6h` | rotation |
| `LOG_LEVEL` | `info` | |
| `RUNNER_DRAIN_TIMEOUT` | `60s` | ≤ 65 s |
| `RUNNER_SPAWN_PATH` | `clone_into_cgroup` | or `cgroup_procs` |
| `RUNNER_CGROUP_ROOT` / `RUNNER_JAIL_DIR` | `/sys/fs/cgroup` / `/jail` | `/jail` is an `emptyDir`, the only place the AppArmor profile admits mounts |
| `RUNNER_FRONT_UID` / `RUNNER_CANARY_EVERY` | `65532` / `5m` | |

## Tests

- Unit tests everywhere (macOS included): the contract codec and validators, the IPC codec, the kill policy and
  classification, the seccomp assembler (with a BPF interpreter), the front's orchestration over a fake spawner.
- **`runner-it`** (CI job; `make runner-it` as root on a Linux VM; build tags `linux && runner_it`,
  `internal/runner/it/`): the hostile corpus (`internal/runner/testdata/corpus/`), the cleanup invariants over 1,000
  case cgroups, the L14 caps, the quiet re-run and the API contract against the real binary in a privileged
  `debian:trixie-slim` container with a private cgroup namespace and the pod's limits. It does not exercise the pod's
  user namespace, the Localhost AppArmor and seccomp profiles or the VAP: m3-15's VM rehearsal and mi-10 do.
