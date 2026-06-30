-- +goose Up
ALTER TABLE reading_order_comics ADD COLUMN tags TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE reading_order_comics DROP COLUMN tags;
