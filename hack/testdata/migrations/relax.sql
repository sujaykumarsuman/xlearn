-- lint-expect: 0
-- A reviewed relaxation: the old CHECK is dropped only after a weaker one is validated,
-- and the drop's own line carries the marker, so the file stays expand.

-- +goose Up
ALTER TABLE demo.mock ADD CONSTRAINT mock_total_check CHECK ((status = 'scored') = (COALESCE(total, total_35) IS NOT NULL)) NOT VALID;
ALTER TABLE demo.mock VALIDATE CONSTRAINT mock_total_check;
ALTER TABLE demo.mock DROP CONSTRAINT IF EXISTS mock_check; -- xlearn:relax replaced by the weaker mock_total_check

-- +goose Down
ALTER TABLE demo.mock DROP CONSTRAINT IF EXISTS mock_total_check;
