# Sprint m2-01 — M2a touch-attempt engine + `touch_concluded` consumer

> **Milestone:** M2 — attempt engine and projections (M2a)   ·   **Track:** product (order 33)
> **Prereqs:** [m1-08](sprint-m1-08.md) (M1 shipped, v1.8.0 live) · [ds-m2-01](sprint-ds-m2-01.md) (AB04–AB06, AB22 frozen)   ·   **Unblocks:** [m2-02](sprint-m2-02.md) · [m2-04](sprint-m2-04.md)
> **Release action:** merge only (ships in **v1.9.0**, tagged by [m2-02](sprint-m2-02.md)) + infra PR(s) of its own (practice → review caller; ACL only if durables change).
> **Calendar:** late October (after v1.8.0, which is booked for week 5, 2026-10-24 → 10-30).
> **Execute with:** [`../prompts/prompt-m2-01.md`](../prompts/prompt-m2-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | practice M2a expand (attempt purpose/state/grade columns, `timer.kind=touch`, unique outcome, race-safe `LogOutcome`) | X | ⬜ |
| 2 | `problem.spec` + DSA probes, solution facts and the pattern alias table | X | ⬜ |
| 3 | Probe key evaluation package (`internal/course/keys`) | X | ⬜ |
| 4 | Touch endpoints (start, view, lock-in, attest, end) + practice → review read | X | ⬜ |
| 5 | Touch-deadline ticker (D15) | X | ⬜ |
| 6 | review: `touch_result` v2 columns, `touch_concluded` consumer, reanchor queries, internal revision read | X | ⬜ |
| 7 | Subject registry, shared event fixture, architecture docs, ADR | X | ⬜ |
| 8 | Infra: practice → review caller (`REVIEW_BASE_URL` + NetworkPolicy check), own PR before v1.9.0 | I | ⬜ |
| 9 | ACL render check (infra ACL PR **only if** durables change), before v1.9.0 | I | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the
> M2 milestone row + the AB04/AB05/AB06/AB22 artboard rows). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

Rollout §3 M2 entry ("M1 shipped; AB04–AB06 and AB22 frozen"), expanded:
- [ ] M1 shipped: **v1.8.0 live** (M1c contract done; golden = v1; floor 1.7.0 recorded in `docs/v2/status.md`)
- [ ] AB04–AB06 and AB22 **frozen** (the [ds-m2-01](sprint-ds-m2-01.md) PR merged by the owner): `design-system/screens/v2/`
      `AB04-touch.html`, `AB05-catalog-agenda.html`, `AB06-public-profile-v2.html` and `AB22-visibility-toggles.html` on `main`
- [ ] On `main` from M1: `internal/course` manifest with `revision.bands[*].{parts, criteria, mock_mode}` and the band timers ([m1-01](sprint-m1-01.md)); item schema with `revision.probes[]` + `solution_facts` ([m1-01](sprint-m1-01.md)); the DSA item layout + loader ([m1-09](sprint-m1-09.md)); `topology.go` + subject-registry test ([mi-05](sprint-mi-05.md)); v2 envelope decoders ([m1-02](sprint-m1-02.md)); practice `GET /attempts/open` ([m1-07](sprint-m1-07.md)); the migration lint with its `-- xlearn:relax <reason>` marker ([m1-02](sprint-m1-02.md)); the dead-letter sink `event_dead_letter` in review ([mi-05](sprint-mi-05.md))

_Informational, not a gate:_ the owner's answer to ds-m2-01's **"Decisions to confirm" #1** (honor-key strictness in M2:
(A) strict until m4-04's claim, or (B) v1-parity until the claim — see Task 3). If the PR review answered it, build that
option; if not, build (A) and record the item as **open** in `docs/v2/status.md` (Update status below).

## Goal

Make revision touches **real attempts**: practice owns a server-timed `purpose=touch` attempt with answer-free probes
(`problem.spec`), probe **lock-ins** evaluated against public keys, a D15 deadline ticker, and a single guarded
conclusion; review gains the `touch_concluded` consumer that scores a touch through the same ladder code as v1's
self endpoint. **Consumer first, producer next tag:** nothing emits `touch_concluded` in v1.9.0
([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)); the
emit lands in [m2-05](sprint-m2-05.md) (v1.10.0), and no BFF route reaches the new endpoints before
[m2-04](sprint-m2-04.md), so v1.9.0 changes nothing a learner can see.

## Scope

**In**
- practice M2a expand ([t1 §4 practice](../research/t1-content-data-model.md#4-schema-deltas-vs-v1-conceptual), [t4 §3.1](../research/t4-judge-contract.md#3-attempt-and-conclusion-lifecycle)): purpose, state, grade/passed/score, `graded_by`/`trust`, `criteria`, `policy_version`/`stage_params`, touch columns, `timer.kind` adds `touch`, `UNIQUE(outcome.attempt_id)`; `LogOutcome` becomes a guarded conclude.
- curriculum `problem.spec` (answer-free probes + solution facts) and the DSA templated probes ([t1 §7.1](../research/t1-content-data-model.md#71-package-format)).
- Touch start / view / lock-in / attest / end endpoints and the touch-deadline ticker (D15).
- review `touch_result` v2 columns, the `touch_concluded` consumer, `ReanchorPendingTouches` / `ReanchorTouch(anchor_at)` queries, and the subject registry + ACL render check.

**Out**
- Emitting `touch_concluded` (and `touch_scored`) → [m2-05](sprint-m2-05.md), merged after the v1.9.0 tag.
- `touch_scored` schema and its consumer, projections v2 → [m2-02](sprint-m2-02.md).
- Gateway BFF touch routes and the Touch UI (AB04) → [m2-04](sprint-m2-04.md) (**no gateway route in this sprint**).
- D2 "every conclusion anchors the ladder", `revision_item.anchor_rule`, `REVISION_ENTRY_RULE` → [m2-05](sprint-m2-05.md) (this sprint only adds the `ReanchorPendingTouches` query it will call).
- Judge-graded re-solve, the watermark drain, course-attempt deadline Miss → [m3-08](sprint-m3-08.md) (extends this sprint's ticker).
- Honor-probe claim ("I meant this") and AI provisional → M4 ([m4-04](sprint-m4-04.md)).

## Tasks

### 1 · practice M2a expand [X]

A new goose migration in `internal/practice/store/migrations/` (take the **next free version** at rebase; CI fails on a
duplicate). Expand-only per ADR-0034 §3 (nullable or constant-default columns, new indexes `CONCURRENTLY` under
`-- +goose NO TRANSACTION` in their own file):

| Table | Change |
|---|---|
| `attempt` | `purpose text NOT NULL DEFAULT 'course' CHECK (purpose IN ('course','touch'))`; `revision_item_id uuid`, `touch_level int CHECK (touch_level BETWEEN 1 AND 5)`, `band text`; `state text NOT NULL DEFAULT 'attempting' CHECK (state IN ('attempting','awaiting_evaluation','provisional','self_grade_pending','concluded','voided'))`; `concluded_at timestamptz` (set once); `grade text CHECK (grade IN ('clean','rough','assisted','miss'))`, `passed boolean`, `score_value numeric`, `score_max numeric`; `graded_by text CHECK (graded_by IN ('self','auto','ai','override'))`, `trust text CHECK (trust IN ('checked','honor'))`; `criteria jsonb`; `policy_version text`, `stage_params jsonb`; `probes_shown_at`, `anchor_at timestamptz`; `resolution text` (validated by a closed Go enum, **no** CHECK, so M3/M4 values need no constraint change) |
| `attempt` backfill | ended attempts → `state='concluded'`, `concluded_at=anchor_at=ended_at`, `grade` from `outcome.value`, `passed = grade <> 'miss'`, `graded_by='self'`, `trust='honor'` — one idempotent statement (`WHERE purpose='course' AND ended_at IS NOT NULL AND concluded_at IS NULL`), also re-run at startup (below) |
| `attempt` index | partial unique `(account_id, problem_id) WHERE purpose='touch' AND state='attempting'` — **one live touch per (account, problem)** (uses the M1a denormalized columns) |
| `timer` | `kind` CHECK widened to `('attempt','hint','touch')` |
| `outcome` | `UNIQUE (attempt_id)` (unique index `CONCURRENTLY`) |

- **Why `self_grade_pending` and `voided` now** (beyond t1's four states): D15 voids never-shown touches in this sprint,
  and declaring M3's `self_grade_pending` now avoids a second CHECK change. `reimplementing` is not added (D16).
- **CHECK widening vs the contract lint.** Widening `timer.kind` is additive (the new set ⊇ the old) but needs
  `DROP CONSTRAINT … , ADD CONSTRAINT …`, which the [m1-02](sprint-m1-02.md) lint (`hack/lint-migrations.sh`) flags.
  Use m1-02's reviewed-relaxation marker on the `DROP CONSTRAINT` line —
  `-- xlearn:relax widen practice.timer.kind for touch` (the same mechanism [p-02](sprint-p-02.md) uses for `path.status`);
  **don't change the lint** and don't use a contract marker (a widening sets no floor). Confirm the constraint name with
  `\d+ practice.timer` in compose. A store test inserts every pre-existing `kind` value and `touch`.
- **Rollback-safe course-attempt state.** The floor stays 1.7.0, so an R-b rollback to v1.8.0 is allowed; v1.8.0 ends
  course attempts without setting `state`/`concluded_at` (the insert default leaves `state='attempting'`). So: (a) "open
  course attempt" keeps its v1 derivation, `ended_at IS NULL`, in every M2 query; `state` is authoritative only for
  `purpose='touch'` (v1.8.0 never writes touches); (b) the backfill statement above also runs at practice startup, after
  migrations under the advisory lock (`store.HealEndedCourseAttempts`, logs the row count), so rows a rolled-back v1.8.0
  wrote are healed on roll-forward. A test ends an attempt v1-style (only `ended_at` + outcome), restarts, and asserts the
  row is `concluded`.
- **Unique outcome on live data.** Before merging, confirm no duplicate outcomes exist (read-only
  `SELECT attempt_id, count(*) FROM practice.outcome GROUP BY 1 HAVING count(*) > 1` against prod via `ssh vps`,
  or ask the owner to run it). The migration must not delete rows; if duplicates exist, stop and report.
- **Every v1 attempt query gains `AND purpose = 'course'`** (`GetOpenAttempt`, `GetLatestAttempt`, the state
  composition in `store.go`, `ListStates`), so a live touch can never be mistaken for the course attempt; a test proves
  the course state is unchanged by touches.
- **`LogOutcome` becomes a guarded conclude** ([t4 §3.2](../research/t4-judge-contract.md#32-practice-consumer-and-ticker)
  `Conclude()`): lock the attempt `FOR UPDATE`, `UPDATE … SET state='concluded', concluded_at=now(), anchor_at=now(),
  grade, passed, graded_by='self', trust='honor' WHERE id=$1 AND concluded_at IS NULL RETURNING`; zero rows → return the
  stored state and emit nothing. The v1 events and response shape are unchanged. A `-race` test: two concurrent
  submits → one outcome row, one `problem_solved`.
- `sqlc generate`; commit the generated code (`sqlc diff` clean).

### 2 · `problem.spec` + DSA probes, solution facts, alias table [X]

Content lives in m1-09's layout (the converter deleted `curriculum/dsa/*.json`): `curriculum/courses/dsa/items/<id>/item.json`
(probes, `solution_facts`), `curriculum/courses/dsa/items/<id>/sections/<stage>/NN-<kind>.md` (prose) and
`curriculum/courses/dsa/items/<id>/_code/…` (reference code).

- **curriculum migration** (next free version): `problem.spec jsonb` (nullable). The seed writes it from each item file:
  `parts`, `grader`, `revision.probes[]`, `solution_facts`; it is covered by `content_hash`, so children rewrite only on
  change ([t1 §3.2](../research/t1-content-data-model.md#32-public-content-layout-and-delivery)).
- **API:** `GET /problems/{id}` (and the bulk reads) add `spec` **answer-free**: probes as
  `{id, type, criterion, prompt_md, config, timer_s}` (no `key_source`), parts config; `solution_facts` only with the
  solution stage — if [m1-06](sprint-m1-06.md) served facts from another source, move it onto `spec.solution_facts` and
  keep its stage-gate test green.
- **DSA probes (templated):** add `p-pattern` (`text`, `key_source: public:pattern`, `criterion: pattern_named_fast`,
  `timer_s: 120`, "Name the pattern.") and `p-complexity` (`blank` with fields `time`/`space`,
  `key_source: public:solution_facts.complexity`, `criterion: complexity_stated`) to **every**
  `curriculum/courses/dsa/items/<id>/item.json` (a one-shot script; commit its output). The content CI (`content` job from
  m1-09) asserts both probes on every live revisable DSA item.
- **Solution facts:** fill `solution_facts.complexity {time[], space[]}` in each `item.json` for every seeded DSA item.
  Five of the 14 v1 items state their complexity in a solution-stage section —
  `curriculum/courses/dsa/items/{1,3,16,40,104}/sections/solution/*-complexity.md`; derive the other nine (and any item
  authored since) from the reference code under `_code/`; list every fact in the PR for an owner glance. Each list holds
  the canonical answer **plus the learner-facing variants** the normaliser can't derive — above all for multi-variable or
  named-variable facts: id 104 `O(amount × len(coins))` also lists `O(n*m)`, `O(nm)`, `O(amount*coins)`; id 40
  `O(capacity)` also lists `O(n)`, `O(k)`; id 7 (Heap / Bucket Sort) lists both approaches' `O(n log k)` and `O(n)`.
  An item with no facts falls back to an **attested** `complexity_stated` (v1 self-affirm, honor); the content CI reports
  it as a warning, and **fails** on any fact the normaliser (Task 3) can't parse (it could never match).
- **Alias table:** `curriculum/courses/dsa/keys/patterns.json` (the course `keys/` dir,
  [t1 §3.2](../research/t1-content-data-model.md#32-public-content-layout-and-delivery)), one entry per distinct DSA `pattern`
  value with generous aliases (e.g. "Hashing" → hash map, hash set, hashmap, hash table, set lookup). Compound patterns
  ("Heap / Bucket Sort", "Design / Hashmap + DLL") accept any listed component. The content CI fails a live item whose
  pattern has no entry.

### 3 · Probe key evaluation package (`internal/course/keys`) [X]

Pure functions, no I/O, keyed by probe type (the closed `key` grader registry of [ADR-0029 §1](../../adr/0029-judge-contract-and-learning-signal.md)):
- `text`: normalise (case, whitespace, punctuation, hyphens, simple plurals) and match any alias.
- `blank.big_o`: a small normaliser applied to **both** the answer and every accepted entry, then set membership.
  Operators: `×`/`·`/implicit spacing = `*`; `^2` = `²` = a repeated factor (`O(n*n)`); `O(…)` wrapper optional; `log`,
  `lg`, `ln`, `log2` are one base-free `log` (`O(nlogn)` = `O(n lg n)`). **Variables are not only `n`:** any identifier
  or call (`k`, `m`, `V`, `E`, `amount`, `capacity`, `len(coins)`) is an atom, case-insensitive; products and sums are
  commutative (factors and terms sorted, so `O(m*n)` = `O(n*m)`, `O(E+V)` = `O(V+E)`). The normaliser never renames
  variables: `O(n*m)` for id 104 matches only because Task 2's accepted list carries it.
- `key_source` resolution: `public:pattern` → the item's pattern + its alias entry; `public:solution_facts.complexity`
  → the facts per field id. Every `public:*` key is **honor-grade** (`trust=honor`).
- **Honor-probe policy** (ds-m2-01 "Decisions to confirm" #1): one pure function
  `honorProbeMet(keyMatch, withinLimit bool, policy)` decides a `public:*` criterion. **(A) strict** (default, T4 §6.6):
  met iff the key matches (and, for `pattern_named_fast`, within 120 s). **(B) v1-parity until the claim:** met iff
  locked in within its limit — the lock-in is v1's self-affirmation — with `source=attest`, `trust=honor`. Either way
  `criteria[]` stores `key_match` so misses stay countable for alias growth and m4-04. The policy is a code constant set
  from the owner's answer (no flag service, ADR-0034 §2) and is part of the stamped `policy_version`; m4-04 switches (B)
  to strict + the claim.
- Table tests over every seeded item's pattern and facts, plus the variant corpus — including ids 7, 40 and 104's
  multi-variable / named-variable forms, `O(m*n)` vs `O(n*m)`, and `V+E` orderings.

**Why practice evaluates lock-ins in M2:** there is no judge before M3. T4 §12 keeps one evaluator ("values frozen where
they're evaluated"); this package is that one implementation, and [m3-06](sprint-m3-06.md) / [m3-08](sprint-m3-08.md)
decide whether lock-ins move behind judge's inline key lane. Record this in the ADR (Task 7).

### 4 · Touch endpoints + practice → review read [X]

practice, JWT-verified like its v1 routes (the account is the token subject). **No gateway BFF route in this sprint.**

| Endpoint | Behaviour |
|---|---|
| `POST /touches/{revision_item_id}/start` | Before any tx, read review's `GET /internal/revisions/{id}` (Task 6): the item must be the caller's (else 404), `pending` (409 `touch_not_pending`), due (409 `touch_not_due`), the **lowest pending level** for that problem (409 `touch_not_lowest`), and its item not `withdrawn` (409 `item_withdrawn`). In one tx: reuse an open touch for (account, problem) if one exists (resume), else insert `attempt(purpose='touch', revision_item_id, touch_level, band, state='attempting')` on the problem's `user_problem_state` row, snapshot `stage_params` (band criteria, timers, `mock_mode`) + `policy_version` from the manifest, and create `timer(kind='touch', deadline_at = now() + band timer)` (DSA 20:00). A unique-violation race re-reads and returns the winner. Review unreachable or `REVIEW_BASE_URL` unset → 503 `touches_unavailable` (fail closed). |
| `GET /touches/{attempt_id}` | The touch view: the server timer, `mock_mode`, format, the probes answer-free with `locked` / `locked_at_secs` only. **The first call sets `probes_shown_at`** (`WHERE probes_shown_at IS NULL`); m2-04's BFF calls it when it serves the view. After conclusion it adds the result: `passed`, per-criterion `met` and value, the accepted canonical answer for unmet probes (never the alias table), and `next: {level, day}` ("advances to Day 21" / "resets to Day 1") from the universal ladder. |
| `POST /attempts/{id}/probes/{probe_id}` | Lock-in, once per probe (409 `probe_locked`); accepted until the deadline (2 s grace, [t4 §3.2](../research/t4-judge-contract.md#32-practice-consumer-and-ticker)), else 409 `touch_expired`. Server-stamps `locked_at`, evaluates inline (Task 3), stores the criterion result (with `key_match`) in `attempt.criteria`; **the response never reveals correctness**. Under policy (A), `pattern_named_fast` is met iff the key matches **and** `locked_at − started_at < timer_s` (120 s, strict as v1 `AutoPass`); under (B), iff locked within 120 s. |
| `POST /attempts/{id}/attest` | `{criterion, claim: bool}` for criteria with no probe and no evaluator yet — in M2 the DSA re-solve (`correct_in_timer`): met iff `claim` and stamped before the deadline. Once per criterion. |
| `POST /attempts/{id}/end` | "End review": conclude now; undecided criteria count as unmet. |

**Conclusion** (one guarded `concludeTouch()`, [t4 §3.6](../research/t4-judge-contract.md#36-parked-q2-resolved-abandoned-attempts) as overridden by D15):
- Concludes **the moment every required criterion is decided** (`resolution=immediate`), on End review (`ended`), or
  from the ticker (`deadline`, or `abandoned` when the probes were shown and no criterion was decided).
  `touch_passed` = every required criterion met. Never shown → `voided` (Task 5).
- `UPDATE … SET state='concluded', concluded_at=now(), passed, score_value/score_max (met/required), graded_by, trust,
  anchor_at, resolution WHERE id=$1 AND concluded_at IS NULL RETURNING`; zero rows = no-op.
- `graded_by = self` while any required criterion rests on an attestation (all DSA touches in M2), else `auto`;
  `trust` = the minimum over criteria (`honor` for `public:*` keys and attestations). `anchor_at` = the deciding
  lock/attest time (immediate), the End-review time, or the deadline.
- It builds the typed **`touch_concluded` payload** ([t0 §5 (b)](../research/t0-extensibility-frame.md#5-the-learning-signal-contract-narrow-waist)):
  v2 envelope (`path_slug`) + `problem_id, revision_item_id, touch_level, attempt_id, touch_passed, resolution,
  criteria[{key, met, source, at_secs}], recall_secs, graded_by, trust, mock_mode, band, anchor_at, policy_version,
  evaluation_id: null` — golden-tested against the shared fixture (Task 7) — but **does not insert the outbox row**
  in this sprint ([m2-05](sprint-m2-05.md) adds that one insert inside the same tx). `TestNoTouchProducerYet` asserts no
  `xlearn.practice.touch_concluded` outbox row is ever written; m2-05 inverts it.
- `GET /attempts/open` ([m1-07](sprint-m1-07.md)) reports an open touch with `purpose: "touch"`, so the coach's
  `locked` mode (D27, 409 `coach_paused`) engages during a live touch.
- A review HTTP client (`internal/practice/reviewclient.go`, 5 s timeout, nil-safe) reads `REVIEW_BASE_URL`
  (config + `docker-compose.yml`); the call happens **before** `pool.Begin` (no HTTP inside a tx).
- practice parses its embedded content at start into an in-memory item index (spec, probes, facts, pattern, role,
  status) plus the alias tables ([t1 §3.2](../research/t1-content-data-model.md#32-public-content-layout-and-delivery) delivery step 3).

### 5 · Touch-deadline ticker (D15) [X]

`internal/practice/ticker.go`, started from `cmd/practice/main.go`, every **30 s**: select started touches whose
`timer.deadline_at + 5 s < now()` with `FOR UPDATE OF attempt SKIP LOCKED LIMIT 50`, each concluded in its own tx:
- `probes_shown_at IS NULL` → `state='voided'`, `resolution='voided'`, nothing built, the revision item stays due;
- shown, no criterion decided → conclude `passed=false`, `resolution='abandoned'`, `anchor_at = deadline`;
- shown with evidence → conclude on the criteria as they stand (`resolution='deadline'`).

Stop the ticker before the pool closes on shutdown. [m3-08](sprint-m3-08.md) extends this same ticker with the judge
watermark drain, the gap reconciler and the course-attempt deadline Miss. A `-race` test runs two tickers concurrently
and asserts each touch concludes exactly once.

### 6 · review: `touch_result` v2, `touch_concluded` consumer, reanchor queries, internal read [X]

- **Migration** (next free version in review): `touch_result` adds `graded_by` (CHECK `self|auto|ai|override`, default
  `'self'`), `trust` (CHECK `checked|honor`, default `'honor'`), `evaluation_id uuid`, `attempt_id uuid`, `criteria jsonb`,
  `anchor_at timestamptz` (backfilled from `scored_at`); the v1 DSA columns (`named_pattern_secs`, `solved_in_timer`,
  `stated_complexity`) become nullable (`DROP NOT NULL`, a widening); `UNIQUE (attempt_id)` via a unique index
  `CONCURRENTLY`. `auto_pass` (`boolean NOT NULL`, no default) and `mock_mode` stay as they are: the consumer fills them.
- **Queries** (`queries/revision_item.sql`, `touch_result.sql`): `ReanchorTouch` takes the anchor (the fail → Day-1
  reset dates from `anchor_at`, not `time.Now()`, [t4 §3.7](../research/t4-judge-contract.md#37-touches-mocks-arena)); new
  **`ReanchorPendingTouches`** re-anchors only `pending` levels (used by m2-05's D2 rule,
  [t4 §6.5](../research/t4-judge-contract.md#65-unsolved-attempts-enter-revision-d2-with-the-revisable-guard)); `ScheduleTouch … DO NOTHING`
  stays as is for `solution_revealed_early`; `InsertTouchResult` carries the new columns. The consumer's row sets
  `auto_pass = touch_passed`, `mock_mode` from the payload, and leaves the three v1 DSA columns **NULL** — review stays
  course-blind (it never reads criteria keys); the per-criterion detail lives in `criteria`. The v1 self path keeps
  writing all v1 columns as today.
- **One scoring path:** extract the body of `Score` into `applyTouchOutcomeTx(item, outcome)` shared by the v1 self
  endpoint (`POST /revisions/{id}/score`: `graded_by=self`, `trust=honor`, `anchor_at=now()`, v1 columns filled —
  **unchanged behaviour**) and the consumer. Pass → `MarkTouchPassed`, `ensureNextTouch` + `revision_scheduled`,
  clean-revisit counting; fail → `ReanchorTouch` × 5 from `anchor_at`, `revision_scheduled` L1, open/re-open the mistake.
- **Consumer:** `internal/review/consumers.go` routes `xlearn.practice.touch_concluded` (on the existing durable with
  the `xlearn.practice.*` filter) to `store.HandleTouchConcluded`: resolve the pattern **before** the tx only when
  `touch_passed=false`; in one tx claim the inbox, load the item `(id, account)` `FOR UPDATE`, require `pending` + the
  same `touch_level` + `problem_id` (else ack as stale and log), insert `touch_result … ON CONFLICT (attempt_id) DO
  NOTHING` (no row → already applied → no-op), then `applyTouchOutcomeTx`. A v2 envelope missing `path_slug` or
  required fields can never succeed, so it is **parked, not dropped** ([t0 §5](../research/t0-extensibility-frame.md#5-the-learning-signal-contract-narrow-waist)
  envelope rule): record a `review.event_dead_letter` row through the mi-05 sink at once (`err_class=decode`, ids only —
  the same path m1-02's decoder uses for a v2 envelope without a path), then `Term()` and log ERROR (D34: read on demand
  via `ListDeadLetters`, no alert). A test feeds a path-less v2 fixture and asserts one dead-letter row, `Term`, and no
  `touch_result`. Transient DB errors return an error → `NakWithDelay` backoff → the mi-05 dead-letter hook on the last
  delivery.
- **Internal read for practice:** `GET /internal/revisions/{id}` (ClusterIP-only, no user JWT — the ADR-0016
  worker-auth model; MI-5a admits only the `xlearn` namespace to `/internal/*`) →
  `{id, account_id, problem_id, path_slug, touch_level, status, due_date, lowest_pending_level}`.

### 7 · Subject registry, shared fixture, docs, ADR [X]

- `internal/platform/events/topology.go` subject registry: `xlearn.practice.touch_concluded` — handled by review;
  explicitly **ignored** by assessment (and by identity and judge once their durables exist). The registry test stays green.
- Shared fixture `internal/platform/events/testdata/touch_concluded.v2.json`: practice's payload golden test marshals to
  it; review's consumer tests and the e2e decode it; m2-05's producer test must match it byte-for-byte.
- e2e (`internal/e2e`, `-tags e2e`, real JetStream): publish the fixture on `XLEARN_PRACTICE` → review writes one
  `touch_result` and moves the ladder; publish it again (same `event_id`) and with a new `event_id` but the same
  `attempt_id` → still one row.
- Docs: `docs/architecture/events.md` (catalogue row + "Flow 5 — touch concluded → review scores", producer idle until
  v1.10.0), `data-model.md` (practice attempt v2 columns, review `touch_result`, curriculum `spec`), `services.md`
  (practice → review internal read; the practice ticker), `api.md` (practice touch endpoints, internal only until m2-04).
- **ADR** "Touch attempts in practice (M2a realization)": interim practice-side key evaluation (T4 §12 note), the M2
  attestation for criteria without an evaluator, the honor-probe policy (A or B, the owner's answer or "open, A until
  answered") until m4-04's claim, `graded_by`/`trust` derivation for touches, conclude-when-decided, the `timer.kind`
  widening under m1-02's `xlearn:relax` marker, the rollback-safe course-attempt heal, practice → review internal read.
  Take the **next free ADR number** after checking peers' PRs and worktrees.

### 8 · Infra: practice → review caller [I]

Its own infra PR in `../infra`, merged **before the v1.9.0 tag** ([m2-02](sprint-m2-02.md) gate) — harmless earlier
(v1.8.0 ignores the env):
- `apps/xlearn-practice.yaml`: env `REVIEW_BASE_URL: http://xlearn-review.xlearn.svc.cluster.local:8084`.
- NetworkPolicy (ADR-0035 §2 standing rule: every new in-cluster HTTP caller has its policy change in its own PR before
  the tag): under MI-5a ([mi-03](sprint-mi-03.md)) review's internal routes admit the whole `xlearn` namespace, so
  **no ingress change is expected** — confirm it from `apps/xlearn-review.yaml` and say so in the PR. If the `xlearn`
  egress matrix ([mi-11](sprint-mi-11.md)) has already merged, add practice → review :8084 to it in this PR.
- GitOps only (never `kubectl apply`); after merge, read-only check that the practice pod restarted with the env.

### 9 · ACL render check [I]

`make nats-acl-render` and diff against `internal/platform/events/testdata/nats-authorization.golden.conf`. Expected:
**no change** — review's durable already filters `xlearn.practice.*`, and practice publishes within `xlearn.practice.>`
([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)).
If `topology.go` gains a durable or filter, re-render the golden and open the infra ACL PR, merged **before v1.9.0**.
Record the outcome (no-op or PR #) in the sprint status.

## Acceptance criteria

- [ ] Touch attempts start **server-timed** (deadline from the band timer), resume idempotently, allow one live touch per (account, problem), and conclude per **D15**: immediate when every criterion is decided, on End review, at deadline + 5 s via the ticker; never-shown touches are **voided**; shown-and-silent touches fail as **abandoned**.
- [ ] Lock-ins are evaluated server-side against public keys, stamped once, and never reveal correctness before conclusion; `pattern_named_fast` enforces < 120 s.
- [ ] review processes a `touch_concluded` fixture **idempotently** (same `event_id`, and same `attempt_id` under a new `event_id`), advancing on pass and resetting to Day 1 from `anchor_at` on fail; the v1 self endpoint behaves exactly as before.
- [ ] **No producer emits the new subject yet** (`TestNoTouchProducerYet` green; no BFF route to the touch endpoints).
- [ ] v1 course flows unchanged (all v1 e2e green); `LogOutcome` double-submit → one outcome, one event.
- [ ] Subject registry and ACL render green; the practice → review infra PR merged (or queued to merge before v1.9.0).

## Release

**Merge only — ships in v1.9.0**, the tag cut by [m2-02](sprint-m2-02.md) with every M2a/M2b consumer and the producers
idle. Before that tag: the Task 8 infra PR (and a Task 9 ACL PR if one was needed) must be merged. Nothing here is
reachable by a learner until [m2-04](sprint-m2-04.md) + [m2-05](sprint-m2-05.md) ship in v1.10.0.

## Definition of Done

CI green (`go build`, `go vet`, `go test -race ./...`, web tests untouched, `sqlc diff`, the migration lint, the content
job, the subject-registry and ACL golden tests) · e2e green · merged to `main` via PR (squash) · infra PR(s) merged ·
statuses updated (this file + [`../status.md`](../status.md), including the AB04/AB05/AB06/AB22 rows → "frozen (PR #, date)"
and ds-m2-01 task 8 ✅; the honor-probe policy as a Decisions-log line, or as an **open owner item** "strict honor keys
until m4-04 — answer before m2-05 turns the producers on" if the owner hasn't answered) · the ADR written.

## Risks / watch-outs

- **Producer shipping with its consumer loses events** (review acks unknown subjects silently today) — hence two tags.
  The only emit is m2-05's; this sprint's guard test fails if anyone adds it early.
- **Touches leaking into v1 course state.** Any attempt query without `purpose='course'` could resume a touch as a
  course attempt or mark state wrongly — the purpose filter test covers every query in `queries/attempt.sql`.
- **Alias / big-O misses reset a ladder with no recourse until M4's claim.** T4 §12 rejected "strict key match with no
  claim" as worse than v1's self-affirm; under policy (A) M2 ships strict lock-ins from v1.10.0 while the claim is
  [m4-04](sprint-m4-04.md) (v1.16.0) — roughly six weeks on the owner's live ladder. That is why it is an explicit owner
  decision (ds-m2-01 "Decisions to confirm" #1), recorded in status.md, and why (B) exists as a one-constant switch.
  Under (A), mitigate with generous aliases, the normaliser corpus and the accepted answer after a fail; either way
  `key_match` is stored so claim-worthy misses are countable for m4-04.
- **Interim evaluator in practice** (vs T4 §12's "one evaluator, in judge") — keep all logic in `internal/course/keys`
  so M3 can move it without a second implementation.
- **CHECK widening vs the contract lint** — do not paper over it with a contract marker (that would set a false floor)
  or a new lint rule; use m1-02's `-- xlearn:relax <reason>` on the `DROP CONSTRAINT` line and record it.
- **R-b rollback to v1.8.0** (floor 1.7.0) writes ended course attempts without `state`/`concluded_at` — keep "open
  course attempt" on `ended_at IS NULL` and heal at startup (Task 1).
- **Unique outcome index on live data** — `CREATE UNIQUE INDEX CONCURRENTLY` leaves an invalid index on duplicates;
  pre-check prod read-only.
- **Synchronous practice → review dependency** on touch start — fail closed with 503, 5 s timeout, never inside a tx.
- **Async pitfalls** (memory of S06/S07): `NakWithDelay` backoff, stop consumers/ticker before `srv.Shutdown()`,
  inbox claim in the same tx, status guards on every settle path.
- **practice memory** (limit 128 Mi): the embedded item index is small for DSA, but check `k3s kubectl top pod -n xlearn`
  (read-only) after v1.9.0; no new pod, so the ADR-0035 memory-sum rule is unaffected.
