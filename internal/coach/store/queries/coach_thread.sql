-- name: UpsertThread :one
-- Get-or-create the thread for (account, page context), labelled with its course
-- (path_slug; NULL for an account-wide context). The DO UPDATE always fires, so the
-- existing row's id comes back via RETURNING on a conflict and concurrent first-messages
-- on the same page can't create duplicate threads (UNIQUE(account_id, page_context)). It
-- keeps an existing path_slug and fills a NULL one (a thread written before m1-03).
INSERT INTO coach.coach_thread AS t (account_id, page_context, path_slug)
VALUES ($1, $2, $3)
ON CONFLICT (account_id, page_context) DO UPDATE
SET path_slug = COALESCE(t.path_slug, EXCLUDED.path_slug)
RETURNING id;

-- name: GetThread :one
-- Resolve an existing thread id for (account, page context). ErrNoRows when the account
-- has never chatted on that page (GET /coach/thread returns empty history).
SELECT id FROM coach.coach_thread
WHERE account_id = $1 AND page_context = $2;
