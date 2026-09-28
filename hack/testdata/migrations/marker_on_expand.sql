-- lint-expect: 1
-- xlearn:contract floor=v1.7.0
-- A floor marker on a file with no contract statement: the marker would raise the
-- rollback floor for nothing, so it is an error.

-- +goose Up
ALTER TABLE demo.thing ADD COLUMN IF NOT EXISTS path_slug text NOT NULL DEFAULT 'dsa';

-- +goose Down
ALTER TABLE demo.thing DROP COLUMN IF EXISTS path_slug;
