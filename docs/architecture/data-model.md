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
| `account` | `id`, `display_name`, `email`, `timezone`, `study_budget_json`, `reminders_json`, `created_at` | The `Account` entity (profile/budget/timezone/reminders). |
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
| `user_problem_state` | `id`, `account_id`, `problem_id`, `status` (`locked`/`available`/`attempting`/`solved`), `first_solved_at`, `current_touch`, `last_outcome`, `unique(account_id,problem_id)` | The `UserProblemState` entity. |
| `attempt` | `id`, `user_problem_state_id`, `stage_reached`, `started_at`, `ended_at` | One attempt run through the stages. |
| `stage_event` | `id`, `attempt_id`, `stage`, `entered_at`, `unlocked_from` | Audit of stage gating. |
| `timer` | `id`, `attempt_id`, `kind` (`attempt`15m/`hint`10m), `deadline_at`, `expired` | **Server-authoritative** countdown ([R-PF3](../prd/xlearn-prd.md#61-guided-problem-flow-gated-stages)). |
| `outcome` | `id`, `attempt_id`, `value` (`clean`/`rough`/`assisted`/`miss`), `revealed_early`, `logged_at` | Below-`clean` ⇒ event that opens a mistake. |
| `outbox` | `event_id`, `subject`, `payload_json`, `created_at`, `sent_at` | Outbox relay to NATS. |

## schema `review`

| Table | Key columns | Notes |
|-------|-------------|-------|
| `revision_item` | `id`, `account_id`, `problem_id`, `touch_level` (1..5 = Day 1/3/7/21/45), `due_date`, `surfaced_at`, `status` (`pending`/`passed`/`failed`) | The five-touch queue ([R-SR1](../prd/xlearn-prd.md#63-five-touch-spaced-repetition)). |
| `touch_result` | `id`, `revision_item_id`, `named_pattern_secs`, `solved_in_timer`, `stated_complexity`, `auto_pass`, `scored_at` | Auto-score inputs ([R-SR2](../prd/xlearn-prd.md#63-five-touch-spaced-repetition)); Day 21/45 mock-mode flag. |
| `mistake_entry` | `id`, `account_id`, `problem_id`, `pattern`, `mistake`, `root_cause`, `insight`, `category` (8-enum), `revisit_date`, `status` (`open`/`closed`), `revisit_count` | The `MistakeEntry` entity ([R-MJ1](../prd/xlearn-prd.md#64-mistake-journal)). |
| `weak_area_snapshot` | `id`, `account_id`, `week_of`, `top_category`, `counts_json` | Weekly weak-area ([R-MJ3](../prd/xlearn-prd.md#64-mistake-journal)). |
| `reminder` | `id`, `account_id`, `kind`, `due_at`, `delivered_at` | Notifications worker (v1 in-app). |
| `outbox` | … | as above. |

## schema `assessment`

| Table | Key columns | Notes |
|-------|-------------|-------|
| `mock_session` | `id`, `account_id`, `set_id`, `problem_id`, `date`, `total_35`, `notes` | The `MockSession` entity. |
| `rubric_score` | `id`, `mock_session_id`, `dimension` (7-enum), `score` (1..5) | 7 dims × 1–5 = /35 ([R-MK2](../prd/xlearn-prd.md#65-mock-interview)). |
| `proj_coverage` | `account_id`, `problem_id`, `solved`, `first_solved_at`, `level1_schedules` | Progress projection — the solved set + ladder anchors. **Keyed per (account, problem)**, not per week — the gateway maps problem → week/phase from curriculum ([ADR-0018](../adr/0018-progress-projection-grain-and-rebuild.md)). |
| `proj_heatmap` | `account_id`, `activity_date`, `solves`, `reviews` | Revision-activity heatmap (per UTC day). |
| `proj_mastery` | `account_id`, `problem_id`, `best_outcome`, `best_rank`, `clean_solves`, `solve_count` | Per-problem solve quality; the gateway rolls it up **by pattern**. |
| `proj_outcome_mix` | `account_id`, `outcome`, `cnt` | First-solve outcome mix. |
| `inbox` | `event_id`, `consumed_at` | Idempotency/dedupe for consumed events. |

> The `proj_*` tables are keyed at the **event grain** (per problem / day / outcome), because the
> practice/review events carry only a bare `problem_id` — the by-week / by-phase / by-pattern
> roll-ups are composed in the gateway from curriculum, keeping the projections a pure function of
> the event log ([ADR-0018](../adr/0018-progress-projection-grain-and-rebuild.md)).

## schema `coach`

| Table | Key columns | Notes |
|-------|-------------|-------|
| `api_key_config` | `id`, `account_id`, `provider`, `enc_key` (ciphertext), `enc_data_key` (wrapped), `masked_key`, `default_model`, `enabled` | Envelope-encrypted; only `masked_key` ever leaves the service ([ADR-0007](../adr/0007-ai-coach-byo-key-and-secrets.md)). |
| `coach_thread` | `id`, `account_id`, `page_context`, `created_at` | One thread per page context. |
| `coach_message` | `id`, `thread_id`, `role`, `content`, `created_at` | Chat history. |

---

## Ownership rules (recap)

- Each schema has **one** owning service; **no** cross-schema reads/writes/FKs.
- Cross-service references are stored as **bare ids** (soft references), resolved via API or events.
- Screens needing several contexts are composed in the **gateway** or read from **`assessment`
  projections** — never via cross-schema SQL.
- See [`events.md`](events.md) for how writes in one schema propagate to others.
