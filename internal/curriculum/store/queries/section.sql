-- name: ListSectionsByProblem :many
-- All content sections for a problem, ordered by stage (attempt -> hint -> solution)
-- then position. A withdrawn item (a takedown) serves no sections: its prose is
-- blanked while its title stays resolvable. language is the code section's language
-- ('' for prose; m1-03 serves it).
SELECT s.stage, s.kind, s."order", s.body_md, s.code, s.language
FROM curriculum.problem_section s
JOIN curriculum.problem p ON p.id = s.problem_id
WHERE s.problem_id = $1 AND p.status <> 'withdrawn'
ORDER BY
    CASE s.stage WHEN 'attempt' THEN 0 WHEN 'hint' THEN 1 WHEN 'solution' THEN 2 ELSE 3 END,
    s."order",
    s.language;

-- name: DeleteSectionsByProblem :exec
-- Every seed rewrites every item's sections (delete, then re-insert), inside the seed
-- transaction. content_hash is NOT a skip key: the v1.5.2 image (a valid R-b target
-- while the floor is "none") rewrites section bodies without touching content_hash, so
-- a hash skip would keep its text after a roll-forward.
DELETE FROM curriculum.problem_section WHERE problem_id = $1;

-- name: InsertSection :exec
INSERT INTO curriculum.problem_section (problem_id, stage, kind, "order", language, body_md, code)
VALUES ($1, $2, $3, $4, $5, $6, $7);
