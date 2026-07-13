-- +goose Up
ALTER TABLE users ADD COLUMN last_login_at TEXT NOT NULL DEFAULT '';

CREATE TABLE audit_events (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER REFERENCES users(id) ON DELETE SET NULL,
    method      TEXT NOT NULL,
    path        TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    occurred_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_events_occurred_at ON audit_events(occurred_at DESC);
CREATE INDEX idx_audit_events_user_occurred ON audit_events(user_id, occurred_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_audit_events_user_occurred;
DROP INDEX IF EXISTS idx_audit_events_occurred_at;
DROP TABLE audit_events;

CREATE TABLE users_without_last_login (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    name              TEXT NOT NULL UNIQUE,
    email             TEXT NOT NULL DEFAULT '',
    email_verified_at TEXT NOT NULL DEFAULT '',
    password_hash     TEXT NOT NULL DEFAULT '',
    is_default        INTEGER NOT NULL DEFAULT 0,
    is_admin          INTEGER NOT NULL DEFAULT 0,
    created_at        TEXT NOT NULL DEFAULT ''
);
INSERT INTO users_without_last_login SELECT id, name, email, email_verified_at, password_hash, is_default, is_admin, created_at FROM users;
DROP TABLE users;
ALTER TABLE users_without_last_login RENAME TO users;
CREATE UNIQUE INDEX idx_users_email ON users(email) WHERE email <> '';
