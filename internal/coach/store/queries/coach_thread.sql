-- name: UpsertThread :one
-- Get-or-create the thread for (account, page context). The no-op DO UPDATE makes the
-- existing row's id come back via RETURNING on a conflict, so concurrent first-messages
-- on the same page can't create duplicate threads (UNIQUE(account_id, page_context)).
INSERT INTO coach.coach_thread (account_id, page_context)
VALUES ($1, $2)
ON CONFLICT (account_id, page_context) DO UPDATE
SET page_context = EXCLUDED.page_context
RETURNING id;

-- name: GetThread :one
-- Resolve an existing thread id for (account, page context). ErrNoRows when the account
-- has never chatted on that page (GET /coach/thread returns empty history).
SELECT id FROM coach.coach_thread
WHERE account_id = $1 AND page_context = $2;
