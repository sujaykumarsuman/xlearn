# Data model

Maps the [PRD domain entities](../prd/xlearn-prd.md#7-domain-model-product-view) to Postgres tables,
**grouped by owning schema** ([ADR-0005](../adr/0005-data-ownership-and-migrations.md)). One shared
database `xlearndb`; a service touches **only** its own schema. Types are indicative (`sqlc`/`pgx`
generate the Go). Every mutating schema has an `outbox` table for the transactional-outbox pattern
([ADR-0004](../adr/0004-inter-service-comms-and-events.md)).

Conventions: `id uuid` PKs (except natural keys like problem id / path slug), `created_at`/`updated_at`
`timestamptz`, all times **UTC**, money-free. Cross-schema references are **soft** (store the id, no FK)
since FKs can't cross ownership.

---

## schema `identity`

| Table | Key columns | Notes |
|-------|-------------|-------|
| `account` | `id`, `display_name`, `email`, `timezone`, `study_budget_json`, `reminders_json`, `created_at`; M1a (`00005`, m1-02): `role` (`learner`/`tester`/`owner`, default `learner`), `status` (`active`/`suspended`, default `active`), `admitted_via` (`grandfathered`/`invite`/`cli`/`dev`), `invite_id`, `accepted_at`, `region`, `profile_visibility` (`public`/`private`, default `public`) | The `Account` entity (profile/budget/timezone/reminders). `admitted_via` was backfilled to `grandfathered` for every existing row (the owner's included, ADR-0033 §2); v1.6.0's create paths (dev login, open-mode signup) write `dev`. **Nothing reads the M1a columns until M1b** (m1-04, m1-05); roles never enter the JWT. |
| `path_enrollment` | PK (`account_id`, `path_slug`); `status`, `started_at`; M1a: `public_visible` (default `true`) | `public_visible` is written on insert from the course manifest's `public_stats.default_visible` (D7), never on a re-start. |
| `admin_audit` | `id`, `at`, `verb`, `target`, `detail jsonb`; index `(at DESC)` | M1a: the owner admin CLI's audit log (m1-04 writes it; ADR-0033 §8). `target` is an id, never an email. |
| `oauth_identity` | `id`, `account_id`, `provider` (`github`/`google`), `provider_user_id`, `unique(provider,provider_user_id)` | Links an account to an OAuth provider; no passwords. |
| `session` | `id` (opaque), `account_id`, `created_at`, `expires_at`, `revoked_at` | Server-side session behind the HttpOnly cookie. |
| `onboarding` | `account_id`, `path_chosen`, `budget_set`, `completed_at` (+ unused `key_added`, see note) | Drives the first-run flow. `key_added` was never set and is no longer read or returned: whether a coach key is connected is coach's `api_key_config` (GET /coach/key `connected`). The column stays until a later migration drops it. |

## schema `curriculum`

| Table | Key columns | Notes |
|-------|-------------|-------|
| `path` | `slug` (PK), `title`, `status` (`active`/`coming_soon`), `problem_total`, `week_total` | e.g. `dsa`. |
| `phase` | `id`, `path_slug`, `order`, `theme`, `week_from`, `week_to` | 4 phases for DSA. |
| `week` | `id`, `path_slug`, `n`, `title`, `thesis`, `unique(path_slug,n)` | Week thesis + rail. |
| `concept` | `id`, `path_slug`, `slug`, `title`, `body_md`, `when_to_use_md`, `code_template` | Concept/pattern reading. |
| `week_concept` | `week_id`, `concept_id` | Week→concept links (M:N). |
| `problem` | `id` (natural, e.g. `16`), `path_slug`, `week_n`, `title`, `difficulty` (`easy`/`med`/`hard`), `pattern`, `leetcode_url`, `neetcode_url`, `is_reinforcement` | Difficulty → UI tokens (green/amber/red). |
| `problem_section` | `id`, `problem_id`, `stage` (`attempt`/`hint`/`solution`), `kind`, `order`, `body_md`/`code` | **Stage-scoped** content — the API only returns sections for unlocked stages ([R-PF1](../prd/xlearn-prd.md#61-guided-problem-flow-gated-stages)). |

## schema `practice`

| Table | Key columns | Notes |
|-------|-------------|-------|
| `user_problem_state` | `id`, `account_id`, `problem_id`, `status` (`locked`/`available`/`attempting`/`solved`), `first_solved_at`, `current_touch`, `last_outcome`, `unique(account_id,problem_id)`; M1a (`00002`): `path_slug` (default `'dsa'`) | The `UserProblemState` entity. |
| `attempt` | `id`, `user_problem_state_id`, `stage_reached`, `started_at`, `ended_at`; M1a: `account_id`, `path_slug`, `problem_id` (nullable, denormalised); M1b (`00003`): `coach_assist_at` (nullable) | One attempt run through the stages. The M1a ids are backfilled from `user_problem_state` and written on insert. `coach_assist_at` (m1-07, D27) is when the coach was first used on this attempt's problem while it was open (the gateway records it before forwarding the chat; `COALESCE` keeps the first time): the outcome then caps a self-reported clean/rough at assisted. Partial index `attempt_open_by_account_idx (account_id, started_at DESC) WHERE ended_at IS NULL` serves `GET /attempts/open`. |
| `stage_event` | `id`, `attempt_id`, `stage`, `entered_at`, `unlocked_from` | Audit of stage gating. |
| `timer` | `id`, `attempt_id`, `kind` (`attempt`15m/`hint`10m), `deadline_at`, `expired` | **Server-authoritative** countdown ([R-PF3](../prd/xlearn-prd.md#61-guided-problem-flow-gated-stages)). |
| `outcome` | `id`, `attempt_id`, `value` (`clean`/`rough`/`assisted`/`miss`), `revealed_early`, `logged_at` | Below-`clean` ⇒ event that opens a mistake. |
| `outbox` | `event_id`, `subject`, `payload_json`, `created_at`, `sent_at`; M1a: `account_id` | Outbox relay to NATS. `account_id` (erase prep, ADR-0027 §6) is backfilled from `payload_json->>'account_id'` and set on insert — also on review's and assessment's outboxes. |

## schema `review`

| Table | Key columns | Notes |
|-------|-------------|-------|
| `revision_item` | `id`, `account_id`, `problem_id`, `touch_level` (1..5 = Day 1/3/7/21/45), `due_date`, `surfaced_at`, `status` (`pending`/`passed`/`failed`); M1a (`00005`): `path_slug` (default `'dsa'`) | The five-touch queue ([R-SR1](../prd/xlearn-prd.md#63-five-touch-spaced-repetition)). `path_slug` is the event's course (envelope v2) or the scored touch's. |
| `touch_result` | `id`, `revision_item_id`, `named_pattern_secs`, `solved_in_timer`, `stated_complexity`, `auto_pass`, `scored_at` | Auto-score inputs ([R-SR2](../prd/xlearn-prd.md#63-five-touch-spaced-repetition)); Day 21/45 mock-mode flag. |
| `mistake_entry` | `id`, `account_id`, `problem_id`, `pattern`, `mistake`, `root_cause`, `insight`, `category` (8-enum), `revisit_date`, `status` (`open`/`closed`), `revisit_count`; M1a: `path_slug` (default `'dsa'`) | The `MistakeEntry` entity ([R-MJ1](../prd/xlearn-prd.md#64-mistake-journal)). |
| `weak_area_snapshot` | `id`, `account_id`, `week_of`, `top_category`, `counts_json`; M1a: `path_slug` (default `'dsa'`), unique index `(account_id, path_slug, week_of)` (`00006`, built `CONCURRENTLY`) | Weekly weak-area ([R-MJ3](../prd/xlearn-prd.md#64-mistake-journal)). The v1 `UNIQUE (account_id, week_of)` stays (v1.5.2's upsert targets it) and is on m1-08's M1c drop list: it would block a second course's snapshot in the same week. |
| `reminder` | `id`, `account_id`, `kind`, `due_at`, `delivered_at`; M1a: `path_slug` (nullable) | Notifications worker (v1 in-app). |
| `outbox` | … | as above. |
| `event_dead_letter` | PK (`durable`, `event_id`); `subject`, `err_class`, `stream_seq`, `at` | mi-05 (`00004`): an event whose handler failed its last delivery on review's durables (`review`, `notifications`); ids only, erase-safe; read on demand ([events.md](events.md#reliability-notes), [ADR-0035 §1.2](../adr/0035-v2-operations-nats-auth-limits-capacity.md)). |

## schema `assessment`

| Table | Key columns | Notes |
|-------|-------------|-------|
| `mock_session` | `id`, `account_id`, `set_id`, `problem_id`, `date`, `total_35`, `notes`; M1a (`00004`): `path_slug` (default `'dsa'`), `rubric_id`, `rubric_snapshot jsonb`, `total`, `max_total`, `scored_by` (`self`/`ai-byo`); `difficulty` nullable | The `MockSession` entity. v1.6.0 dual-writes `total`/`max_total`/`scored_by` beside `total_35` and reads `COALESCE(total, total_35)` / `COALESCE(max_total, 35)`; M1c drops `total_35`. The status/total CHECK is `mock_session_scored_total_check` on `COALESCE(total, total_35)` (the v1 `mock_session_check` was relaxed away, `xlearn:relax`). |
| `mock_session_item` | PK (`session_id`, `ordinal`); `item_id` (nullable), `path_slug`, `contract_hash` | M1a: a session's ordered items, superseding `problem_id`/`set_id`. v1 sessions have one row (ordinal 1), `item_id NULL` for a mixed set (`problem_id = ''`); written with the session in one transaction. |
| `rubric_score` | `id`, `mock_session_id`, `dimension` (7-enum), `score` (1..5) | 7 dims × 1–5 = /35 ([R-MK2](../prd/xlearn-prd.md#65-mock-interview)). |
| `proj_coverage` | `account_id`, `problem_id`, `solved`, `first_solved_at`, `level1_schedules` | Progress projection — the solved set + ladder anchors. **Keyed per (account, problem)**, not per week — the gateway maps problem → week/phase from curriculum ([ADR-0018](../adr/0018-progress-projection-grain-and-rebuild.md)). |
| `proj_heatmap` | `account_id`, `activity_date`, `solves`, `reviews` | Revision-activity heatmap (per UTC day). |
| `proj_mastery` | `account_id`, `problem_id`, `best_outcome`, `best_rank`, `clean_solves`, `solve_count` | Per-problem solve quality; the gateway rolls it up **by pattern**. |
| `proj_outcome_mix` | `account_id`, `outcome`, `cnt` | First-solve outcome mix. |
| `inbox` | `event_id`, `consumed_at` | Idempotency/dedupe for consumed events. |
| `event_dead_letter` | PK (`durable`, `event_id`); `subject`, `err_class`, `stream_seq`, `at` | mi-05 (`00003`): an event whose handler failed its last delivery on assessment's durables (on `XLEARN_PRACTICE` and `XLEARN_REVIEW`); ids only, erase-safe; read on demand. |

> The `proj_*` tables are keyed at the **event grain** (per problem / day / outcome), because the
> practice/review events carry only a bare `problem_id` — the by-week / by-phase / by-pattern
> roll-ups are composed in the gateway from curriculum, keeping the projections a pure function of
> the event log ([ADR-0018](../adr/0018-progress-projection-grain-and-rebuild.md)).

## schema `coach`

| Table | Key columns | Notes |
|-------|-------------|-------|
| `api_key_config` | `id`, `account_id`, `provider`, `enc_key` (ciphertext), `enc_data_key` (wrapped), `masked_key`, `default_model`, `name`, `enabled`, `is_default` (nullable since M1a, **dead since M1b**); M1b (`00006`): `enc_key_ad`, `enc_data_key_ad`, `kek_id`, `ad_src_digest` | Envelope-encrypted; only `masked_key` ever leaves the service ([ADR-0007](../adr/0007-ai-coach-byo-key-and-secrets.md)). **Two sealed pairs** (m1-10): the LEGACY unbound pair (`enc_key`, `enc_data_key`), dual-written so v1.6.0 can still decrypt every key (rollback floor, [ADR-0034](../adr/0034-v2-release-labelling-gating-and-rollback.md) §3), and the AD-BOUND pair sealed with associated data `xlearn/coach/key/v1\|<account_id>\|<provider>` under keyring entry `kek_id`. `ad_src_digest` = `sha256(enc_data_key)` as it stood when the AD pair was sealed: the AD pair is CURRENT ⇔ it exists and (`enc_data_key IS NULL` or the digest matches), which is how a pair left stale by a v1.6.0 key replace during a rollback is detected. `rewrap.go` backfills and repairs; the legacy pair is contracted by [l-01](../v2/sprints/sprint-l-01.md)/[l-02](../v2/sprints/sprint-l-02.md). |
| `key_default` | PK (`account_id`, `feature` ∈ `coach`/`interview`); `key_id` → `api_key_config` ON DELETE CASCADE, `model`, `updated_at` | M1a (`00004`): the per-feature default key, backfilled from `is_default`. v1.6.0 dual-wrote it and fell back to `is_default`; **m1-10 makes it the ONLY source** — coach neither reads nor writes `is_default`, and there is no fallback (no row = no default), which is M1c's precondition for dropping the column. `coach` is maintained implicitly (first key in; earliest survivor promoted on delete). `interview` is set EXPLICITLY only — never auto-assigned, never auto-promoted — so "not set" is a real state; its `model` may be a catalog id with the `interview_brain` capability or a custom id ("custom" = absent from the catalog at read time, never a stored flag). |
| `coach_thread` | `id`, `account_id`, `page_context`, `created_at`; M1a: `path_slug` (nullable) | One thread per page context. m1-03 writes `path_slug`. |
| `coach_message` | `id`, `thread_id`, `role`, `content`, `created_at`; M1a: `path_slug` (nullable); M1b (`00006`): `provider`, `model`, `input_tokens`, `output_tokens`, `est_cost_micros`, `stop_reason` (all nullable); M1b (`00007`): `prompt_v`, `attempt_id` (nullable) | Chat history. The M1b columns record what the provider turn that produced an assistant message cost: tokens as the stream reported them, `est_cost_micros` = catalog price × tokens (NULL for a custom model id or a stream that reported no usage, which is what makes the month-to-date figure a floor rather than a total), and the raw stop reason. Display only; `GET /keys` returns the month summary as `usage_month`. m1-07 writes `prompt_v` (`coach-prompt@2`), `path_slug` and `attempt_id` (the open attempt a recorded D27 assist is on, a bare practice id) on both rows of a turn. |
| `message_quota_day` | PK (`account_id`, `day` — the **UTC** date); `n` | M1b (`00008`, m1-07): L18's durable daily count, ≤ 300 messages per account per UTC day, bumped by one atomic conditional upsert (`… DO UPDATE SET n = n + 1 WHERE n < 300 RETURNING n`; no row ⇒ cap reached). The per-minute bucket and the stream counter are in process (services.md, L24). |

**Erase (by `account_id`).** `coach.message_quota_day` and `coach.key_default` are per-user rows keyed by
`account_id`, like `api_key_config`, `coach_thread` (and its messages) and `outbox`; an account erase must
delete them too. [l-01](../v2/sprints/sprint-l-01.md)'s coach delete list doesn't name these two yet
(m1-07 hand-off in `docs/v2/status.md`).

---

## Ownership rules (recap)

- Each schema has **one** owning service; **no** cross-schema reads/writes/FKs.
- Cross-service references are stored as **bare ids** (soft references), resolved via API or events.
- Screens needing several contexts are composed in the **gateway** or read from **`assessment`
  projections** — never via cross-schema SQL.
- See [`events.md`](events.md) for how writes in one schema propagate to others.
- **Expand / contract** ([ADR-0034 §3](../adr/0034-v2-release-labelling-gating-and-rollback.md)): an expand
  migration holds only nullable or constant-default columns, new tables and `CREATE INDEX CONCURRENTLY`
  under `-- +goose NO TRANSACTION`, with set-based idempotent backfills; writers dual-write and readers
  read `COALESCE(new, old)` until the contract drops the old shape one tag later.
- **Migration markers** (`hack/lint-migrations.sh`, CI `go` job and `make lint`; m1-02). Every goose
  file newer than v1.5.2 (`hack/migrations-baseline.txt` lists the exempt ones) is classified from its
  `+goose Up` section: a line with `DROP …` (anything but `DROP NOT NULL` — so `DROP COLUMN`/`TABLE`/
  `CONSTRAINT`/`DEFAULT`, a bare column drop and **every** `DROP INDEX`), `SET NOT NULL`, **every**
  `ALTER [COLUMN] … TYPE`, or `RENAME` is a **contract** statement.
  - `-- xlearn:contract floor=vX.Y.Z` — required on a file with a contract statement (its rollback
    floor; record it in `docs/v2/status.md`), and an error on a file without one.
  - `-- xlearn:relax <reason>` — on a contract statement's own line: a reviewed relaxation (e.g. m1-02's
    mock CHECK swap, which drops the old CHECK only after a weaker one is validated). The reason is
    required; the marker on a line without a contract statement is an error.
