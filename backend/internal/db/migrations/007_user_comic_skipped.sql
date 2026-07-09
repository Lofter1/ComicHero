-- +goose Up
ALTER TABLE user_comics ADD COLUMN skipped INTEGER NOT NULL DEFAULT 0;
CREATE INDEX idx_user_comics_user_skipped
ON user_comics(user_id, skipped);

-- +goose Down
DROP INDEX IF EXISTS idx_user_comics_user_skipped;
CREATE TABLE user_comics_without_skipped (
    comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read     INTEGER NOT NULL DEFAULT 0,
    read_at  TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (comic_id, user_id)
);
INSERT INTO user_comics_without_skipped (comic_id, user_id, read, read_at)
SELECT comic_id, user_id, read, read_at
FROM user_comics;
DROP TABLE user_comics;
ALTER TABLE user_comics_without_skipped RENAME TO user_comics;
