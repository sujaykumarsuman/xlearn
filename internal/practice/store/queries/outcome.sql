-- name: CreateOutcome :exec
INSERT INTO practice.outcome (attempt_id, value, revealed_early)
VALUES ($1, $2, $3);
