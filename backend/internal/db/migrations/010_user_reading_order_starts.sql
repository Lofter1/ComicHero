-- +goose Up
CREATE TABLE user_reading_orders (
    reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
    user_id          INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at       TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (reading_order_id, user_id)
);
CREATE INDEX idx_user_reading_orders_user_started
ON user_reading_orders(user_id, started_at);

-- +goose Down
DROP INDEX IF EXISTS idx_user_reading_orders_user_started;
DROP TABLE user_reading_orders;
