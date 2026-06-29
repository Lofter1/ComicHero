-- +goose Up
ALTER TABLE series ADD COLUMN metron_series_id INTEGER;
ALTER TABLE series ADD COLUMN publisher TEXT NOT NULL DEFAULT '';
ALTER TABLE series ADD COLUMN volume INTEGER NOT NULL DEFAULT 0;
ALTER TABLE series ADD COLUMN year_end INTEGER NOT NULL DEFAULT 0;
ALTER TABLE series ADD COLUMN issue_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE series ADD COLUMN description TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX idx_series_metron_series_id
ON series(metron_series_id)
WHERE metron_series_id IS NOT NULL;

-- +goose Down
DROP INDEX idx_series_metron_series_id;
