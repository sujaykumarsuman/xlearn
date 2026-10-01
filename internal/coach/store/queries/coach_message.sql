-- name: InsertMessage :one
-- Append one message to a thread, labelled with the thread's course (path_slug, copied
-- from the thread so the two never disagree; NULL for an account-wide thread). seq
-- (identity) orders it; created_at is the wall time. No row when the thread is missing.
INSERT INTO coach.coach_message (thread_id, role, content, path_slug)
SELECT t.id, sqlc.arg(role), sqlc.arg(content), t.path_slug
FROM coach.coach_thread t
WHERE t.id = sqlc.arg(thread_id)
RETURNING id, seq, role, content, created_at;

-- name: ListMessages :many
-- A thread's messages oldest-first (seq is the stable total order). Used both for
-- GET /coach/thread history and to build the provider request's prior turns.
SELECT id, role, content, created_at
FROM coach.coach_message
WHERE thread_id = $1
ORDER BY seq;

-- --- m1-10 (M1b task 4): per-turn provider usage and cost ---

-- name: InsertAssistantMessage :one
-- Append the coach's reply with the usage of the provider turn that produced it
-- (migration 00006). Every usage field is nullable and passed NULL when the stream
-- reported none: a reply still persists if the provider sent no usage frame, and v1 rows
-- keep NULL forever. est_cost_micros is the catalog price × tokens computed in Go and is
-- NULL for a custom model id (no published price) — "cost unknown" in the UI.
--
-- Separate from InsertMessage so the user's turn cannot accidentally be stamped with the
-- assistant turn's usage (which would double-count the month-to-date total).
INSERT INTO coach.coach_message (
    thread_id, role, content, path_slug,
    provider, model, input_tokens, output_tokens, est_cost_micros, stop_reason
)
SELECT t.id, sqlc.arg(role), sqlc.arg(content), t.path_slug,
       sqlc.narg(provider), sqlc.narg(model), sqlc.narg(input_tokens),
       sqlc.narg(output_tokens), sqlc.narg(est_cost_micros), sqlc.narg(stop_reason)
FROM coach.coach_thread t
WHERE t.id = sqlc.arg(thread_id)
RETURNING id, seq, role, content, created_at;

-- name: UsageMonthForAccount :one
-- Month-to-date spend on the account's own keys, for "This month on your keys" (AB01
-- F13). The window is the current UTC CALENDAR month — date_trunc on a timestamptz is
-- evaluated in the session TimeZone, so it is pinned to UTC here rather than inheriting
-- whatever the connection happens to carry; the UI labels it as a UTC month.
--
-- has_unknown_cost marks that at least one answered turn had no price (a custom model, or
-- a turn whose stream reported no usage), so the displayed total is a FLOOR and the UI
-- says custom models are not estimated. Display only; always an estimate.
-- Every aggregate is COALESCEd, bool_or included: over ZERO rows — a learner who has
-- just connected a key and not chatted yet — sum() and bool_or() both return NULL, and
-- sqlc types these columns as non-null, so an un-COALESCEd bool_or makes GET /keys fail
-- with "cannot scan NULL into *bool" for every brand-new account. (Found by the m1-10
-- compose rehearsal; the handler unit tests run against an in-memory fake and never
-- execute this SQL. TestUsageMonthOnAnEmptyAccount pins it.)
SELECT count(*)::bigint                                   AS messages,
       COALESCE(sum(m.input_tokens), 0)::bigint           AS input_tokens,
       COALESCE(sum(m.output_tokens), 0)::bigint          AS output_tokens,
       COALESCE(sum(m.est_cost_micros), 0)::bigint        AS est_cost_micros,
       -- The ::boolean cast is required, not decorative: without it sqlc cannot infer the
       -- COALESCE's type and generates `interface{}` for the field.
       COALESCE(bool_or(m.est_cost_micros IS NULL), false)::boolean AS has_unknown_cost
FROM coach.coach_message AS m
JOIN coach.coach_thread AS t ON t.id = m.thread_id
WHERE t.account_id = $1
  AND m.role = 'assistant'
  AND m.created_at >= date_trunc('month', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC';
