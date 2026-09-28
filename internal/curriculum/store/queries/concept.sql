-- name: GetConcept :one
-- code_template is nullable since 00002 but dual-written from templates.go by the seed.
SELECT slug, path_slug, title, body_md, when_to_use_md,
       COALESCE(code_template, templates ->> 'go', '')::text AS code_template
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
-- Keyed on (path_slug, slug) (00003's unique), so a concept is never re-parented. The v1
-- UNIQUE(slug) stays until M1c, so a slug is still global until then. Dual-writes
-- code_template from templates.go.
INSERT INTO curriculum.concept (path_slug, slug, title, body_md, when_to_use_md, code_template, templates)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (path_slug, slug) DO UPDATE SET
    title          = EXCLUDED.title,
    body_md        = EXCLUDED.body_md,
    when_to_use_md = EXCLUDED.when_to_use_md,
    code_template  = EXCLUDED.code_template,
    templates      = EXCLUDED.templates
RETURNING id;

-- name: DeleteMissingConcepts :exec
-- Delete-missing per course (content-only table; week_concept rows cascade).
DELETE FROM curriculum.concept
WHERE path_slug = sqlc.arg(path_slug) AND NOT (slug = ANY(sqlc.arg(keep)::text[]));

-- name: DeleteWeekConceptsByPath :exec
-- The seed relinks a course's week <-> concept pairs from scratch on every run.
DELETE FROM curriculum.week_concept wc
USING curriculum.week w
WHERE wc.week_id = w.id AND w.path_slug = $1;

-- name: LinkWeekConcept :exec
INSERT INTO curriculum.week_concept (week_id, concept_id)
VALUES ($1, $2)
ON CONFLICT (week_id, concept_id) DO NOTHING;
