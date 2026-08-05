-- +goose Up
ALTER TABLE comics ADD COLUMN comic_vine_synced_at TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_comics_comic_vine_synced_at
ON comics(comic_vine_synced_at)
WHERE comic_vine_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_comics_comic_vine_synced_at;
ALTER TABLE comics DROP COLUMN comic_vine_synced_at;
