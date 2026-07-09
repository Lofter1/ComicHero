-- +goose Up
ALTER TABLE users ADD COLUMN email_verified_at TEXT NOT NULL DEFAULT '';

UPDATE users
SET email_verified_at = CURRENT_TIMESTAMP
WHERE email_verified_at = ''
  AND (
    is_default = 1
    OR id = (SELECT MIN(id) FROM users)
  );

CREATE TABLE user_email_verifications (
    token_hash TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT    NOT NULL,
    used_at    TEXT    NOT NULL DEFAULT '',
    created_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_email_verifications_user_expires
ON user_email_verifications(user_id, expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_user_email_verifications_user_expires;
DROP TABLE user_email_verifications;

ALTER TABLE users DROP COLUMN email_verified_at;
