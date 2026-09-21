-- name: LockMistakeJournal :exec
-- Serialise all journal mutations for one (account, problem) across BOTH entry points
-- (the below-clean/fail opener and the clean-revisit incrementer) with a transaction
-- advisory lock, so a concurrent open can't collide with a close→open re-open on the
-- one-open-entry partial unique index. The salted string namespaces the key away from
-- goose's migration lock and other services' advisory locks. Re-acquiring it within the
-- same tx is a harmless no-op (advisory locks are re-entrant, released at commit).
SELECT pg_advisory_xact_lock(hashtextextended('review.mistake:' || sqlc.arg(account_id)::text || '|' || sqlc.arg(problem_id)::text, 0));

-- name: OpenMistake :one
-- Open a mistake for (account, problem). ON CONFLICT on the partial unique index
-- (one OPEN entry per account+problem) DO NOTHING, so a repeat below-clean solve or a
-- redelivered event never opens a duplicate: on conflict it returns no row
-- (pgx.ErrNoRows), which the caller reads as "already open — don't re-emit
-- mistake_opened". A closed prior entry does not conflict, so a recurring problem
-- opens a fresh entry.
INSERT INTO review.mistake_entry (account_id, problem_id, pattern, category, revisit_date, status, revisit_count)
VALUES ($1, $2, $3, $4, $5, 'open', 0)
ON CONFLICT (account_id, problem_id) WHERE status = 'open'
DO NOTHING
RETURNING *;

-- name: GetLatestMistake :one
-- The most-recent entry for (account, problem), open or closed — the Score fail path
-- reads this to decide open-vs-reopen-vs-reset.
SELECT * FROM review.mistake_entry
WHERE account_id = $1 AND problem_id = $2
ORDER BY created_at DESC
LIMIT 1;

-- name: ReopenMistake :one
-- Re-open a closed entry after a later fail (R-MJ4): status back to open, the clean
-- revisit count reset to 0, and revisit_date re-anchored to the fresh Day-1 schedule.
-- Pattern/category/root-cause/insight are preserved (the learner's classification
-- survives the re-open).
UPDATE review.mistake_entry
SET status = 'open', revisit_count = 0, revisit_date = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ResetCleanRevisitCount :one
-- A fail on an already-OPEN entry resets its clean-revisit streak to 0 (a fail never
-- counts toward the 2). No re-emit — the entry is already open.
UPDATE review.mistake_entry
SET revisit_count = 0, revisit_date = $3, updated_at = now()
WHERE account_id = $1 AND problem_id = $2 AND status = 'open'
RETURNING *;

-- name: IncrementCleanRevisit :one
-- A clean revisit (an auto-passed re-solve for account+problem) increments the open
-- entry's count; at 2 it closes (R-MJ4). Returns the new count + status so the caller
-- emits mistake_closed exactly on the close transition. At most one open row exists
-- (the partial unique index), so this affects a single entry; no open entry → no row.
UPDATE review.mistake_entry
SET revisit_count = revisit_count + 1,
    status = CASE WHEN revisit_count + 1 >= 2 THEN 'closed' ELSE status END,
    updated_at = now()
WHERE account_id = $1 AND problem_id = $2 AND status = 'open'
RETURNING id, problem_id, revisit_count, status;

-- name: ListMistakes :many
-- The journal for an account, newest first (GET /mistakes with no status filter).
SELECT * FROM review.mistake_entry
WHERE account_id = $1
ORDER BY created_at DESC;

-- name: ListMistakesByStatus :many
-- The journal filtered to open|closed (GET /mistakes?status=).
SELECT * FROM review.mistake_entry
WHERE account_id = $1 AND status = $2
ORDER BY created_at DESC;

-- name: GetMistake :one
-- One entry scoped to its owner (the account from the gateway-minted JWT).
SELECT * FROM review.mistake_entry
WHERE id = $1 AND account_id = $2;

-- name: CreateMistake :one
-- Manually create a journal entry (POST /mistakes). A plain insert: if the learner
-- already has an open entry for the problem the partial unique index rejects it
-- (mapped to 409 by the handler).
INSERT INTO review.mistake_entry (
    account_id, problem_id, pattern, mistake, root_cause, insight, category, revisit_date, status, revisit_count
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateMistake :one
-- Overwrite the editable fields of an entry (POST edit / PATCH). The handler loads the
-- current row, overlays the request's fields, and writes the full set — so this is a
-- straight assignment, no COALESCE ambiguity (category can be set back to NULL).
UPDATE review.mistake_entry
SET pattern = $3, mistake = $4, root_cause = $5, insight = $6,
    category = $7, status = $8, revisit_count = $9, revisit_date = $10,
    updated_at = now()
WHERE id = $1 AND account_id = $2
RETURNING *;

-- name: ListAccountsWithMistakes :many
-- Every account that has at least one mistake entry — the recompute job iterates these
-- to build a weak-area snapshot per account.
SELECT DISTINCT account_id FROM review.mistake_entry;

-- name: CountOpenMistakesByCategoryInRange :many
-- Per-category counts of an account's OPEN entries opened within [start, end) (the
-- week window in the account timezone, computed by the caller). Uncategorised entries
-- (category IS NULL) are excluded — they don't define a weak area until classified.
SELECT category, count(*) AS n
FROM review.mistake_entry
WHERE account_id = $1
  AND status = 'open'
  AND category IS NOT NULL
  AND created_at >= $2 AND created_at < $3
GROUP BY category;

-- name: ListOpenMistakesByCategory :many
-- The supporting entries behind the weak-area banner: an account's open entries in one
-- category, newest first (GET /weak-area).
SELECT * FROM review.mistake_entry
WHERE account_id = $1 AND status = 'open' AND category = $2
ORDER BY created_at DESC;
