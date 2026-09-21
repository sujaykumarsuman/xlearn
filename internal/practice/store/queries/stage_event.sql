-- name: CreateStageEvent :exec
INSERT INTO practice.stage_event (attempt_id, stage, unlocked_from)
VALUES ($1, $2, $3);

-- name: ListStageEvents :many
-- The stages entered for an attempt; the distinct content stages here are the
-- authoritative "unlocked stages" for the problem (R-PF1).
SELECT * FROM practice.stage_event
WHERE attempt_id = $1
ORDER BY entered_at;
