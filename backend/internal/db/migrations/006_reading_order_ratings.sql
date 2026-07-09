-- +goose Up
ALTER TABLE reading_orders ADD COLUMN rating REAL NOT NULL DEFAULT 0;
ALTER TABLE reading_orders ADD COLUMN rating_count INTEGER NOT NULL DEFAULT 0;
CREATE INDEX idx_reading_orders_rating
ON reading_orders(rating);

-- +goose Down
DROP INDEX IF EXISTS idx_reading_orders_rating;
CREATE TABLE reading_orders_without_ratings (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    name                   TEXT    NOT NULL,
    description            TEXT    NOT NULL DEFAULT '',
    favorite               INTEGER NOT NULL DEFAULT 0,
    metron_reading_list_id INTEGER,
    image                  TEXT    NOT NULL DEFAULT '',
    author_user_id         INTEGER REFERENCES users(id) ON DELETE SET NULL
);
INSERT INTO reading_orders_without_ratings (
    id,
    name,
    description,
    favorite,
    metron_reading_list_id,
    image,
    author_user_id
)
SELECT
    id,
    name,
    description,
    favorite,
    metron_reading_list_id,
    image,
    author_user_id
FROM reading_orders;
DROP TABLE reading_orders;
ALTER TABLE reading_orders_without_ratings RENAME TO reading_orders;
CREATE UNIQUE INDEX idx_reading_orders_metron_reading_list_id
ON reading_orders(metron_reading_list_id)
WHERE metron_reading_list_id IS NOT NULL;
CREATE INDEX idx_reading_orders_author_user_id
ON reading_orders(author_user_id);
