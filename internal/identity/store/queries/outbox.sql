-- name: InsertOutbox :exec
INSERT INTO identity.outbox (event_id, subject, payload_json)
VALUES ($1, $2, $3);

-- name: ListUnsentOutbox :many
SELECT * FROM identity.outbox
WHERE sent_at IS NULL
ORDER BY created_at
LIMIT $1;

-- name: MarkOutboxSent :exec
UPDATE identity.outbox
SET sent_at = now()
WHERE event_id = $1;
