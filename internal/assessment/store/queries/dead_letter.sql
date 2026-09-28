-- name: InsertDeadLetter :exec
-- Record an event whose handler failed its last delivery (mi-05, ADR-0035 §1.2). Ids
-- only. ON CONFLICT DO NOTHING: a replayed-then-dead-lettered event stays one row.
INSERT INTO assessment.event_dead_letter (event_id, subject, durable, err_class, stream_seq, at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (durable, event_id) DO NOTHING;

-- name: ListDeadLetters :many
-- The on-demand read (D34: no alerting): newest first, capped.
SELECT event_id, subject, durable, err_class, stream_seq, at
FROM assessment.event_dead_letter
ORDER BY at DESC, durable, event_id
LIMIT $1;
