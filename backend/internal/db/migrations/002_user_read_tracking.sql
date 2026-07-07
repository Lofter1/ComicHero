-- +goose Up
CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL DEFAULT '',
    is_default    INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE app_settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE user_sessions (
    token      TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_comics (
    comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (comic_id, user_id)
);

INSERT OR IGNORE INTO users (name, is_default)
VALUES ('Default', 1);

INSERT INTO user_comics (comic_id, user_id, read)
SELECT c.id, u.id, 1
FROM comics AS c
JOIN users AS u ON u.name = 'Default'
WHERE c.read = 1;

ALTER TABLE comics DROP COLUMN read;

-- +goose Down
ALTER TABLE comics ADD COLUMN read INTEGER NOT NULL DEFAULT 0;

UPDATE comics
SET read = 1
WHERE id IN (
    SELECT comic_id
    FROM user_comics
    WHERE read = 1
);

DROP TABLE user_comics;
DROP TABLE user_sessions;
DROP TABLE app_settings;
DROP TABLE users;
