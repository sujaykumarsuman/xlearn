-- v2 M1a EXPAND (sprint m1-09; ADR-0034 §3, t1 §4): the per-course content model's
-- columns land BESIDE the v1 ones. Additive only: nullable or constant-default columns,
-- backfills, and DROP NOT NULL on the columns M1c (m1-08) drops, so writers can stop
-- writing them in M1b. Nothing is dropped or tightened, so the v1.5.2 image (the R-b
-- target while the rollback floor is "none") still migrates-noop, seeds and serves:
-- its INSERTs omit the new columns (their defaults apply) and its upserts still target
-- the v1 uniques, which stay until M1c. The seed dual-writes the old columns from the
-- new fields, so no NULL exists before M1c.
--
-- The new composite uniques are built CONCURRENTLY in 00003 (NO TRANSACTION).

-- +goose Up

-- problem: role (backfilled from is_reinforcement), lifecycle status + retired_at,
-- links (backfilled from the two URL columns) and the item's content_hash (written by
-- the seed; canon.ContentHash).
-- +goose StatementBegin
ALTER TABLE curriculum.problem
    ADD COLUMN IF NOT EXISTS role text NOT NULL DEFAULT 'core'
        CONSTRAINT problem_role_check CHECK (role IN ('core', 'reinforcement', 'drill')),
    ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'live'
        CONSTRAINT problem_status_check CHECK (status IN ('live', 'retired', 'withdrawn')),
    ADD COLUMN IF NOT EXISTS retired_at timestamptz,
    ADD COLUMN IF NOT EXISTS links jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS content_hash text NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE curriculum.problem SET role = 'reinforcement' WHERE is_reinforcement;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE curriculum.problem
SET links =
      (CASE WHEN leetcode_url <> '' THEN jsonb_build_array(jsonb_build_object('kind', 'leetcode', 'url', leetcode_url))
            ELSE '[]'::jsonb END)
   || (CASE WHEN neetcode_url <> '' THEN jsonb_build_array(jsonb_build_object('kind', 'neetcode', 'url', neetcode_url))
            ELSE '[]'::jsonb END);
-- +goose StatementEnd

-- Write-optional for M1b; dropped in M1c (m1-08).
-- +goose StatementBegin
ALTER TABLE curriculum.problem
    ALTER COLUMN is_reinforcement DROP NOT NULL,
    ALTER COLUMN leetcode_url DROP NOT NULL,
    ALTER COLUMN neetcode_url DROP NOT NULL;
-- +goose StatementEnd

-- path: the course's item-id prefix (from its manifest; written by the seed). Nullable;
-- its unique index is built in 00003.
-- +goose StatementBegin
ALTER TABLE curriculum.path ADD COLUMN IF NOT EXISTS id_prefix text;
-- +goose StatementEnd

-- concept: per-language code templates (backfilled from code_template, which becomes
-- write-optional; dropped in M1c). The (path_slug, slug) unique is built in 00003; the
-- v1 UNIQUE(slug) stays until M1c.
-- +goose StatementBegin
ALTER TABLE curriculum.concept
    ADD COLUMN IF NOT EXISTS templates jsonb NOT NULL DEFAULT '{}'::jsonb,
    ALTER COLUMN code_template DROP NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE curriculum.concept SET templates = jsonb_build_object('go', code_template) WHERE code_template <> '';
-- +goose StatementEnd

-- problem_section: the language of a code section ('' for prose). The
-- (problem_id, stage, "order", language) unique is built in 00003; the v1
-- (problem_id, stage, "order") unique stays until M1c.
-- +goose StatementBegin
ALTER TABLE curriculum.problem_section ADD COLUMN IF NOT EXISTS language text NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE curriculum.problem_section SET language = 'go' WHERE kind = 'code';
-- +goose StatementEnd

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
-- +goose StatementBegin
UPDATE curriculum.problem SET is_reinforcement = coalesce(is_reinforcement, role = 'reinforcement'),
    leetcode_url = coalesce(leetcode_url, ''), neetcode_url = coalesce(neetcode_url, '');
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE curriculum.concept SET code_template = coalesce(code_template, templates ->> 'go', '');
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE curriculum.problem_section DROP COLUMN IF EXISTS language;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE curriculum.concept DROP COLUMN IF EXISTS templates, ALTER COLUMN code_template SET NOT NULL;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE curriculum.path DROP COLUMN IF EXISTS id_prefix;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE curriculum.problem
    ALTER COLUMN is_reinforcement SET NOT NULL,
    ALTER COLUMN leetcode_url SET NOT NULL,
    ALTER COLUMN neetcode_url SET NOT NULL,
    DROP COLUMN IF EXISTS content_hash,
    DROP COLUMN IF EXISTS links,
    DROP COLUMN IF EXISTS retired_at,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS role;
-- +goose StatementEnd
