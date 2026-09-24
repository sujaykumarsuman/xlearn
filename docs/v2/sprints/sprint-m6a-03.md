# Sprint m6a-03 — Assessment deltas + proposal/accept + ScoreMock once + twin fairness gate

> **Milestone:** M6a — text interviewer plus failsafes (v2.1 line; ships dark in a v2.0.x patch)   ·   **Track:** product (order 76)
> **Prereqs:** [m6a-02](sprint-m6a-02.md) (the brain, classifier, probe and `store:false` registry this review call reuses) · [m6a-01](sprint-m6a-01.md) (FSM `finished → proposed → scored`, retention)
> **Unblocks:** [m6a-05](sprint-m6a-05.md) (the UI needs a scoreable interview and the Mock-v2 data) · [m6a-06](sprint-m6a-06.md) (AB27 debrief/proposal; its tag requires this sprint's twin gate to be green)
> **Release action:** **merge only (ships dark in the next v2.0.x patch)** — normally [m6a-06](sprint-m6a-06.md)'s M6a patch. Expand-only migrations (assessment + coach); no contract, no infra change.
> **Calendar:** Q1 2027. One **owner action**: an explicit go-ahead to spend ≤ $20 on the owner's own keys for the live twin-gate run (≈ 180 calls, expected ≈ $11–18; the harness stops at the approved amount), or the owner runs the one command.
> **Execute with:** [`../prompts/prompt-m6a-03.md`](../prompts/prompt-m6a-03.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | assessment deltas (`status`, `format`, `time_multiplier`, `caveats`; snapshot rubric; aggregates `scored` only) + start orchestration + terminal mirror | X | ⬜ |
| 2 | Proposal flow: review call → validated `mock_review@1` → explicit accept / edit / re-propose once / self → `ScoreMock` once | X | ⬜ |
| 3a | Twin fairness gate: harness, synthetic twins, comparator, catalog gate rule, CI replay | X | ⬜ |
| 3b | Twin gate live run per catalog `interview_brain` model (≤ $20, owner's keys, explicit go-ahead) | O | ⬜ |
| 4 | Public route unchanged (D31 count only; `scored` only) + allowlist test extended | X | ⬜ |
| 5 | Docs: data-model, api/openapi, ADR-0032 dated update (twin tolerance, `ai-byo` as built), ADR-0017 update line | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the
> M6a milestone row, pending contracts, decisions log). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m6a-02](sprint-m6a-02.md) **merged**: a cohort interview runs end to end through the API to `finished`; the classifier, probe,
      `callKinds` registry and the exported never-list are on `main`.
- [ ] ADR-0032 is Accepted (with m6a-01/m6a-02's dated updates).
- [ ] v2.0.0's M4 pieces are live: the quote verifier and injection heuristic from [m4-02](sprint-m4-02.md) (NFKC, strip zero-width,
      collapse whitespace, fold quotes/dashes, substring) and the `llm` catalog ([m4-01](sprint-m4-01.md)).
- [ ] **[m6a-04](sprint-m6a-04.md) status known** (it runs in parallel): if merged, its evidence seam (`POST /interviews/{id}/evidence`,
      `SnapshotAt`, `ensureMockFinals`, the picker's `contract_hash` pins) is what this sprint consumes; if not, this sprint defines the
      evidence storage and m6a-04 adopts it (whichever merges second wires the seam).
- [ ] Parallel sessions: no open peer PR adds an **assessment** or **coach** goose migration (else take the next free version at rebase);
      none edits `internal/assessment/store/store.go` `ScoreMock` or `internal/gateway/public.go`.

## Goal

Turn an interview into **one honest mock score** ([ADR-0032 §5](../../adr/0032-realtime-ai-mock-interviewer.md#5-assessment-and-scoring),
[t6 §6](../research/t6-realtime-interviewer.md#6-assessment)): assessment learns the new session shapes (D30 `incomplete`, formats,
time multiplier, caveats) and keeps counting **only `scored`** sessions; coach makes an AI **proposal on public rubric descriptors
only**, validated against the transcript, which the learner must **explicitly accept or edit** (no auto-accept, unlike D14); the
gateway calls **`ScoreMock` exactly once** with `scored_by ∈ {self, ai-byo}` under strict conditions; and the **twin fairness gate**
proves, per catalog model, that irrelevant attributes don't move bands — a model that fails never proposes. D31 (public count only)
is unchanged.

## Scope

**In**
- assessment expand migration + `POST /mocks` for `text|voice`, `POST /mocks/{id}/close`, `ScoreMock` for text/voice, aggregates and
  `GET /mocks/live` restricted correctly; the classic path unchanged.
- Gateway: start orchestration (assessment session ↔ coach `mock_session_id`), the terminal mirror, `POST /api/interviews/{id}/submit`,
  the classic score route hardened.
- Coach: `interview_proposal`, the review call (`mock-review@1`), validation, re-propose once, self-score, the `ai-byo` rule, the
  incomplete paid partial view (click only), the evidence seam if m6a-04 hasn't built it.
- Twin gate harness + CI replay + the live run + the catalog gate rule.
- Public route: nothing new is exposed; the allowlist test grows.

**Out (later sprints)**
- The screens (AB13 Mock-v2, AB27 debrief/proposal, the `incomplete` view) → [m6a-05](sprint-m6a-05.md), [m6a-06](sprint-m6a-06.md).
- Judge finals, evidence aggregation, the picker and `SnapshotAt` → [m6a-04](sprint-m6a-04.md) (consumed here).
- Voice-mode specifics: learner transcript self-edit before review, Communication `self_only_voice` in practice → [m6b-02](sprint-m6b-02.md),
  [m6b-03](sprint-m6b-03.md); the multi-speaker fairness check → [m6c-02](sprint-m6c-02.md).
- Any public relabel: D31 (O4 resolved) already reduced the public mock tile to a count; nothing else changes.
- Contracting `live` into `open` (a later contract; recorded as pending).

## Tasks

### 1 · assessment deltas + orchestration [X]

Sources: [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (T1 and T4 amendment rows),
[t4 §6.7](../research/t4-judge-contract.md#67-mock-evidence--scoremock), [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (expand/contract),
[ADR-0017](../../adr/0017-mock-model-and-projection-consumer-scaffold.md) (score-once), [m1-02](sprint-m1-02.md) / [m1-08](sprint-m1-08.md) (`rubric_snapshot`, `total`/`max_total`, `scored_by`, `mock_session_item` already live).

**Migrations** (next free versions, **expand only**; [m1-02](sprint-m1-02.md)'s migration lint must pass):
- `internal/assessment/store/migrations/0000N_mock_formats.sql`:
  - `status` CHECK widened to `('live','open','scored','incomplete','abandoned')` — `live` stays for classic, `open` is the
    text/voice non-terminal; folding `live` into `open` is a **pending contract** (status.md). A widening is a drop + add, and m1-02's
    lint classifies any `DROP CONSTRAINT` as contract unless its line carries the reviewed marker. So: `ADD CONSTRAINT
    mock_session_status_v2_check CHECK (status IN (…)) NOT VALID`, then `VALIDATE CONSTRAINT mock_session_status_v2_check`, then
    `DROP CONSTRAINT mock_session_status_check -- xlearn:relax widened CHECK (superset of the old set)`. Confirm the v1 name
    (auto-generated from the inline column CHECK in `00001_init.sql`) with `\d+ assessment.mock_session` in compose first — a wrong
    name in `DROP CONSTRAINT IF EXISTS` silently no-ops ([m1-08](sprint-m1-08.md)'s rule).
  - `format text NOT NULL DEFAULT 'classic' CHECK (format IN ('classic','text','voice'))`;
    `time_multiplier numeric(3,2) NOT NULL DEFAULT 1.00 CHECK (time_multiplier IN (1.00,1.25,1.50,2.00))`;
    `caveats text[] NOT NULL DEFAULT '{}'`; `client_ref uuid NULL` (the coach interview id — the idempotency key; **no inline
    `UNIQUE`**, which would build the index under a lock inside the transaction).
- `internal/assessment/store/migrations/0000(N+1)_mock_client_ref_idx.sql` — `-- +goose NO TRANSACTION` and only
  `CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS mock_session_client_ref_uq ON assessment.mock_session (client_ref) WHERE
  client_ref IS NOT NULL` (the ADR-0034 §3 expand form; m1-02's second-file pattern).
- `scored_by` (CHECK `self`/`ai-byo`) already exists (m1-02). `mock_session_scored_total_check` is untouched.

**Store + handlers** (`internal/assessment/{mock.go,handlers.go,store/store.go}`, queries in `store/queries/mock_session.sql`):
- `POST /mocks` accepts `{format: text|voice, path_slug, difficulty, time_multiplier, items[{item_id, contract_hash}], client_ref}` →
  `status='open'`, `path_slug` from the body ([m1-08](sprint-m1-08.md) dropped `DEFAULT 'dsa'`, so every writer sets it; required,
  422 otherwise), `difficulty` from the interview's create, `set_id = 'interview'` (v1's `set_id text NOT NULL` has no default; a
  fixed, non-authoritative value — the items live in `mock_session_item`), `problem_id` left `''`, `mock_session_item` rows from
  `items` (m6a-04's picker pins), `rubric_snapshot` from the manifest, and `deadline_at = started_at + 30 d` (**non-authoritative**
  for text/voice — coach's clock is; documented). A repeat with the same `client_ref` returns the same session.
- `POST /mocks/{id}/close {status: incomplete|abandoned}` — only from `open`, idempotent (a repeat returns 200 with the row).
- **`ScoreMock`** for text/voice: from `open` → score; from `scored` → return the stored result (the existing idempotent re-submit
  path in `store.go`, unchanged — no re-insert, no new outbox row); from `incomplete`/`abandoned` → 409 `mock_not_open`. The gateway
  treats a `scored` response as success and proceeds to submit step 3. Dimensions are validated against the session's
  **`rubric_snapshot`** (N dims, each in the snapshot's scale — confirm m1-07/m1-08 already removed the exactly-7 check at
  `store.go` `ValidateRubric`; finish it if not); accepts `scored_by ∈ {self, ai-byo}`, `caveats[]` (a closed enum: `pause_exposure`,
  `transcript_edited`, `client_turns`, `review_flag`, `cut_short_cap`, `cut_short_pause_limit`, `report_pending_quota`,
  `time_multiplier`, `custom_model`, `snapshot_capped`), `notes` ≤ 2 KiB; same `FOR UPDATE` score-once transaction, outbox row and
  `proj_activity.mocks` increment as today. **Classic path unchanged** (forced `scored_by='self'`).
- **Aggregates count only `status='scored'`**: `Trend`, `MockStats`, `proj_activity.mocks` (already in `ScoreMock`), `/public/stats`
  `header.mocks` — add a test per reader that `open`/`incomplete`/`abandoned` rows never count.
- `GET /mocks/live` (m1-07's coach mode input) returns **classic** sessions only; text/voice locking is coach's (m6a-01/m6a-04).
- **L16 counts classic only:** the L16 checks live in assessment — `CreateMock`'s transaction takes the advisory lock and counts
  ([m3-09](sprint-m3-09.md) task 6). Add `format = 'classic'` to both count queries (≤ 1 live per course, ≤ 3 started per UTC day) and
  skip the L16 checks for a text/voice create (their caps are L19, in coach). A test: two text sessions today don't consume the classic
  daily limit.
- Real-PG tests: status/format matrix, `client_ref` idempotency, close idempotency, ScoreMock from each status (`open` scores,
  `scored` replays the stored result, `incomplete`/`abandoned` → 409), N-dim snapshot validation, aggregate exclusion, the L16
  classic-only count, a create without `path_slug` → 422, and a v1-shaped classic row still scoring. The migration lint passes on both
  new files.

**Gateway orchestration** (`internal/gateway/interview.go`):
- **Start:** `POST /api/interviews/{id}/start` → assessment `POST /mocks {format:"text", path_slug, difficulty, time_multiplier, items,
  client_ref: id}` (idempotent; `path_slug`/`difficulty` from coach's interview row) → coach `start {mock_session_id}` (stores m6a-01's
  `interview.mock_session_id`). A retry after a partial failure reuses
  the same session through `client_ref`; a coach refusal (e.g. `consent_required`) leaves an `open` session that the next start reuses.
- **Terminal mirror:** `abandon` → coach, then assessment `close(abandoned)`; the sweeper-driven `incomplete` (24 h pause, 30-day
  unscored) is mirrored **lazily** — on the next `GET /api/interviews/{id}`, `GET /api/interviews/active` or `GET /api/mocks*`, the gateway
  closes any `open` session whose interview is terminal (idempotent). Aggregates count only `scored`, so a lagging mirror never corrupts
  trends; a test covers the lag.
- **Classic score route hardened:** `POST /api/mocks/{id}/score` refuses `format≠classic` (409 `use_interview_review`) and always sends
  `scored_by='self'` — a learner can never self-label `ai-byo`.

### 2 · Proposal flow [X]

Sources: [t6 §6 Flow steps 1–5](../research/t6-realtime-interviewer.md#6-assessment), [t6 §7 S1–S3](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility),
[t6 §13 D29–D31](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict),
[t4 §7](../research/t4-judge-contract.md#7-structured-feedback-shape-for-t5) (quote verification, injection heuristic), [t1 §10](../research/t1-content-data-model.md#10-security-and-threat-notes)
and [§12](../research/t1-content-data-model.md#12-what-t1-constrains-downstream) (BYO calls see public descriptors only, `ai-byo`),
[t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) "Structured outputs: a portable subset".

**Coach migration** (next free coach version, expand): `interview_proposal((interview_id, n) n ∈ {1,2}, account_id, kind (full|partial),
model, prompt_v, review jsonb ≤ 8 KiB, review_flag, dropped_quotes, created_at)`; `interview.submission jsonb NULL` (frozen scores,
`scored_by`, caveats, notes); `interview.review_started_at timestamptz NULL` (the once-only guard for the automatic review);
`interview.judge_evidence jsonb NULL` **only if m6a-04 hasn't added evidence storage** (then also the intake
`POST /interviews/{id}/evidence`, m6a-04's route name). All join m6a-01's retention purge and the erase list.

**Review call** (`internal/coach/interview/review/`), on the learner's `interview` key, `store:false` (add `review`, `repropose`,
`partial` to m6a-02's `callKinds`):
- **Trigger** (automatic, once; the learner started the interview and its consent covers it). Coach can't call the gateway or
  `ensureMockFinals`, and a rail-end, debrief-end, pause-limit or `$`-cap `finished` happens **inside coach** (sweeper, `AddSpend`) with
  no gateway call. So the review starts when **both** hold — the interview is `finished` **and** judge evidence is stored — on
  whichever of the two events comes second, as a CAS on `review_started_at IS NULL`:
  - the **`finished` transition** (any path) starts it if evidence was already stored (the gateway may post it during `wrapping`)
    **or** no evidence is needed (every item in `interview.items` is `runs: false`, m6a-04's picker flag);
  - the **evidence intake** `POST /interviews/{id}/evidence` starts it if the interview is already `finished`.
  The **gateway** is the only caller of `ensureMockFinals`, and posts the evidence once every final is terminal, from: (a) its
  `finish` proxy, (b) the two evidence routes, and (c) a **lazy check** on `GET /api/interviews/{id}`, `GET /api/interviews/active` and
  `GET /api/interviews/{id}/proposal` when coach's read reports `state: finished` with `evidence_pending: true` (a new field on coach's
  read) — this is m6a-04's "(c) m6a-03's review trigger". So a sweeper-driven finish with the tab closed is reviewed, with evidence,
  on the learner's next read. Until [m6a-04](sprint-m6a-04.md) merges there are no finals: the gateway posts an empty evidence set
  (`items: []`, so `judge`-only dimensions go `null`) and m6a-04 replaces it with `ensureMockFinals`. `report_pending=quota` → no
  automatic call; the learner retries with a click (`POST /interviews/{id}/proposal`, probe first).
- **Inputs** (delimited data with a random boundary): the snapshot's **public** rubric descriptors and `evidence` lists; the stored
  transcript (non-deleted turns with `source`/`edited`); the final code and the on-screen state at each turn via `SnapshotAt` (D29); judge
  **aggregate** evidence (hidden passed/total, first-failure class, `evidence_hints` caps); the timeline (phases, per-phase ms, pauses);
  the hint ledger. **No pack material, ever** (a test plants canary pack text and asserts it never reaches the request).
- **Output** `xlearn.mock_review@1` (strict JSON schema, ≤ 8 KiB): per dimension `band 1–5 | null` + `null_reason`, `quotes[]`
  (verbatim candidate text, ≤ 200 chars each), `why`; ≤ 3 strengths; ≤ 3 improvements `{behaviour, drill_ref}`; `summary`; `caveats[]`.
  The schema is in [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task)'s **portable subset** (reviews run on
  Anthropic keys too, and `output_config.format` rejects `minimum`/`maximum`, `minLength`/`maxLength`, `maxItems`, `pattern`): `band`
  is `anyOf:[{enum:[1,2,3,4,5]}, null]`, every field required, `additionalProperties:false`, and the 200-char quote, ≤ 3 strengths /
  improvements and 8 KiB limits are enforced by `review/validate.go` in Go (truncate at a rune boundary or drop extras), as
  [m4-02](sprint-m4-02.md)'s validator does. A schema test forbids those keywords.
  Stored with `model` and `prompt_v` in `interview_proposal`; `proposal_ready` → `proposed`.
- **Validation** (`review/validate.go`): quotes checked with **M4's normalise-then-substring verifier** against candidate turns only
  (if it lives under `internal/judge/ai`, lift the pure function into `internal/platform/llm` with no behaviour change; judge tests stay
  green); unverified quotes dropped; a dimension with no verified evidence (or whose evidence kinds are all absent) → **`band: null`**
  (**no clamp ≤ 2 for mocks**); `evidence_hints` act as caps; the **injection heuristic** ("ignore previous", "score 5", role markers,
  or a band ≥ 4 resting on one quote < 20 chars) → `review_flag`; the **deny-lexicon** (m6a-01's exported never-list) on `why`,
  strengths, improvements and summary → drop the sentence + `review_flag`; `drill_ref`s validated against the course's curriculum ids
  (unknown → dropped); hints mentioned in the interviewer transcript but absent from the ledger → `review_flag` + caveat.
- **Per-dimension `ai` mode** (manifest): `self_only` (and `self_only_voice` in voice) → quotes only, band left `null` for the learner.
  **No AI bands at all** when the model is a **custom id** or hasn't **passed the twin gate** at the current `mock-review@v` (task 3a) —
  the learner scores with the AI's quotes.
- **Review API** (coach, proxied by the gateway, cohort-gated): `GET /interviews/{id}/proposal` (current proposal + the frozen
  submission if any); `POST /interviews/{id}/proposal/repropose` (**once**; blind — the prompt never sees proposal 1 — fresh sample,
  `n=2` becomes current; both kept); `POST /interviews/{id}/partial-feedback` (only for `incomplete`, **explicit click**, probe first;
  improvements only, never bands, never scoreable).
- **Submit** — gateway `POST /api/interviews/{id}/submit {mode: accept_all | edited | self, scores?, notes?}`:
  1. coach freezes `submission` (idempotent; state must be `finished`|`proposed`; `qa_slice` → 409 `qa_session_unscored`) and computes
     **`scored_by = ai-byo` only if all hold**: the model is a non-custom catalog model that passed the twin gate; `mode = accept_all`
     with every band AI-supplied and unchanged; zero `edited`, `deleted` or `source=client` turns; no `pause_exposure`; no `review_flag`;
     the explicit Accept click (this request). Otherwise `self`. Caveats from the interview (task 1's enum) + `time_multiplier` if ≠ 1.
     Improvement areas pre-fill `notes` (≤ 2 KiB, **no quotes**).
  2. gateway → assessment `ScoreMock` (idempotent score-once) with scores, `scored_by`, caveats, notes; a replay on an already-`scored`
     session returns the stored result and counts as success;
  3. gateway → coach `submitted` → `scored` (`scored_at`, retention `purge_at`).
  A crash between steps leaves a frozen `submission` on a `finished`/`proposed` interview; the next read or submit resumes at step 2
  (idempotent). "AI proposed 24 · you scored 27" stays in coach and never reaches assessment. `trust=honor` is always shown.
- **No auto-accept** (D14's 24 h auto-submit does not extend to mocks): an untouched proposal stays `proposed`; `GET /interviews/active`
  exposes `pending_since` so the UI can nudge at 24 h and 7 d (read-time only; no reminder job, D34); 30 days → `incomplete` (m6a-01).
- **Tests:** quote verification (hits, near-misses, zero-width tricks); null bands; caps; injection twin → `review_flag`; deny-lexicon;
  custom id and un-gated model → quotes only; re-propose once (the second is refused); every `ai-byo` condition flipped one at a time →
  `self`; **score-once**: double submit, concurrent submits, a crash between steps 2 and 3 → exactly one `ScoreMock` effect and one
  `mock_completed` outbox row; `store:false` on the new call kinds (both layers of m6a-02's test); **trigger**: a sweeper-driven
  finish with no client → no review yet; a later `GET /api/interviews/{id}` ensures finals and posts evidence → exactly one review,
  with evidence in its request; a second read or a duplicate evidence post → no second review; evidence posted during `wrapping` →
  the review starts on `finished`; an all-`runs:false` interview reviews on `finished` with no evidence.

### 3a · Twin fairness gate [X]

Sources: [t6 §6 Fairness gates](../research/t6-realtime-interviewer.md#6-assessment), [t6 §14](../research/t6-realtime-interviewer.md#14-risks)
(ASR/accent bias, learner steering), [ADR-0032 §5](../../adr/0032-realtime-ai-mock-interviewer.md#5-assessment-and-scoring).

- **Fixtures** `internal/coach/interview/review/twingate/testdata/twins/`: for 3 DSA items, a **clean base** transcript + final code and
  three twins that differ **only** in an irrelevant attribute — **ASR-noisy** (homophones, dropped words), **non-native grammar**, and an
  **injected instruction** in a candidate turn and a code comment. All synthetic (no real person's words).
- **Harness** (`twingate.go`): `mock_review@1` × **3 samples** per transcript per model; per dimension take the median band. **Pass** =
  for every non-injection twin, every dimension's median is within **±1 band** of the base's median, and null-vs-non-null mismatches
  count as failures; the injection twin gets `review_flag` in every sample. Output: a JSON result per `(model, prompt_v)` (bands and
  flags only). Live-mode flags `-models`, `-items` (default 3) and `-budget-usd` (hard stop, task 3b); a unit test pins the call count
  (models × items × 4 × 3) and that the budget stop fires before the crossing call.
- **CI:** a replay mode serves recorded outputs through the fake provider and exercises the comparator, including a planted failing twin
  (must report `fail`). Runs in `go test ./...`.
- **Gate rule** (code, not config): the `llm` catalog entry gains `mock_review_gate{prompt_v, result: pass|fail, date}`; a model without a
  `pass` at the **current** `mock-review@v` is treated as un-gated (quotes only, never `ai-byo`). A test fails if the prompt version is
  bumped without resetting every gate to un-gated. A failing model gets `ai: self_only` behaviour, the M6a ship is **not** blocked by one
  model failing — only by the gate not having run.
- **Reconciling with the register:** the register's twin-gate task says it "blocks the M6a ship if red". This sprint reads **red** as
  "the gate hasn't run live, or the harness/CI replay is failing", per [t6 §6](../research/t6-realtime-interviewer.md#6-assessment)
  "Fairness gates" ("A model that fails gets `ai: self_only`"): a model that runs and fails is **demoted** (quotes only, never
  `ai-byo`), not a ship blocker. [m6a-06](sprint-m6a-06.md)'s entry gate reads the same way.
- **Tolerance** (±1 band, median of 3, over 3 items × 4 transcripts per model — or 1 item if the live run was cut to fit a smaller
  go-ahead) is recorded in the ADR-0032 dated update (the register's open risk).

### 3b · Twin gate live run [O]

- **Needs the owner's explicit go-ahead** to spend ≤ **$20** on the owner's own keys. The count is models × (items × 4 transcripts) ×
  3 samples = ≈ 5 catalog `interview_brain` models × (3 × 4 = 12 transcripts) × 3 = **≈ 180 calls**, at ≈ $0.06–0.10 each ≈
  **$11–18**. No agent uses a key without it. Keys come from env vars on the owner's machine
  (`go test -tags twinlive ./internal/coach/interview/review/twingate -run TestTwinGateLive -models=… -budget-usd=<approved>`), never
  from coach's store.
- **Budget stop:** the harness first prints a dry-run estimate (calls × catalog price × the fixtures' measured tokens) and asks for no
  more than the approved amount; it runs model by model, prices each call with `llm.Cost`, and stops **before** a call that would cross
  `-budget-usd`. A model it didn't finish stays un-gated (quotes only) and is listed as pending — the remainder runs later on a new
  go-ahead. (If the owner would rather approve ≤ $6: `-items=1` runs 1 item × 4 transcripts × 3 samples × 5 models = 60 calls, and the
  ADR-0032 tolerance note must say the live gate used one item.)
- Commit the results under `twingate/testdata/results/` and set each model's `mock_review_gate`. If the go-ahead doesn't come in-session,
  mark 3b ⛔ and carry "**twin gate run live, results committed**" as an entry gate of [m6a-06](sprint-m6a-06.md)'s tag.

### 4 · Public route [X]

Sources: [t6 §13 D31](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict),
[rollout §10 P1, P10](../rollout-plan.md#10-public-dashboard-tasks), [m2-03](sprint-m2-03.md) (`/public/stats`).

- **Mock count only, unchanged**: `header.mocks` counts `status='scored'` sessions (an AI proposal that wasn't accepted leaves the
  session unscored, so it never counts); no `scored_by` split, no format, no best/average.
- Extend the public-shape allowlist test (`internal/gateway/public_test.go`, P10): no `format`, `scored_by`, `caveats`,
  `time_multiplier`, transcript, proposal or interview fields in any public payload.
- The **authed** `GET /api/mocks/trend` and mock views gain `format`, `scored_by`, `caveats` for the learner's own sessions (m6a-05/06
  render them).

### 5 · Docs [X]

- `docs/architecture/data-model.md` (assessment columns, coach proposal tables), `api.md` + `openapi.yaml` (submit, proposal, repropose,
  partial feedback, close; the classic 409), `events.md` (`mock_completed` unchanged).
- [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) dated update: twin-gate tolerance and the per-model gate rule; the `ai-byo`
  conditions as built; the lazy terminal mirror; the automatic review trigger (`finished` + evidence stored, whichever comes second;
  the gateway's lazy evidence check); the live twin-gate budget and item count. [ADR-0017](../../adr/0017-mock-model-and-projection-consumer-scaffold.md):
  an append-only "Update — <date>" line (status/format set; score-once unchanged).

## Acceptance criteria

- [ ] **A mock scores exactly once**: double, concurrent and crash-window submits produce one `ScoreMock` effect and one `mock_completed`
      row; for text/voice `ScoreMock` scores from `open`, replays the stored result from `scored`, and refuses `incomplete`/`abandoned`
      (409); the classic route refuses text/voice and never sends `ai-byo`.
- [ ] The automatic review runs exactly once, with evidence, whichever of `finished` and the evidence post comes second (incl. a
      sweeper-driven finish reviewed on the next read).
- [ ] **Twin gate green**: the harness and comparator pass in CI (the planted failure is caught); the live run is recorded for every
      catalog `interview_brain` model at `mock-review@1` (or 3b is ⛔ and carried as an m6a-06 gate); failing/un-gated models propose no bands.
- [ ] `ai-byo` only when every condition holds (table-tested one condition at a time); no auto-accept; re-propose at most once.
- [ ] Proposals use public descriptors only (pack-canary test); unverified quotes dropped; missing evidence → `null` (no clamp);
      injection and deny-lexicon → `review_flag`.
- [ ] Trends, stats, `proj_activity.mocks` and `/public/stats` count only `scored`; `GET /mocks/live` returns classic only; the public
      allowlist test covers the new fields.
- [ ] Start orchestration idempotent via `client_ref`; abandon mirrored at once, `incomplete` lazily (tested).
- [ ] Both assessment migrations pass m1-02's lint (the `xlearn:relax` CHECK swap; the concurrent `client_ref` index); L16 counts
      classic sessions only; text/voice creates carry `path_slug`.

## Release

**Merge only — ships dark in the next v2.0.x patch** (normally [m6a-06](sprint-m6a-06.md)'s). Expand-only migrations in assessment and
coach (no contract, no floor change, no snapshot); no new NATS subject (`mock_completed` is unchanged), no infra change, no new pod.
Everything stays behind m6a-01's cohort gate. **Pending contract** recorded: fold `live` into `open` at least one release after every
reader handles both.

## Definition of Done

CI green (`go test -race ./...` incl. real-PG assessment and coach tests, the twin-gate CI replay, the `store:false` test with the new
kinds, the public allowlist test, `sqlc diff`, the migration lint, the OpenAPI drift and route-enumeration tests) · e2e lane green (a
cohort interview → finished → proposal → accept → one `ScoreMock`) · merged via PR (squash) · twin-gate live results committed (or 3b ⛔
and carried to m6a-06) · statuses updated (this file + [`../status.md`](../status.md): Sprint board, M6a row "scoring merged (dark)",
pending contracts `mock_session.status live→open`, decisions log) · ADR updates written.

## Risks / watch-outs

- **Twin-gate tolerance choice** — ±1 band on medians of 3 is T6's proposal; document it in the ADR-0032 follow-up and re-run the gate on
  every `mock-review@v` bump (enforced by test).
- **The learner steering `ai-byo`** (custom model, edits, re-propose, pause lookup, a forged `scored_by`) — strict conditions computed in
  coach, the classic route forcing `self`, `model` and `prompt_v` stored.
- **Two services, one score** — coach freezes first, assessment scores once, coach marks last; every step idempotent; crash-window test.
- **Lagging mirror** — harmless for aggregates (`scored` only); the UI must read coach's state for text/voice sessions.
- **Evidence seam race with m6a-04** — whichever merges second adopts the first's route/storage and deletes stubs.
- **Spending the owner's keys** — only with an explicit go-ahead, capped by `-budget-usd`; results contain bands and flags only
  (synthetic transcripts).
- **Review before evidence** — never start the review on `finished` alone; the CAS on `review_started_at` plus "second of the two
  events" keeps it once and with evidence.
- **Custom model ids** — allowed to interview (m6a-02) but never to propose bands (tested).
