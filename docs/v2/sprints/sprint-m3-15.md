# Sprint m3-15 — Runner release: reproducible image, runner-release.yml, acceptance suite, TL baselines → runner-v1.0.0

> **Milestone:** M3 — judge plus code grader (runner track; the image half of rollout step **MI-12**) · **Track:** product · **Order:** 32
> **Prereqs:** [m3-04](sprint-m3-04.md) (profiles, harness codecs, allowlists), and through it [m3-03](sprint-m3-03.md) (runner core, ADR-0030 Accepted)
> **Unblocks:** [mi-10](sprint-mi-10.md) (its gate "`runner-v1.0.0` image exists"; it runs this sprint's acceptance suite on production). Also feeds [m3-02](sprint-m3-02.md)/[m3-07](sprint-m3-07.md)/[m3-13](sprint-m3-13.md) (the TL gate uses the runner image and the TL-baselines doc) and [m3-06](sprint-m3-06.md) (the compose `runner` service for its e2e)
> **Release action:** **tag `runner-v1.0.0`** on the runner stream ([ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)), preceded by a rehearsal prerelease `runner-v1.0.0-rc.1`. The tag builds and pushes `ghcr.io/sujaykumarsuman/xlearn-runner:1.0.0` and **deploys nothing**: no runner ImagePolicy exists until [mi-10](sprint-mi-10.md) (image before policy). No infra PR.
> **Calendar:** late October – early November: after m3-04, before mi-10 (which follows the Sat 2026-10-24 host window)
> **Execute with:** [`../prompts/prompt-m3-15.md`](../prompts/prompt-m3-15.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Reproducible `deploy/runner.Dockerfile` | X | ⬜ |
| 2 | `.github/workflows/runner-release.yml` + runner major-line guard + no `v*` collision + git-strategy runner section | X | ⬜ |
| 3 | CI: reproducibility check and in-image acceptance lane | X | ⬜ |
| 4 | Acceptance suite `make runner-acceptance` (+ calibration mode) | X | ⬜ |
| 5 | Compose `runner` service (dev mode) | X | ⬜ |
| 6 | Full-shape rehearsal on a throwaway arm64 VM (k3s + host block + guards) | H | ⬜ |
| 7 | TL-baselines doc `docs/architecture/runner-tl-baselines.md` | X | ⬜ |
| 8 | Rehearsal prerelease `runner-v1.0.0-rc.1`; the image is anonymously pullable | X + O | ⬜ |
| 9 | Tag `runner-v1.0.0` (release checklist) and record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the M3 milestone row, the MI-12 row, and the **runner stream** tag row).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m3-04](sprint-m3-04.md) merged: the three profiles, the extended `internal/platform/harness`, `runnerapi/lint`, the `runner seed-gocache` subcommand and the amd64 allowlists (arm64 dev lists KILL-default) are on `main`, and `runner-it` is green with Go, C++ and Python.
- [ ] ADR-0030 is **Accepted** ([m3-03](sprint-m3-03.md) task 1), so the image implements the mechanism the spike chose (go-sandbox `forkexec`, or the pinned `nsjail` for R1-N).
- [ ] **No runner tag exists yet**: `git ls-remote --tags origin 'refs/tags/runner-v*'` is empty, and no peer PR touches `deploy/runner*`, `.github/workflows/`, `docker-compose.yml` or `Makefile` (`gh pr list`, `git worktree list`, ListAgents).
- [ ] MI-2a live ✅: `.release-line` = `1` and `deploy.yml`'s guard refuses non-`vX.Y.Z` tags (xlearn#53).
- [ ] *(For task 6)* local `../infra` `main` is synced and holds [mi-09](sprint-mi-09.md)'s sandbox block in `hack/host-bootstrap.sh` and [mi-14](sprint-mi-14.md)'s `infrastructure/sandbox/` manifests. If either is still a PR, use its branch (or t3 §16.1 and the t3 §8.2 draft) and note it; the rehearsal is local and read-only toward `../infra`.
- [ ] *(For task 8)* the owner is reachable to flip the GHCR package to **Public** if the anonymous pull fails (an account setting the agent doesn't change).

This sprint is not gated by the [M3 hard entry checklist](../rollout-plan.md#5-m3-hard-entry-checklist) (that gates [m3-11](sprint-m3-11.md)); it supplies MI-12's image, which the checklist reads through [mi-10](sprint-mi-10.md).

## Goal

Release the runner **on its own stream** with a **reproducible image**, and ship the **acceptance suite** that production runs dark in [mi-10](sprint-mi-10.md):
- a single-layer, digest-pinned, setuid-free `xlearn-runner` image whose digest is identical across two clean builds;
- a self-contained `runner-release.yml` driven only by `runner-v*` tags, proven not to collide with the fleet's `v*` pipeline;
- `make runner-acceptance`: network and syscall probes, the P2 corpus, cross-job markers, cleanup invariants, references through the image, canary timing and a calibration mode;
- published per-profile TL baselines (CI column now, production column by mi-10) for the pack TL gate;
- `runner-v1.0.0` tagged, anonymously pullable, **not deployed**.

## Scope

**In**
- `deploy/runner.Dockerfile` (pins declared as `ARG *_VERSION` / `*_SHA256` / `DEBIAN_SNAPSHOT` / the base digest, for [mi-11](sprint-mi-11.md)'s Renovate regex manager).
- `.github/workflows/runner-release.yml`, `deploy/runner.release-line`, a workflow-trigger test, the runner-stream subsection of `docs/git-strategy.md`.
- CI jobs `runner-repro` and `runner-image-acceptance` (path-filtered), on one pinned BuildKit image shared with the release workflow.
- `internal/runner/acceptance/` + `make runner-acceptance` (`SUBSET=full|prod` required; + `CALIBRATE=1`, `REQUIRE_PROD=1`), rotation-aware.
- The compose `runner` service (`RUNNER_MODE=dev`, compose profile `runner`).
- A full-shape rehearsal on a throwaway arm64 multipass VM.
- `docs/architecture/runner-tl-baselines.md`.
- Tags `runner-v1.0.0-rc.1` (rehearsal) and `runner-v1.0.0`.

**Out**
- Deploying the runner (the `runner` Kustomization, HelmRelease, bearer secret, ImageRepository/ImagePolicy, the 2nd IUA) and the production acceptance run + calibration → [mi-10](sprint-mi-10.md).
- Host files and the host window → [mi-09](sprint-mi-09.md). Guard objects and the VAP → [mi-14](sprint-mi-14.md).
- `go-race` / `gotest@1` → `runner-v1.1.0`, [p-01](sprint-p-01.md).
- judge's use of the runner and the compose e2e through judge → [m3-05](sprint-m3-05.md)/[m3-06](sprint-m3-06.md). The evalpack TL gate itself → [m3-02](sprint-m3-02.md) (E), re-gated in [m3-13](sprint-m3-13.md).
- Renovate config → [mi-11](sprint-mi-11.md) (it reads this sprint's `ARG` pins and the BuildKit image pin).

## Tasks

### 1 · Reproducible `deploy/runner.Dockerfile` [X]

Sources: [t3 §6.1](../research/t3-sandbox.md#61-image-and-release), [ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams) ("reproducible build; digest output"), [rollout §12](../rollout-plan.md#12-downstream-constraints-for-the-build-plan-session) ("`runner-release.yml` uses `type=match` and builds reproducibly").

- **Pins at the top** (the only place versions live): `ARG BASE=debian:trixie-slim@sha256:…` (the **multi-arch index** digest, so task 5's compose service and task 6's arm64 VM build the same file), `ARG DEBIAN_SNAPSHOT=<YYYYMMDDTHHMMSSZ>` (apt from `snapshot.debian.org`, so package versions are fixed by date), `ARG GO_VERSION=1.26.8`, `ARG GO_SHA256_AMD64=…` and `ARG GO_SHA256_ARM64=…` (the tarball checksum is picked by `TARGETARCH`; the release builds amd64 only), and, for R1-N only, the nsjail package or source pin.
- **Stages:**
  - `build` — `golang:<GO_VERSION>@sha256:…`: `CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -trimpath -buildvcs=false -ldflags="-s -w -X main.version=${VERSION}" -o /out/runner ./cmd/runner` (this exact line shape keeps `deploy/version_test.go` covering the image).
  - `rootfs` — `FROM ${BASE}`: apt from the snapshot, `--no-install-recommends` `g++` and `python3` (+ `nsjail` for R1-N); the Go tarball checked with `sha256sum -c` into `/opt/xl/go`, pruned (no `test/`, `api/`, `doc/`, `misc/`); the runner binary, profile files, harness templates and preludes; the Go cache seed from **`runner seed-gocache -out <seed dir>`** (the subcommand [m3-04](sprint-m3-04.md) added; the build fails on its non-zero exit, and the printed tree hash goes into the build log — if the subcommand is somehow missing, add it to `cmd/runner/main.go` here with m3-04's flags and exit rules rather than scripting the recipe in the Dockerfile, so the seed stays covered by `ProfileSHA`); Python stdlib bytecode rebuilt with `compileall --invalidation-mode checked-hash`; empty mount points the spawner uses under a read-only root (e.g. `/run/xl/…`); then **strip every setuid/setgid bit**, delete apt lists, caches, logs, docs and man pages, and **normalize every mtime** to `SOURCE_DATE_EPOCH` with `touch -h -d @…` — then set the Go cache seed to its fixed *future* mtime (t3 §6.1).
  - final — `FROM scratch`, `COPY --from=rootfs / /` (**one layer**), `ENTRYPOINT ["/usr/local/bin/runner"]`, `EXPOSE 8090`, labels `org.opencontainers.image.source`, `.version`, `.revision` (from build args, never a wall-clock `created`).
- **Do not use buildkit's `rewrite-timestamp`:** it would clamp the cache seed's future mtime. Explicit normalization gives the same determinism.
- **The C++ prelude PCH:** build it in the image only if it is byte-identical across two builds (task 3 decides); otherwise build it at container start into a `runner/`-charged read-only location, or drop it (record the compile-time cost). Record the choice in the decisions log.
- **No secrets, no pack, no evalpack path** in the build context (`.dockerignore` already drops `.git`, `docs/`, `*.md`; add any evalpack path m3-01 listed). Assert with a grep over the final rootfs in CI.
- Record the compressed size (t3 estimated 200–250 MB for Go only; D20 adds g++ and Python).

### 2 · `runner-release.yml` + runner major-line guard + no `v*` collision [X]

Sources: [ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams), [§1.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#13-accidental-major-guard-shipped-2026-09-24-xlearn53-infra29) (the fleet guard it mirrors), [`../../git-strategy.md`](../../git-strategy.md).

- `on: push: tags: ["runner-v*"]` only; `permissions: contents: read` at the top.
- **Job `guard`** (every other job `needs:` it): the tag must match `^runner-v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$`; a **stable** tag's major must equal the one-line `deploy/runner.release-line` (`1`); a prerelease builds with a notice (no ImagePolicy selects it). A runner major is a judge↔runner contract break and moves this file in a reviewed PR, then follows "GA in miniature" (ADR-0034 §1.5). Reuse the fleet guard's script if it has been factored out by then.
- **Job `image`** (`packages: write`): pinned `docker/setup-buildx-action` on task 3's pinned BuildKit image (`driver-opts: image=moby/buildkit:<version>@sha256:…`), `docker/login-action` (GITHUB_TOKEN), `docker/metadata-action` with `images: ghcr.io/sujaykumarsuman/xlearn-runner` and `tags: type=match,pattern=runner-v(\d+\.\d+\.\d+.*),group=1` (so `runner-v1.0.0` → `1.0.0`, which mi-10's `filterTags ^\d+\.\d+\.\d+$` selects), `flavor: latest=false`; `docker/build-push-action` with `file: deploy/runner.Dockerfile`, `platforms: linux/amd64`, `build-args: VERSION=${{ github.ref_name }}, SOURCE_DATE_EPOCH=<commit time>, REVISION=${{ github.sha }}`, **`provenance: false`, `sbom: false`** (attestations would wrap the image in an index with non-deterministic content), and `outputs: type=image,push=true,oci-mediatypes=true,compression=gzip,compression-level=9,force-compression=true` (task 3's settings, so a local rebuild can match); write the digest to the job summary.
- **Job `release`**: `gh release create` for stable tags, title `runner-v1.0.0 — v2 build · M3 runner (dark)`, notes with the digest, the profile ids with their `profile_sha256`, and the TL-baselines link.
- **Docs:** add a **runner stream** subsection to [`docs/git-strategy.md`](../../git-strategy.md) (rollout §12 asks it to cover the runner and evalpack streams): `runner-v*` tags and the image they build, the guard file `deploy/runner.release-line` (not named in ADR-0034 §1.5; this is its record), the `-rc` rehearsal (task 8), "never move or re-push a tag", and how to go to a runner major (a contract break: move `deploy/runner.release-line` in a reviewed PR, then GA in miniature per ADR-0034 §1.5). If the build-plan session's rewrite already has a runner section, extend it; don't duplicate it.
- **No collision, both directions:** `deploy/workflows_test.go` parses `.github/workflows/*.yml` and asserts, with GitHub's glob semantics, that `deploy.yml`'s tag patterns never match `runner-v1.0.0` / `runner-v1.0.0-rc.1` and `runner-release.yml`'s never match `v1.13.0` / `v2.0.0-rc.1`. The live proof is task 8 (no `deploy.yml` run for the rc ref). `deploy.yml`'s own `release-line` regex (`^v…`) would refuse a runner tag anyway.

### 3 · CI: reproducibility check and in-image acceptance lane [X]

- **One pinned builder everywhere.** BuildKit versions and compression settings change layer bytes even when the content is identical, so `runner-repro`, `runner-release.yml` and any local rebuild use the **same BuildKit image**: `docker/setup-buildx-action` (pinned by SHA) with `driver-opts: image=moby/buildkit:<version>@sha256:…`, and every output sets `oci-mediatypes=true,compression=gzip,compression-level=9,force-compression=true` (so a pushed `type=image` manifest and a local `type=oci` one are comparable). Locally: `docker buildx create --name xl-repro --driver docker-container --driver-opt image=<the same ref>`. The BuildKit pin is another `ARG`-style pin for [mi-11](sprint-mi-11.md)'s Renovate.
- **`runner-repro`** (in `ci.yml`, path-filtered to `deploy/runner*`, `cmd/runner/**`, `internal/runner/**`, `internal/platform/runnerapi/**`, `curriculum/**`): build twice on the pinned builder with `--no-cache`, the same `SOURCE_DATE_EPOCH`, `provenance=false`, to `type=oci` outputs; compare the manifest digests. On a mismatch, run a pinned `diffoci` to name the differing files, and fail.
- **`runner-image-acceptance`** (same filter): build the image, `docker run -d --restart=always --privileged --cgroupns=private --memory=3g --cpus=2 -e RUNNER_MODE=dev` with a CI-only token file (**`--restart=always`**: the runner exits on purpose after a SIGSYS, task 4's rotation rule), then `make runner-acceptance SUBSET=full` (every section except the prod-only ones) and `CALIBRATE=1` once. This is the **authoritative in-image run** of m3-04's references and of m3-04's public content check ("the reference passes the samples in the public production runner image", t1 §7.2), and the source of the TL doc's CI column (task 7). Upload the JSON report as an artifact (`retention-days: 7`).
- The privileged container is CI-only; production never runs privileged (the VAP forbids it).

### 4 · Acceptance suite `make runner-acceptance` (+ calibration mode) [X]

Sources: [t3 §5.10](../research/t3-sandbox.md#510-cleanup-invariants-release-gate-a8), [§5.3](../research/t3-sandbox.md#53-api-judge-is-the-only-caller) (`/readyz` canaries), [§7.3](../research/t3-sandbox.md#73-calibration), [t3 §8.8](../research/t3-sandbox.md#88-rollout-order-that-keeps-v1-up) A8, and what [mi-10](sprint-mi-10.md) task 5 expects.

A **black-box** HTTP client (`internal/runner/acceptance/`, `//go:build runner_acceptance`): it submits jobs straight to the runner (bypassing judge's lint on purpose: the suite tests the jail, not the lint) and reads `GET /v1/stats` and `GET /v1/profiles`.
`make runner-acceptance RUNNER_URL=… RUNNER_TOKEN_FILE=… SUBSET=full|prod [REQUIRE_PROD=1] [CALIBRATE=1]` → `go test -tags runner_acceptance -count=1 ./internal/runner/acceptance/ -args …`; a JSON report in `bin/` (never committed) and a Markdown summary on stdout for status.md. **`SUBSET` has no default**: the Makefile and the test refuse to start without it, so nobody runs the wrong set on production by omission.

| Section | Checks | Pass |
|---|---|---|
| A · contract | 401 without token; 400 bad profile; 503 on a third concurrent job; client disconnect | typed errors; `pids.current` 0 afterwards |
| B · network | per language (one job each): TCP `1.1.1.1:443`, apiserver VIP `10.43.0.1:443`, kube-dns `10.43.0.10:53` (UDP and TCP), a DNS lookup, binding loopback | every attempt fails (empty netns, loopback down, sockets denied → `signal(SIGSYS)`); one rotation per job |
| C · syscalls | `io_uring_setup`, `bpf`, `perf_event_open`, `userfaultfd`, `keyctl`/`add_key`, `ptrace`, `process_vm_readv`, `mount`, `unshare(CLONE_NEWUSER)`, `setns`, `clone(CLONE_NEWNS)`, `socket(AF_NETLINK)` | `signal(SIGSYS)` each, as cases of one job per language; one rotation per job (new `BootEpoch`); `/readyz` still reports `core_pattern` not a pipe |
| D · P2 corpus | 1 GiB balloon **× 100**, fork bomb, thread bomb, tmpfs fill, inode fill, stdout flood, orphan double-fork, spin, sleep | **MLE 100/100**; container `oom_kill` delta **0**; OLE, TLE, TLE (idle), RE as specified; **0 survivors** |
| E · cross-job markers | job N writes `/w`, `/tmp`, `/dev/shm`, SysV shm, POSIX mq, an abstract socket, a keyring; job N+1 looks | nothing visible (or the call was denied) |
| F · cleanup | ≥ 1,000 case cgroups, then `/v1/stats` | slots at baseline ±5 MiB, `pids.current` 0, `nr_dying_descendants` → ~0 within 60 s, `runner/` at baseline ±5 MiB |
| G · references | m3-04's items × Go/C++/Python, references and wrong solutions; m3-04 task 8's public content check (starters compile, public `_code` references pass their samples) | pass / classified as in m3-04 — **inside the image's toolchains** (t1 §7.2's "in the public production runner image") |
| H · canary | `canary_median` and its CV from `/v1/profiles` over the run | reported (thresholds are mi-10's, from S0) |
| I · prod-only (`REQUIRE_PROD=1`) | `/v1/profiles` `mode=prod`; `/readyz` green: AppArmor label `xlearn-runner`, non-identity UID map, userns/`fsopen`/SCTP denied | all green; `mode=dev` fails the run |
| Calibration (`CALIBRATE=1`) | 5 kernels (integer loop, sort 1e6, map-heavy, alloc/GC-heavy, BFS on 2e5 edges) × 30 runs × Go/C++/Python, one slot, sequential | per-kernel median CPU ms + CV, steal during the window (`Telemetry.StealPct`), `baseline@1` keyed to each `profile_sha256`, kernel-derived language ratios |

**`SUBSET` values:**

| `SUBSET` | Sections | Counts | Used by |
|---|---|---|---|
| `full` | A–H, plus I with `REQUIRE_PROD=1` | D: balloon × 100 + every other corpus program once; F: ≥ 1,000 case cgroups; G: every m3-04 item × language, plus m3-04's public content check | CI in-image lane (task 3), VM rehearsal (task 6) |
| `prod` | A–F and H, plus I with `REQUIRE_PROD=1`; **no G** (the image is digest-identical to the CI in-image run, so G would only spend CPU on the shared node) | **not reduced**: t3 §5.10 makes 100 hostile jobs and 1,000 case cgroups the A8 release gate, and each balloon is capped by its case cgroup, not by the node. D: balloon × 100 + every other program once; F: ≥ 1,000 case cgroups | [mi-10](sprint-mi-10.md) on production; task 8's rc smoke |

**Rotation.** [m3-03](sprint-m3-03.md) has the runner drain and exit after **any** SIGSYS so the container restarts with a new `BootEpoch`. The suite plans for it:
- **Batch SIGSYS jobs.** Every case expected (or allowed) to end in SIGSYS — C's syscall probes, B's network probes (the exec filters admit no socket family), E's denied markers (keyring, abstract socket), the fork/thread bombs, G's runtime-SIGSYS wrong solutions — runs as cases of **one job per language per section**, in a correctness group so every case runs (a draining runner finishes its in-flight job). One rotation per such job; schedule them last within their section.
- **E needs one epoch.** Allowed-channel markers (`/w`, `/tmp`, `/dev/shm`, SysV shm, POSIX mq if allowlisted) are written by job N and looked for by job N+1 **under the same `BootEpoch`** (asserted), or the isolation check proves nothing.
- **Wait across each rotation:** after a Result with a `signal(SIGSYS)` case, poll `GET /readyz` and `GET /v1/profiles` until 200 with a **new `boot_epoch`**; timeout 6 min per rotation (kubelet's CrashLoopBackOff grows to 5 min after a few quick restarts; Docker's `--restart=always` back-off is shorter). Stats baselines (`oom_kill`, slot memory) restart per epoch.
- **Count them:** the suite computes the expected rotation count from its plan and reports expected vs observed. An epoch change no SIGSYS explains is a **failure** (treat it as a possible container OOM); on the VM and on prod the pod's `lastState.terminated.reason` must never be `OOMKilled`.

- **Stop conditions built in:** abort at once if the container `oom_kill` delta goes above 0, an unexplained epoch change happens, or `/v1/stats` stops answering outside an expected rotation (or past its timeout); print what ran.
- **The exact production command** (for [mi-10](sprint-mi-10.md) task 5; the port-forward can drop when the container restarts, so it runs in a retry loop):
  ```
  while :; do ssh -L 18090:127.0.0.1:18090 vps 'k3s kubectl -n xlearn-runner port-forward deploy/xlearn-runner 18090:8090'; sleep 2; done &
  make runner-acceptance SUBSET=prod REQUIRE_PROD=1 CALIBRATE=1 RUNNER_URL=http://127.0.0.1:18090 \
    RUNNER_TOKEN_FILE=<(sops -d --extract '["stringData"]["token"]' ../infra/runner/secrets/runner-auth.enc.yaml)
  ```
  Expect the pod's `restartCount` to rise by the suite's expected rotation count (not a failure) and the run to take longer than the CI lane because of kubelet back-off.
- Kernel and probe sources live in **`internal/runner/acceptance/testdata/{kernels,probes}/{go,cpp,python}/`**, loaded with `embed` (or `os.ReadFile`) and written against `func-json@1` so they take the normal profile path. `testdata` keeps these learner-style files (`package main` without `func main`, several per directory) out of `go build ./...`, `go vet ./...` and CI's `go test -race ./...`.

### 5 · Compose `runner` service (dev mode) [X]

- `docker-compose.yml`: a `runner` service built from `deploy/runner.Dockerfile`, `privileged: true`, `cgroup: private`, `mem_limit: 3g`, `cpus: 2`, **`restart: unless-stopped`** (the runner exits on purpose after a SIGSYS and after `RUNNER_MAX_JOBS`/`RUNNER_MAX_AGE`), `RUNNER_MODE=dev`, `RUNNER_TOKEN_FILE` → a committed dev-only token file (`deploy/local/runner-token.dev`, an obviously non-secret value), port 8090 internal only, compose profile **`runner`** (a plain `docker compose up` stays unchanged). [m3-05](sprint-m3-05.md)/[m3-06](sprint-m3-06.md) wire judge to it and give judge the **same dev token value** (its compose `RUNNER_TOKEN`, or the same file).
- `deploy/local/README.md`: how to start it, why it's privileged (compose only), and that Apple Silicon builds use the arm64 dev allowlist delta and run in `dev` mode (Docker Desktop's kernel lacks Ubuntu's AppArmor restriction).

### 6 · Full-shape rehearsal on a throwaway arm64 VM [H]

The CI lanes run the runner in a privileged container, which exercises the runner's own layers but not the **pod shape**. Rehearse that locally before mi-10 does it on production. Sources: [t3 §8.1–§8.3](../research/t3-sandbox.md#82-guard-objects-infrastructuresandbox), [t3 §8.7](../research/t3-sandbox.md#87-host-level-changes-manual-scripted-recorded), mi-10's value table.

1. `multipass launch 24.04 --name xl-runner-rehearsal --cpus 4 --memory 8G --disk 30G`; `apt full-upgrade` and reboot (mirror H0); install k3s with `host-bootstrap.sh --upgrade --with-k3s`, i.e. **whatever `PIN_K3S_VERSION` `../infra/hack/host-bootstrap.sh` pins on `main`** (v1.36.4+k3s1 today; v1.36.5 if [mi-09](sprint-mi-09.md) task 8 bumped it in the Oct 24 window). Never hardcode the version here.
2. Run **only the sandbox block and L23**: `host-bootstrap.sh --with-sandbox` inside the VM (containerd drop-in, AppArmor `xlearn-runner`, the pod seccomp profile, subuid range, sysctls, kubelet args). That writes the **amd64** pod seccomp file, so then **replace `/var/lib/kubelet/seccomp/profiles/xlearn-runner.json` with the arm64 pod profile from t3 §16.1** (same path, `0644 root`). `systemctl restart k3s`; run `host-verify --expect-sandbox` inside the VM: expect **exactly one FAIL, `sandbox.seccomp`** (its sha256 ≠ the BOM because it's the arm64 file); every other check green. Record that.
3. Apply `../infra/infrastructure/sandbox/` (mi-14) **inside the VM only**; build the image for arm64 locally, import it with `k3s ctr images import`, reference it by digest with `imagePullPolicy: Never`. If the VAP's image rule refuses the imported image, use a VM-only VAP copy with **only** the image rule widened (spk-01's precedent) and record it.
4. Render the pod with `helm template ../infra/charts/project` and mi-10's values (its task 4 table), with a VM-only `runner-auth` token and `RUNNER_MODE=prod`. The pod must be admitted and **Ready** (every `/readyz` canary green).
5. Port-forward (in a retry loop, as in task 4) and run `make runner-acceptance … SUBSET=full REQUIRE_PROD=1`. The image's arm64 exec filters are **KILL-default** ([m3-04](sprint-m3-04.md) task 5), so section C must pass unchanged here; a probe that doesn't SIGSYS on arm64 is a failure to fix in m3-04's arm64 list, not a known difference. Record the report and the observed rotation count. **No timing conclusions** (arm64 on HVF).
6. `multipass delete --purge xl-runner-rehearsal`. Nothing is committed except the results summary in the PR and status.md. No production credential ever enters the VM.

### 7 · TL-baselines doc `docs/architecture/runner-tl-baselines.md` [X]

Sources: [ADR-0030 §2–§3](../../adr/0030-runner-technology-and-host-hardening.md#3-timing-on-a-noisy-4-vcpu-vm), [t3 §7.2–§7.3](../research/t3-sandbox.md#73-calibration).

- The rule: pack TL = max(3 × the Go reference's CPU time on the **production** profile, 1 s); a language's TL = that × its `tl_multiplier`; CI references are scaled by `speed_index(prod) / speed_index(CI)`; TLs only scale up; a patch bump re-runs the speed index (±5% → TLs carry forward, else scale up and flag).
- One table per profile (`go@1.26`, `cpp@g++14`, `python@3.13`): `profile_sha256`, toolchain versions, the `runner-v1.0.0` digest, the memory baseline, the provisional `tl_multiplier` (m3-04), the **CI column** (task 3's calibration: kernel medians, CV, `speed_index(CI)`), and a **production column marked "pending mi-10"** (`baseline@1`, `CanaryMedian`, calibrated multipliers, steal during the run).
- A line for pack authors: **until the production column exists, every TL computed from it is provisional** ([m3-02](sprint-m3-02.md)'s flag), cleared by [m3-13](sprint-m3-13.md)'s re-gate.
- Link it from `docs/architecture/runner.md`; [mi-10](sprint-mi-10.md) fills the production column in its docs PR.

### 8 · Rehearsal prerelease `runner-v1.0.0-rc.1`; the image is anonymously pullable [X + O]

After the PR (tasks 1–7) merges and CI is green:
- `git tag -a runner-v1.0.0-rc.1 -m "runner-v1.0.0-rc.1 — rehearsal"` on `main`, push the tag.
- Verify: `runner-release.yml` built `1.0.0-rc.1` and printed a digest; **`deploy.yml` did not run** for that ref (`gh run list --workflow deploy.yml` shows nothing for it); the guard printed the prerelease notice.
- **Anonymous pull** (the VAP forbids `imagePullSecrets`, so the image must be public): `docker logout ghcr.io && docker pull ghcr.io/sujaykumarsuman/xlearn-runner:1.0.0-rc.1`, or an anonymous `crane manifest`. **If it's refused (a new GHCR package defaults to private), the owner switches `xlearn-runner` to Public** in the package settings and confirms it's linked to `sujaykumarsuman/xlearn` (task **O**); re-check.
- Rebuild the rc commit locally for `linux/amd64` on the **same pinned BuildKit image** (task 3's `xl-repro` builder) with the same `SOURCE_DATE_EPOCH`, build args and output settings, and compare the manifest digest with the pushed one (end-to-end reproducibility). A mismatch → `diffoci`, fix the cause.
- `docker run --restart=always` the rc image in `dev` mode and run `make runner-acceptance SUBSET=prod` against it as a smoke.
- The rc tag stays (tags are never moved or re-pushed); no ImagePolicy will ever select it.

### 9 · Tag `runner-v1.0.0` (release checklist) and record [X]

- Run the release checklist below, then `git tag -a runner-v1.0.0 -m "runner-v1.0.0 — v2 build · M3 runner (dark)"` on the same commit as the rc (or a later `main` commit whose runner paths are unchanged; if they changed, cut `-rc.2` first).
- Verify the release workflow, the digest, the anonymous pull and the GitHub release.
- **Record** in `docs/v2/status.md`: the **runner stream** row `runner-v1.0.0 → sha256:… → not deployed (mi-10 deploys dark)`; the TL-baselines doc link; hand-offs to [mi-10](sprint-mi-10.md) (the digest; `RUNNER_IMAGE_DIGEST`; no `emptyDir` needed, or which; **task 4's exact production command** with `SUBSET=prod REQUIRE_PROD=1 CALIBRATE=1` and the port-forward retry loop; the expected rotation / `restartCount` rise for a `prod` run), [m3-02](sprint-m3-02.md) (run the TL gate in this image at this digest, scaled by the CI column, flagged provisional), [m3-07](sprint-m3-07.md) and [m3-13](sprint-m3-13.md).

## Acceptance criteria

- [ ] **Reproducible image digest across two builds** (CI `runner-repro`), and the pushed rc digest equals a local rebuild on the same pinned BuildKit image and output settings.
- [ ] **Acceptance suite green against a local jail-capable VM** (task 6, prod mode, full pod shape) and in the CI in-image lane (task 3, dev mode).
- [ ] `deploy.yml` never runs for a `runner-v*` tag and `runner-release.yml` never for a `v*` tag (test + the rc observation).
- [ ] **`runner-v1.0.0` image exists** at `ghcr.io/sujaykumarsuman/xlearn-runner:1.0.0`, anonymously pullable, digest recorded; **nothing deployed** (no fleet image or ImagePolicy moved).
- [ ] The TL-baselines doc is published with the CI column and a "pending mi-10" production column.
- [ ] The compose `runner` service starts in dev mode, answers `/healthz` and `/v1/profiles`, and comes back (new `boot_epoch`) after a SIGSYS rotation.
- [ ] The suite waits across every SIGSYS rotation (expected = observed rotation count; no unexplained epoch change); `SUBSET` is required and `prod` is defined; `docs/git-strategy.md` has the runner-stream subsection.
- [ ] The image holds no setuid/setgid file, no secret and no pack content (CI grep).

## Release

**Tag `runner-v1.0.0`** on the runner stream ([ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams); [rollout §7](../rollout-plan.md#7-indicative-tag-timeline): "runner-v1.0.0 · runner dark (MI-12)"). Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist), verbatim, plus the ADR-0035 §2 standing rule):

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

**For this tag (runner stream):**
- *Peers / next free:* `git ls-remote --tags origin 'refs/tags/runner-v*'` shows only `runner-v1.0.0-rc.1`; `runner-v1.0.0` is the first stable runner tag. Its major (`1`) equals **`deploy/runner.release-line`**; the fleet's `.release-line` governs `v*` tags only and is untouched.
- *ACL PRs:* n/a — the runner has no NATS.
- *Image before policy:* **this tag is the image.** No runner ImageRepository/ImagePolicy exists; [mi-10](sprint-mi-10.md) merges them (and the 2nd IUA) after this tag.
- *Contract / erase / GA:* none of them, so no `host-verify` or snapshot is required: nothing deploys. (`runner-v1.0.0` does *start* the judge↔runner contract: from here a runner major is a contract break.)
- *From M6:* n/a.
- *After the tag, by looking:* the point is that **nothing moved** — healthz reports the same fleet version as before; `k3s kubectl get deploy -n xlearn` shows the same images; every `xlearn-*` ImagePolicy's latest is unchanged and the HelmReleases are Ready; login, dashboard and coach smoke-test fine. Plus: GHCR has `xlearn-runner:1.0.0` at the recorded digest, pullable anonymously.
- *Record:* status.md runner stream row (`runner-v1.0.0` → digest → floor n/a → snapshot n/a → "not deployed"); no flag change.
- *NetworkPolicy rule:* n/a — no new in-cluster caller. `judge-to-runner` already exists ([mi-14](sprint-mi-14.md)); judge's egress to the runner is [m3-07](sprint-m3-07.md)'s infra PR.

## Definition of Done

CI green (including `runner-repro` and `runner-image-acceptance`) · the rc rehearsal observed (no `deploy.yml` run, anonymous pull, digest match) · `runner-v1.0.0` tagged, released and recorded · VM rehearsal green and the VM deleted · no production change, no infra PR · statuses updated (this file + [`../status.md`](../status.md): board, M3 row, MI-12 row, runner stream row, hand-offs) · the PCH choice, image size, BuildKit pin and guard file in the decisions log, and the guard file documented in `docs/git-strategy.md`.

## Risks / watch-outs

- **A `v*` filter collision** would build (and, once ranges allow, deploy) the runner through the fleet pipeline. The workflow test and the rc observation are the guards; `deploy.yml`'s `^v` regex is the backstop.
- **Reproducibility traps:** apt logs and caches, `ldconfig`'s aux cache, Python `.pyc` timestamps, a GCC PCH that isn't byte-stable, buildx provenance attestations, a wall-clock `created` label, `rewrite-timestamp` clamping the Go cache seed's future mtime, and a different BuildKit version or layer compression between CI and a local rebuild (hence the pinned builder image and explicit output settings). `diffoci` names the file; fix the cause, never relax the check.
- **Rotation back-off on production.** Every SIGSYS restarts the container; kubelet's CrashLoopBackOff grows to 5 min, so a `prod` run can sit in back-off for a while and judge would see `saturated` (dark, so harmless). If it makes mi-10's run impractical, m3-03's recorded fallback (an in-process front restart with a new `BootEpoch`, an ADR-0030 delta) is the fix, not skipping probes.
- **`snapshot.debian.org` availability.** It can be slow or rate-limited; retry, and keep the snapshot date pinned (changing it is a runner patch with the ±5% speed check).
- **A private GHCR package** would make the pod unpullable (the VAP forbids pull secrets). Task 8 catches it before mi-10.
- **Image size and first pull.** g++ and Python make the image several hundred MB; mi-10's first pull and every runner bump (`Recreate`) mean a few minutes of `saturated`, acceptable while dark.
- **The VM rehearsal is arm64 and the CI lane is a privileged container.** Neither gives production timings; the production column of the TL doc is mi-10's. Never publish CI numbers as final TLs.
- **The suite is hostile code.** It stops on the first container OOM; on production (mi-10) the same stop conditions apply.
- **Parallel sessions** could push a runner tag; re-check `git ls-remote` right before tagging and take the next free version. Never move or re-push a tag.
