-- +goose Up
CREATE TABLE reading_order_sections (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
    position         INTEGER NOT NULL DEFAULT 0,
    title            TEXT    NOT NULL,
    description      TEXT    NOT NULL DEFAULT ''
);
CREATE INDEX idx_reading_order_sections_order_position
ON reading_order_sections(reading_order_id, position);

-- +goose Down
DROP INDEX IF EXISTS idx_reading_order_sections_order_position;
DROP TABLE IF EXISTS reading_order_sections;
