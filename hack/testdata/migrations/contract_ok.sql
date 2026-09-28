-- lint-expect: 0
-- xlearn:contract floor=v1.7.0
-- A contract file carrying its floor marker: drops, SET NOT NULL, a type change.

-- +goose Up
ALTER TABLE demo.thing DROP COLUMN IF EXISTS total_35;
ALTER TABLE demo.thing ALTER COLUMN path_slug SET NOT NULL;
ALTER TABLE demo.thing ALTER COLUMN score TYPE smallint;
DROP INDEX IF EXISTS demo.thing_old_idx;

-- +goose Down
ALTER TABLE demo.thing ADD COLUMN IF NOT EXISTS total_35 integer;
