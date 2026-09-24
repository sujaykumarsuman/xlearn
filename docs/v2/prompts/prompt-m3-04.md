# Prompt — Sprint m3-04 · Runner profiles Go/C++/Python + harness codecs + amd64 allowlists

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-04.md`](../sprints/sprint-m3-04.md)   ·   **Milestone:** M3 (runner track; code for rollout step MI-12)   ·   **Prereqs:** [m3-03](../sprints/sprint-m3-03.md), [spk-02](../sprints/sprint-spk-02.md), [m1-01](../sprints/sprint-m1-01.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync.
- The plan: [`../sprints/sprint-m3-04.md`](../sprints/sprint-m3-04.md) (the type table, the profile table, the lint rules and the item list are there).
- [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md) (Accepted in m3-03: §2 languages, §3 timing, the spike-results section).
- [t3 §16.2](../research/t3-sandbox.md) — **the amd64 allowlists** (per-profile sorted syscall lists, justified additions, compile sets). You copy them; you don't regenerate them.
- [t3](../research/t3-sandbox.md) §2.4 (A1, A5, A18), §6.1–§6.2 (image, profile table, harnesses, ASLR), §7.2–§7.3 (TL policy, calibration), §12 (D20: Go + C++ + Python).
- [t1 §7.1](../research/t1-content-data-model.md#71-package-format) (the closed type and checker registries, harness `@v`), [t4 §2.6](../research/t4-judge-contract.md#26-evaluation), [§4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide), [§5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start), [§11.1](../research/t4-judge-contract.md#111-t3-the-runner-contract-not-the-technology).
- Code from m3-03: `internal/runner/profile` (the `Profile` interface and registry), `internal/runner/seccomp`, `internal/runner/it`, `internal/platform/runnerapi` (`names.go`, types), `docs/architecture/runner.md`, the `runner-it` job in `.github/workflows/ci.yml`.
- Code from m1-01/m3-01: `internal/course` (`Signature`, `Harness`, `Languages`), `docs/v2/authoring.md`, and m3-01's Out list (compiled starters + reference-passes-samples were handed to you).
- **Code from m3-02 (it ran first; you extend it, never fork it):** `internal/platform/harness` — **its `README.md` wire format and goldens are the `@1` spec** — plus `internal/platform/checker` (the closed checker registry you compare with) and `internal/judge/testdata/pack` (the fixture items). [p-01](../sprints/sprint-p-01.md) later adds `internal/platform/harness/gotest/` to the same package tree.
- [t1 §7.2](../research/t1-content-data-model.md#72-ci-validation) (the public content CI checks you add in step 11).

## Context

m3-03 built the runner's trusted core: the privilege-split supervisor, userns-less per-case jails, per-case cgroups with outside
measurement, the job API, `throttled`, and a test-only `testgo@0` profile. This sprint makes it grade the **three launch
languages (D20)**. judge (built later in m3-05/m3-06) sends the runner the learner's file plus a **judge-generated, public
harness** (`func-json@1` / `class-ops@1`) and streams each case's arguments on fd 3; the harness writes one canonical JSON frame
on fd 4, which judge decodes and compares with a checker. m3-02 already created that codec's package, `internal/platform/harness`,
with the Go codec, and packs are gated against its bytes; you **extend the same package** to C++ and Python without moving a byte;
you add the three profiles with their toolchain specs, lints and the **amd64 seccomp allowlists the spike generated (spk-02)**;
and you prove it with reference and wrong solutions through the real jail. The toolchain version pins and the image are m3-15's.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] m3-03 merged (`git log origin/main`): `internal/runner/profile`, `internal/runner/seccomp`, the jail and the `runner-it` job exist; ADR-0030 is Accepted.
- [ ] t3 §16.2 has the amd64 `go`, `cpp` and `python` lists and the compile sets, with 0 unexpected SIGSYS under KILL.
- [ ] `internal/course` exposes the frozen `Signature` (`Mode`, `Name`, `Params`, `Returns`, `Ops`), `Harness` and `Languages`.
- [ ] m3-02 merged: `internal/platform/harness` (Go codec, README wire format, goldens) and `internal/platform/checker` exist. The README's wire
      format is t3 §6.2's (a `u32`-BE-framed canonical JSON frame in on fd 3, one out on fd 4); if it is anything else, **stop and
      report** (`harness@1` is in `contract_hash`). If m3-02 hasn't merged, create `internal/platform/harness` yourself (with the
      README), use a test-only comparator until `internal/platform/checker` lands, and record the order.
- [ ] No open peer PR touches `internal/runner/profile/`, `internal/platform/harness/`, `internal/platform/checker/`, `internal/platform/runnerapi/lint/` (`gh pr list`, `git worktree list`, ListAgents).

## Do this (in order)

1. **[X] Branch** `feat/runner-profiles` from an up-to-date `main`.
2. **[X] Type registry + canonical JSON** (plan task 1): read `internal/platform/harness/README.md` and its goldens first, reuse what
   exists and add what's missing (`types.go`, `canon.go`, `encode.go`, `decode.go` in that package) — `ParseType` over the closed set
   only; the per-language mapping; canonical JSON (no whitespace, minimal string escaping, level-order trees with trailing `null`s
   trimmed, 1-indexed graph adjacency); floats never canonical (`CanonicalizableOutput`); `EncodeInput` (strict, bounded: depth ≤ 64,
   nodes ≤ 10⁶) and `DecodeOutput` + `Canonical`. **Where m3-02's bytes differ from these rules, its README wins for `@1`**; fix the
   rule and log a `@2` candidate. Golden tests for every type and every malformed case, plus a **compatibility golden**: m3-02's Go
   fixtures give the same bytes before and after. Extend `internal/runner/imports_test.go`: the harness package imports only the
   stdlib, `internal/course` and `internal/platform/runnerapi` (judge and packlint import it).
3. **[X] Harness codecs** (plan task 2): the `u32`-BE frame protocol; `{"ok"}` / `{"panic": class}` / `{"error"}` frames with the
   closed per-language panic classes; templates under `internal/platform/harness/templates/{go,cpp,python}/{func-json,class-ops}.tmpl`
   and the preludes (`xl_prelude.hpp`, `xl_prelude.py`, Go node types in the harness); `Generate(lang, harness, sig)` (m3-02's Go
   generator moves into the template shape only if its output bytes are unchanged); `Starter(lang, harness, sig)` (the learner-file
   skeleton). Golden files per language × harness × fixture signature; non-float frames byte-identical across languages **and equal
   to m3-02's Go bytes**. **No reflection** in generated Go.
4. **[X] Profiles** (plan task 3): `internal/runner/profile/{go,cpp,python}/profile.go` implementing m3-03's `Profile` with the plan's
   compile/exec commands, env, compile limits (Go 15 s/768 MiB/pids 256; C++ 15 s/1 GiB/pids 64; Python 15 s/256 MiB/pids 16), case
   limits (pids 32/4/4, `RLIMIT_STACK` = the memory limit for C++ and Python), diagnostics parsers (`go build -json`, GCC JSON,
   `SyntaxError`), detected toolchain versions, the `languages[]` → profile table, and the Go cache seed recipe (`profile/go/seed.go`:
   `go build std` with the profile's flags, fixed future mtimes, a fresh `trim.txt` per job) **plus the `runner seed-gocache`
   subcommand** in `cmd/runner/main.go` (`-out <dir>` required, `-goroot` defaulting to the profile's toolchain; prints the seed's
   tree hash; non-zero exit and no partial seed on failure; m3-15's Dockerfile calls it), with a `runner_it` test that a compile
   against the seeded overlay rebuilds no `std` package. `ArtifactMode` per profile: `0111` for Go/C++ `bin`, `0444` for
   `app.pyz`, one test per language. Toolchain paths overridable by `RUNNER_TOOLCHAINS_DIR` for tests.
5. **[X] Lint** (plan task 4): `internal/platform/runnerapi/lint` with `Check` (judge, pre-enqueue → REJECTED) and `CheckNames` (the
   front's re-check → 400); Go per t3 A5 + the import allowlist; C++ header/directive/`asm`/`main` rules; Python import allowlist.
   Wire `CheckNames` into the front. One golden fixture per rule, one evasion fixture per language.
6. **[X] Allowlists** (plan task 5): `seccomp_amd64.go` per profile, copied verbatim from t3 §16.2 (sorted, unique, a comment per
   justified addition); the dangerous set in one list, **including `splice`, `vmsplice` and `tee`** (t3 A1); the compile filters
   (ENOSYS default, KILL for the dangerous set — ENOSYS instead, recorded, only for a splice-family call spk-02 saw a toolchain make —
   fork/exec allowed); `seccomp_arm64.go` (`//go:build arm64`, dev-VM delta, documented "never in a release image", **KILL-default
   like amd64**; retire m3-03's dev-only LOG switch) with a guard test; invariant tests (names resolve on amd64, the dangerous set is
   in no exec list). Move `testgo@0`'s list reference to the real `go` list.
7. **[X] Items + tests** (plan task 6): the seven synthetic items under `internal/runner/testdata/items/` (reuse m3-02's fixture
   items where they fit; **never copy from `../xlearn-evalpack`**), references and `wrong/{wa,tle,re,ce,mle}` per language, plus a
   runtime-SIGSYS probe and the lint-evasion fixtures; `internal/runner/profile/refs_it_test.go` (`//go:build linux && runner_it`)
   asserting runner terms, outputs compared through **`internal/platform/checker`** (no second comparator), the test-only term→class mapper,
   diagnostics positions and the missing-function CE shape. Extend the `runner-it` CI job: `apt-get install -y g++ python3` from
   trixie inside the same container.
8. **[X] Multipliers + ProfileSHA** (plan task 7): compute provisional `tl_multiplier` from the CI perf cases (max ratio vs Go, up to
   the next 0.5, floor 1.0), `calibrated: false`; extend `ProfileSHA` to templates, preludes, lists and the seed (a one-byte template
   change must change it); `GET /v1/profiles` serves both.
9. **[X] Allowlist gaps:** if a C++/Python reference or wrong solution hits a SIGSYS P3 didn't log, add it only if safe, with a
   justification comment, and log it in the PR and the decisions log. Never add from the dangerous set; if the fix needs one, stop and
   report (that is an ADR-0030 question).
10. **[X] Docs:** "Profiles", "Harness protocol" and "Type registry" sections in `docs/architecture/runner.md` (link the harness
    README; don't restate it); the type table in `docs/v2/authoring.md`. Hand-off lines in `docs/v2/status.md`: m3-06, **m3-13** (the
    receiver for packlint's C++/Python gates and `lint.Check` over references, since m3-01/m3-02 ran earlier), m3-15.
11. **[X] Content CI** (plan task 8, m3-01's hand-off): `internal/platform/harness/contentcheck_it_test.go` (`linux && runner_it`) in
    the `runner-it` lane (path filter + `curriculum/**`): for every code part of every public item and each listed language, the
    starter (`_starter/`, else `harness.Starter`) compiles with the harness and doesn't pass the samples; every public
    `_code/solution.<ext>` passes the samples through the jail via `internal/platform/checker`. Public data only; vacuous pass if no code item.
12. **[X] Verify:** `gofmt`, `go vet ./...`, `go test -race ./...` (macOS and Linux), `sqlc diff` unchanged, `runner-it` green with
    all three languages. Don't mark a jail task ✅ from a macOS run.
13. **[X] PR** → conventional commits (`feat(runner): …`, `feat(harness): …`) with the attribution lines → CI green → squash-merge.

## Constraints

- **Service boundaries** ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)): `internal/platform/harness` and `runnerapi/lint`
  are stdlib-only shared leaf libraries (plus `internal/course` and `runnerapi` types; the import guard enforces it); the runner still
  has no DB, NATS, egress or secrets.
- **One definition** (m3-02's rule): one harness package, one checker registry; extend, never duplicate.
- **goose + sqlc:** no migration; `sqlc diff` unchanged. **Outbox/inbox, NATS ACL PRs, consumers-before-producers:** n/a (no events).
- **Contract discipline:** harness bytes on fd 3/fd 4 and the registry are public and versioned; any change to them is `@2`, never a
  patch. `runnerapi` stays append-only.
- **Security:** the lint is not the boundary; never loosen a jail or seccomp rule to make a reference pass; never allowlist the
  dangerous set; SIGSYS stays RE (counted).
- **Content hygiene:** every item, case and solution here is synthetic and public; never copy from `../xlearn-evalpack` (private).
- **GitOps / production:** no infra PR, no `kubectl`, no host change.
- **D34:** no alerting of any kind. **Memory-sum rule:** n/a (no pod). **theme.css:** n/a (no UI).
- **Parallel sessions:** check peers' PRs and worktrees before touching shared packages; no ADR expected (if one is, check peers'
  numbers first).

## Deliverables

- `internal/platform/harness`, extended (registry, canonical JSON, C++/Python codecs, generators, starters, templates, preludes, goldens
  incl. the m3-02 compatibility golden) and the content check.
- `internal/runner/profile/{go,cpp,python}` (profiles, diagnostics, seed recipe, artifact modes, amd64 + arm64-dev seccomp lists);
  `runner seed-gocache` in `cmd/runner`.
- `internal/platform/runnerapi/lint` (+ the front's `CheckNames` wiring).
- `internal/runner/testdata/items/` (7 synthetic items × 3 languages, references and wrong solutions) and `refs_it_test.go`.
- `runner-it` CI with g++ and python3; provisional multipliers and `ProfileSHA` in `/v1/profiles`.
- Docs sections in `runner.md` and `authoring.md`.

## Update status

- [`../sprints/sprint-m3-04.md`](../sprints/sprint-m3-04.md): each task ⬜ → 🔄 → ✅ (or ⛔ with a reason); _Overall_ ✅ at the end.
- [`../status.md`](../status.md): the Sprint board row (m3-04 ✅); M3 🔄; the MI-12 "runner code" part (profiles ✅); **Decisions log**:
  provisional multipliers (values, `calibrated: false`), allowlist additions with justifications, the Python lint location, the
  PCH/pyc hand-offs, any m3-02-vs-plan byte difference (a `@2` candidate); **hand-offs** to m3-06, m3-13 and m3-15.
- No flag, tag-floor or content-status change.

## Done when (acceptance)

- [ ] Reference solutions pass samples and synthetic hidden cases in all 3 languages through the real jail in CI.
- [ ] Wrong solutions are classified WA / TLE / RE / CE (plus MLE, REJECTED) in all 3 languages; runtime SIGSYS → RE.
- [ ] Both harnesses round-trip every registry type; non-float outputs are byte-identical across languages and equal to m3-02's Go
      bytes; one harness package; tests compare through `internal/platform/checker`.
- [ ] The content check passes (starters compile, public references pass samples); `runner seed-gocache` works; artifact modes tested.
- [ ] Exec allowlists = t3 §16.2 + justified additions; the dangerous set is in no exec list; no arm64 delta in amd64 builds.
- [ ] Lint goldens pass; evasion fixtures are stopped by the jail.
- [ ] `/v1/profiles` serves versions, `ProfileSHA`, baselines and provisional multipliers; CI green; docs and hand-offs recorded.

Ship per AGENT.md land-and-sync with **this sprint's release action: merge only** (the profiles ship in `runner-v1.0.0`, cut in
[m3-15](../sprints/sprint-m3-15.md)) — no tag, no infra PR; then `git checkout main && git pull`.
