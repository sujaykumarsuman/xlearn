# Prompt — Sprint m3-01 · Authoring tooling T25/T26: canonical hashes, contract_hash, packlint, pre-push fingerprint hook

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-01.md`](../sprints/sprint-m3-01.md)   ·   **Milestone:** M3 (content track)   ·   **Prereqs:** [m1-01](../sprints/sprint-m1-01.md) (schema frozen), [m1-09](../sprints/sprint-m1-09.md) (the whole sprint builds on it)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync.
- The plan: [`../sprints/sprint-m3-01.md`](../sprints/sprint-m3-01.md) — the canon@1 rules, the contract/content field table, the
  nine `packlint check` rules, the hook design and the migration are spelled out there. Follow them.
- [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) (Accepted) §1 (public/private rule), §2 (hashes, `contract_hash`,
  `accepts_contract_hashes[≤2]`), §5 (leak controls), §8 (authoring and rights).
- Research: [t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack) (pack layout),
  [§3.4](../research/t1-content-data-model.md#34-versioning-and-pinning) (hash table, mismatch rules 1–5),
  [§7.1](../research/t1-content-data-model.md#71-package-format), [§7.2](../research/t1-content-data-model.md#72-ci-validation)
  (public CI, the pre-push hook), [§7.3](../research/t1-content-data-model.md#73-where-ai-may-help),
  [§8](../research/t1-content-data-model.md#8-content-rights-stance), [§10](../research/t1-content-data-model.md#10-security-and-threat-notes)
  (leak runbook); [t4 §5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start) (CI lints, #9 contract scope).
- [`../rollout-plan.md`](../rollout-plan.md) [§5](../rollout-plan.md#5-m3-hard-entry-checklist) (the T25/T26 line),
  [§6](../rollout-plan.md#6-critical-path-parallel-tracks-owner-calendar) (the owner-hours chain starts at the schema freeze).
- Neighbours: [m1-01](../sprints/sprint-m1-01.md) (item/manifest types, fixtures, freeze guard), [m1-09](../sprints/sprint-m1-09.md)
  (on `main`: the loader and its resolved item type, `canon.ContentHash`, migrations `00002`/`00003`, `cmd/contentlint`, the
  deferred content gates), [m3-02](../sprints/sprint-m3-02.md) (extends `packspec` and packlint next).
- Code: `internal/course/` (m1-01; `canon/` and the resolved item type from m1-09), `internal/curriculum/{load.go,seed.go,handlers.go}`,
  `internal/curriculum/store/{migrations,queries}/`, `cmd/contentlint/`, `.github/workflows/ci.yml`, `Makefile`, `.gitignore`, `.dockerignore`, `AGENT.md`, `curriculum/README.md`,
  `internal/gateway/aggregate.go` (problem passthrough / m1-06's DTO).

## Context

The owner's content track (~10 h/week) starts at the M1a schema freeze (`ev-schema-freeze`, ≈ 2026-10-05) and has to produce 14
stamped pilot packs before the M3 UI sprint — the longest owner-hours chain to GA. Authoring those packs safely needs three
things that don't exist yet: **one** hash definition that the seed, practice, judge and packlint share (so a pack written against
an item keeps matching until the item's *contract* changes), a linter that sees the public item and the private pack together,
and a **pre-push hook** — the only check that sees both repos before anything becomes public (a slip into the public repo is
permanent). This sprint builds them, plus the public content gates m1-09 deferred (stamp gate, label-edit flag, structure lints),
the repo guards, an authoring guide and the `problem.contract_hash` column. v2 is owner-only (D35); no infra, no image change.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] m1-01 merged: `internal/course` types, `curriculum/_schema/*.schema.json`, the freeze guard; `ev-schema-freeze` recorded in `docs/v2/status.md`.
- [ ] m1-09 merged — a gate for the **whole** sprint: its resolved item type and `canon.ContentHash`, `cmd/contentlint` + the
      `content` CI job, `curriculum/README.md`, and curriculum `00002`/`00003` with `problem.content_hash` are on `main`.
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR creates `internal/course/canon` (beyond
      m1-09's `ContentHash`), `internal/packspec`, `cmd/packlint`, `hack/git-hooks/` or a curriculum migration.
- [ ] Check (not a stop): `internal/course/canon` normally exists (m1-09) — extend it. If m1-09 kept a local hash helper instead,
      create the package and move the helper in (same bytes).

## Do this (in order)

1. **[X] Branch** `feat/m3-authoring-tooling` off an up-to-date `origin/main`.
2. **[X] `internal/course/canon`** — extend m1-09's package: `Bytes`, `PolicyVersion`, `ContractHash`, `Prefix` beside its `ContentHash`
   (over m1-09's resolved item type — never a second one), per the plan's canon@1 table (typed re-marshal, `omitempty` views,
   `UseNumber` + the number normalization for open values, sorted sets, domain-separated `sha256("xlearn.<kind>@1\n"+…)`).
   Add `classify.go` (every `course.Item` JSON path → contract or content-only) and the tests: classification (reflection), golden
   vectors over m1-01's original fixtures + a class-mode fixture + the DSA manifest (`-update` flag), the mutation table (incl. the
   non-edits: reformatting, `1e-6` ↔ `0.000001`), round trip, self-path → `""`. Keep `ContentHash`'s bytes unless they break a rule
   (then update m1-09's golden). Generator `@v` stays out of `contract_hash` and `constraints[]` is content-only (decided here; the
   owner may revisit it later) — both per the plan, both logged.
3. **[X] `internal/packspec` + `cmd/packlint`** — source-format types (root and item `pack.json`, strict, allowed subdirs);
   `packlint check` with the plan's nine rules and exit codes, `packlint hash`; tests against a temp **synthetic** pack.
   Never name a package or directory `evalpack`.
4. **[X] Content gates** — `internal/course/lint` + `cmd/contentlint`: the stamp gate (added/changed hint and editorial files need
   the stamp; v1 files grandfathered; `--report-unstamped`), the label-edit flag (PR body token `label-edit-ok: <item>/<part>/<id>`),
   and t4 §5.6 lints #1–#8, #12–#14 (public-registry half). Wire them into the `content` job in `.github/workflows/ci.yml`.
5. **[X] Hook** — `packlint fingerprint` (`--pre-push` / `--diff` / `--tree`, corpus from `tests/edge.jsonl`, keys, anchors/exemplar
   shingles and any `build/` cases; min-len 24; hits print file:line and the pack-relative path, **never the payload**);
   `hack/git-hooks/pre-push` (no pack dir → exit 0); `Makefile` targets `install-hooks`, `uninstall-hooks`, `packlint` (m1-09 owns `contentlint`);
   the plan's five fingerprint tests.
6. **[X] Hygiene** — `.gitignore` (`evalpack/`, `xlearn-evalpack/`), `.dockerignore` (`**/evalpack`, `**/xlearn-evalpack`), the
   AGENT.md never-copy rule (plan task 5 wording), leak-lint extensions (denylist names; the repo-wide pack-artefact pass over
   `git ls-files`, excepting only `internal/**/testdata/` trees marked by a `SYNTHETIC.md`, with a test for both sides; embed allowlist
   over every `//go:embed` package).
7. **[X] Guide** — `docs/v2/authoring.md` (the plan's eight sections, incl. the reference file names `_code/solution.{go,cpp,py}`)
   and a pointer from m1-09's `curriculum/README.md`.
8. **[X] curriculum columns (task 7)** — migration `00004_contract_hash.sql` (or the next free number after m1-09's; rebase first)
   with `contract_hash text NOT NULL DEFAULT ''` and `grading_summary jsonb NOT NULL DEFAULT '{}'`; seed both via `canon`
   (and `content_hash` via `canon`); `UpsertProblem`/`GetProblem`/`GetProblemsByIDs`; `sqlc generate`; `GET /problems/{id}` adds
   `contract_hash_prefix` and `grading_summary` (and the gateway DTO allowlist if m1-06 made one); store + handler tests.
9. **[X] Verify** — `gofmt -l .` empty, `go vet ./...`, `go test -race ./...`, `sqlc diff`, web typecheck/lint/test/build (then
   `git checkout -- web/dist/.gitkeep`), e2e on PG 18, `make contentlint`. Manual hook check with a scratch pack dir
   (`XLEARN_EVALPACK_DIR`), a planted **synthetic** string and a **local bare remote** — blocked, payload not printed; never push
   the plant to `origin`.
10. **[X] Update status** (below), including the post-ship owner event `ev-hook-install` (the owner installs the hook after the
    merge; it doesn't gate this sprint), then **Ship:** see **Ship** below (commit e.g. `feat(content): canonical hashes,
    contract_hash, packlint, pre-push fingerprint hook`).

## Constraints

- **Answer secrecy (ADR-0027 §1, §5):** never read, copy or print real pack data from `../xlearn-evalpack` into this repo, tests, logs,
  commit messages or the PR — every fixture here is synthetic and original. packlint output carries ids, paths and hashes only.
- **One definition:** `canon` is the only hash implementation and m1-09's resolved item type the only resolved type; you extend both.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** `canon`, `course/lint` and `packspec` are
  stdlib-only libraries (no DB, no service imports); the migration touches schema `curriculum` only.
- **goose + sqlc:** additive expand migration (constant defaults, ADR-0034 §3), next free version at rebase (never `00002`/`00003`);
  `sqlc generate` committed, `sqlc diff` clean.
- **No behaviour change** beyond two additive, answer-free fields on `GET /problems/{id}`; no event, route or UI change.
- **Dev tools only:** packlint, contentlint and the hook never enter a Dockerfile or service binary.
- **GitOps / infra:** no `../infra` change; never `kubectl apply`. D34: no alerting of any kind (the hook is a local check, not an alert).
  No new pod, so the memory-sum rule is untouched; no events, so outbox/inbox, consumers-before-producers and ACL-before-consumer
  do not apply.
- **Parallel sessions:** check peers' PRs, tags and worktrees (`gh pr list`, `git ls-remote --tags origin`, `git worktree list`,
  ListAgents) before merging; rebase onto a moved `main`. No ADR number is claimed here.

## Deliverables

- `internal/course/canon/{canon.go,contract.go,classify.go,*_test.go,testdata/…}`; `internal/course/lint/…`.
- `internal/packspec/source.go` (+ tests); `cmd/packlint/{main.go,check.go,hash.go,fingerprint.go,*_test.go,testdata/…}`.
- `cmd/contentlint` extensions + the `content` CI job wiring.
- `hack/git-hooks/pre-push`; `Makefile` targets; `.gitignore`; `.dockerignore`; `AGENT.md` rule.
- `docs/v2/authoring.md`; pointer in `curriculum/README.md`.
- `internal/curriculum/store/migrations/00004_contract_hash.sql`, queries + sqlc output, seed and handler changes (task 7).

## Update status

- [`../sprints/sprint-m3-01.md`](../sprints/sprint-m3-01.md): each task 🔄 → ✅; _Overall_ ✅ when all are (the hook install is a
  post-ship owner event, not a task).
- [`../status.md`](../status.md): the Sprint board row; the **M3** milestone row (🔄, content track started) and its **T25/T26**
  checklist line (✅ on merge); **content status** (tooling live; grandfathered unstamped hint/editorial items: N);
  **owner events** (`ev-packs-14` can use the tooling; add **`ev-hook-install`**: the owner installs the hook on the authoring
  machine after the merge, ~5 min, per the plan's task 8); **Decisions log**: canon@1, number normalization and domain separation;
  the field classification (limits content-only with advisory; **`constraints[]` content-only — decided here, flagged for the owner
  to revisit**); generator `@v` outside `contract_hash` (t4 §5.6 #9 deviation); the `packspec` name; migration `00004`; stamp-gate
  grandfathering; the `label-edit-ok:` token; the `SYNTHETIC.md` marker rule; the reference file names.
- No ADR (ADR-0027 is Accepted and this implements it). A change to a decided shape needs a new ADR — check peers for the number first.

## Done when (acceptance)

- [ ] Hash golden vectors stable; the classification test covers every `course.Item` field.
- [ ] A prompt edit changes `content_hash` only; a signature, harness, checker-param or option-id edit changes `contract_hash`;
      reformatting or respelling a number moves neither.
- [ ] `packlint check` flags a stale contract hash (exit 1, naming both hashes).
- [ ] The hook blocks a planted hidden-case string without printing it; no pack dir → no-op.
- [ ] Stamp gate, label-edit flag and t4 §5.6 structure lints run in the public `content` job.
- [ ] `.gitignore`/`.dockerignore` entries, the AGENT.md rule and `docs/v2/authoring.md` are in place; the repo-wide pack-artefact
      pass honours only `SYNTHETIC.md`-marked `internal/**/testdata/` trees.
- [ ] `problem.contract_hash` + `grading_summary` seeded (migration `00004`); `GET /problems/{id}` shows the prefix; `sqlc diff` clean; every v1 e2e green.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: xlearn only, on `feat/m3-authoring-tooling`.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships dark in the next app tag):** nothing deploys; it ships in `v1.6.0` (cut by [m1-02](../sprints/sprint-m1-02.md)) if merged before that tag, else in `v1.7.0` (cut by [m1-07](../sprints/sprint-m1-07.md)). Don't tag. No infra PR, no evalpack change; packlint, contentlint and the hook never enter an image.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way (including the post-ship owner event `ev-hook-install`).
5. Run `git checkout main && git pull` in every repo touched (xlearn). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
