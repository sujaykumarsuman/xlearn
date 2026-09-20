-- name: ListPhasesByPath :many
SELECT "order", name, theme, week_from, week_to
FROM curriculum.phase
WHERE path_slug = $1
ORDER BY "order";

-- name: UpsertPhase :exec
INSERT INTO curriculum.phase (path_slug, "order", name, theme, week_from, week_to)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (path_slug, "order") DO UPDATE SET
    name      = EXCLUDED.name,
    theme     = EXCLUDED.theme,
    week_from = EXCLUDED.week_from,
    week_to   = EXCLUDED.week_to;
