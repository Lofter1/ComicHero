-- +goose Up
CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL DEFAULT '',
    is_default    INTEGER NOT NULL DEFAULT 0,
    is_admin      INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE app_settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE user_sessions (
    token      TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT    NOT NULL DEFAULT '',
    created_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_sessions_expires_at
ON user_sessions(expires_at);

CREATE TABLE user_comics (
    comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (comic_id, user_id)
);

INSERT OR IGNORE INTO users (name, is_default, is_admin)
VALUES ('Default', 1, 1);

INSERT INTO user_comics (comic_id, user_id, read)
SELECT c.id, u.id, 1
FROM comics AS c
JOIN users AS u ON u.name = 'Default'
WHERE c.read = 1;

ALTER TABLE comics DROP COLUMN read;

CREATE TABLE user_metron_permissions (
    user_id      INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    allowed      INTEGER NOT NULL DEFAULT 0,
    scopes       TEXT    NOT NULL DEFAULT '',
    hourly_limit INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_metron_request_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scope      TEXT    NOT NULL,
    endpoint   TEXT    NOT NULL,
    created_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_metron_request_log_user_created
ON user_metron_request_log(user_id, created_at);

INSERT INTO user_metron_permissions (user_id, allowed, scopes, hourly_limit)
SELECT id, 1, '*', 0
FROM users
WHERE is_admin = 1
ON CONFLICT(user_id) DO NOTHING;

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

CREATE TABLE user_invites (
    token              TEXT PRIMARY KEY,
    created_by_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    expires_at         TEXT NOT NULL DEFAULT '',
    used_at            TEXT NOT NULL DEFAULT '',
    created_at         TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_invites_unused_expiry
ON user_invites(used_at, expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_user_invites_unused_expiry;
DROP TABLE user_invites;

DROP INDEX IF EXISTS idx_reading_orders_author_user_id;
ALTER TABLE reading_orders DROP COLUMN author_user_id;

DROP INDEX idx_user_metron_request_log_user_created;
DROP TABLE user_metron_request_log;
DROP TABLE user_metron_permissions;

ALTER TABLE comics ADD COLUMN read INTEGER NOT NULL DEFAULT 0;

UPDATE comics
SET read = 1
WHERE id IN (
    SELECT comic_id
    FROM user_comics
    WHERE read = 1
);

DROP TABLE user_comics;
DROP INDEX IF EXISTS idx_user_sessions_expires_at;
DROP TABLE user_sessions;
DROP TABLE app_settings;
DROP TABLE users;
