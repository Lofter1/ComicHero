-- +goose Up
CREATE TABLE reading_order_ratings (
    reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
    user_id          INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating           REAL    NOT NULL,
    created_at       TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (reading_order_id, user_id),
    CHECK (rating >= 1 AND rating <= 5)
);
CREATE INDEX idx_reading_order_ratings_order
ON reading_order_ratings(reading_order_id);

-- +goose Down
DROP INDEX IF EXISTS idx_reading_order_ratings_order;
DROP TABLE reading_order_ratings;
