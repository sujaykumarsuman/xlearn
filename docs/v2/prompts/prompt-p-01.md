# Prompt — Sprint p-01 · go-race runner profile → runner-v1.1.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-p-01.md`](../sprints/sprint-p-01.md)   ·   **Milestone:** P   ·   **Prereqs:** [m3-13](../sprints/sprint-m3-13.md) (v1.14.0), [spk-02](../sprints/sprint-spk-02.md) (P3), [ds-p-01](../sprints/sprint-ds-p-01.md) (Q5 + freeze)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] Only for a re-run after an earlier p-01 session found the pod-level seccomp gap (p-01 task 7 ⛔ in `status.md`): today is the host window you booked for the change (a short calendar event). A first run needs nothing from you.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, land-and-sync.
- The plan: [`../sprints/sprint-p-01.md`](../sprints/sprint-p-01.md) — tasks 1–8 and the Release section are the brief.
- Runner: [t3 §5.7](../research/t3-sandbox.md#57-throttled-and-the-quiet-re-run-learner-proof-bounded),
  [§5.8](../research/t3-sandbox.md#58-goroutine-dump-go-race), [§5.10](../research/t3-sandbox.md#510-cleanup-invariants-release-gate-a8),
  [§6.1](../research/t3-sandbox.md#61-image-and-release), [§6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them)
  (the go-race column), [§7.3](../research/t3-sandbox.md#73-calibration), [§10](../research/t3-sandbox.md#10-what-t3-constrains-downstream),
  and **§16.2** (spk-02's P3 amd64 replay: the TSAN verdict, the ASLR policy, the go-race allowlist) in
  [`../research/t3-sandbox.md`](../research/t3-sandbox.md).
- Contract and judge: [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) (go-concurrency row),
  [§5.2](../research/t4-judge-contract.md#52-grader-contract-go), [§5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start),
  [§5.7](../research/t4-judge-contract.md#57-cost-of-adding-capability-inferred), [§6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only).
- Content: [t1 §3.4](../research/t1-content-data-model.md#34-versioning-and-pinning), [§7.1](../research/t1-content-data-model.md#71-package-format)
  (go-concurrency row), [§7.2](../research/t1-content-data-model.md#72-ci-validation).
- ADRs: [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md) (Accepted at m3-03, with the spike table),
  [ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md), [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md),
  [ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams) and
  [§6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist),
  [ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses).
- The sprints you build on: [m3-03](../sprints/sprint-m3-03.md) (runner core, `internal/platform/runnerapi`),
  [m3-04](../sprints/sprint-m3-04.md) (profiles, allowlists, the jail-capable CI job), [m3-15](../sprints/sprint-m3-15.md)
  (reproducible image, `runner-release.yml`, `make runner-acceptance`, TL-baseline doc), [mi-10](../sprints/sprint-mi-10.md)
  (runner dark on prod, the ssh tunnel, calibration), [mi-09](../sprints/sprint-mi-09.md) (the host's pod-level seccomp
  profile as a heredoc in `../infra/hack/host-bootstrap.sh`, `hack/host-bom.txt`, `host-verify`), [m3-01](../sprints/sprint-m3-01.md) (`internal/course/canon`, packlint),
  [m3-06](../sprints/sprint-m3-06.md) (judge `code` grader), [m1-01](../sprints/sprint-m1-01.md) (item schema, freeze guard).
- Code: `cmd/runner`, `internal/runner/`, `internal/platform/runnerapi/`, `deploy/runner.Dockerfile`,
  `.github/workflows/runner-release.yml`, `internal/judge/`, `internal/course/`, `curriculum/_schema/`, `cmd/packlint`.

## Context

M3 shipped the judge and the runner with the `go`, `cpp` and `python` profiles (`runner-v1.0.0`, live dark since mi-10;
judge on for the owner/tester cohort since `v1.14.0`). Milestone **P** adds a second course, go-concurrency, as manifest +
content with widget/profile code only. Its code items are **multi-file Go modules graded by in-process tests under the
race detector**: that needs a new runner profile, **`go-race@1.26`**, and a new harness, **`gotest@1`** — this sprint.
The spike's P3 replay (t3 §16.2) says whether TSAN works under production's `vm.mmap_rnd_bits=32` and which **per-process**
ASLR policy it needs; the host sysctl is never lowered. Results are **honor-grade** (the tests run inside the learner's
program), so judge labels them `trust=honor` and judge-checked % excludes them. The runner releases on its own stream:
`runner-v1.1.0` is picked by the existing `xlearn-runner` ImagePolicy and deployed by the 2nd ImageUpdateAutomation, with
no infra PR. The judge and schema changes merge now and ship dark in `v1.15.0` (p-03). Nothing learner-visible changes:
the pilot's items arrive in [p-02](../sprints/sprint-p-02.md) and its pack in [p-03](../sprints/sprint-p-03.md).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] M3 shipped (v1.14.0) — `curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz` reports ≥ 1.14.0
- [ ] P3 TSAN result GO for go-race (spk-02) — t3 §16.2 on `main`
- [ ] PRD Q5 recorded (`ev-q5`, resolved in ds-p-01 before AB15 was drafted): PRD §7 Q5 reads resolved **go-concurrency**
  on `main`, i.e. ds-p-01's record PR (`docs/ds-p-01-q5`) is merged (ds-p-01 merges it on CI green, D40). If it isn't,
  ds-p-01 isn't done: stop and report.
- [ ] AB14–AB15 frozen — the ds-p-01 board PR is merged (the merge is the freeze)
- [ ] `runner-v1.0.0` live dark with `baseline@1` and the Go/C++/Python multipliers recorded; ADR-0030 Accepted
- [ ] **The pod-level seccomp profile permits the ASLR mechanism.** The production host file
  `/var/lib/kubelet/seccomp/profiles/xlearn-runner.json` (mi-09, the amd64 file recorded verbatim in t3 §16.2) must allow
  `personality` with `ADDR_NO_RANDOMIZE` (0x0040000), plus the self-`execve` if §16.2 needed it. RuntimeDefault admits
  only 0x0/0x8/0x20000/0x20008/0xffffffff. Check it read-only: compare the heredoc in `../infra/hack/host-bootstrap.sh`
  with §16.2, and confirm the host matches it (`host-verify --cluster --expect-sandbox`'s `sandbox.seccomp` sha256 = BOM,
  or `ssh vps 'sudo cat /var/lib/kubelet/seccomp/profiles/xlearn-runner.json'`)
- [ ] Parallel sessions: no open peer PR on `internal/runner/`, `deploy/runner.Dockerfile`, `runner-release.yml`, `ci.yml` (the `runner-it` / `runner-image-acceptance` jobs), `internal/judge/grader/`; no `runner-v*` tag in flight

If Q5 recorded **SQL**, stop: p-01 must be re-planned to a `sql-pg` profile first. If status.md marks p-01 ⛔ "needs
owner decision" (ds-p-01 found P3 inconclusive for go-race), or t3 §16.2 says go-race failed even with a per-process
`setarch -R` launcher, stop and report: Q5 goes back to the owner. If the **pod-level profile** lacks the `personality`
rule, do steps 1–9 (the code ships dark) and step 2's host-change prep, **skip steps 10–11** (no tag in this session),
do step 12 with task 7 ⛔ in status.md ("the tag waits for the go-race seccomp host change; owner to book a host
window"), and run **Ship**. Nothing waits. A re-run on the booked date (see **Before you launch (owner)**) finds the code
(steps 2–9) on `main`, runs step 2's runbook, and picks up at step 10. The tag waits until `host-verify` shows the new file.

## Do this (in order)

1. **[X] Branch** `feat/p-01-go-race-profile` from an up-to-date `main`.
2. **[X] go-race profile** (plan task 1): add `go-race@1.26` to the runner's profile registry — compile `go test -c -race
   -vet=off -trimpath` (15 s, 1.25 GiB); one slot, `cpu.max` 1 CPU, `GOMAXPROCS=2`; one process per declared test
   (`-test.run '^Name$' -test.count=N -test.timeout=…`); the `go` env + `GORACE="halt_on_error=1 atexit_sleep_ms=0
   exitcode=66"`, `GOTRACEBACK=all`; 1 GiB, pids 128, job ≤ 45 s, Run `-count=1` ≤ 20 s; the learner-code lint allowlist
   is t3 §6.2's go-race column: the `go` list + `sync sync/atomic time context` and a limited `runtime`. **Not**
   `testing/synctest`: that is for the visible and hidden test files, which aren't linted as learner code. In `deploy/runner.Dockerfile` (still reproducible):
   a **separate** race `GOCACHE` seed, `goleak` pinned in the read-only `GOMODCACHE`, and only the C toolchain pieces
   `cpp` didn't already install. Implement the ASLR policy t3 §16.2 requires (a static `personality(ADDR_NO_RANDOMIZE)`
   + `execve` launcher for go-race processes only, or the runtime's own re-exec) and the amd64 KILL-default exec
   allowlist from §16.2 with `personality` **arg-filtered**. Add the profile-health canary (a prebuilt race canary; on
   failure `GET /v1/profiles` omits `go-race@1.26`, the runner stays Ready). Report `vm.mmap_rnd_bits` and the go-race
   `baseline@v` in `/v1/profiles`. Size the slots so a go-race job plus the other slot's job fit 3 GiB − baseline −
   256 MiB; if two go-race jobs don't, cap go-race at one concurrent job (`503 saturated` for the second).
   **Pod-level profile (entry gate):** the image's exec filter only narrows the pod-level profile. If the host file lacks
   the `personality` rule:
   - **[I]** push an infra branch (e.g. `chore/host-seccomp-go-race`) with **no PR** (mi-09's window-branch pattern)
     that changes only that file's heredoc in `../infra/hack/host-bootstrap.sh`, plus its sha256 in `hack/host-bom.txt`
     and the `host-verify` constant. The commit message carries the §16.2 row and a JSON diff. Merged early, it would
     make every `host-verify --expect-sandbox` FAIL `sandbox.seccomp`, so the runbook merges it in the window.
   - **[X]** write `docs/v2/runbooks/host-seccomp-go-race.md`: open and merge the PR from that branch (its body carries
     the §16.2 row and the JSON diff), run `host-bootstrap.sh --with-sandbox` (no k3s restart; the file is read at pod
     create), then `host-verify --cluster --expect-sandbox`.
   - The session that finds the gap doesn't write to the host: it records the gap and the owner action (book a host
     window) in status.md. On a **re-run in the booked window** (the owner attests
     it under **Before you launch (owner)**), run the runbook yourself: its host steps are pre-approved by launching this
     prompt (D40).
3. **[X] gotest@1 harness** (plan task 2): module assembly (rendered `go.mod` + `go.sum`, editable + read-only files,
   `visible_test.go`, `HiddenFiles` in Submit only; clean listed paths or REJECTED); the generated `TestMain` (fd 4
   frames, `goleak.VerifyNone` per test → LEAK, missing pass = fail); Run = visible tests `-count=1` with the learner's
   own `test2json` output; Submit = visible once then hidden `-count=N`, `TestEvent.Output` dropped; SIGQUIT dump →
   DEADLOCK (blocked in a learner-package frame) or TLE; exit 66 + `WARNING: DATA RACE` → RACE; precedence RACE >
   DEADLOCK > TLE > LEAK > RE > WA; frames reduced to function + module-relative file:line **in learner-editable files
   only** (hidden-test frames collapsed to a count). Front-side A5 lint re-check (no `TestMain`, no `_test.go` among
   editable files, no `//go:linkname`, `//go:embed`, `import "C"`, `unsafe`, imports within the allowlist). Additive
   `runnerapi` fields; `gotest@1` in the profile's harness majors.
4. **[X] Item schema + contract_hash** (plan task 3): the additive `config.module{path, go, files[{path, role}], api[]}`
   for `harness: "gotest@1"` in `curriculum/_schema/item.schema.json` and `internal/course/item.go` (freeze guard green;
   `module` required for `gotest@1`); the `items/<id>/_starter/module/` and `_code/module/` layout in the embed
   allowlist and leak lints; `internal/course/canon` hashes `harness@v`, `module.path`, `module.go`, `api[]` (golden
   vectors); packlint checks `tests[]`, go-race params and that hidden tests use only `api[]`; the content-CI "reference
   passes visible tests in the runner image" gate for modules.
5. **[X] judge** (plan task 4): the `code@1` grader (in m3-06's closed-registry file in `internal/judge/grader`; no new
   subpackage or registry entry) builds `gotest@1` jobs and maps results (hidden passed/total, first
   failure class, learner-file locations); `TrustOf` → `honor`; signals `race`, `deadlock`, `leak`; `Present()`
   denylist tests with a hidden-frame fixture (no hidden test name, source, line, output or frame); the evaluable index
   marks `gotest@1` items `unsupported` while `GET /v1/profiles` lacks `go-race@1.26`; lints Σ TL ≤ 40 CPU-s and
   `mem_mb` ≥ 2 × reference peak in packlint, judge start and content CI; lint #15's compile-leak fixture for `gotest@1`.
   Add the synthetic `zz-fixture` `gotest@1` item (public half `internal/course/testdata/`, pack half
   `internal/judge/testdata/pack/`) with racy / deadlocking / leaking / wrong / correct solutions and a judge + runner
   integration test.
6. **[X] Acceptance + CI** (plan task 5): extend `make runner-acceptance` (race ≥ 19/20, deadlock 20/20, leak 20/20,
   correct 20/20, go-race balloon → MLE with 0 container OOMs, two concurrent go-race jobs → 0 container OOMs, cleanup
   invariants after 100 go-race jobs, the canary and `/v1/profiles`). In `.github/workflows/ci.yml`:
   - extend **`runner-it`** (m3-03's jail-capable `ubuntu-24.04` job) with the go-race integration block and the judge
     fixture test;
   - extend **`runner-image-acceptance`** (m3-15) so the in-image `make runner-acceptance` includes the go-race block.
   - In both, run `sysctl -w vm.mmap_rnd_bits=32` first (a global sysctl: set it on the VM before the container, or
     inside the `--privileged` container, and record which works).

   Check `go`/`cpp`/`python` speed within ± 5 % and `profile_sha256` unchanged, and that `runner-repro` still gives one
   digest for two builds.
7. **[X] ADR** (plan task 6): check `ls docs/adr`, open PRs and peers for the next free number (≥ 0036), then write
   "go-race runner profile and `gotest@1` harness (pilot)", Accepted, recording the ASLR mechanism, the arg-filtered
   `personality` rule, honor trust, the frame filter, the concurrency cap, the `module` schema + `contract_hash` inputs,
   and the separate race seed.
8. **[X] Verify locally:** `gofmt -l .` empty · `go vet ./...` · `go test -race ./...` · `sqlc diff` (unchanged) · web
   typecheck/test (unchanged) · the jail-capable job green · packlint self-tests · `docker compose up --build` still
   serves the DSA flow (the `gotest@1` fixture is test-only).
9. **[X] Ship the code** (**Ship** steps 1–2 below): conventional commits with the attribution lines (e.g.
   `feat(runner): go-race profile and gotest@1 harness`, `feat(judge): map gotest@1 results as honor-grade`,
   `docs(adr): …`); PR; CI green; squash-merge; `git checkout main && git pull`.
10. **[X] Tag `runner-v1.1.0`** on the merge commit after the Release checklist in the plan (runner-stream reading).
    Before tagging, the pod-profile gate must pass, either originally or after step 2's runbook (in the booked window)
    shows the new sha256 in `host-verify`. Watch `runner-release.yml` build and push; confirm `deploy.yml` did not run for this tag.
11. **[H] Verify dark on prod:** wait for the 2nd IUA commit on infra `main`, then through the tunnel
    (`ssh -L 18090:127.0.0.1:18090 vps 'k3s kubectl -n xlearn-runner port-forward deploy/xlearn-runner 18090:8090'`):
    `GET /v1/profiles` lists `go-race@1.26` + `gotest@1` + the new digest; `make runner-acceptance
    RUNNER_URL=http://127.0.0.1:18090 RUNNER_TOKEN_FILE=<(sops -d --extract '["stringData"]["token"]' ../infra/runner/secrets/runner-auth.enc.yaml)`
    passes. `k3s kubectl get deploy -n xlearn-runner` shows 1.1.0; `flux get images policy xlearn-runner` latest 1.1.0;
    `runner` Kustomization Ready; `host-verify --cluster --expect-sandbox` green (memory sum unchanged, no OOMKills,
    disk < 70 %); one DSA Submit still grades (as a cohort account in an already-signed-in browser session; you never
    sign in, so otherwise record "owner login smoke pending" in status.md's pending-smoke notes). **If `/v1/profiles`
    omits `go-race@1.26` after the Recreate** while CI's in-image canary passed, suspect the **pod-level seccomp profile** (EPERM on `personality`), not the
    image. Re-check the gate and follow step 2's host-change path; don't fix forward with a runner patch. **Calibrate**
    the go-race baseline at a quiet hour (sar steal recorded; re-run if > 5 %) and publish it in the TL-baseline doc (a
    docs PR, merge on green).
12. **[X] Record** (plan task 8) and update this sprint's Status, then ship: see **Ship** below.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** the runner never initiates a
  connection and holds no secrets or pack; judge is its only caller; comparison and presentation happen in judge.
  No cross-schema reads. No new service, stream, subject, consumer or in-cluster caller.
- **goose + sqlc:** no migration is expected; if one appears, it is expand-only, embedded, `sqlc diff` clean.
- **Outbox/inbox:** `evaluation_completed` keeps its existing outbox path; you only add signal values.
- **Private data:** hidden tests, their names and their frames never reach a learner DTO or a log line. Never copy
  content from `../xlearn-evalpack` into this repo (the fixture pack is synthetic).
- **Runner hardening is not negotiable:** the host `vm.mmap_rnd_bits` is never lowered; `personality` is arg-filtered
  at both layers (the image's exec filter and, if it must change, the pod-level host file, which gains exactly
  `ADDR_NO_RANDOMIZE` and nothing broader); never exec into the runner (the VAP denies it; delete the pod instead).
- **GitOps:** no `kubectl apply`; the runner deploys through the 2nd IUA; `ssh vps` is read-only except the port-forward
  and, on a re-run in a booked host window, step 2's runbook (pre-approved by launching this prompt, D40).
  **Never move or re-push a tag** — fix forward with `runner-v1.1.1`.
- **Memory-sum rule:** the runner's limits (2 CPU / 3 GiB) don't change; the slot budget or the go-race cap absorbs the
  new profile. Any limit change would be an infra PR plus a `host-verify --cluster` memory-sum check.
- **Consumers before producers / ACL before consumer:** not triggered (no NATS change). Judge only sends go-race jobs
  when the runner lists the profile, so the two deploy in either order.
- **No alerting (D34):** verification is by looking and `host-verify`; no timer, CronJob or push channel.
- **Parallel sessions:** check peers' PRs, tags, worktrees and ListAgents before tagging and before claiming the ADR number.
- **No `v*` tag** in this sprint: `v1.15.0` is p-03's.

## Deliverables

- `go-race@1.26` profile, race `GOCACHE` seed, `goleak` seed, the ASLR launcher (if needed), the go-race amd64 allowlist,
  the profile canary and the slot budget/cap, in `internal/runner/` and `deploy/runner.Dockerfile`.
- The `gotest@1` harness (module assembly, TestMain, dump/race parsing, frame filter, A5 lint) and additive `runnerapi` fields.
- The `module` item-schema block, `contract_hash` inputs, packlint and content-CI rules.
- judge `gotest@1` mapping, honor trust, signals, denylist tests, evaluable-vs-profiles check, lints, the `zz-fixture` item.
- The acceptance-suite go-race block and the CI job; the ADR; the `runner-v1.1.0` tag, live dark; the go-race baseline doc.

## Update status

- [`../sprints/sprint-p-01.md`](../sprints/sprint-p-01.md): task rows 🔄 → ✅ (⛔ with a reason), _Overall_.
- [`../status.md`](../status.md): the **runner stream** row (`runner-v1.1.0`, digest, date, go-race baseline, steal);
  the **Sprint board** row; **Milestones** P 🔄; if [ds-p-01](../sprints/sprint-ds-p-01.md)'s session didn't already
  record them (skip any edit already done): the **Artboards** rows AB14, AB15, AB02 (full), AB05 (full) →
  "frozen (PR #, date)", ds-p-01's _Overall_ ✅ (in its plan file too) and owner event **`ev-q5`** ✅ with its date;
  **Decisions log** lines (ASLR mechanism, the pod-level `personality` check and any host change, go-race cap, `module`
  schema, frame filter, ADR number); if the pod-level gate found a gap, task 7 ⛔ with the owner action (book a host
  window) and the infra branch named. No flag changes.
- The new ADR under `docs/adr/` (MADR-style, append-only).

## Done when (acceptance)

- [ ] Race fixture detected ≥ 19/20; deadlock 20/20
- [ ] runner-v1.1.0 deployed dark by image automation
- [ ] Leak 20/20; correct 20/20; 0 container OOMs (balloon, two concurrent go-race jobs)
- [ ] `GET /v1/profiles` on prod lists `go-race@1.26` + `gotest@1`; go/cpp/python within ± 5 %, `profile_sha256` unchanged
- [ ] judge maps `gotest@1` as honor-grade with `race`/`deadlock`/`leak`; the denylist test proves no hidden test name, source, line, output or frame leaks
- [ ] A `gotest@1` item is `unsupported` while the runner lacks the profile
- [ ] Schema extension additive (freeze guard green); `contract_hash` vectors cover `module`; the lints run in packlint, judge start and content CI
- [ ] go-race baseline published; ADR merged; status.md updated

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). (Branch `feat/p-01-go-race-profile`; the only infra PR is step 2's conditional seccomp change, opened and merged by its runbook in the booked window.)
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — `runner-v1.1.0` (runner stream):** Follow the stream's own tag procedure: the plan's Release checklist (runner-stream reading), then tag `runner-v1.1.0` on the merge commit; `runner-release.yml` builds it (confirm `deploy.yml` didn't run) and the 2nd IUA deploys it dark, with no infra PR (step 10). Verify on prod by looking through the tunnel and calibrate the go-race baseline (step 11); record it under **release streams** (digest, go-race baseline). No app tag: the judge and schema changes ship dark in `v1.15.0` ([p-03](../sprints/sprint-p-03.md)). If the pod-level gate found a gap, don't tag: task 7 stays ⛔ in status.md, and a re-run in the owner-booked window runs step 2's runbook, then this step.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn, and `../infra`, which the IUA commits to). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
