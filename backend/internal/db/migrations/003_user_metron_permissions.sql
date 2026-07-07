-- +goose Up
ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;

UPDATE users
SET is_admin = 1
WHERE is_default = 1
   OR id = (SELECT MIN(id) FROM users);

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

-- +goose Down
DROP INDEX idx_user_metron_request_log_user_created;
DROP TABLE user_metron_request_log;
DROP TABLE user_metron_permissions;
ALTER TABLE users DROP COLUMN is_admin;
