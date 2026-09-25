# Prompt — Sprint m1-08 · M1c contract → v1.8.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m1-08.md`](../sprints/sprint-m1-08.md) · **Milestone:** M1 (M1c contract — closes M1) · **Prereqs:** [m1-07](../sprints/sprint-m1-07.md) (`v1.7.0` live), [mi-02](../sprints/sprint-mi-02.md) (MI-8 `host-verify --cluster`)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] The October host window (Sat 2026-10-24) has settled: ≥ 24 h since its last k3s / PG restart, with a green `host-verify --cluster` after it (status.md). Otherwise don't launch yet.
- [ ] The last Hostinger weekly image is ≤ 7 days old (hPanel).
- [ ] **`ev-snap-v1.8.0`:** take a Hostinger manual snapshot in hPanel (one at a time, 1-day retention; it replaces the window's pre-change snapshot) right before you launch, and put its name/time and the weekly image's date in your launch message. The contract tag lands the same day.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync, status updates.
- The plan: [`../sprints/sprint-m1-08.md`](../sprints/sprint-m1-08.md) — the per-service drop table, rehearsal phases, release checklist.
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) — [§3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (contract ≥ 1 release after the last reader/writer; floor marker; rehearse; snapshot), [§4.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#41-mechanisms-fastest-first) / [§4.2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#42-r-d-is-a-procedure-not-a-button) / [§4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule) (rollback, R-d procedure, snapshot rule), §4.4 ("M1c contract: no, below 1.7.0"), §6 (release checklist).
- [`../research/t1-content-data-model.md`](../research/t1-content-data-model.md) §4 — per-service deltas and the *Expand → backfill → contract* table (the M1c list).
- [`../rollout-plan.md`](../rollout-plan.md) — [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (snapshots, "verify the host", no hand-applied changes), [§3](../rollout-plan.md#3-milestone-map) (M1 exit), [§7](../rollout-plan.md#7-indicative-tag-timeline) (`v1.8.0` row).
- [ADR-0035 §3](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34) (the MI-8 `host-verify --cluster` checks; D34 — no alerting).
- [ADR-0005](../../adr/0005-data-ownership-and-migrations.md) (goose on startup under the advisory lock; schema per service), [`../../runbooks/projection-rebuild.md`](../../runbooks/projection-rebuild.md).
- What M1 built: [m1-02](../sprints/sprint-m1-02.md) (expand, `mock_session_scored_total_check`, `weak_area_snapshot_account_path_week_uq`, `hack/lint-migrations.sh` + `xlearn:contract` / `xlearn:relax` markers, `internal/e2e/testdata/v1-events.jsonl` replay test), [m1-09](../sprints/sprint-m1-09.md) (curriculum expand), [m1-03](../sprints/sprint-m1-03.md) (writers stopped; `hack/lint-dropped-columns.sh`, which you extend) / [m1-10](../sprints/sprint-m1-10.md) (`is_default` gone; `internal/coach/contract_test.go`), [m1-07](../sprints/sprint-m1-07.md) task 5 (the M1c-readiness record) — and the decisions log in [`../status.md`](../status.md).
- Code: `internal/{curriculum,review,assessment,coach,practice,identity}/store/{migrations,queries,gen}/`, `internal/curriculum/` loader, `internal/review/{handlers.go,mistakes.go,consumers.go}`, `internal/assessment/{handlers.go,mock.go,store/store.go}`, `docker-compose.yml`, `deploy/local/`, `internal/e2e/`, `sqlc.yaml`, `Makefile`; infra `../infra/hack/host-verify.sh` (read-only).

## Context

M1a (`v1.6.0`) expanded every schema, and M1b (`v1.7.0`) moved every reader and writer to the new shape. M1c now
**drops** what nobody uses any more:
- curriculum: `problem.is_reinforcement`, `leetcode_url`, `neetcode_url`; `concept.code_template`; `UNIQUE(concept.slug)`;
  the v1 `problem_section UNIQUE (problem_id, stage, "order")` (replaced by m1-09's `(problem_id, stage, "order", language)`;
  conditional, like the weak-area unique);
- review: the `mistake_entry.category` CHECK; the v1 `weak_area_snapshot UNIQUE(account_id, week_of)` (conditional);
- assessment: `mock_session.total_35` (re-homing the scored/total invariant onto `total`) and the `rubric_score.dimension` CHECK;
- coach: `api_key_config.is_default`;
- all services: the `'dsa'` defaults.

The Go side, validating against the course manifest and the rubric snapshot, replaces each CHECK.

A contract is the one M1 step that can't be undone below its floor. After `v1.8.0` the rollback floor is **1.7.0,
hard**, and only a Hostinger snapshot (R-d, ~1 day) can go lower. So this sprint:
- proves in compose that the `v1.7.0` images still run on the contracted schema;
- runs `host-verify --cluster` on a settled host;
- relies on the owner's manual snapshot, taken right before launch (D40), and tags the same day.

It also closes M1: golden = v1, every v1 e2e green, events replay.

**Tag names are indicative** ([ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline): the next free minor at tag time).
`v1.6.0` / `v1.7.0` / `v1.8.0` here mean the M1a tag, the M1b tag and this tag. Read the actual M1a and M1b tags from
[`../status.md`](../status.md) and substitute them everywhere: the `floor=` marker, `--ref`, the rehearsal image
tags, the R-b target and the floor record.

Calendar: week 5, **after** the Sat 2026-10-24 host window has settled (≥ 24 h, `host-verify` green). Target
Mon 10-26 → Wed 10-28. The owner's part is ~5 minutes before launch (`ev-snap-v1.8.0`).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] `v1.7.0` live and verified (m1-07 ✅; `curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz`), and m1-07's M1c-readiness record says `v1.7.0` neither reads nor writes any drop-list column (re-proved in step 3).
- [ ] mi-02 merged: `../infra/hack/host-verify.sh` has the MI-8 `--cluster` checks (`git -C ../infra log --oneline -5 -- hack/host-verify.sh`).
- [ ] Host settled: the Oct 24 window (mi-09) is done and ≥ 24 h old with a green `host-verify --cluster` after it (status.md) — or it is rebooked to ≥ 2 days after this tag.
- [ ] The launch message carries the `ev-snap-v1.8.0` snapshot's name/time and the weekly image's date (≤ 7 d); the snapshot was taken before launch (D40).
- [ ] Parallel sessions: `gh pr list --state open`, `git ls-remote --tags origin`, `git worktree list`, ListAgents — no peer adds a migration to these five services, and no peer plans to tag from `main` during the merge → snapshot → tag window (tell them).

## Do this (in order)

1. **[X] Branch** `feat/m1-08-m1c-contract` from an up-to-date `main`.
2. **[X] Confirm the real names.** `docker compose up -d postgres` plus the services at HEAD (or use m1-07's stack),
   then `\d+` each affected table (`curriculum.problem`, `curriculum.concept`, `review.mistake_entry`,
   `curriculum.problem_section`, `review.weak_area_snapshot`, `assessment.mock_session`, `assessment.rubric_score`,
   `coach.api_key_config`, the `path_slug` columns) and write down the constraint names and which columns have
   `DEFAULT 'dsa'`.
3. **[X] Assertion (task 2).** Extend m1-03's `hack/lint-dropped-columns.sh` (don't fork it). Add a `--ref <git-ref>`
   mode (scratch worktree of the ref), plus the surfaces it misses: `internal/*/store/gen/*.go`, raw SQL in
   `internal/*/store/*.go`, the curriculum loader. It checks `is_reinforcement`, `leetcode_url`, `neetcode_url`,
   `code_template`, `total_35` and `is_default` (off the allowlist since m1-10), plus the v1 conflict targets
   `ON CONFLICT (slug)` on `curriculum.concept`, `ON CONFLICT (account_id, week_of)` on `review.weak_area_snapshot`
   and `ON CONFLICT (problem_id, stage, "order")` on `curriculum.problem_section` (the 4-column target with
   `language` must not match). The allowlist (hit → reason) covers migrations, event-payload decoders and JSON API
   names. Run it with `--ref v1.7.0` (the M1b tag) and on HEAD. Any `v1.7.0` hit → drop that item from this contract
   and log it with its deadline: the `weak_area_snapshot` unique before p-02; the `problem_section` unique before M3
   seeds a second language at the same stage and order (m3-01's Go/C++/Python packs).
4. **[X] Contract migrations (task 1).** One file per service — `internal/<svc>/store/migrations/0000N_m1c_contract.sql`
   (next free version at rebase) for curriculum, review, assessment, coach and practice — with
   `-- xlearn:contract floor=<the M1b tag, e.g. v1.7.0>` on the first Up line, exactly the drops in the plan's table
   (minus any conditional unique step 3 kept). Assessment adds
   `mock_session_scored_total_check CHECK ((status = 'scored') = (total IS NOT NULL)) NOT VALID` then
   `VALIDATE CONSTRAINT` in the same file (dropping `total_35` also drops m1-02's COALESCE check). Down re-adds the
   columns nullable and CHECKs `NOT VALID` (dev only; never run in prod). No identity file (confirm there is no
   `'dsa'` default there). `sqlc generate`; fix any compile errors in HEAD; `sqlc diff` clean; `hack/lint-migrations.sh` green.
5. **[X] Validation proof.** Confirm the Go-side replacements exist: review `category` ∈ the course manifest's
   categories (`internal/course`) on `POST`/`PATCH /mistakes` and in the consumer's pre-fill; assessment
   `ScoreMock` dimensions ∈ the session's `rubric_snapshot`. Add `internal/<svc>/store/contract_m1c_integration_test.go`
   (env-gated on `XLEARN_TEST_DATABASE_URL`): fresh schema → HEAD; `information_schema` / `pg_constraint` show each
   dropped object gone and the re-homed CHECK `convalidated`; where validation lives in the store (review's
   `store.ErrInvalidCategory`), a bad value is still refused. The **422** assertions go in the handler packages,
   not in `store` (`internal/review` imports its `store`, so a `package store` test importing the handler is an
   import cycle): `internal/review/mistakes_handlers_test.go` (unknown category on `POST`/`PATCH /mistakes`) and
   `internal/assessment/handlers_test.go` (unknown rubric dimension on score), on the existing fakes.
6. **[X] Rehearsal tooling (task 3).** `deploy/local/compose.rehearse.yml` (per-service
   `image: ghcr.io/sujaykumarsuman/xlearn-<svc>:${XLEARN_IMAGE_TAG}` with `platform: linux/amd64` — the images are
   amd64-only and this Mac runs them emulated; HEAD uses `build:`). **GHCR tags have no leading `v`** (`deploy.yml`
   strips it; live is `xlearn-identity:1.5.2`): `XLEARN_IMAGE_TAG=1.5.2` / `1.6.0` / `1.7.0`. Then `hack/rehearse-contract.sh`
   (project `xlearn-rehearse`, own volumes) and a `rehearse-contract` target in the `Makefile`. Write
   `internal/e2e/rehearse_test.go` (`//go:build rehearse`, `XLEARN_REHEARSE_BASE_URL`): dev login; enrol DSA;
   attempt → hint → outcome (clean + below-clean); revisions due + score; mistakes create + patch; mock start +
   score; progress, dashboard, `/u/<username>`; coach key PUT / set-default / DELETE and thread GET — v1 routes
   only (the `v1.5.2` surface), so it runs against `v1.5.2`, `v1.6.0`, `v1.7.0` and HEAD. If a pull still fails,
   build the tag from a scratch worktree of it.
7. **[X] Rehearse** — the plan's phases, in order, starting from fresh volumes and keeping them throughout (the
   production lineage):
   **−1** `v1.5.2` + driver (incl. a scored mock and a default coach key) → **0** `v1.6.0` on those volumes: the
   expand and its backfills apply (check `total` from `total_35`, `key_default` from `is_default`, the curriculum
   backfills), driver → **1** `v1.7.0` + driver, drain, dump schema + `assessment.proj_*` (A); on new rows the old
   columns are **not written by `v1.7.0`** (`NULL` or the column's remaining default: `is_default` false, the URLs
   `''`, `is_reinforcement` false, `code_template` `''`) → **2** HEAD: check `migration applied … m1c_contract` in every
   service's log and that `VALIDATE CONSTRAINT` passed on the backfilled mock, dump `proj_*` (B) before any write,
   A = B, driver → **3 (R-b)** `v1.7.0` on the contracted schema: goose no-ops, all Ready, driver, restart each
   service once, driver → **4** HEAD again + driver. Also at HEAD:
   `XLEARN_TEST_DATABASE_URL=… go test -tags e2e ./internal/e2e/...` (without the env var these tests silently skip,
   per `internal/e2e/coreloop_test.go`) and m1-02's 19-event replay test (v1 and v2 encodings) against a contracted
   fresh schema. Keep every output for the PR.
8. **[X] Verify + PR.** `gofmt -l`, `go vet ./...`, `go test -race ./...`, `sqlc diff`, the migration lint,
   web tests (unchanged). Open the PR (e.g. `feat(db): M1c contract — drop v1 columns and CHECKs (m1-08)`, with
   the attribution lines) with the drop table, the assertion output for `v1.7.0` and HEAD, the rehearsal phase
   table and logs, and the replay result. CI green. **Merge it in step 10**, after `host-verify`, right before the tag.
9. **[H] `host-verify --cluster` (task 4).** With `../infra` on an up-to-date `main`, from the xlearn root (agent
   shells reset there, and xlearn has no `hack/host-verify.sh`):
   `ssh sujaykumar-vps 'bash -s -- --cluster --expect-sandbox --json --nats-stage=<live stage from status.md: n3, or n4 after mi-11>' < ../infra/hack/host-verify.sh`
   → no FAIL (a `legacy` NATS connection is a FAIL at n3); note WARNs; confirm ≥ 24 h since the window's last
   restart. Drop `--expect-sandbox` only if the window was rebooked past this tag. Read-only only.
10. **[X] Merge (the [O] snapshot, task 5, was taken before launch).** Record the snapshot name/time and the weekly date
    from the launch message. Re-check peers (the entry-gate commands). Squash-merge the PR (1.x deploys only on a tag) and
    go straight to the tag. If the tag can't land the same day as the snapshot, don't merge: record ⛔ "snapshot lapsed;
    re-take it and relaunch".
11. **[X] Tag `v1.8.0` (task 6)** right after the merge, with the plan's release checklist (verbatim; the contract lines
    are tasks 3–5). Re-check peers and `.release-line` just before `git push origin v1.8.0`; create the GitHub release
    **`v1.8.0 — v2 build · M1c contract`** (notes: the dropped objects; floor 1.7.0 hard; snapshot id). After Flux:
    healthz reports the version; `ssh sujaykumar-vps 'k3s kubectl get deploy,pods -n xlearn -o wide'` shows the new images and no
    crash-loop; ImagePolicies' latest = the tag and HelmReleases Ready; `… logs deploy/xlearn-<svc> --since=30m | grep "migration applied"`
    for the five services; smoke login, dashboard, coach (plus a mistake edit and a mock page load).
12. **[X] If anything fails after the tag:** R-c (revert + patch tag) by default. R-b to `1.7.0` is proven safe;
    **never below `1.7.0`**. R-d only within the snapshot's day, and only as ADR-0034 §4.2 (pin git back first).
    Log it in status.md.

## Constraints

- **Contract discipline ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)):** drop only what `v1.7.0` neither reads nor writes; one marked file per service; never run `Down` in production; fix forward.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** each service contracts its own schema only; no cross-schema DDL.
- **goose + sqlc:** next free version per service at rebase; commit `sqlc generate` output; `sqlc diff` clean; the migration lint green.
- **No scope creep:** no feature, no `SET NOT NULL` beyond the list, no event or API rename (`total_35` / `total35` stay in payload decoders and `/api/mocks/*`).
- **Events:** nothing changes on NATS — no subject, stream or consumer (no ACL PR); envelope append-only, decoders forever.
- **GitOps:** no `kubectl apply`; `ssh sujaykumar-vps` is read-only (`get`, `logs`, `host-verify`); the snapshot is the owner's hPanel action, taken before launch.
- **D34 / D12:** no alert, timer, CronJob or push channel; no off-node `pg_dump`, no backups beyond the manual snapshot.
- **Memory-sum rule:** no new pod; unchanged.
- **Parallel sessions:** check peers' PRs, tags and worktrees (and ListAgents) before merging the contract, before the snapshot and again right before pushing the tag; don't claim an ADR number without checking.
- The throwaway fake-provider override (if used) lives in the scratchpad only, never in the repo.

## Deliverables

- Five contract migrations with the `floor=<M1b tag>` marker; the re-homed `mock_session_scored_total_check`; regenerated sqlc code.
- Contract store integration tests (dropped objects gone, CHECK validated) and handler tests (422 on bad category / dimension).
- `hack/lint-dropped-columns.sh` extended (`--ref`, sqlc / raw-SQL surfaces, the three v1 conflict targets, `is_default`); `hack/rehearse-contract.sh` + `deploy/local/compose.rehearse.yml` + `make rehearse-contract`; `internal/e2e/rehearse_test.go` (reusable by l-02 and ga-02).
- A PR with the assertion output, the rehearsal phase table and logs, the replay result, and the `host-verify` summary.
- Snapshot recorded; tag `v1.8.0` deployed and verified.

## Update status

- Set each task in [`../sprints/sprint-m1-08.md`](../sprints/sprint-m1-08.md) 🔄 / ✅ / ⛔; _Overall_ ✅ at the end.
- [`../status.md`](../status.md): Sprint board row; **Milestones: M1 ✅** (M1a `v1.6.0` → M1b `v1.7.0` → M1c `v1.8.0`); **milestone → tag → rollback floor → snapshot**: `v1.8.0` → **1.7.0 (hard)** → snapshot `<name>` taken `<time>`, "preceded v1.8.0"; owner event `ev-snap-v1.8.0` ✅ (taken before launch); the `host-verify --cluster` run (date, WARNs); no flag changes; the accepted-risk register untouched (D12, D34).
- **Decisions log:** any drop-list item moved to a later contract and its deadline ("before p-02" for the weak-area unique, "before M3 seeds a second language" for the `problem_section` unique), the constraint names found, the rehearsal result. An ADR only if something genuinely new was decided (check the next free number with peers first).

## Done when

- [ ] M1 exit: golden = v1; every v1 e2e green; events replay (replay test on the contracted schema; projection dumps equal across the contract).
- [ ] Rehearsed forward along the production lineage (`v1.5.2` → `v1.6.0` backfills → `v1.7.0` → HEAD), R-b (`v1.7.0` on the contracted schema) and forward again; driver green in every phase; outputs in the PR.
- [ ] Every contract file carries `-- xlearn:contract floor=<the M1b tag>`; lint, `sqlc diff`, CI green.
- [ ] Each dropped CHECK is replaced by Go validation with a 422 handler test; the re-homed CHECK is validated (also on the backfilled rows).
- [ ] `hack/lint-dropped-columns.sh --ref v1.7.0` and the HEAD run are clean (incl. the three v1 conflict targets); each conditional unique dropped or logged with its deadline.
- [ ] `host-verify --cluster` green on a settled host; weekly image ≤ 7 d; the owner's before-launch snapshot recorded, the same day as the tag.
- [ ] `v1.8.0` verified; floor 1.7.0 (hard) and the snapshot in status.md; M1 ✅.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/m1-08-m1c-contract`, then conventional commit(s) with the attribution lines, then push, then the PR. This repo only: no `../infra` PR.
2. Once CI is green (fix, then merge, on failure) **and** `host-verify --cluster` (step 9) is green, squash-merge. Never enable auto-merge. A merged but untagged contract is exposed to any peer's tag, so the tag follows at once. (An owner "hold" given in the session still overrides: stop before the merge.)
3. **Release action — tag `v1.8.0`, a contract tag** (the next free minor): walk the release checklist (ADR-0034 §6, in the plan; the contract lines are the rehearsal, `host-verify` and the owner's before-launch snapshot), push the tag right after the merge, the same day as the snapshot, let Flux deploy, then verify live by looking (step 11). Floor after: 1.7.0, hard.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way (the tag → floor → snapshot record needs the follow-up).
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
