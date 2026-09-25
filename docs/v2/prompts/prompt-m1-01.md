# Prompt — Sprint m1-01 · Curriculum spine part 1: compose parity, course manifest + golden, item schema freeze (M1a)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m1-01.md`](../sprints/sprint-m1-01.md)   ·   **Milestone:** M1 (M1a expand)   ·   **Prereqs:** [mi-01](../sprints/sprint-mi-01.md) (MI-2 merged)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync.
- The plan: [`../sprints/sprint-m1-01.md`](../sprints/sprint-m1-01.md) — the manifest block table (DSA values + `file:line`
  sources), the item field table and the six schema tests are spelled out there. Follow them.
- [`../rollout-plan.md`](../rollout-plan.md) [§3](../rollout-plan.md#3-milestone-map) (M1 entry gates), §4 M1 (scope T0/T1),
  [§6](../rollout-plan.md#6-critical-path-parallel-tracks-owner-calendar) (owner-hours chain: the freeze starts the content track).
- [ADR-0026](../../adr/0026-per-course-extensibility-model.md) §1 (manifest = data, compiled in), §4 (universal vs per course), §6 (M1).
- [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) §1 (no field can hold an answer), §2 (layout
  `curriculum/courses/<slug>/`), §8 (authoring, provenance).
- [ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md) §1 (registries, `composite@1`), §3 (D18 DSA defaults),
  §4 (mistake pre-fill via `mistakes.prefill`, concepts to revise).
- Research: [t0 §4](../research/t0-extensibility-frame.md#4-universal-vs-per-course-method), [t0 §6](../research/t0-extensibility-frame.md#6-six-course-fit)
  (dsa row: the golden values), [t0 §9](../research/t0-extensibility-frame.md#9-how-live-dsa-migrates);
  [t1 §3.2](../research/t1-content-data-model.md#32-public-content-layout-and-delivery), [§3.7](../research/t1-content-data-model.md#37-local-dev-and-public-ci)
  (compose parity), [§4](../research/t1-content-data-model.md#4-schema-deltas-vs-v1-conceptual), [§7.1](../research/t1-content-data-model.md#71-package-format)
  (item.json), [§8](../research/t1-content-data-model.md#8-content-rights-stance) (provenance);
  [t4 §6.2](../research/t4-judge-contract.md#62-strategies-a-closed-go-set-the-manifest-picks-one-and-sets-thresholds),
  [§6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only), [§6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course),
  [§6.7](../research/t4-judge-contract.md#67-mock-evidence--scoremock), [§11.4 #19](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag)
  (the content/manifest fields), §13 (D18 overrides the 15 + 10 golden rows for M3, not for M1);
  [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (the `coach` block).
- Code: `curriculum/{embed.go,paths.json,dsa/*.json}`, `internal/curriculum/seed.go`, `docker-compose.yml`, `deploy/local/`,
  `.github/workflows/ci.yml`, `go.mod`; v1 constants: `internal/practice/store/store.go:39-44`,
  `internal/review/store/store.go:27-43,712`, `internal/review/store/migrations/00002_mistakes_notifications.sql`,
  `internal/assessment/mock.go:30-43,147`, `internal/assessment/store/migrations/00001_init.sql`,
  `internal/coach/prompt.go:56-57`, `internal/gateway/dashboard.go:23`, `web/src/nav.ts`,
  `web/src/screens/{Problem.tsx:18-26,Revision.tsx:14}`.

## Context

M1 makes DSA "a course like any other" with **no behaviour change** (M1a expand `v1.6.0` → M1b `v1.7.0` → M1c contract
`v1.8.0`). This sprint is the first M1a slice and the **content-authoring unblock**: the owner's content track (~10 h/week,
14 pilot packs before M3) cannot start until the item schema is frozen, and every later sprint that touches courses
reads the manifest types defined here. It adds a new stdlib-only `internal/course` package, the DSA manifest (golden =
v1: 15:00 attempt / 10:00 hint / 3-day penalty, Day 1·3·7·21·45, 4 grades, 8 categories, 7×1–5 = /35, 45-minute rail,
W13 24 / W15 28 / Pre 30), and the frozen item schema carrying every field ADR-0029 needs later, so authored items never
need re-stamping. The owner's D18 timing (45:00 / hint at 15) is recorded as **dormant** `verdict_timer@1` params — it
becomes operative only in M3. Nothing reads the manifest at runtime yet. Compose moves to production parity (PG 18,
NATS 2.14), which mi-05's NATS-auth test needs. v2 is owner-only (D35); production has 1 account and 19 events.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] MI-2 merged: `kustomize.toolkit.fluxcd.io/prune: disabled` on the CNPG Cluster, `databases` and `messaging`
      (check `../infra` `main`, read-only; [mi-01](../sprints/sprint-mi-01.md) task 1).
- [ ] MI-2a live ✅ (`.release-line` = `1` in this repo; the xlearn ImagePolicies in `../infra/apps/image-automation.yaml`
      carry `range: ">=1.0.0 <2.0.0"` — the `apps/xlearn-*.yaml` HelmReleases hold only the image tag).
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR creates `internal/course/`
      or conflicts on `docker-compose.yml` / `ci.yml` / `go.mod` (else agree an order).

## Do this (in order)

1. **[X] Branch** `feat/m1-course-manifest-schema` off an up-to-date `origin/main`.
2. **[X] Compose + CI parity** — `docker-compose.yml`: `postgres:18-alpine` with the volume at `/var/lib/postgresql`
   (PG 18 images reject `/var/lib/postgresql/data`), `nats:2.14-alpine` (same `-js -sd /data -m 8222`); add the
   "PG major bump → `docker compose down -v`" note to `deploy/local/README.md`; CI e2e service → `postgres:18`; bump the
   embedded `github.com/nats-io/nats-server/v2` to the latest v2.14.x and `go mod tidy`. Bring compose up, run every
   `internal/*/store/*_integration_test.go` against it with `XLEARN_TEST_DATABASE_URL`, fix PG 18 differences.
   Recommended: add the store integration tests to the e2e job.
3. **[X] `internal/course`** — `manifest.go`, `load.go` (`Load(fs.FS)`: glob `courses/*/course.json`, strict decode with
   `DisallowUnknownFields` + trailing-data check, then `Validate`), `validate.go` (the plan's rules, including the
   `paths.json` consistency check), `universal.go` (D3 constants, closed nav screens, closed strategy ids). Stdlib only.
4. **[X] Manifests** — `curriculum/courses/dsa/course.json` with exactly the plan's DSA values (dormant blocks included:
   `grading.params["verdict_timer@1"]` = D18, `plan.est_minutes` keyed `course_attempt` / `touch_<band format>` / `mock` —
   the only place minutes live; bands carry no `est_minutes`), and `coming_soon` stubs for `system-design`,
   `go-concurrency`, `lld-ood`, `sql`, `behavioral` (prefixes `sd`, `gc`, `lld`, `sql`, `beh`; behavioral
   `default_visible: false`). Extend `curriculum/embed.go` **additively**:
   `//go:embed paths.json dsa/*.json courses/*/course.json _schema/*.json` — the v1 loader keeps reading `dsa/*.json`.
5. **[X] Two-sided golden** — `internal/course/golden_test.go` (literal v1 table with `file:line` comments) plus mirror
   tests inside `internal/practice/store`, `internal/review/store`, `internal/assessment` and `internal/assessment/store`
   that compare their own constants and parsed migration CHECK lists to the loaded manifest. Break one value on purpose,
   see both sides fail, revert.
6. **[X] Item schema freeze** — `curriculum/_schema/item.schema.json` + `course.schema.json` (2020-12,
   `additionalProperties: false` everywhere), `internal/course/item.go` + `Validate`, fixtures under
   `internal/course/testdata/`, and `schema_test.go` with the plan's six tests: types⇄schema parity, answer-free denylist
   over the item **and** manifest schemas and Go types (with the plan's five allowlisted paths — three item, two manifest:
   `stages.solution`, `revision.bands[].criteria[].key` — matched by full JSON path), ADR-0029 / t4 §11.4 #19 field presence, the freeze guard against
   `testdata/item.schema.v1.frozen.json`, pass/fail fixtures (the code fixture is an **original** example — never copy a
   LeetCode statement or example), and fixture validation with a **test-only** pure-Go JSON Schema validator.
7. **[X] Verify** — `gofmt -l .` empty, `go vet ./...`, `go test -race ./...`, `sqlc diff`, web typecheck/lint/test/build
   (then `git checkout -- web/dist/.gitkeep`), e2e on PG 18, compose click-through (login, Today, a problem, revision,
   mock, coach). Confirm no handler, query or route changed.
8. **[X] Update status** (below), then commit (conventional, e.g. `feat(course): course manifest + golden, item schema v1
   frozen, compose PG 18 / NATS 2.14`) with the attribution lines, push, open the PR, and ship it (see Ship).

## Constraints

- **No behaviour change:** no handler, query, route or event changes; the manifest is compiled in but unread at runtime.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** `internal/course` is a shared,
  stdlib-only library of types and constants (ADR-0026: compiled into every image) — no DB access, no service imports.
  Mirror tests live in each owning package; production code gains no new cross-service import.
- **goose + sqlc:** no migration in this sprint; `sqlc diff` must stay clean.
- **Answer secrecy (ADR-0027 §1):** the schema has no field that can hold an answer; fixtures use original content only.
- **Freeze discipline:** after merge, item-schema changes are additive and optional only (the freeze guard enforces it).
- **Layout:** `curriculum/courses/<slug>/…` per ADR-0027 §2; do not move `curriculum/dsa/*` (m1-09 does).
- **Dependencies:** the JSON Schema validator is test-only; note it in the PR and the Decisions log.
- **GitOps / infra:** no `../infra` change (read-only here); never `kubectl apply`. D34: no alerting work. No new pod, so
  the memory-sum rule is untouched. Consumers-before-producers and ACL-before-consumer do not apply (no events).
- **Parallel sessions:** check peers' PRs, tags and worktrees before merging (`gh pr list`, `git ls-remote --tags origin`,
  `git worktree list`, ListAgents); rebase onto a moved `main`. No ADR number is claimed here.

## Deliverables

- `docker-compose.yml`, `deploy/local/README.md`, `.github/workflows/ci.yml`, `go.mod`/`go.sum` (PG 18, NATS 2.14 parity).
- `internal/course/{manifest.go,load.go,validate.go,universal.go,item.go,golden_test.go,schema_test.go,testdata/…}`.
- `curriculum/courses/{dsa,system-design,go-concurrency,lld-ood,sql,behavioral}/course.json`; `curriculum/_schema/{item,course}.schema.json`;
  `curriculum/embed.go` (additive embed).
- Mirror golden tests in `internal/{practice,review}/store`, `internal/assessment`, `internal/assessment/store`.

## Update status

- [`../sprints/sprint-m1-01.md`](../sprints/sprint-m1-01.md): each task 🔄 → ✅; _Overall_ ✅ once all are.
- [`../status.md`](../status.md): the Sprint board row; the **M1** milestone row (🔄); the **content status** table
  ("item schema v1 frozen — <date>, <commit>"); the **owner events** table (`ev-schema-freeze` ✅ — the owner content track
  starts); the M1 size note ("10 build/release + 1 design sprint; m1-01 and m1-07 were split"); **Decisions log** lines for
  the `curriculum/courses/<slug>/` layout, the dormant D18 params, the coming-soon prefixes and the test-only validator.
- No ADR: this implements ADR-0026/0027/0029 as decided. If you must change a decided shape, amend the ADR instead.

## Done when (acceptance)

- [ ] DSA manifest golden test green (the `internal/course` table **and** the service-side mirror tests).
- [ ] The schema test proves every ADR-0029 / t4 §11.4 #19 content field exists and none can hold an answer.
- [ ] Freeze guard in place (frozen snapshot + additive-only test).
- [ ] Compose on `postgres:18` + `nats:2.14`; CI e2e on PG 18; every store integration test green on PG 18.
- [ ] No API behaviour change.
- [ ] `docs/v2/status.md` records the freeze → `ev-schema-freeze`.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/m1-course-manifest-schema`, then conventional commit(s) with the attribution lines, then push, then the PR. This repo only: no `../infra` PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only:** nothing deploys (`main` is build-only; nothing reads the manifest at runtime yet). It ships dark in **`v1.6.0`**, which [m1-02](../sprints/sprint-m1-02.md) tags. No tag here.
4. Update status: the sprint file and `docs/v2/status.md` (with `ev-schema-freeze` ✅), in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
