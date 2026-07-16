-- +goose Up
ALTER TABLE comics ADD COLUMN comic_vine_id INTEGER;
CREATE UNIQUE INDEX idx_comics_comic_vine_id
ON comics(comic_vine_id)
WHERE comic_vine_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_comics_comic_vine_id;
ALTER TABLE comics DROP COLUMN comic_vine_id;
