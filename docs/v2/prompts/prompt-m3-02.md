# Prompt — Sprint m3-02 · Evalpack pipeline: private CI gates, generic validator, image build (E) + compose fixture pack

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-02.md`](../sprints/sprint-m3-02.md)   ·   **Milestone:** M3 (content track)   ·   **Prereqs:** [mi-07](../sprints/sprint-mi-07.md) (MI-9), [m3-01](../sprints/sprint-m3-01.md) (canon, packspec, packlint, hook)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync, and the
  **never-copy rule** for `../xlearn-evalpack` (added by m3-01).
- The plan: [`../sprints/sprint-m3-02.md`](../sprints/sprint-m3-02.md) — the pack layout, the `packspec` file table, the gate table,
  the image allowlist and the fixture layout are spelled out there. Follow them.
- [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) (Accepted) §1, §2 (the pack, the image volume, `contract_hash`),
  §5 (leak controls), §8 (CI gates, what AI may draft); [ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)
  (the evalpack stream: `v1.x` tags, `>=1.0.0 <2.0.0`, image before policy).
- Research: [t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack) (layout, image, flow),
  [§3.4](../research/t1-content-data-model.md#34-versioning-and-pinning), [§3.7](../research/t1-content-data-model.md#37-local-dev-and-public-ci)
  (compose + public CI), [§7.1](../research/t1-content-data-model.md#71-package-format) (case format, type and checker registries),
  [§7.2](../research/t1-content-data-model.md#72-ci-validation) (the private-repo gate table),
  [§7.3](../research/t1-content-data-model.md#73-where-ai-may-help), [§10](../research/t1-content-data-model.md#10-security-and-threat-notes);
  [t4 §2.6](../research/t4-judge-contract.md#26-evaluation) (case order, perf block), [§4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide)
  (seeded perf specs, the generator registry), [§9.4](../research/t4-judge-contract.md#94-the-per-case-input-cap-t1-flag-resolved);
  [t3 §6.1](../research/t3-sandbox.md#61-image-and-release) (go1.26.8), [§7.2](../research/t3-sandbox.md#72-time-limit-policy) (TL = max(3 × ref, 1 s), pack lints),
  [§7.3](../research/t3-sandbox.md#73-calibration).
- Neighbours: [mi-07](../sprints/sprint-mi-07.md) (the scaffold, `build.yml`, the probe), [m3-01](../sprints/sprint-m3-01.md) (what
  `packlint` already does), [m3-04](../sprints/sprint-m3-04.md) tasks 1–2 (**the spec** for the Go half of `internal/platform/harness`
  you build here; m3-04 extends it), [m3-06](../sprints/sprint-m3-06.md) (imports the checkers and generators you create),
  [m3-05](../sprints/sprint-m3-05.md) (the compose overlay hand-off), [spk-02](../sprints/sprint-spk-02.md) (multi-arch request, if any).
- Code: `cmd/packlint/`, `internal/packspec/`, `internal/course/{canon,lint}/`, `internal/curriculum/load.go` (m1-09's loader and its
  `_code/` handling), `.github/workflows/ci.yml`, `docker-compose.yml` (the `x-pg-env` anchor style), `deploy/local/README.md`; in `../xlearn-evalpack`: `README.md`, `pack.json`, `hack/build.sh`,
  `Dockerfile`, `.github/workflows/build.yml`.

## Context

After mi-07 the private repo is only a pipe: a `FROM scratch` scaffold image and a probe proving the package isn't public. After m3-01
the owner has hashes, `packlint check` and the pre-push hook. What's missing is the pipeline that turns authored material into a trusted
pack: reproducible cases with expected outputs computed from the public reference (never by AI), one validator derived from the public
constraints, the oracle / wrong-solution / time-limit gates, the review-stamp gate, and the data image judge will mount. The owner
authors the 14 pilot packs (28–41 h) against this pipeline from mid-October to mid-November — the owner-hours chain to M3.
**Public code, private data:** every tool is public Go in this repo (shared with judge later); only pack contents stay private. The
runner (m3-03/m3-04/m3-15) doesn't exist yet, so programs run in pinned stock toolchain containers with `--network none`, and every TL is
**provisional** until m3-13. Three shared packages land here first because the pipeline needs them, each at **one** path that later
sprints extend or import, never duplicate: `internal/packspec/gen` (t4 §4.1's generator registry; m3-06 imports it),
`internal/platform/checker` (m3-06's `code@1` imports it) and the Go half of `internal/platform/harness`, built to m3-04's frame
protocol (m3-04 adds C++/Python). v2 is owner-only (D35).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] MI-9 done: `gh repo view sujaykumarsuman/xlearn-evalpack --json visibility,isFork,isTemplate` → `PRIVATE/false/false`;
      `../xlearn-evalpack` checked out; its latest probe run green; `0.1.0` exists; the infra pull secrets + ImageRepository merged.
- [ ] m3-01 merged: `internal/course/canon`, `internal/packspec`, `cmd/packlint check|hash|fingerprint` on `main`; `git config core.hooksPath`
      → `hack/git-hooks` in this clone (if it's unset, run `make install-hooks` yourself first: a per-clone setting, not an owner step).
- [ ] Parallel sessions: `gh pr list --state open` in **both** repos, `git worktree list`, ListAgents — nothing conflicting on `cmd/packlint`,
      `internal/packspec`, `internal/platform/checker`, `internal/platform/harness`, `docker-compose.yml` or the evalpack workflows; no
      peer in `../xlearn-evalpack`.
- [ ] Check (not a stop): do `internal/platform/harness`, `internal/platform/checker` or `internal/packspec/gen` already exist (normally
      not — m3-04/m3-06 run later)? If one does, extend it. Did spk-02 request a multi-arch pack?

## Do this (in order)

1. **[X] Branch** `feat/m3-evalpack-pipeline` in xlearn off an up-to-date `origin/main`.
2. **[X] `internal/packspec`** — `cases.go`, `lock.go`, `validate.go`, `manifest.go` (`ReadManifest`, `VerifyFiles`), `build.go`,
   `listing.go` per the plan's table (case id = `sha256("xlearn.case@1\n"+input)`, order edge → random → perf, 256 KiB literal cap,
   deterministic zstd, expected from the Go reference `_code/solution.go` only, only stamped items built). `/manifest.json` has
   **exactly** t1 §3.3's fields (no `course` — judge may decode strictly); `ReadManifest` is strict and `VerifyFiles` streams sha256.
   Extend m3-01's item `pack.json` type with `gen[]` and `large_case_exception`. Add `github.com/klauspost/compress`.
3. **[X] Generator registry** — `internal/packspec/gen` (the one registry; m3-06 imports it): `int_array@1`, `string@1`,
   `permutation@1`, `tree@1`, `graph@1` (the edge-list generator), `op_sequence@1`; an explicit list; streaming, deterministic from
   `(params, seed)`, params checked against the item's constraints, golden hash per `gen@v`.
4. **[X] Executor, checkers, codec** — `packlint exec`: one `docker run --network none --read-only …` per program with digest-pinned
   images, one process per case, CPU and peak RSS per case. Go env `HOME=/w GOCACHE=/w/gocache GOFLAGS=-mod=readonly GOTOOLCHAIN=local
   GOPROXY=off CGO_ENABLED=0`, a 512 MiB tmpfs, and a read-only std-cache volume per image digest (`packlint exec --warm-cache`)
   copied in at start. `internal/platform/checker` (exact, unordered, unordered_deep, float abs/rel, set_equal, any_of; typed verdicts
   on `harness.Value`). The Go half of `internal/platform/harness` to **m3-04's plan tasks 1–2**: `ParseType`, canonical JSON,
   `EncodeInput`/`DecodeOutput`/`Canonical`, the `u32`-BE fd-3/fd-4 frames with `ok`/`panic`/`error`, `templates/go/*.tmpl`, and
   `Generate` returning the package's own `File` (runnerapi doesn't exist yet). C++/Python gates report `pending(harness)`.
   Gates `oracle`, `wrong`, `tl` (TL ≥ max(3 × ref max, 1 s), Σ TL ≤ 40 CPU-s, memory ≥ 2 × ref peak + baseline, TLE wrong
   solutions exceed TL; `timing.json` with `provisional: true`).
5. **[X] Subcommands** — `packlint lock (--write|--verify)`, `validate`, `exec --gate …`, `build`, `listing`; `fingerprint` also reads
   the pack's `build/` cases. Unit tests per file.
6. **[X] Fixture** — `internal/judge/testdata/{content,packsrc,pack}` in a separate **`fixture` course** (not real DSA items — the
   plan says why) with the three **original, synthetic** items (`fx-001` exact, `fx-002` unordered_deep with a WA and a TLE wrong
   solution, `fx-003` class-ops), each with `_code/solution.{go,cpp,py}`. The content root passes m1-09's loader and guards:
   `paths.json` (a `fixture` row, status `active`), `ids.lock.json`, `courses/fixture/{course.json,phases,weeks,concepts}.json`; no
   `_schema/` copy. If m1-09's loader treats every `_code/` file as a section, make it skip the reserved `solution.*` names (tested).
   Add the `internal/judge/testdata/SYNTHETIC.md` marker (m3-01's exact first line). `internal/packspec/fixture_test.go` (no
   docker): every case line hashes to `tests.lock`, re-compression and `manifest.json` recompute identically. The `pack-fixture` job
   in `.github/workflows/ci.yml` (docker, no secrets, the std-cache restored via `actions/cache`) runs check → lock --verify →
   validate → exec --gate all → build to a temp dir + `diff -r` against `pack/` → listing.
7. **[X] Compose** — the `x-evalpack-mount` anchor (plan task 7) at the top of `docker-compose.yml`; `docker compose config` passes;
   "Eval pack in compose" in `deploy/local/README.md`, including the **`FIXTURE_CONTENT_DIR` overlay contract** for m3-05 (DEV_AUTH
   only; read by judge, curriculum and practice; loads the fixture root beside the embedded curriculum). Don't build the overlay:
   judge doesn't exist yet.
8. **[X] Verify + PR (xlearn)** — `gofmt -l .`, `go vet ./...`, `go test -race ./...`, `sqlc diff`, web checks (then
   `git checkout -- web/dist/.gitkeep`), e2e unchanged; commit (`feat(content): evalpack pipeline — packspec, generator registry,
   executor, shared harness/checker, fixture pack`) with the attribution lines; push; PR; CI green; squash-merge.
   **Cut line:** this xlearn PR stands alone. If the session can't finish steps 9–12 too, stop here: don't touch
   `../xlearn-evalpack` `main`, set plan tasks 1, 5, 6, 8 ⛔ "carried: evalpack half", and a follow-up session runs steps 9–13 only.
   This session still runs step 13 and **Ship** below (xlearn only).
9. **[E] Branch** `feat/pack-pipeline` in `../xlearn-evalpack`. Format v1: `pack.json` → `format_major: 1`, the item layout and `drafts/`,
   `tests.lock` with its tool-version header, the `Makefile` (`packcheck`, `lock`, `build`, `image`, `selftest`) calling
   `go -C ../xlearn run ./cmd/packlint`; replace `hack/build.sh` with `packlint build` (keep the version == tag check).
10. **[E] `packcheck.yml`** — public checkout at `main` or `public-ref:`; changed-items detection + full-run triggers; the gate table in
    order; the `selftest` job on the public fixture; a job summary with ids and counts only; add a weekly `schedule` to mi-07's probe.
11. **[E] Image** — extend `build.yml`: `packlint build` → generated Dockerfile (one `COPY` per course + `manifest.json`), `buildx`
    `linux/amd64,linux/arm64`, `provenance: false`, `sbom: false`, `rewrite-timestamp` + `SOURCE_DATE_EPOCH`, the listing test
    **before** the push, then the probe. Prove a planted `gen/` file fails the listing test, then remove it.
12. **[E] README** — the authoring rules (plan task 8). Commit (`feat: pack format v1, packcheck gates, image build`) with the attribution
    lines; PR; the `selftest` + probe green; squash-merge. Optional: tag `v0.2.0` and check build → push → probe. **Never tag `>=1.0.0`.**
13. **[X] Update status** (below) as an xlearn docs commit/PR if the code PR has already merged; then **Ship:** see **Ship** below.

## Constraints

- **Secrecy (ADR-0027 §1, §5):** never read real pack data into this repo, tests, logs, the PR, commit messages or chat. The fixture is
  synthetic and original. packlint output and CI summaries carry ids, counts and hashes only. Never change the package's or repo's
  visibility; the probe stays green.
- **Programs never run on the host:** references, oracles, wrong solutions, generators and validators run only through `packlint exec`
  (`--network none`, read-only, non-root, pinned digests).
- **Expected outputs are computed, never written** by you or any AI. Don't stamp `review.tests` — that's the owner's.
- **One definition, one path:** `internal/platform/harness` (Go half, to m3-04's spec), `internal/platform/checker`,
  `internal/packspec/gen` (t4 §4.1's six generators). Never `internal/runner/harness`, `internal/checker` or `internal/judge/gen`.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** `packspec`, `platform/checker`, `platform/harness`
  are libraries (no DB, no service imports); no migration here (`sqlc diff` unchanged).
- **Releases:** evalpack stays below `1.0.0` (image before policy — m3-07 tags `v1.0.0`, then its ImagePolicy merges). No xlearn tag.
- **GitOps / infra:** no `../infra` change; never `kubectl apply`. D34: the weekly probe is a CI check, not an alert — no push channel,
  no opscheck. No new pod, so the memory-sum rule is untouched; no events, so outbox/inbox, consumers-before-producers and
  ACL-before-consumer don't apply.
- **Parallel sessions:** check peers' PRs, tags and worktrees in both repos before merging or tagging (`gh pr list`,
  `git ls-remote --tags origin`, `git worktree list`, ListAgents); rebase onto a moved `main`. No ADR number is claimed.

## Deliverables

- xlearn: `internal/packspec/{cases,lock,validate,manifest,build,listing}.go` + `gen/` + tests; `internal/platform/checker/`;
  `internal/platform/harness/` (Go half: `types.go`, `canon.go`, frames, `templates/go/`, tests); `cmd/packlint` subcommands (incl.
  `exec --warm-cache`); `internal/judge/testdata/{SYNTHETIC.md,content,packsrc,pack}`; the `pack-fixture` CI job; the compose anchor;
  `deploy/local/README.md`.
- xlearn-evalpack: `pack.json` (format 1), layout + `drafts/`, `tests.lock`, `Makefile`, `.github/workflows/packcheck.yml`, the
  extended `build.yml`, `README.md` authoring rules; optionally a `v0.2.0` image.

## Update status

- [`../sprints/sprint-m3-02.md`](../sprints/sprint-m3-02.md): each task 🔄 → ✅; _Overall_ ✅ when all are.
- [`../status.md`](../status.md): the Sprint board row; the **M3** row; **content status** (pipeline live, format 1, items stamped 0/14,
  TLs provisional until m3-13, C++/Python gates pending m3-04); the **evalpack stream** row (`v0.2.0` + digest if tagged); **owner events**
  (`ev-packs-14`: pipeline ready — the owner's pack authoring continues; it doesn't gate this sprint); **Decisions log**: public code / private data; the three shared-package paths and who extends
  them; the manifest kept to t1 §3.3's shape; the `fixture` course instead of DSA items and the `FIXTURE_CONTENT_DIR` overlay
  contract for m3-05; the std-cache seed; the provisional TL scale; multi-arch; the `public-ref:` token.
- No ADR (ADR-0027 is Accepted). A change to a decided shape needs a new ADR — check peers for the next free number first.

## Done when (acceptance)

- [ ] A sample item passes every gate (the synthetic fixture, in public CI and in the private `selftest`); a planted wrong solution fails as declared.
- [ ] `lock --verify` reproduces `tests.lock` byte-for-byte; every `invalid/*` is rejected.
- [ ] The image contains only allowed files (listing test), one layer per course, with a verifying `/manifest.json`.
- [ ] Probe green on every push (and weekly).
- [ ] The compose anchor is in place; public CI needs no secrets; `pack-fixture` rebuilds `pack/` identically; `fixture_test.go` passes without docker.
- [ ] `internal/platform/harness` (Go half), `internal/platform/checker` and `internal/packspec/gen` exist once, at those paths, with golden tests.
- [ ] The pack README carries the authoring rules; `docs/v2/status.md` is updated.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: the xlearn PR first (step 8 above, `feat/m3-evalpack-pipeline`), then the `../xlearn-evalpack` PR (step 12 above, `feat/pack-pipeline`); no `../infra` PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. (The evalpack PR's checks are its `selftest` and probe jobs.)
3. **Release action — merge only + an optional evalpack `v0.2.0`:**
   - xlearn: nothing deploys; the PR ships in the next app tag with no runtime change: `v1.6.0` (cut by [m1-02](../sprints/sprint-m1-02.md)) if merged before that tag, else `v1.7.0` (cut by [m1-07](../sprints/sprint-m1-07.md)). Don't tag xlearn.
   - `../xlearn-evalpack`: the pipeline lands on `main`. `v0.2.0` is optional; if you tag it, follow the evalpack stream's own procedure: check peers' evalpack tags first (`git ls-remote --tags` there), tag `v0.2.0`, let `build.yml` run `packlint build`, the listing test, the multi-arch push and the probe (anonymous GET 401/403), and record it under **release streams** in status.md (digest, `validated_against`). It sits below every `>=1.0.0 <2.0.0` range, so nothing deploys. **Never tag `>=1.0.0`** ([m3-07](../sprints/sprint-m3-07.md) cuts `v1.0.0`).
   - At the cut line (step 8 above): xlearn only; plan tasks 1, 5, 6 and 8 are ⛔ "carried: evalpack half", and a follow-up session runs steps 9–13 above.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn and `../xlearn-evalpack`; at the cut line, xlearn only). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
