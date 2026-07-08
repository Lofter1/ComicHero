-- +goose Up
ALTER TABLE user_comics ADD COLUMN read_at TEXT NOT NULL DEFAULT '';

UPDATE user_comics
SET read_at = CURRENT_TIMESTAMP
WHERE read = 1 AND read_at = '';

-- +goose Down
CREATE TABLE user_comics_without_read_at (
    comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (comic_id, user_id)
);

INSERT INTO user_comics_without_read_at (comic_id, user_id, read)
SELECT comic_id, user_id, read
FROM user_comics;

DROP TABLE user_comics;

ALTER TABLE user_comics_without_read_at RENAME TO user_comics;
