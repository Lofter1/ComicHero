-- +goose Up
ALTER TABLE reading_orders ADD COLUMN is_public INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_reading_orders_is_public ON reading_orders(is_public);

-- +goose Down
DROP INDEX IF EXISTS idx_reading_orders_is_public;
ALTER TABLE reading_orders DROP COLUMN is_public;
