-- lint-expect: 1
-- A contract statement (a bare column drop, no COLUMN keyword) with no floor marker.

-- +goose Up
ALTER TABLE demo.thing
    DROP total_35;

-- +goose Down
ALTER TABLE demo.thing ADD COLUMN IF NOT EXISTS total_35 integer;
