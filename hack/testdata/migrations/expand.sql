-- lint-expect: 0
-- An expand-only file: new nullable / constant-default columns, a DROP NOT NULL (a
-- relaxation by definition), a new table and a CONCURRENTLY index. Comments that
-- mention DROP COLUMN or SET NOT NULL are ignored.

-- +goose Up
ALTER TABLE demo.thing
    ADD COLUMN IF NOT EXISTS path_slug text NOT NULL DEFAULT 'dsa',
    ADD COLUMN IF NOT EXISTS dropped_at timestamptz,
    ALTER COLUMN legacy DROP NOT NULL;
CREATE TABLE IF NOT EXISTS demo.item (id uuid PRIMARY KEY);
UPDATE demo.thing SET path_slug = 'dsa' WHERE path_slug IS NULL;

-- +goose Down
-- The Down section is ignored: it never runs in production.
ALTER TABLE demo.thing DROP COLUMN IF EXISTS path_slug, ALTER COLUMN legacy SET NOT NULL;
DROP TABLE IF EXISTS demo.item;
