# Prompt — Sprint m3-15 · Runner release: reproducible image, runner-release.yml, acceptance suite, TL baselines → runner-v1.0.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-15.md`](../sprints/sprint-m3-15.md)   ·   **Milestone:** M3 (runner track; the image half of rollout step MI-12)   ·   **Prereqs:** [m3-04](../sprints/sprint-m3-04.md) (and [m3-03](../sprints/sprint-m3-03.md))

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync.
- The plan: [`../sprints/sprint-m3-15.md`](../sprints/sprint-m3-15.md) (the Dockerfile stages, the workflow jobs, the acceptance-suite table, the rehearsal steps and the release checklist are there).
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) **§1.3** (the fleet guard you mirror), **§1.5** (runner stream: `runner-v*`, self-contained workflow, `type=match`, reproducible, digest output; a major is a contract break), **§6** (release checklist).
- [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md) (Accepted; spike results, mechanism) and [ADR-0035 §2, §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) (NetworkPolicy rule, memory sum).
- [t3](../research/t3-sandbox.md) §5.3 (API, `/readyz` canaries), §5.10 (cleanup invariants), **§6.1** (image and release: toolchains, setuid strip, cache seed with a fixed future mtime, `profile_sha256`, ±5% speed check), §7.2–§7.3 (TL policy, calibration), §8.1–§8.3 and §8.7 (Flux layout, guards, runner values, host block), §8.8 A8, §16.1 (the arm64 pod seccomp profile for the VM rehearsal).
- [rollout §2.2, §7, §12](../rollout-plan.md) (operating rules, tag timeline, "`runner-release.yml` uses `type=match` and builds reproducibly").
- Neighbours: [mi-10](../sprints/sprint-mi-10.md) (task 4's value table and task 5's use of `make runner-acceptance`), [mi-11](../sprints/sprint-mi-11.md) (Renovate reads your `ARG` pins), [m3-02](../sprints/sprint-m3-02.md)/[m3-13](../sprints/sprint-m3-13.md) (TL gate, provisional flags).
- Code: `.github/workflows/{deploy,ci}.yml`, `deploy/*.Dockerfile`, `deploy/version_test.go`, `.dockerignore`, `docker-compose.yml`, `deploy/local/README.md`, `Makefile`, `.release-line`, `docs/git-strategy.md`, `cmd/runner` (incl. m3-04's `runner seed-gocache`), `internal/runner/**` (incl. `internal/platform/harness`), `internal/platform/runnerapi`, `docs/architecture/runner.md`.
- Infra (read-only, for the rehearsal): `../infra/hack/host-bootstrap.sh` (sandbox block, `PIN_K3S_VERSION`), `../infra/hack/host-verify.sh` (`sandbox.seccomp` checks the BOM hash), `../infra/infrastructure/sandbox/`, `../infra/charts/project/`.
- [m3-03](../sprints/sprint-m3-03.md) task 5 (the runner **drains and exits after any SIGSYS**: your suite must wait across those rotations).

## Context

m3-03 built the runner core and accepted ADR-0030; m3-04 added the Go, C++ and Python profiles, the harness codecs and the amd64
allowlists. The runner has its **own release stream** (ADR-0034 §1.5): `runner-v*` tags build `ghcr.io/sujaykumarsuman/xlearn-runner`
through a self-contained workflow, never through the fleet's `deploy.yml`. This sprint makes the image reproducible, adds that
workflow, ships the acceptance suite that production runs dark in mi-10 (after the Oct 24 host window), publishes the TL baselines
the pack TL gate needs, adds a dev-mode compose service for judge's e2e, rehearses the full pod shape on a throwaway VM, and tags
**`runner-v1.0.0`**. Nothing deploys: the runner's ImagePolicy doesn't exist until mi-10 (image before policy). The fleet is on
1.x; production has one user (the owner, D35).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] m3-04 merged (profiles, the extended `internal/platform/harness`, `runner seed-gocache`, arm64 dev lists KILL-default); `runner-it` green with Go, C++ and Python; ADR-0030 Accepted.
- [ ] `git ls-remote --tags origin 'refs/tags/runner-v*'` is empty; no peer PR touches `deploy/runner*`, `.github/workflows/`, `docker-compose.yml`, `Makefile` (`gh pr list`, `git worktree list`, ListAgents).
- [ ] `.release-line` = `1` and `deploy.yml`'s `release-line` job is in place (MI-2a ✅).
- [ ] For step 8: `../infra` `main` is synced (`git -C ../infra pull --ff-only`) and has mi-09's sandbox block and mi-14's `infrastructure/sandbox/`; otherwise use their PR branches or t3 §16.1 / §8.2 and note it.

No owner needs to be reachable (D40): if the new GHCR package comes up private in step 11, that step records the owner action as ⛔ and the session carries on.

## Do this (in order)

1. **[X] Branch** `feat/runner-release` from an up-to-date `main`.
2. **[X] Dockerfile** (plan task 1): `deploy/runner.Dockerfile` with the pins at the top (`BASE` multi-arch index digest,
   `DEBIAN_SNAPSHOT`, `GO_VERSION`, `GO_SHA256_AMD64` / `GO_SHA256_ARM64` picked by `TARGETARCH`; nsjail only for R1-N); stages `build` (the exact `go build -trimpath -buildvcs=false -ldflags="-s -w -X main.version=${VERSION}" -o /out/runner ./cmd/runner` line), `rootfs` (snapshot apt, `g++`, `python3`, checksummed Go tarball, pruned; runner binary, profiles, templates, preludes; `runner seed-gocache -out …` (m3-04's subcommand; fail the build on its exit code; add it here only if it is missing); `compileall --invalidation-mode checked-hash`; mount-point dirs; strip setuid/setgid; delete caches/logs/docs; `touch -h -d @$SOURCE_DATE_EPOCH` everything, then the cache seed's fixed future mtime) and a single-layer `FROM scratch` final. **No `rewrite-timestamp`, no wall-clock labels.** Decide the C++ PCH (in-image only if byte-stable). `deploy/version_test.go` must still pass.
3. **[X] Workflow** (plan task 2): `.github/workflows/runner-release.yml` (`runner-v*` only; jobs `guard` → `image` → `release`;
   `docker/metadata-action` `type=match,pattern=runner-v(\d+\.\d+\.\d+.*),group=1`, `flavor: latest=false`; `platforms: linux/amd64`;
   the pinned builder (`driver-opts: image=moby/buildkit:<v>@sha256:…`); `provenance: false`, `sbom: false`; `outputs:
   type=image,push=true,oci-mediatypes=true,compression=gzip,compression-level=9,force-compression=true`; digest in the job
   summary); `deploy/runner.release-line` (`1`); `deploy/workflows_test.go` (glob semantics, both directions: `deploy.yml` never
   matches `runner-v…`, `runner-release.yml` never matches `v…`). Add the **runner-stream subsection to `docs/git-strategy.md`**
   (`runner-v*`, `deploy/runner.release-line`, the `-rc` rehearsal, never move a tag, how to go to a runner major); extend an
   existing runner section rather than duplicating it.
4. **[X] Acceptance suite** (plan task 4): `internal/runner/acceptance/` (`//go:build runner_acceptance`) with sections A–I and the
   calibration mode exactly as the plan's table; probes and kernels under
   **`internal/runner/acceptance/testdata/{probes,kernels}/{go,cpp,python}/`** (loaded with `embed`; `testdata` keeps these
   `func-json@1` solution files out of `go build/vet/test ./...`). **`SUBSET=full|prod` is required** (no default; the plan's
   table defines both — `prod` = A–F + H (+ I) at the t3 §5.10 A8 counts, no G). **Rotation-aware:** batch every SIGSYS-expected
   case into one job per language per section (a correctness group, so every case runs); keep E's allowed-channel markers in one
   `BootEpoch`; after a SIGSYS result, wait for `/readyz` 200 with a new `boot_epoch` (6 min timeout, back-off tolerant); report
   expected vs observed rotations; an unexplained epoch change fails. Stop conditions: container `oom_kill` delta > 0, an
   unexplained epoch change, or `/v1/stats` gone outside an expected rotation → abort. JSON report in `bin/` and a Markdown summary;
   `make runner-acceptance RUNNER_URL=… RUNNER_TOKEN_FILE=… SUBSET=full|prod [REQUIRE_PROD=1] [CALIBRATE=1]`.
5. **[X] CI lanes** (plan task 3): both on the **pinned BuildKit image** (same as the release workflow) with the same output
   settings. `runner-repro` (two `--no-cache` builds → identical OCI manifest digest; pinned `diffoci` on a mismatch) and
   `runner-image-acceptance` (`docker run -d --restart=always --privileged --cgroupns=private --memory=3g --cpus=2`,
   `RUNNER_MODE=dev`, `SUBSET=full` without prod-only sections — including m3-04's public content check in section G — then
   `CALIBRATE=1`; report as an artifact), both path-filtered to `deploy/runner*`, `cmd/runner/**`, `internal/runner/**`,
   `internal/platform/runnerapi/**`, `curriculum/**`. Add a rootfs grep for setuid bits, secrets and any evalpack path.
6. **[X] Compose** (plan task 5): the `runner` service (compose profile `runner`, privileged, private cgroupns, 3g/2 CPUs,
   **`restart: unless-stopped`**, `RUNNER_MODE=dev`, the committed dev-only token file `deploy/local/runner-token.dev`); update
   `deploy/local/README.md`.
7. **[X] TL-baselines doc** (plan task 7): `docs/architecture/runner-tl-baselines.md` with the rule, one table per profile, the CI
   column from step 5's calibration, and a production column marked "pending mi-10"; link it from `docs/architecture/runner.md`.
8. **[H] VM rehearsal** (plan task 6): a throwaway arm64 multipass VM; k3s via `host-bootstrap.sh --upgrade --with-k3s` (whatever
   `PIN_K3S_VERSION` `../infra` `main` pins; never hardcode it); `host-bootstrap.sh --with-sandbox` (sandbox block + L23), then
   replace `/var/lib/kubelet/seccomp/profiles/xlearn-runner.json` with t3 §16.1's **arm64** pod profile, restart k3s;
   `host-verify --expect-sandbox` in the VM with exactly one expected FAIL (`sandbox.seccomp`, the BOM hash is amd64's);
   `infrastructure/sandbox/` applied **inside the VM only**; the arm64 image imported and referenced by digest; the pod rendered
   from `../infra/charts/project` with mi-10's values, `RUNNER_MODE=prod`; Ready with every canary green; `make runner-acceptance
   SUBSET=full REQUIRE_PROD=1` via a port-forward in a retry loop. The image's arm64 exec filters are KILL-default, so section C
   must pass unchanged. Record the summary and rotation count, then `multipass delete --purge`. No production credential in the VM;
   no timing conclusions.
9. **[X] Verify** locally and in CI: `gofmt`, `go vet ./...`, `go test -race ./...`, `sqlc diff` unchanged, the new lanes green.
10. **[X] PR** (Ship steps 1–2 below) → conventional commits (`build(runner): …`, `ci(runner): …`, `test(runner): acceptance suite`, `docs(runner): TL baselines`)
    with the attribution lines → CI green → squash-merge.
11. **[X] Rehearsal prerelease** (plan task 8): tag `runner-v1.0.0-rc.1` on `main` and push it. Check: the workflow built
    `1.0.0-rc.1`; **no `deploy.yml` run** for that ref; the anonymous pull works (`docker logout ghcr.io`, then pull). **If the pull is
    refused** (a new GHCR package defaults to private), don't wait (D40) and don't change account or package settings yourself: making
    the `xlearn-runner` package Public (package settings, linked to `sujaykumarsuman/xlearn`) is an owner-only action that can only
    follow this push. Record ⛔ in status.md naming it (the MI-12 row and Blocked / needs input), set plan task 8 ⛔ with that reason,
    and carry on: the tag doesn't depend on visibility. The follow-up is the same anonymous pull against `:1.0.0`; whichever session
    re-runs it after the owner's flip ticks task 8 ✅, and it must pass before mi-10 deploys. Rebuild the rc commit locally for
    `linux/amd64` on the same pinned BuildKit image (a `docker-container` builder created with that image) and output settings, and
    compare digests; `docker run --restart=always` the image in dev mode (the digest-identical local rebuild if the package is still
    private) with `SUBSET=prod` as a smoke.
12. **[X] Tag `runner-v1.0.0`** (plan task 9): run every release-checklist line in the plan, including its "for this tag" notes
    (re-check `git ls-remote --tags origin 'refs/tags/runner-v*'` right before tagging; major = `deploy/runner.release-line`). Tag the rc
    commit (or cut `-rc.2` if runner paths changed since), push, verify the workflow, the digest, the anonymous pull (or step 11's ⛔
    owner item stays open) and the GitHub release (`runner-v1.0.0 — v2 build · M3 runner (dark)`). After the tag, **by looking**: fleet
    healthz, images and ImagePolicies are **unchanged**, HelmReleases Ready, smoke login/dashboard/coach (through an already-signed-in
    browser session if the session has one: never enter credentials; else the credential-free checks and an "owner login smoke
    pending" pending-smoke note in status.md).
13. **[X] Record** (docs PR, merge only): status.md rows and hand-offs (below).

## Constraints

- **Release streams:** the runner is built only by `runner-release.yml` on `runner-v*`; never add it to `deploy.yml`; never touch
  `.release-line` (fleet). **Never move or re-push a tag**; a fix is a new patch or rc.
- **Image before policy:** this sprint creates the image; the ImageRepository/ImagePolicy and the 2nd IUA are mi-10's infra PRs, after
  the tag. **No infra PR here.**
- **GitOps / production:** no `kubectl apply` against production; `kubectl apply` happens only inside the throwaway VM. Read-only
  `ssh vps` for the after-tag checks. `../infra` is read-only.
- **Security:** the image carries no secret, no pack content, no setuid bit; privileged containers are CI/compose-only; the dev token is
  obviously non-secret and never used outside compose.
- **Service boundaries** ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)): the runner has no DB, NATS or egress.
  **goose + sqlc:** no migration; `sqlc diff` unchanged. **Outbox/inbox, consumers-before-producers, NATS ACL PR before a consuming
  tag:** n/a (no events).
- **D34:** no alerting, no timer, no CronJob, no push channel; the suite and CI report, nothing pages.
- **Memory-sum rule** (ADR-0035 §5): no new always-on pod here; the runner's 3 GiB is counted when mi-10 deploys it.
- **theme.css:** n/a (no UI).
- **Parallel sessions:** check peers' tags (fleet and runner), PRs, worktrees and ListAgents before tagging or claiming an ADR number
  (none expected).

## Deliverables

- `deploy/runner.Dockerfile`, `deploy/runner.release-line`, `deploy/workflows_test.go`.
- `.github/workflows/runner-release.yml`; CI jobs `runner-repro` and `runner-image-acceptance`.
- `internal/runner/acceptance/` (+ probes and kernels under its `testdata/`) and `make runner-acceptance`.
- The runner-stream subsection of `docs/git-strategy.md`.
- The compose `runner` service, `deploy/local/runner-token.dev`, the README update.
- `docs/architecture/runner-tl-baselines.md` (+ the link from `runner.md`).
- Tags `runner-v1.0.0-rc.1` and `runner-v1.0.0`; the GitHub release; the VM rehearsal summary.

## Update status

- [`../sprints/sprint-m3-15.md`](../sprints/sprint-m3-15.md): each task ⬜ → 🔄 → ✅ (or ⛔ with a reason); _Overall_ ✅ at the end.
- [`../status.md`](../status.md): the Sprint board row (m3-15 ✅); M3 🔄; the MI-12 row ("image `runner-v1.0.0` ✅; deploy = mi-10");
  the **runner stream** tag row (`runner-v1.0.0` → digest → floor n/a → snapshot n/a → not deployed); the TL-baselines link;
  **Decisions log**: the PCH choice, image size, the BuildKit pin, `deploy/runner.release-line` (documented in `docs/git-strategy.md`),
  the package visibility (public at first push, or step 11's ⛔ owner item and its re-check), the VM rehearsal result (incl. the expected `sandbox.seccomp` FAIL) and any VM-only
  VAP widening; **hand-offs** to mi-10 (digest, `RUNNER_IMAGE_DIGEST` marker, `emptyDir` needs or none, **the plan's exact
  production command** — `SUBSET=prod REQUIRE_PROD=1 CALIBRATE=1` with the port-forward retry loop — and the expected rotation /
  `restartCount` rise), m3-02 (TL gate in this image, CI column, provisional), m3-06 (compose `runner` profile), m3-07, m3-13.
- No flag change; no fleet tag-floor change.

## Done when (acceptance)

- [ ] Reproducible image digest across two builds, and the pushed rc digest equals a local rebuild on the same pinned BuildKit image.
- [ ] The acceptance suite is green on the local jail-capable VM (full pod shape, prod mode) and in the CI in-image lane.
- [ ] `deploy.yml` never runs for `runner-v*`, and `runner-release.yml` never for `v*` (test + rc observation).
- [ ] `runner-v1.0.0` exists at `ghcr.io/sujaykumarsuman/xlearn-runner:1.0.0`, anonymously pullable (or, for a private package, step 11's ⛔ owner item recorded), digest recorded; nothing deployed or moved.
- [ ] The TL-baselines doc is published (CI column; production column pending mi-10); the compose `runner` service works in dev mode
      and restarts after a SIGSYS rotation.
- [ ] The suite waits across every rotation (expected = observed); `SUBSET` is required; `docs/git-strategy.md` has the runner
      stream.
- [ ] No setuid bit, secret or pack content in the image; statuses and hand-offs recorded.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). This repo only: step 10's PR on `feat/runner-release`, then step 13's docs/status PR; no `../infra` PR (read-only here; the ImagePolicy is mi-10's).
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — tag `runner-v1.0.0` (runner stream):** Follow the stream's own tag procedure: after step 10's merge, the `runner-v1.0.0-rc.1` rehearsal (step 11: `runner-release.yml` built it, no `deploy.yml` run, the anonymous pull or its ⛔ owner item, the local digest match, the `SUBSET=prod` smoke), then `runner-v1.0.0` via `runner-release.yml` (step 12: the plan's release checklist and its "for this tag" notes; by looking, the fleet is unchanged); record it under **release streams** (digest; `validated_against` / notes: the TL-baselines link; "not deployed"). No app tag; nothing deploys until mi-10.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way (step 13's docs PR).
5. Run `git checkout main && git pull` in every repo touched (xlearn; `../infra` stays read-only, synced in the entry gates). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
