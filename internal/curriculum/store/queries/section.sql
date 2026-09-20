-- name: ListSectionsByProblem :many
-- All content sections for a problem, ordered by stage (attempt -> hint -> solution)
-- then position. S05 will filter to only the user's unlocked stages; this sprint
-- returns the full content set (the shape is already stage-keyed).
SELECT stage, kind, "order", body_md, code
FROM curriculum.problem_section
WHERE problem_id = $1
ORDER BY
    CASE stage WHEN 'attempt' THEN 0 WHEN 'hint' THEN 1 WHEN 'solution' THEN 2 ELSE 3 END,
    "order";

-- name: UpsertSection :exec
INSERT INTO curriculum.problem_section (problem_id, stage, kind, "order", body_md, code)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (problem_id, stage, "order") DO UPDATE SET
    kind    = EXCLUDED.kind,
    body_md = EXCLUDED.body_md,
    code    = EXCLUDED.code;
