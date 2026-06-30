-- +goose Up
ALTER TABLE reading_orders ADD COLUMN image TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE reading_orders DROP COLUMN image;
