-- name: InsertOutbox :exec
-- account_id is erase prep (m1-02, ADR-0027 §6): the envelope's account, as a column.
INSERT INTO practice.outbox (event_id, subject, payload_json, account_id)
VALUES ($1, $2, $3, $4);

-- name: ListUnsentOutbox :many
SELECT * FROM practice.outbox
WHERE sent_at IS NULL
ORDER BY created_at
LIMIT $1;

-- name: MarkOutboxSent :exec
UPDATE practice.outbox
SET sent_at = now()
WHERE event_id = $1;
