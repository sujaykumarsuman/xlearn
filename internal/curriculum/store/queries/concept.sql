-- name: GetConcept :one
SELECT slug, path_slug, title, body_md, when_to_use_md, code_template
FROM curriculum.concept
WHERE slug = $1;

-- name: ListConceptsByWeek :many
SELECT c.slug, c.title
FROM curriculum.concept c
JOIN curriculum.week_concept wc ON wc.concept_id = c.id
JOIN curriculum.week w          ON w.id = wc.week_id
WHERE w.path_slug = $1 AND w.n = $2
ORDER BY c.title;

-- name: UpsertConcept :one
INSERT INTO curriculum.concept (path_slug, slug, title, body_md, when_to_use_md, code_template)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (slug) DO UPDATE SET
    path_slug      = EXCLUDED.path_slug,
    title          = EXCLUDED.title,
    body_md        = EXCLUDED.body_md,
    when_to_use_md = EXCLUDED.when_to_use_md,
    code_template  = EXCLUDED.code_template
RETURNING id;

-- name: LinkWeekConcept :exec
INSERT INTO curriculum.week_concept (week_id, concept_id)
VALUES ($1, $2)
ON CONFLICT (week_id, concept_id) DO NOTHING;
