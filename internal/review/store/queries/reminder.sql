-- name: InsertReminder :one
-- Write one in-app reminder (the notifications worker, inside the dedupe tx).
INSERT INTO review.reminder (account_id, kind, due_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListDueReminders :many
-- An account's undelivered reminders that are due (due_at <= now), soonest-due first,
-- capped — the Dashboard "revisions due today" surface (GET /dashboard).
SELECT * FROM review.reminder
WHERE account_id = $1 AND delivered_at IS NULL AND due_at <= now()
ORDER BY due_at ASC
LIMIT $2;
