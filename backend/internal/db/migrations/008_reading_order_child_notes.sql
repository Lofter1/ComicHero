-- +goose Up
ALTER TABLE reading_order_children ADD COLUMN note TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE reading_order_children DROP COLUMN note;
