-- name: InsertRubricScore :exec
-- Record one dimension's score (1..5) for a session. Called once per dimension inside
-- the scoring transaction; the UNIQUE (mock_session_id, dimension) is the backstop.
INSERT INTO assessment.rubric_score (mock_session_id, dimension, score)
VALUES ($1, $2, $3);

-- name: ListRubricScores :many
-- All rubric rows for a session (the results radar + per-dimension meters).
SELECT dimension, score
FROM assessment.rubric_score
WHERE mock_session_id = $1
ORDER BY created_at, id;
