-- lint-expect: 1
-- The m1-02 reader shape m1-03 retires: COALESCE over the v1 /35 column.

-- name: MockAggregate :one
SELECT COALESCE(MAX(COALESCE(total, total_35)), 0)::int AS best
FROM assessment.mock_session
WHERE account_id = $1;
