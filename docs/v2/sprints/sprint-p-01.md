# Sprint p-01 — go-race runner profile → runner-v1.1.0

> **Milestone:** P — pilot course (go-concurrency)   ·   **Track:** product (runner stream; order 57)
> **Prereqs:** [m3-13](sprint-m3-13.md) (M3 shipped, `v1.14.0`) · [spk-02](sprint-spk-02.md) (P3 TSAN verdict) · [ds-p-01](sprint-ds-p-01.md) (PRD Q5 recorded; AB14–AB15 frozen)
> **Unblocks:** [p-02](sprint-p-02.md) (multi-course + multi-file workspace on a runner that grades `gotest@1`) → [p-03](sprint-p-03.md) (`v1.15.0`)
> **Release action:** **`runner-v1.1.0`** (the runner stream; the 2nd ImageUpdateAutomation deploys it dark, no infra PR). The judge / `internal/course` / curriculum changes merge to `main` and ship dark in **`v1.15.0`**, tagged by [p-03](sprint-p-03.md).
> **Calendar:** December (after v1.14.0 and the `ds-p-01` merge, which records Q5 and is the AB14–AB15 freeze). No owner time, unless the pod-level seccomp gate finds a gap: then this session lands everything that doesn't depend on the host change and records the tag ⛔ in status.md, the owner books a short host window, and a re-run of the prompt on that date applies the host change (pre-approved by launching it, D40) and tags.
> **Execute with:** [`../prompts/prompt-p-01.md`](../prompts/prompt-p-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | `go-race@1.26` profile: image deltas, ASLR launcher, exec allowlist, pod-profile check, limits, profile health, slot memory budget | X (+ I, and O before a re-run, only if the pod profile lacks the rule) | ⬜ |
| 2 | `gotest@1` harness: multi-file module, TestMain on fd 4, per-test exec, RACE / DEADLOCK / LEAK, goroutine dump | X | ⬜ |
| 3 | Item schema + `contract_hash` for multi-file modules (additive) | X | ⬜ |
| 4 | judge: map `gotest@1`, honor trust, signals, `Present()` denylist, evaluable vs runner profiles, lints | X | ⬜ |
| 5 | Acceptance suite + CI: race ≥ 19/20, deadlock 20/20, leak 20/20, INV-14, ±5 % speed, reproducible digest | X | ⬜ |
| 6 | ADR: go-race profile and `gotest@1` harness | X | ⬜ |
| 7 | Tag `runner-v1.1.0`; verify dark on prod; calibrate the go-race baseline | X + H | ⬜ |
| 8 | Record (status.md: runner stream, freeze rows, decisions) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row,
> milestone P, the runner stream row). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] M3 shipped (v1.14.0)
- [ ] P3 TSAN result GO for go-race (spk-02)
- [ ] PRD Q5 recorded (`ev-q5`, resolved in ds-p-01 before AB15 was drafted): ds-p-01's Q5 record PR (`docs/ds-p-01-q5`)
  is merged, so PRD §7 Q5 reads resolved on `main`. ds-p-01 merges it on CI green (D40); if it isn't merged, ds-p-01
  isn't done, which is a gate failure
- [ ] AB14–AB15 frozen: ds-p-01's board PR is merged (the merge is the freeze)
- [ ] `runner-v1.0.0` live dark ([mi-10](sprint-mi-10.md)) with `baseline@1` and the Go/C++/Python multipliers recorded; ADR-0030 **Accepted** ([m3-03](sprint-m3-03.md))
- [ ] **The production pod-level seccomp profile permits the ASLR mechanism t3 §16.2 recorded.** The host file
  `/var/lib/kubelet/seccomp/profiles/xlearn-runner.json` ([mi-09](sprint-mi-09.md), the amd64 file recorded verbatim in
  t3 §16.2) must allow `personality` with `ADDR_NO_RANDOMIZE` (0x0040000), plus the self-`execve` if §16.2 needed it.
  It is built from RuntimeDefault, whose `personality` rule admits only 0x0, 0x8, 0x20000, 0x20008 and 0xffffffff, so
  any other value gets EPERM at the pod layer, whatever the image's exec allowlist says. Check it read-only: compare the
  heredoc in `../infra/hack/host-bootstrap.sh` with §16.2's text, and confirm the host file matches that heredoc
  (`host-verify --cluster --expect-sandbox`'s `sandbox.seccomp` sha256 = `hack/host-bom.txt`, or
  `ssh sujaykumar-vps 'sudo cat /var/lib/kubelet/seccomp/profiles/xlearn-runner.json'`). Skip this gate only if §16.2 says go-race
  needs no `personality` call.
- [ ] **Parallel sessions:** no open peer PR touches `internal/runner/`, `deploy/runner.Dockerfile`, `.github/workflows/{runner-release,ci}.yml` or `internal/judge/grader/`; no peer `runner-v*` tag in flight (`git ls-remote --tags origin 'refs/tags/runner-v*'`, `gh pr list`, `git worktree list`, ListAgents)

**Stop and report** if Q5 recorded **SQL** (p-01 is re-planned to a `sql-pg` profile before it starts), if status.md
marks p-01 ⛔ "needs owner decision" (ds-p-01 found the P3 result inconclusive for go-race), or if t3 §16.2
says go-race failed even with a per-process `setarch -R` launcher (go-concurrency needs a redesign; Q5 goes back to the owner).
If the **pod-level profile gate** fails, tasks 1–6 still merge (they ship dark). **Don't tag in this session** (task 7):
prepare the host change (task 1, "Pod-level profile"), land everything that doesn't depend on it, and record task 7 ⛔
in status.md with the owner action named (book a host window). Nothing waits. A re-run of the prompt on the booked date
applies the host change and picks up at task 7. The tag waits until `host-verify` shows the new file.

## Goal

Give the runner the pilot's **`go-race@1.26`** profile — `go test -race` in the jail under production's
`vm.mmap_rnd_bits=32`, with the per-process ASLR policy P3 required — and the **`gotest@1`** harness: multi-file Go
modules, a harness-owned `TestMain` reporting on fd 4, one process per declared test, and the RACE / DEADLOCK / LEAK
classes with a goroutine dump reduced to learner-package frames. Teach judge to map `gotest@1` results as
**honor-grade** evidence, add the item-schema and lint pieces, prove it with the acceptance suite, and release
**`runner-v1.1.0`**, deployed dark by image automation. Nothing learner-visible changes: no `gotest@1` item exists until
[p-02](sprint-p-02.md) and its pack until [p-03](sprint-p-03.md).

## Scope

**In**
- Runner: the `go-race@1.26` profile, its image deltas (race-instrumented `GOCACHE` seed, pinned `goleak` in the
  read-only `GOMODCACHE`, the C toolchain if not already present for `cpp`), the ASLR launcher, the exec seccomp
  allowlist from P3, limits, and a profile-level health canary.
- Runner: the `gotest@1` harness (multi-file module, TestMain, test2json in Run, per-test deadlines, dump parsing, classes).
- `internal/course` + curriculum schema: the additive `module` block for `gotest@1` code parts and its `contract_hash` inputs.
- judge: result mapping, `TrustOf` = honor, signals `race` / `deadlock` / `leak`, the `Present()` denylist, the evaluable
  index checked against the runner's `GET /v1/profiles`, and the lints (Σ TL ≤ 40 CPU-s, `mem_mb` ≥ 2 × reference peak,
  the go-race A5 file/directive rules, the compile-leak fixture).
- The acceptance-suite extension, the CI job, a new ADR, the `runner-v1.1.0` tag and the prod dark verification.

**Out**
- `sql-pg` (the SQL fallback) — only if Q5 flips, as a re-planned sprint.
- The go-concurrency manifest, items, `preview` gating and the multi-file editor → [p-02](sprint-p-02.md).
- The quiz widget, the race-verdict / dump **UI** (AB15 F3–F16) and the pilot pack with its go-race detection-rate gate
  in the evalpack CI → [p-03](sprint-p-03.md) (E).
- Flipping the pilot to `active` → GA ([ga-01](sprint-ga-01.md)).
- Any infra PR: the runner's ImagePolicy (`>=1.0.0 <2.0.0`), 2nd IUA, quota and policies already exist ([mi-10](sprint-mi-10.md)).
  The one exception is conditional: task 1's pod-level seccomp heredoc change, only if the entry gate finds the shipped host
  file lacks the `personality` rule. It waits as a pushed branch; its runbook opens and merges the PR and applies it on
  the host in an owner-booked window.

## Tasks

### 1 · `go-race@1.26` profile [X; + I, and O before a re-run, only if the pod profile lacks the rule]

Sources: [t3 §6.1](../research/t3-sandbox.md#61-image-and-release), [t3 §6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them)
(go-race column), [t3 §16.2](../research/t3-sandbox.md) (spk-02: TSAN verdict, ASLR policy, the go-race allowlist),
[ADR-0030 §2–§3](../../adr/0030-runner-technology-and-host-hardening.md#2-languages-at-m3-owner), D20.

- **Profile** (next to m3-04's `go`, `cpp`, `python` in the runner's profile registry, e.g. `internal/runner/profile/`):

  | Field | Value |
  |---|---|
  | Compile | `go test -c -race -vet=off -trimpath -o /job/out/t.test`; 15 s; 1.25 GiB; pids 256; compile seccomp ENOSYS-default as for `go` |
  | Slot | 1 slot; `cpu.max` = 1 CPU; `GOMAXPROCS=2` (interleavings) |
  | Exec unit | one process **per declared test**: `-test.run '^Name$' -test.count=N -test.timeout=<per-test deadline>`; fresh procfs `hidepid=invisible,subset=pid` |
  | Env | the `go` env (`TZ=UTC`, `LANG=C.UTF-8`, `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOFLAGS=-mod=readonly`, `GOTELEMETRY=off`, `GOMEMLIMIT` unset) + `GORACE="halt_on_error=1 atexit_sleep_ms=0 exitcode=66"`, `GOTRACEBACK=all` |
  | Limits | per-test deadline ≥ 10 × the published go-race baseline; 1 GiB per case; pids 128; **job ≤ 45 s**; **Run: `-count=1`, job ≤ 20 s** |
  | Lint allowlist (learner-editable files) | t3 §6.2's go-race column: the `go` list + `sync sync/atomic time context`, a limited `runtime` (`Gosched`, `NumGoroutine`); still denied: `os/*`, `syscall`, `unsafe`, `net/*`, `plugin`, `embed`, `reflect`, `C`. `testing/synctest` is **not** added. It is pack guidance for time-based tests, and those live in `visible_test.go` and the hidden `_test.go` files, which are not learner-editable and not linted as learner code (A5 already forbids `_test.go` among editable files) |

- **Image deltas** in `deploy/runner.Dockerfile` (stay reproducible: pinned digests, `SOURCE_DATE_EPOCH`, fixed mtimes):
  - a **separate race `GOCACHE` seed** (race-instrumented std, built at image time with a fixed future mtime) so go-race
    compiles fit 15 s and the `go`/`cpp`/`python` seeds — and therefore their `profile_sha256` — stay byte-identical to 1.0.0;
  - `go.uber.org/goleak` (MIT) pinned in the read-only `GOMODCACHE` seed (the harness imports it; `GOPROXY=off`);
  - the C toolchain the race build needs per t3 §16.2 (`gcc`, `libc6-dev`): D20's `cpp` profile likely installed it
    already — add only what is missing.
- **ASLR policy (per process, never the host sysctl):** if P3 showed the Go 1.26 race runtime fails under
  `mmap_rnd_bits=32` without help, add a tiny static launcher (e.g. `internal/runner/launcher/norand`) that calls
  `personality(ADDR_NO_RANDOMIZE)` and `execve`s the test binary, used **for go-race processes only**. If P3 showed the
  runtime re-execs itself, allow exactly that path instead. Record which in the ADR (task 6).
- **Exec seccomp** `go-race` allowlist (amd64, from t3 §16.2, KILL-default): the `go` set + threads (`clone` with
  `CLONE_THREAD`, `futex`, `sched_yield`, …) as P3 logged; `clone3` → ENOSYS (glibc falls back); `personality` allowed
  **arg-filtered** to `ADDR_NO_RANDOMIZE` and the `0xffffffff` query, and a self-`execve` only if P3 needed it. Runtime
  SIGSYS → RE (counted), as for every profile.
- **Pod-level profile (host; a read-only check unless it lacks the rule).** The in-image exec filter only narrows what
  the pod-level profile already allows. The pod layer ([mi-09](sprint-mi-09.md)'s host file, built from RuntimeDefault)
  must also allow `personality(ADDR_NO_RANDOMIZE)` (and the self-`execve`, if needed). That is the entry gate. If the
  shipped file lacks the rule:
  - **[I]** push an infra branch (e.g. `chore/host-seccomp-go-race`) with **no PR**, as mi-09 does with its window
    changes. It changes only that file's heredoc in `../infra/hack/host-bootstrap.sh` (add the arg-filtered
    `personality` value §16.2 recorded, nothing else) and its sha256 in `hack/host-bom.txt` and the `host-verify`
    constant. The commit message carries the §16.2 row and a before/after diff of the JSON, and the runbook's PR repeats
    them. Merged before the host file changes, it would make every `host-verify --expect-sandbox` FAIL `sandbox.seccomp`,
    so it merges in the window. The session that finds the gap never writes to the host.
  - **[X]** add a short runbook, `docs/v2/runbooks/host-seccomp-go-race.md`: open and merge the PR from that branch, run
    `host-bootstrap.sh --with-sandbox` (compare-then-write; no k3s restart, since the file is read at pod create), then
    `host-verify --cluster --expect-sandbox`. The `runner-v1.1.0` Recreate in task 7 then loads the new file.
  - **[O, before the re-run]** the owner books the host change: a short calendar event, the date the re-run runs on.
    Nothing waits in this session: record task 7 ⛔ in status.md
    ("the tag waits for the go-race seccomp host change; owner to book a host window"), and the gap and the branch in the
    Decisions log. A re-run of the prompt on the booked date runs the runbook (its host steps are pre-approved by
    launching the prompt, D40) and picks up at task 7, whose tag waits until `host-verify` reports the new sha256.
- **Profile health:** at runner start (and each front restart) run a **prebuilt race canary** test binary from the image
  (a known race → must report RACE; a clean test → must pass) through the launcher. If it fails, `GET /v1/profiles` omits
  `go-race@1.26` and logs at ERROR; the runner stays Ready for `go`/`cpp`/`python` (a profile problem never takes the
  runner down). `GET /v1/profiles` also reports the observed `vm.mmap_rnd_bits` (read-only) and the go-race `baseline@v`.
- **Slot memory budget (INV-14: 0 container OOMs):** a go-race compile (1.25 GiB) or test (1 GiB) plus a concurrent
  job in the other slot must fit the pod's 3 GiB limit minus the `runner/` baseline and a 256 MiB margin. If two
  concurrent go-race jobs don't fit, **cap go-race at one concurrent job**: the second gets `503 saturated` (judge
  re-queues with `run_after`; never a verdict). The pod's limits stay 2 CPU / 3 GiB, so the
  [memory-sum rule](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses) is unchanged.

### 2 · `gotest@1` harness [X]

Sources: [t3 §5.8](../research/t3-sandbox.md#58-goroutine-dump-go-race), [t3 §6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them)
(harnesses, verdicts), [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) (go-concurrency row),
[t3 §10](../research/t3-sandbox.md#10-what-t3-constrains-downstream) (T4 §11.1 amendments: go-race Run `-count=1`, ≤ 20 s).

- **Module assembly** (runner, e.g. `internal/platform/harness/gotest/`): the job carries the rendered `go.mod` (from the
  item's `go.mod.tmpl` + the harness `require` for goleak and its `go.sum` lines), the learner's editable files, the
  read-only starter files, `visible_test.go`, and — **Submit only** — the pack's hidden `*_test.go` as `HiddenFiles`
  (compile jail only, per m3-03's contract). Paths are clean relative paths from the item's file list; anything else is
  REJECTED before any learner code runs (uncounted).
- **Harness-owned `TestMain`** (`zz_xl_testmain_test.go`, generated into the test package): runs `m.Run()`, runs
  `goleak.VerifyNone` after each declared test (LEAK), and writes length-prefixed result frames on **fd 4**. A timeout or
  a non-zero exit is a failure whatever the pass lines say; a declared test with no pass frame fails.
- **Run vs Submit:** Run = visible tests only, `-count=1`, job ≤ 20 s, `test2json` events with the learner's **own**
  output (capped 8 KiB / 2 KiB as for Run today). Submit = visible tests once (a failure stops: `sample_failed`), then
  every hidden test with `-count=N` from the pack's go-race params; `TestEvent.Output` is **dropped**.
- **Timeouts and the dump:** `-test.timeout` sits below the external deadline; backstop SIGQUIT (`GOTRACEBACK=all`),
  500 ms, then `cgroup.kill`. The front parses goroutine headers and frames: a goroutine blocked in a **learner-package**
  frame → **DEADLOCK**, otherwise **TLE**. The Result carries, per goroutine group, the wait state, the wait duration and
  frames reduced to **function name + module-relative file:line in learner-editable files** — no argument words, no PC
  offsets, no runtime/stdlib frames, and never a frame in a hidden `_test.go` (collapsed to a count).
- **Race:** exit 66 + the `WARNING: DATA RACE` block → **RACE**, with the two access sites and the goroutine creation
  sites filtered the same way. "A race in any run fails", even if other runs passed.
- **Precedence** within a test: RACE > DEADLOCK > TLE > LEAK > RE > WA; CE and REJECTED are uncounted. Throttling uses
  the existing `throttled` / quiet re-run path (a go-race deadline is a time verdict; t3 §5.7).
- **Lint (A5 rules for go-race)**, front-side re-check of what judge lints first: only the item's listed editable files;
  `package` clause matches the module; no `func TestMain`, no `_test.go` among editable files, no `//go:linkname`,
  `//go:embed`, `import "C"` or `unsafe`; imports within the go-race allowlist; `go.mod` never learner-supplied.
- Additive `internal/platform/runnerapi` fields (e.g. `Result.Tests[].Class`, `.Dump`, `.Race`); judge tolerates their
  absence. Harness majors appear in `GET /v1/profiles` (`gotest@1`).

### 3 · Item schema + `contract_hash` for multi-file modules [X]

Sources: [t1 §7.1](../research/t1-content-data-model.md#71-package-format) (go-concurrency: `_starter/module/` with
`go.mod.tmpl`, editable file list, `visible_test.go`), [t1 §3.4](../research/t1-content-data-model.md#34-versioning-and-pinning),
[m1-01](sprint-m1-01.md) task 3 (freeze guard: additive only).

- `curriculum/_schema/item.schema.json` + `internal/course/item.go`: a `code` part with `harness: "gotest@1"` carries
  `config.module{path, go, files[{path, role: editable|readonly|visible_test}], api[]}` — `api[]` lists the exported
  identifiers the hidden tests may use (`NewPool`, `(*Pool).Submit`). `signature`, `samples` and `checker` are not
  required for `gotest@1` (visible tests replace samples; tests are the checker). Additive only: the freeze-guard test
  stays green; a `gotest@1` part without `module` fails `Validate`.
- Layout under `items/<id>/`: `_starter/module/{go.mod.tmpl, *.go, visible_test.go}` and the solution-stage reference
  `_code/module/…` (public, stage-gated). The embed allowlist and the leak lints ([t1 §7.2](../research/t1-content-data-model.md#72-ci-validation)) cover the new paths.
- `internal/course/canon` ([m3-01](sprint-m3-01.md)): `contract_hash` for a `gotest@1` part covers `harness@v`,
  `module.path`, `module.go` and `api[]` (what hidden tests compile against) — not file bodies, prompts or limits.
  Golden vectors updated.
- `cmd/packlint`: a `gotest@1` pack item declares `tests[]` and go-race params `{count, per_test_deadline_ms, mem_mb}`;
  hidden tests reference only `api[]` (checked by compiling against a stub built from `api[]`).
- Public content CI: the reference (`_code/module`) passes `visible_test.go` in the **runner image** with `--network none`
  (the "reference passes samples" gate, for modules).

### 4 · judge support + lints [X]

Sources: [t4 §5.2](../research/t4-judge-contract.md#52-grader-contract-go) (`TrustOf`, `Present`), [t4 §5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start)
(#11 denylist, #15 compile-leak fixture), [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only)
(signals), [t3 §10](../research/t3-sandbox.md#10-what-t3-constrains-downstream) (T1 lints), [PRD §5.5 R-CT2](../../prd/xlearn-v2-prd.md#55-content-arena-profile-privacy-from-t1--adr-0027).

- `internal/judge/grader` (the `code@1` grader in [m3-06](sprint-m3-06.md)'s closed-registry file; no new subpackage
  and no new registry entry, since `gotest@1` is a harness of `code@1`): build the `gotest@1` job (files, `HiddenFiles`, `Tests[]`,
  `Count`, per-test deadlines from the pack, `profile: go-race@1.26`); map per-test results to checks and the step
  `Class`; `Hidden` = passed / total over hidden declared tests; `FirstFailure` = the class plus learner-file locations.
- **Trust:** `TrustOf` returns **`honor`** for `gotest@1` (in-process tests), so the evaluation is `trust=honor` and
  judge-checked % excludes it ([rollout §10 P7](../rollout-plan.md#10-public-dashboard-tasks)).
- **Signals** (`evaluation_completed`, [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only)):
  `race`, `deadlock`, `leak` (and the existing `ce_only`, `sample_failed`, `wa`, `tle_small`, `re_*`). The manifest's
  prefill rules map them (p-02); no review change.
- **`Present()` denylist** (per endpoint; lint #11): Submit exposes hidden passed/total, the first failure class and
  learner-file locations only — **never** hidden test names, sources, lines or output, and no hidden-test frame. Run
  exposes visible test names and the learner's own output. Extend the denylist tests with a `gotest@1` fixture.
- **Evaluable index:** at start and on each runner `saturated`/profile refresh, judge reads `GET /v1/profiles`; a
  `gotest@1` item is `unsupported` (self path, "grading pending: self-report" badge) while the runner lacks
  `go-race@1.26`. So judge and runner can deploy in either order.
- **Lints** (`cmd/packlint`, judge start, public content CI): Σ over declared hidden tests of (per-test deadline × count)
  ≤ **40 CPU-s**; `mem_mb` ≥ **2 × the reference peak**; the go-race A5 rules of task 2; **#15 compile-leak fixture for
  `gotest@1`**: a hidden test calling a missing learner symbol shows only the fixed CE text ("Hidden tests don't compile
  against your code…"), never the hidden file's name or line.
- **Fixture:** a synthetic `zz-fixture` `gotest@1` item (public half in `internal/course/testdata/`, pack half in
  `internal/judge/testdata/pack/`), with a racy, a deadlocking, a leaking, a wrong and a correct solution, used by a
  judge + runner integration test in the jail-capable CI job. Never embedded in an image.

### 5 · Acceptance suite + CI [X]

Sources: [t3 §9](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead) (P2 pass bars:
RACE ≥ 19/20, DEADLOCK 20/20), [t3 §5.10](../research/t3-sandbox.md#510-cleanup-invariants-release-gate-a8), [t3 §6.1](../research/t3-sandbox.md#61-image-and-release) (±5 %).

- Extend `make runner-acceptance RUNNER_URL=…` ([m3-15](sprint-m3-15.md)) with a go-race block:
  - race fixture (unsynchronised counter) → **RACE in ≥ 19/20** runs at the suite's `-count`;
  - lock-order deadlock → **DEADLOCK 20/20**, with the dump naming only learner frames;
  - leaked goroutine → **LEAK 20/20**; a correct solution → pass 20/20 (no false positives);
  - a go-race 1.5 GiB balloon → MLE in its case cgroup, **0 container OOMs**; two concurrent go-race jobs (or the cap's
    `503 saturated`) → **0 container OOMs**;
  - cleanup invariants after 100 go-race jobs (TSAN shadow memory released: `runner/` back to baseline ± 5 MiB,
    0 survivors, `nr_dying_descendants` → ~0 within 60 s);
  - the profile canary and `GET /v1/profiles` (`go-race@1.26`, `gotest@1`, `mmap_rnd_bits`).
- **CI** (`.github/workflows/ci.yml`):
  - extend **`runner-it`** ([m3-03](sprint-m3-03.md)'s jail-capable `ubuntu-24.04` job; m3-04 added `g++`/`python3`)
    with the go-race integration block and the judge fixture integration test of task 4. Set
    `sysctl -w vm.mmap_rnd_bits=32` first, to mirror production. It is a global sysctl, so set it on the runner VM in a
    step before the container, or inside the `--privileged` job container. Record which works;
  - extend **`runner-image-acceptance`** ([m3-15](sprint-m3-15.md)) so its in-image `make runner-acceptance` run includes
    the go-race block (with the same sysctl set first). This is the authoritative in-image go-race acceptance.
- **Speed check:** the canary / speed index for `go`, `cpp`, `python` within **± 5 %** of 1.0.0 and their
  `profile_sha256` unchanged, so TLs carry forward (no recalibration).
- **Reproducible image:** two builds of the tagged commit give the same digest (m3-15's `runner-repro` job, as it proved for 1.0.0).

### 6 · ADR [X]

A new ADR — **next free number** (≥ 0036; check `ls docs/adr`, open PRs and peers first) — "go-race runner profile and
`gotest@1` harness (pilot)": [t4 §5.7](../research/t4-judge-contract.md#57-cost-of-adding-capability-inferred) requires
an ADR for a new runner profile and harness. Record: the ASLR mechanism P3 required (launcher vs self re-exec) and why
the host sysctl is never lowered; the arg-filtered `personality` rule; honor trust and its judge-checked exclusion; the
dump/race frame filter (learner-editable files only); the go-race concurrency cap (if taken); the `module` schema and
its `contract_hash` inputs; the separate race `GOCACHE` seed. Status Accepted (it implements Accepted ADR-0030 §2).

### 7 · Tag `runner-v1.1.0` + verify dark on prod [X + H]

- The PR (tasks 1–6) merges first: branch `feat/p-01-go-race-profile`, CI green, squash-merge.
- Run the Release checklist below, then tag **`runner-v1.1.0`** on the merge commit. `runner-release.yml` builds it;
  `deploy.yml`'s `v*` filter does not match `runner-v*` (m3-15 tested this). The existing ImagePolicy
  `xlearn-runner` (`>=1.0.0 <2.0.0`) selects 1.1.0 and the 2nd IUA commits tag + digest to `runner/xlearn-runner.yaml`
  (**no infra PR**). The pod is `Recreate`d (≈ 70 s + start): judge sees `saturated` and re-queues, so tag at a quiet hour.
- **Verify by looking (D34)**, through mi-10's tunnel (the VAP denies exec):
  `ssh -L 18090:127.0.0.1:18090 vps 'k3s kubectl -n xlearn-runner port-forward deploy/xlearn-runner 18090:8090'`, then
  `GET /v1/profiles` shows `go-race@1.26` and the new digest, and `make runner-acceptance RUNNER_URL=http://127.0.0.1:18090 …`
  passes (go-race block included). `k3s kubectl get deploy -n xlearn-runner` shows 1.1.0; `flux get images policy
  xlearn-runner` latest = 1.1.0; the `runner` Kustomization is Ready.
- **If the canary fails after the Recreate** (`/v1/profiles` omits `go-race@1.26`, with the canary's ERROR in the logs)
  while CI's in-image canary passed, suspect the **pod-level seccomp profile**, not the image. The in-image run has no
  pod profile, so the likely cause is EPERM on `personality`. Re-run the entry gate's check (the host file's sha256
  against the BOM, and its `personality` rule against §16.2). Don't fix forward with a runner patch. Follow task 1's
  host-change path, and don't record `runner-v1.1.0` as "live dark with go-race" until a Recreate after the host change
  shows the profile.
- **Calibrate** the go-race baseline on prod (the A8 method of [t3 §7.3](../research/t3-sandbox.md#73-calibration): quiet
  hour, sar steal recorded, re-run if steal > 5 %) and publish it in the TL-baseline doc m3-15 started — p-03's pack sets
  per-test deadlines ≥ 10 × this baseline.
- `host-verify --cluster --expect-sandbox`: memory sum unchanged, no OOMKills, node disk < 70 % after the larger image
  pull, runner Ready. A DSA Submit through judge still grades (as a cohort account in an already-signed-in browser
  session; the agent never signs in, so otherwise record "owner login smoke pending" in status.md's pending-smoke notes).

### 8 · Record [X]

In [`../status.md`](../status.md): the **runner stream** row (`runner-v1.1.0` live dark, digest, date, go-race baseline,
steal during calibration); the Sprint board row; milestone **P** 🔄; if [ds-p-01](sprint-ds-p-01.md)'s session didn't
already record them (skip any edit already done): the **Artboards** rows AB14, AB15, AB02 (full) and AB05 (full) →
"frozen (PR #, date)", ds-p-01's _Overall_ ✅ and owner event **`ev-q5`** ✅ with its date; Decisions-log lines (ASLR
mechanism, the pod-level `personality` check result and any host change, go-race cap, `module` schema, frame filter, ADR
number); if the pod-level gate found a gap, task 7 ⛔ with the owner action (book a host window) and the infra branch
named. This sprint file's Status.

## Acceptance criteria

- [ ] Race fixture detected ≥ 19/20; deadlock 20/20
- [ ] runner-v1.1.0 deployed dark by image automation
- [ ] Leak fixture 20/20; a correct solution passes 20/20; 0 container OOMs (balloon, two concurrent go-race jobs)
- [ ] `GET /v1/profiles` on prod lists `go-race@1.26` + `gotest@1`; go/cpp/python within ± 5 % and `profile_sha256` unchanged
- [ ] judge maps `gotest@1` as honor-grade with `race`/`deadlock`/`leak` signals; no hidden test name, source, line, output or frame in any DTO (denylist test)
- [ ] A `gotest@1` item is `unsupported` (self path) while the runner lacks the profile (test)
- [ ] Item schema extended additively (freeze guard green); `contract_hash` golden vectors cover `module`; lints (Σ TL ≤ 40 CPU-s, `mem_mb` ≥ 2 × peak, #15) in packlint, judge start and content CI
- [ ] The go-race baseline is published; the ADR is merged

## Release

**`runner-v1.1.0`** — the runner stream ([ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)):
a minor adds a profile (no judge↔runner contract break), so the existing range selects it and the 2nd IUA deploys it.
The judge, `internal/course`, curriculum-schema and packlint changes merged here **ship dark in `v1.15.0`** ([p-03](sprint-p-03.md));
until then production judge (1.14.x) never sends a go-race job, and after it judge only sends one when the runner lists the profile.

Release checklist (ADR-0034 §6, copied; runner-stream reading below):

- [ ] Before the tag: peers' tags and PRs are checked (parallel sessions; `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents)
- [ ] Before the tag: it is the next free version, and its major equals `.release-line`
- [ ] Before the tag: ACL PRs for new streams and consumers are merged
- [ ] Before the tag: a new service's image comes before its policy
- [ ] Before the tag: for a contract: rehearsed in compose, floor marked
- [ ] Before the tag: for a contract, erase or GA tag: `host-verify --cluster` is green (ADR-0035), the host has settled, and the snapshot is taken
- [ ] Before the tag: from M6: no live interviews
- [ ] After the tag (by looking, D34): `/xlearn/api/v1/healthz` reports the version
- [ ] After the tag: `k3s kubectl get deploy -n xlearn` shows the new images
- [ ] After the tag: every `xlearn-*` ImagePolicy's latest equals the tag, and the HelmReleases are Ready
- [ ] After the tag: smoke-test login, the dashboard and coach
- [ ] Record milestone → tag → floor → snapshot and any flag changes in `docs/v2/status.md`
- [ ] (ADR-0035 §2 standing rule, not part of ADR-0034 §6) Every new in-cluster HTTP or NATS caller this tag introduces has its NetworkPolicy (ingress and egress) change in its own infra PR, merged before the tag

**Runner-stream reading** (how each line applies to `runner-v1.1.0`):

| Line | Reading |
|---|---|
| next free version / `.release-line` | next free `runner-v1.x.y` (`git ls-remote --tags origin 'refs/tags/runner-v*'`); the runner's own major stays **1** (range `<2.0.0`); `.release-line` governs `v*` app tags only |
| ACL PRs | none: no NATS stream or consumer changes |
| image before policy | n/a: the policy exists since mi-10; a minor needs no range change |
| contract / erase / GA | n/a (not a contract); `host-verify --cluster --expect-sandbox` is still run after the Recreate (task 7) |
| M6 live interviews | n/a (pre-M6) |
| healthz / `get deploy -n xlearn` | read instead: `GET /v1/profiles` via the tunnel and `k3s kubectl get deploy -n xlearn-runner`; the `xlearn` fleet is unchanged |
| ImagePolicy latest / Ready | `xlearn-runner` latest = 1.1.0; the `runner` Kustomization and HelmRelease Ready |
| smoke test | login, dashboard, coach, plus one DSA Submit graded through judge (in an already-signed-in browser session; otherwise "owner login smoke pending" in status.md) |
| record | the runner stream row (tag → digest → baseline); no floor or snapshot (not a contract) |
| NetworkPolicy rule | no new in-cluster caller (judge → runner already allowed) → no infra PR |
| (runner-stream addition) host pod profile | the entry gate's `personality` check passed, or the host change landed (task 1's runbook, in the owner-booked window) and `host-verify --cluster --expect-sandbox` shows the new `sandbox.seccomp` sha256 **before** the tag |

## Definition of Done

CI green (Go, sqlc diff unchanged, content CI, packlint, the jail-capable job) · PR squash-merged · `runner-v1.1.0`
tagged and live dark via image automation (no hand `kubectl`) · acceptance suite green on prod through the tunnel ·
go-race baseline published · ADR merged · statuses updated (this file + [`../status.md`](../status.md)) · `main` synced.

## Risks / watch-outs

- **If P3 failed for go-race**, go-concurrency is redesigned or SQL (the fallback) is chosen — stop at the entry gate.
- **The pod-level seccomp profile blocks the ASLR call.** RuntimeDefault's arg filter on `personality` rejects
  `ADDR_NO_RANDOMIZE` unless spk-02 added it to the file mi-09 shipped. CI's in-image run has no pod profile, so it
  can't catch this. The entry gate and task 7's "canary fails after Recreate" rule are the controls. The fix is a host
  change in an owner-booked window (task 1's runbook), never a runner patch.
- **Container OOM from two race builds.** 2 × 1.25 GiB plus the baseline is close to the 3 GiB limit; take the go-race
  concurrency cap rather than raising the pod's limits (a limit change is an infra PR and a memory-sum re-check).
- **Leaking hidden tests through dumps or race reports.** Goroutines of hidden `_test.go` functions appear in both; the
  frame filter is the control, and the denylist test must include a hidden-frame fixture.
- **Honor-grade integrity:** in-process tests can be subverted by learner code; that's accepted (t0 §6) and labelled —
  results are `trust=honor`, never in judge-checked %.
- **TSAN flakiness:** detection is probabilistic; ≥ 19/20 is the bar at the suite's `-count`, and packs choose `count`
  with the detection-rate gate in p-03's evalpack CI. A missed race in one run is not a false pass of the item (other runs).
- **Seed drift changing the other profiles' hashes** would force a recalibration of Go/C++/Python TLs: keep the race
  seed separate and verify `profile_sha256` unchanged.
- **Tag collision:** a `v*` filter matching `runner-v*` would push the runner through the app pipeline (m3-15 tested it;
  re-check the filters if `deploy.yml` changed since).
- **Recreate blip:** grading pauses ~70 s + start during the bump; judge re-queues `saturated` jobs. Tag at a quiet hour.
