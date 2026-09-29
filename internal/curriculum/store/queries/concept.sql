-- name: GetConcept :one
-- Keyed on (path_slug, slug) (00003's concept_path_slug_slug_key; m1-03): a concept
-- resolves only under its own course. Reads the per-language templates only; the v1
-- single-template field is derived from them in Go (the course's primary language), so
-- no reader touches the column M1c drops.
SELECT slug, path_slug, title, body_md, when_to_use_md, templates
FROM curriculum.concept
WHERE path_slug = $1 AND slug = $2;

-- name: ListConceptsByWeek :many
SELECT c.slug, c.title
FROM curriculum.concept c
JOIN curriculum.week_concept wc ON wc.concept_id = c.id
JOIN curriculum.week w          ON w.id = wc.week_id
WHERE w.path_slug = $1 AND w.n = $2
ORDER BY c.title;

-- name: UpsertConcept :one
-- Keyed on (path_slug, slug) (00003's unique), so a concept is never re-parented. The v1
-- UNIQUE(slug) stays until M1c, so a slug is still global until then. Since m1-03 the
-- seed writes templates only: the v1 single-template column (nullable since 00002) gets
-- its 00001 default ('') on a new row and keeps what an older seed wrote on an existing
-- one. A v1.6.0 reader (the R-b target, or a pod mid-rollout) reads it through COALESCE,
-- so it never fails on it, and a v1.6.0 boot's own seed rewrites it for every concept
-- that image carries. M1c drops the column.
INSERT INTO curriculum.concept (path_slug, slug, title, body_md, when_to_use_md, templates)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (path_slug, slug) DO UPDATE SET
    title          = EXCLUDED.title,
    body_md        = EXCLUDED.body_md,
    when_to_use_md = EXCLUDED.when_to_use_md,
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
