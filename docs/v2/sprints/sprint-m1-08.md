# Sprint m1-08 — M1c contract → v1.8.0

> **Milestone:** M1 — spine (**M1c contract**; closes M1) · **Track:** product (release sprint) · **Order:** 29
> **Prereqs:** [m1-07](sprint-m1-07.md) (`v1.7.0` live, M1c-readiness record) · [mi-02](sprint-mi-02.md) (MI-8 `host-verify --cluster` extension)
> **Unblocks:** [m2-01](sprint-m2-01.md) (M2 entry gate: M1 shipped, `v1.8.0` live)
> **Release action:** **tag `v1.8.0`** — a **contract** tag: compose rehearsal, `host-verify --cluster`, manual snapshot first; rollback floor after: **1.7.0, hard** ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.6, §3, §4.3)
> **Calendar:** week 5 (2026-10-24 → 10-30), **after** the Sat 2026-10-24 host window ([mi-09](sprint-mi-09.md)) has settled — target Mon 10-26 → Wed 10-28 · owner event **`ev-snap-v1.8.0`** (~5 min, right before the tag)
> **Execute with:** [`../prompts/prompt-m1-08.md`](../prompts/prompt-m1-08.md) — one prompt, one session.
> **Tag names are indicative** ([ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline): take the next free minor at tag time). `v1.6.0` (M1a), `v1.7.0` (M1b) and `v1.8.0` (this tag) below stand for the tags **recorded in [`../status.md`](../status.md)**. Read the actual M1a / M1b tags there and substitute them everywhere, including the migration marker (`-- xlearn:contract floor=<M1b tag>`), the floor record, the `--ref` run, the rehearsal image tags and the R-b target.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Contract migrations (`-- xlearn:contract floor=v1.7.0`) + Go validation for every dropped CHECK | X | ⬜ |
| 2 | Assert no old-shape reader or writer in HEAD **and** `v1.7.0` | X | ⬜ |
| 3 | Compose rehearsal: forward (1.5.2 → 1.6.0 → 1.7.0 → HEAD) and R-b (1.7.0 on the contracted schema) | X | ⬜ |
| 4 | `host-verify --cluster` green, host settled, weekly image ≤ 7 d | H | ⬜ |
| 5 | Manual Hostinger snapshot right before the tag (`ev-snap-v1.8.0`) | O | ⬜ |
| 6 | Tag `v1.8.0` + M1 exit record (floor 1.7.0 hard, snapshot id) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + M1 milestone + tag → floor → snapshot row).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **`v1.7.0` live and verified** ([m1-07](sprint-m1-07.md) ✅), and **`v1.7.0` neither reads nor writes any column this contract drops** ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules): contract ≥ 1 release after the last reader or writer) — m1-07 task 5's record, re-proved against the `v1.7.0` tree in task 2 here.
- [ ] **MI-8 `host-verify --cluster` extension available** ([mi-02](sprint-mi-02.md) merged in `../infra` `hack/host-verify.sh`).
- [ ] **Host settled:** the October host window ([mi-09](sprint-mi-09.md), Sat 2026-10-24: host sandbox block, L23 kubelet args, pid limits, k3s/CNPG bumps) is finished and `host-verify --cluster` has been green for ≥ 24 h since its last restart — **or** the window is rebooked to ≥ 2 days after this tag. The snapshot must never straddle host changes (a restore also rewinds the host, [ADR-0034 §4.2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#42-r-d-is-a-procedure-not-a-button)).
- [ ] **Owner available** at tag time for the manual snapshot (`ev-snap-v1.8.0`).
- [ ] **No peer is about to tag from `main`:** once the contract merges, any tag cut from `main` ships it. Check `gh pr list`, `git ls-remote --tags origin`, `git worktree list`, ListAgents, and tell active peers the merge → snapshot → tag window.

## Goal

Contract the M1a expand one release after its last old reader and writer. Drop the v1 columns, uniques, CHECKs
and `'dsa'` defaults that M1a/M1b made redundant. The Go side, validating against the course manifest and the
session's rubric snapshot, replaces every dropped CHECK. Prove it in compose on seeded data along the production
lineage, forward from `v1.5.2` → `v1.6.0` (expand + backfills) → `v1.7.0` → HEAD, and **backward**: the `v1.7.0`
images run on the contracted schema (the R-b floor). Then
`host-verify --cluster`, a manual snapshot, and **`v1.8.0`**. This closes M1: golden = v1, every v1 e2e green,
events replay. From here the rollback floor is **1.7.0, hard**. Only R-d (the snapshot, ~1 day) can go below it.

## Scope

**In**
- Contract migrations in curriculum, review, assessment, coach and practice (per the M1c list in
  [t1 §4 *Expand → backfill → contract*](../research/t1-content-data-model.md) plus m1-02's `weak_area_snapshot`
  addition and [m1-09](sprint-m1-09.md)'s old `problem_section` unique), each marked `-- xlearn:contract floor=v1.7.0`
  (the actual M1b tag; m1-02's lint enforces the marker).
- The Go-side validation that replaces each dropped CHECK (present since `v1.7.0` per m1-07 task 5; proven here on
  the contracted schema).
- The assertion: m1-03's `hack/lint-dropped-columns.sh`, extended with a `--ref` mode and the sqlc / raw-SQL surfaces; `make rehearse-contract` (`hack/rehearse-contract.sh` +
  `deploy/local/compose.rehearse.yml`) and the black-box HTTP driver `internal/e2e/rehearse_test.go`. These are
  reusable by the next contract / erase / GA tags ([l-02](sprint-l-02.md), [ga-02](sprint-ga-02.md)).
- `host-verify --cluster`, weekly-image check, manual snapshot, tag `v1.8.0`, M1 exit record.

**Out**
- Any new feature → M2 starts at [m2-01](sprint-m2-01.md).
- Tightening nullable expand columns (`SET NOT NULL` on `attempt.account_id`, `coach_message.path_slug`, …) — not
  on the M1c list; a later contract if ever needed.
- The review `touch_result` v1 DSA columns → nullable in M2a ([m2-01](sprint-m2-01.md)); nothing of M2 is contracted here.
- ImagePolicy range changes — ordinary releases never touch a range ([ADR-0034 §1.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure)); the floor lives in status.md.
- Erase (the other snapshot-first tag) → [l-02](sprint-l-02.md).

## Tasks

### 1 · Contract migrations + replacement validation [X]

Sources: [t1 §4](../research/t1-content-data-model.md) (curriculum, review, assessment, coach; *Expand → backfill → contract*),
[ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules), [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md),
[m1-02](sprint-m1-02.md) (what the expand added; the lint), [m1-09](sprint-m1-09.md) (curriculum expand).
One file per service, **next free goose version at rebase**, first line of the Up section
`-- xlearn:contract floor=v1.7.0` (substitute the M1b tag recorded in status.md). **Confirm every constraint name
with `\d+` in compose first.** Auto-generated names (`concept_slug_key`, `problem_section_problem_id_stage_order_key`,
`mistake_entry_category_check`, `rubric_score_dimension_check`, …) are likely but not guaranteed, and
`DROP CONSTRAINT IF EXISTS` on a wrong name silently no-ops.

| Service | File | Drops | What replaces it |
|---|---|---|---|
| curriculum | `internal/curriculum/store/migrations/0000N_m1c_contract.sql` | `problem.is_reinforcement`, `problem.leetcode_url`, `problem.neetcode_url`, `concept.code_template`, the v1 `UNIQUE(concept.slug)` (`00001_init.sql:68`); the v1 `problem_section UNIQUE (problem_id, stage, "order")` (`00001_init.sql:120`, conditional, see below) | `problem.role` CHECK, `links jsonb`, `templates jsonb`, `UNIQUE(path_slug, slug)` (m1-09); `problem_section (problem_id, stage, "order", language)` unique (m1-09 `00003`); loader + schema validation |
| review | `internal/review/store/migrations/0000N_m1c_contract.sql` | `mistake_entry.category` CHECK (`00002_mistakes_notifications.sql:28-30`); the v1 `UNIQUE(account_id, week_of)` on `weak_area_snapshot` (`00002:72`, m1-02's addition to this list); `DEFAULT 'dsa'` on `revision_item`, `mistake_entry`, `weak_area_snapshot` `.path_slug` | category ∈ the course manifest's categories (universal core + course, `internal/course`) in `POST`/`PATCH /mistakes` and the consumer's pre-fill; `weak_area_snapshot_account_path_week_uq` (m1-02); writers set `path_slug` (m1-03) |
| assessment | `internal/assessment/store/migrations/0000N_m1c_contract.sql` | `mock_session.total_35` (its range CHECK and m1-02's `mock_session_scored_total_check` go with it — both reference the column); `rubric_score.dimension` CHECK (`00001_init.sql:65-67`); `DEFAULT 'dsa'` on `mock_session.path_slug` and `mock_session_item.path_slug` | `total`/`max_total`; the state invariant **re-homed** as `mock_session_scored_total_check CHECK ((status = 'scored') = (total IS NOT NULL)) NOT VALID` + `VALIDATE CONSTRAINT` in the same file; dimension ∈ the session's `rubric_snapshot` in `ScoreMock` |
| coach | `internal/coach/store/migrations/0000N_m1c_contract.sql` | `api_key_config.is_default` | `coach.key_default(account_id, feature)` (m1-02, m1-10) |
| practice | `internal/practice/store/migrations/0000N_m1c_contract.sql` | `DEFAULT 'dsa'` on `user_problem_state.path_slug` (and on `attempt.path_slug` if m1-02 gave it one) | writers set `path_slug` explicitly (m1-03) |
| identity | — | none: `path_enrollment.path_slug` never had a `'dsa'` default (`00002_path_enrollment.sql:13`); confirm, and add no file | — |

- **Conditional items** (the v1 uniques; m1-07 task 5 should already have switched every upsert off them):
  - drop the `weak_area_snapshot` v1 unique only if task 2 proves `v1.7.0` upserts on
    `(account_id, path_slug, week_of)`. Otherwise leave it and record in the decisions log that it must go before
    [p-02](sprint-p-02.md) writes non-DSA snapshots;
  - drop the `problem_section (problem_id, stage, "order")` unique only if task 2 proves `v1.7.0` no longer upserts on
    that target (it targets `(problem_id, stage, "order", language)` or inserts after the delete). Otherwise leave it
    and record that it must go before M3 seeds a second language at the same stage and order (the
    [m3-01](sprint-m3-01.md) pilot packs: Go, C++, Python). Until then it blocks every multi-language code section.

  The same rule covers any drop-list item that task 2 finds still read or written by `v1.7.0`: move it to the next
  contract tag, never widen this one.
- **Down sections** re-add the columns as nullable and the CHECKs `NOT VALID`, for local dev only. They are
  **never run in production** (ADR-0034 §3; the lint ignores Down).
- `sqlc generate`: the generated models lose the dropped fields, so any HEAD code still using them fails to
  compile. Fix the code, not the migration. `sqlc diff` clean.
- **Validation proof**, in two places (a `package store` test can't import the handler: `internal/review` imports
  `internal/review/store`, so that would be a cycle):
  - **Schema:** an env-gated integration test per service (`internal/<svc>/store/contract_m1c_integration_test.go`,
    `XLEARN_TEST_DATABASE_URL`) migrates a fresh schema to HEAD and asserts via `information_schema.columns` /
    `pg_constraint` that each dropped object is gone and the re-homed CHECK is `convalidated`. Where the validation
    lives in the store (today review's `store.ErrInvalidCategory`), it also shows that a bad value is still refused
    on the contracted schema, with no DB CHECK behind it any more.
  - **422:** handler tests on the existing fakes: `internal/review/mistakes_handlers_test.go` (unknown category on
    `POST`/`PATCH /mistakes` → 422) and `internal/assessment/handlers_test.go` (unknown rubric dimension on score → 422).

### 2 · Assert no old-shape reader or writer [X]

- **Extend m1-03's `hack/lint-dropped-columns.sh`**; don't write a second script. m1-03 wrote it to scan
  `internal/*/store/queries/*.sql` and planned for this sprint to reuse it against the `v1.7.0` tree. Add:
  - a `--ref <git-ref>` mode that checks the ref out into a scratch worktree;
  - the surfaces it misses: the sqlc output `internal/*/store/gen/*.go` (sqlc expands `SELECT *` / `RETURNING *`
    into column lists), raw SQL in `internal/*/store/*.go`, and the curriculum loader;
  - `is_default` (off the allowlist since m1-10);
  - the conflict targets `ON CONFLICT (slug)` on `curriculum.concept`, `ON CONFLICT (account_id, week_of)` on
    `review.weak_area_snapshot` and `ON CONFLICT (problem_id, stage, "order")` on `curriculum.problem_section`
    (whitespace-tolerant; the 4-column `(problem_id, stage, "order", language)` target must **not** match).
  Its allowlist names each allowed hit with a reason: migrations, event-payload decoders (`total_35` in
  `mock_completed` payloads is decoded forever, ADR-0034 §3), JSON API names (`total35` in `/api/mocks/*`). CI keeps
  running it on HEAD.
- Run it on **`v1.7.0`** and on **HEAD**. Both must be clean (HEAD also fails to compile if it isn't, via sqlc).
  Any `v1.7.0` hit takes that item out of this contract (task 1, conditional items).
- The executable proof is task 3's R-b phase: the `v1.7.0` images run and pass the driver on the contracted schema.

### 3 · Compose rehearsal: forward + R-b [X]

`make rehearse-contract` → `hack/rehearse-contract.sh` (compose project `xlearn-rehearse`, own volumes; compose
parity from m1-01: PG 18, NATS 2.14; `DEV_AUTH=1`, `SIGNUP_MODE=open` as in `docker-compose.yml`), with the override
`deploy/local/compose.rehearse.yml` pinning each service to `ghcr.io/sujaykumarsuman/xlearn-<svc>:${XLEARN_IMAGE_TAG}`
with **`platform: linux/amd64`**. The images are amd64-only (`GOARCH=amd64` in `deploy/*.Dockerfile`) and the owner's
machine is macOS, so Docker runs them emulated. **GHCR tags carry no leading `v`** (`deploy.yml` strips it; live is
`xlearn-identity:1.5.2`), so use `XLEARN_IMAGE_TAG=1.5.2` / `1.6.0` / `1.7.0`, never `v1.7.0`. If a pull still
fails, build the tag from a scratch worktree (`git worktree add <scratch>/xlearn-v1.7.0 v1.7.0`); HEAD builds from
the branch.

The driver is `internal/e2e/rehearse_test.go` (`//go:build rehearse`; `XLEARN_REHEARSE_BASE_URL`), a black-box
HTTP run over the gateway using only v1 routes (the `v1.5.2` surface), so it works against every phase. It covers: dev
login; enrol DSA; attempt start → hint → outcome (clean, and a below-clean); revisions due + score; mistakes create
+ patch with a manifest category; mock start + score; progress, dashboard and the public profile `/u/<username>`;
coach key `PUT` / set-default / `DELETE` (the `is_default` → `key_default` writer path) and thread `GET`. Coach chat
is optional: use a throwaway fake-provider override kept in the scratchpad, never committed.

| Phase | Images | Do | Check |
|---|---|---|---|
| −1 | `v1.5.2` (live today) | fresh volumes; driver, including a **scored mock** and a default coach key | v1 rows as production has them: `total_35` only, `is_default`, the v1 curriculum columns |
| 0 | `v1.6.0` on −1's volumes | start (the M1a expand and its backfills apply); driver | the backfills ran on the v1.5.2 rows (`total` from `total_35`, `key_default` from `is_default`, `role` / `links` / `templates` / `language`); v1 envelopes in the outboxes; old columns populated (dual-write) |
| 1 | `v1.7.0` | driver again; let the consumers drain; dump `pg_dump --schema-only` + `assessment.proj_*` data (**A**) | v2 envelopes; on new rows the old columns are **not written by `v1.7.0`**: `NULL`, or the column's remaining default (`is_default` false, `leetcode_url` / `neetcode_url` `''`, `is_reinforcement` false, `code_template` `''`; m1-02 / m1-09 only dropped NOT NULL) |
| 2 | HEAD | start (the contract applies); dump `proj_*` (**B**) **before** any new write; then driver | each service logs `migration applied … m1c_contract`; `VALIDATE CONSTRAINT mock_session_scored_total_check` passed on the **backfilled** v1.5.2 mock; **A = B** (the contract alters no projection row); driver green |
| 3 (R-b) | `v1.7.0` on the contracted schema | start; driver; restart every service once more | goose sees newer versions and no-ops (no error); all Ready; driver green |
| 4 | HEAD again | start; driver | roll-forward after R-b is a no-op; driver green |

Also, at HEAD: `XLEARN_TEST_DATABASE_URL=… go test -tags e2e ./internal/e2e/...` (every v1 e2e green, golden = v1;
without the env var the e2e tests silently skip, per `internal/e2e/coreloop_test.go`) and m1-02's 19-event
replay test (`internal/e2e/testdata/v1-events.jsonl`, v1 and v2-encoded) on the contracted schema. That is the
M1 "events replay" exit, together with the A/B projection check. Paste the phase table with outputs, the replay
result and the migration logs into the PR body.

### 4 · `host-verify --cluster` [H]

- With `../infra` on `main` (with mi-02's MI-8 checks), from the xlearn root (agent shells reset there):
  `ssh vps 'bash -s -- --cluster --expect-sandbox --json --nats-stage=<live stage from status.md: n3, or n4 once mi-11 has run>' < ../infra/hack/host-verify.sh`.
  `--expect-sandbox` applies because the window has run (mi-09: every run from the window on passes it); drop it
  only if the window was rebooked past this tag. It must show **no FAIL**: memory sum, Flux Ready, pods (no OOMKill, ≤ 3 restarts / 24 h), CNPG healthy, PVC / disk,
  NATS auth stage, NetworkPolicies present. Record the WARNs (e.g. TR-STEAL) in the PR; a WARN doesn't block.
- **Host settled:** ≥ 24 h since the host window's last k3s / PG restart, with a green run after it
  ([ADR-0035 §3](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34), [rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)).
- **Weekly image ≤ 7 d:** the owner reads the last Hostinger weekly image date in hPanel (as part of task 5).
- Read-only throughout: no `kubectl apply`, no edits on the node.

### 5 · Manual snapshot (`ev-snap-v1.8.0`) [O]

Right **after** the contract PR is squash-merged (merging deploys nothing; 1.x is tag-only) and task 4 is green,
**right before** the tag: the owner takes a manual Hostinger snapshot in hPanel (one at a time, kept ~1 day; inside
D12, [ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule)) and replies with its name/time and the weekly image
date. The agent records both, then tags within the hour. No off-node `pg_dump` (D12).

### 6 · Tag `v1.8.0` + M1 exit record [X]

Run the release checklist (§Release). GitHub release title **`v1.8.0 — v2 build · M1c contract`**; notes list the
dropped objects and "rollback floor 1.7.0 (hard); snapshot <id> taken <time>". After Flux rolls, beyond the checklist:
`ssh vps 'k3s kubectl logs -n xlearn deploy/xlearn-<svc> --since=30m | grep "migration applied"'` for curriculum,
review, assessment, coach and practice; `k3s kubectl get pods -n xlearn` shows no crash-loop. In
[`../status.md`](../status.md): **M1 ✅**; milestone M1c → tag `v1.8.0` → floor **1.7.0, hard** → snapshot (id, time,
"preceded v1.8.0"); the M2 entry gate is satisfied ([m2-01](sprint-m2-01.md)); the decisions log records any item moved
out of this contract.

## Acceptance criteria

- [ ] **M1 exit:** golden = v1; every v1 e2e green; events replay (m1-02's replay test on the contracted schema, and
      the compose projection dumps equal before and after the contract).
- [ ] Contract rehearsed in compose **forward** along the production lineage (`v1.5.2` → `v1.6.0` backfills →
      `v1.7.0` → HEAD) and **R-b** (`v1.7.0` images on the contracted schema), then forward again; driver green in
      every phase; outputs in the PR.
- [ ] Every contract file carries `-- xlearn:contract floor=<the M1b tag>` (indicatively `v1.7.0`); the migration
      lint, `sqlc diff` and CI are green.
- [ ] Each dropped CHECK has Go-side validation: the store integration test shows the CHECK gone on the contracted
      schema, and a handler test returns 422 for the bad value; the re-homed `mock_session_scored_total_check` is
      validated (including on the backfilled v1.5.2 rows).
- [ ] `hack/lint-dropped-columns.sh --ref v1.7.0` and the HEAD run are clean (allowlist reviewed; the three v1
      conflict targets included); each conditional unique is dropped or logged with its deadline.
- [ ] `host-verify --cluster` green on a settled host; weekly image ≤ 7 d; the manual snapshot taken right before the
      tag and recorded.
- [ ] `v1.8.0` verified; floor **1.7.0 (hard)** and the snapshot recorded in status.md; M1 ✅.

## Release

**Tag `v1.8.0`** ([ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline): M1c **contract**, snapshot first; rollback floor after:
**1.7.0, hard**). Take the next free minor at tag time.
Release checklist ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §6, verbatim, plus the ADR-0035 §2 standing rule):

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

**For this tag:** this **is** a contract. "Rehearsed in compose, floor marked" = task 3 + the file markers;
"`host-verify --cluster` green, host settled, snapshot taken" = tasks 4 and 5, in that order, immediately before the
tag. ACL PRs, new service, new callers and M6: **n/a** (no subject, stream, consumer, service or caller changes;
no new pod, so the memory-sum rule is unaffected). No flag changes.

**If it goes wrong** ([ADR-0034 §4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#41-mechanisms-fastest-first)): R-c (revert + patch tag) is the default. R-b to
`1.7.0` is safe (task 3 phase 3). R-b below `1.7.0` is **forbidden**. R-d (the snapshot) only within its day and only
as the §4.2 procedure: pin git back first, confirm the tags match, list erases since the snapshot (none in M1),
restore, `host-verify --cluster`, re-run the erases.

## Definition of Done

CI green (incl. the migration lint and `sqlc diff`) · rehearsal outputs in the PR · PR squash-merged right before
the snapshot · `host-verify --cluster` green on a settled host · snapshot taken and recorded · `v1.8.0` tagged,
deployed by Flux (no hand `kubectl`) and verified · acceptance criteria met · statuses updated (this file +
[`../status.md`](../status.md): board, **M1 ✅**, tag → floor → snapshot) · anything moved out of the contract logged
in the decisions log.

## Risks / watch-outs

- **Irreversible below 1.7.0.** Once the migrations run, only R-d (the snapshot, ~1 day, loses every write since)
  can undo them. Tag within the hour of the snapshot and verify at once.
- **A late reader or writer crash-loops.** During the rollout the new pod migrates while the old `v1.7.0` pod still
  serves, and every other service keeps running `v1.7.0` until its own pod rolls. The rehearsal must cover **all**
  services and their startup paths (curriculum's seed, coach's key writers). That is why phase 3 restarts everything.
- **Wrong constraint names** silently no-op with `IF EXISTS`. Task 1's `pg_constraint` assertions catch it.
- **Wrong image tag or arch** silently sends the session to the rebuild fallback: GHCR tags have no `v`, and the
  images are amd64-only (`platform: linux/amd64` in the override).
- **An un-snapshotted contract via a peer tag.** A patch tag cut from `main` after the merge would ship the contract
  without the snapshot. Merge only when you are ready to snapshot and tag in the same sitting, and tell peers.
- **Host window interplay.** A snapshot taken before the Oct 24 window, then restored, would also undo the window's
  host changes. Snapshot only after the window has settled. Hostinger keeps **one** manual snapshot, so this one
  replaces the window's pre-change snapshot (mi-09). Declare the window good (`host-verify --cluster` green,
  ≥ 24 h) before taking it.
- **Event payload names ≠ columns.** Don't "clean up" `total_35` in `mock_completed` decoding or `total35` in the
  `/api/mocks/*` JSON; decoders stay forever and the API stays `/api/v1`.
- **Lock time** is negligible (tiny tables: 1 account). Keep each file one transaction, with no data rewrite
  (`DROP COLUMN` is catalog-only; the re-homed CHECK is added `NOT VALID` then validated).
