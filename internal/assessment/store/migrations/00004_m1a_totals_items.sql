-- v2 M1a EXPAND (sprint m1-02; ADR-0034 §3, t1 §4 assessment): the course-agnostic mock
-- shape lands BESIDE the v1 columns, which M1c (m1-08) drops.
--
--   * mock_session gains path_slug, rubric_id + rubric_snapshot (the rubric the session
--     is scored against), total / max_total (any rubric width, not just /35) and
--     scored_by. difficulty becomes write-optional (a course may have none).
--   * mock_session_item is the ordered item list that supersedes problem_id / set_id:
--     one backfilled row per session (ordinal 1). v1 never pins an item — problemId is
--     optional setup input and '' means a mixed set — so a '' session still gets its row,
--     with item_id NULL ("no pinned item").
--   * outbox.account_id is erase prep (ADR-0027 §6).
--
-- The one constraint change of M1a: the v1 table CHECK (status = 'scored') =
-- (total_35 IS NOT NULL) would reject a v1.7.0 scored row that no longer writes total_35.
-- It is swapped for the same coupling on COALESCE(total, total_35): added NOT VALID,
-- validated, and only then is the old CHECK dropped, on a line marked xlearn:relax
-- (hack/lint-migrations.sh). v1.5.2 writes total_35 on every scored row and never
-- writes total, so it satisfies the new CHECK and the R-b to v1.5.2 stays safe.
--
-- The rubric snapshot literal is curriculum/courses/dsa/course.json's mock.rubric
-- (id dsa-mock@1, 7 dims x 1..5); TestManifestGoldenMirror pins it to the manifest.
-- Backfills are set-based and idempotent (WHERE ... IS NULL / ON CONFLICT DO NOTHING).

-- +goose Up

-- +goose StatementBegin
ALTER TABLE assessment.mock_session
    ADD COLUMN IF NOT EXISTS path_slug text NOT NULL DEFAULT 'dsa',
    ADD COLUMN IF NOT EXISTS rubric_id text,
    ADD COLUMN IF NOT EXISTS rubric_snapshot jsonb,
    ADD COLUMN IF NOT EXISTS total integer,
    ADD COLUMN IF NOT EXISTS max_total integer,
    ADD COLUMN IF NOT EXISTS scored_by text
        CONSTRAINT mock_session_scored_by_check CHECK (scored_by IN ('self', 'ai-byo')),
    ALTER COLUMN difficulty DROP NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE assessment.mock_session
SET rubric_id       = 'dsa-mock@1',
    rubric_snapshot = '{"id": "dsa-mock@1", "dims": [{"id": "communication", "label": "Communication"}, {"id": "problem_understanding", "label": "Problem understanding"}, {"id": "brute_force", "label": "Brute force"}, {"id": "optimisation", "label": "Optimisation"}, {"id": "code_quality", "label": "Code quality"}, {"id": "edge_cases", "label": "Edge cases"}, {"id": "complexity", "label": "Complexity"}], "scale": [1, 5]}'::jsonb,
    max_total       = 35
WHERE rubric_id IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE assessment.mock_session
SET total = total_35, scored_by = 'self'
WHERE status = 'scored' AND total IS NULL AND total_35 IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE assessment.mock_session
    ADD CONSTRAINT mock_session_scored_total_check
        CHECK ((status = 'scored') = (COALESCE(total, total_35) IS NOT NULL)) NOT VALID;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE assessment.mock_session VALIDATE CONSTRAINT mock_session_scored_total_check;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE assessment.mock_session DROP CONSTRAINT IF EXISTS mock_session_check; -- xlearn:relax replaced by the weaker mock_session_scored_total_check
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS assessment.mock_session_item (
    session_id    uuid    NOT NULL REFERENCES assessment.mock_session (id) ON DELETE CASCADE,
    ordinal       integer NOT NULL CHECK (ordinal >= 1),
    item_id       text,
    path_slug     text    NOT NULL DEFAULT 'dsa',
    contract_hash text,
    PRIMARY KEY (session_id, ordinal)
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO assessment.mock_session_item (session_id, ordinal, item_id, path_slug)
SELECT id, 1, NULLIF(problem_id, ''), path_slug
FROM assessment.mock_session
ON CONFLICT (session_id, ordinal) DO NOTHING;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE assessment.outbox ADD COLUMN IF NOT EXISTS account_id uuid;
-- +goose StatementEnd

-- Every v1 envelope carries a uuid account_id; the regex guard keeps a malformed row
-- from failing the migration (it just stays NULL).
-- +goose StatementBegin
UPDATE assessment.outbox
SET account_id = (payload_json ->> 'account_id')::uuid
WHERE account_id IS NULL
  AND payload_json ->> 'account_id' ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';
-- +goose StatementEnd

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
-- +goose StatementBegin
ALTER TABLE assessment.outbox DROP COLUMN IF EXISTS account_id;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS assessment.mock_session_item;
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE assessment.mock_session SET total_35 = COALESCE(total_35, total) WHERE status = 'scored';
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE assessment.mock_session
    ADD CONSTRAINT mock_session_check CHECK ((status = 'scored') = (total_35 IS NOT NULL));
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE assessment.mock_session DROP CONSTRAINT IF EXISTS mock_session_scored_total_check;
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE assessment.mock_session SET difficulty = 'med' WHERE difficulty IS NULL;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE assessment.mock_session
    ALTER COLUMN difficulty SET NOT NULL,
    DROP COLUMN IF EXISTS scored_by,
    DROP COLUMN IF EXISTS max_total,
    DROP COLUMN IF EXISTS total,
    DROP COLUMN IF EXISTS rubric_snapshot,
    DROP COLUMN IF EXISTS rubric_id,
    DROP COLUMN IF EXISTS path_slug;
-- +goose StatementEnd
