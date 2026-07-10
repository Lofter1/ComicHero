-- +goose NO TRANSACTION

-- +goose Up
PRAGMA foreign_keys = OFF;

CREATE TABLE users_new (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    name              TEXT    NOT NULL UNIQUE,
    password_hash     TEXT    NOT NULL DEFAULT '',
    is_default        INTEGER NOT NULL DEFAULT 0,
    is_admin          INTEGER NOT NULL DEFAULT 0,
    created_at        TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    email             TEXT    NOT NULL DEFAULT '',
    email_verified_at TEXT    NOT NULL DEFAULT ''
);

INSERT INTO users_new (
    id,
    name,
    password_hash,
    is_default,
    is_admin,
    created_at,
    email,
    email_verified_at
)
SELECT
    id,
    name,
    password_hash,
    is_default,
    is_admin,
    CASE WHEN created_at = '' THEN CURRENT_TIMESTAMP ELSE created_at END,
    email,
    email_verified_at
FROM users;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

CREATE UNIQUE INDEX idx_users_email
ON users(email)
WHERE email <> '';

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

CREATE TABLE users_old (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    name              TEXT    NOT NULL UNIQUE,
    password_hash     TEXT    NOT NULL DEFAULT '',
    is_default        INTEGER NOT NULL DEFAULT 0,
    is_admin          INTEGER NOT NULL DEFAULT 0,
    created_at        TEXT    NOT NULL DEFAULT '',
    email             TEXT    NOT NULL DEFAULT '',
    email_verified_at TEXT    NOT NULL DEFAULT ''
);

INSERT INTO users_old (
    id,
    name,
    password_hash,
    is_default,
    is_admin,
    created_at,
    email,
    email_verified_at
)
SELECT
    id,
    name,
    password_hash,
    is_default,
    is_admin,
    created_at,
    email,
    email_verified_at
FROM users;

DROP TABLE users;
ALTER TABLE users_old RENAME TO users;

CREATE UNIQUE INDEX idx_users_email
ON users(email)
WHERE email <> '';

PRAGMA foreign_keys = ON;
