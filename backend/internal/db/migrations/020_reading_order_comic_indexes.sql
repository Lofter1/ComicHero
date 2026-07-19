-- +goose Up
CREATE INDEX IF NOT EXISTS idx_reading_order_comics_order_position
ON reading_order_comics(reading_order_id, position);

CREATE INDEX IF NOT EXISTS idx_reading_order_comics_comic_order
ON reading_order_comics(comic_id, reading_order_id);

-- +goose Down
DROP INDEX IF EXISTS idx_reading_order_comics_comic_order;
DROP INDEX IF EXISTS idx_reading_order_comics_order_position;
