-- +goose Up
ALTER TABLE reading_orders ADD COLUMN author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;

UPDATE reading_orders
SET author_user_id = (
    SELECT id
    FROM users
    WHERE is_default = 1 OR name = 'Default'
    ORDER BY is_default DESC, id
    LIMIT 1
)
WHERE author_user_id IS NULL;

CREATE INDEX idx_reading_orders_author_user_id
ON reading_orders(author_user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_reading_orders_author_user_id;
ALTER TABLE reading_orders DROP COLUMN author_user_id;
