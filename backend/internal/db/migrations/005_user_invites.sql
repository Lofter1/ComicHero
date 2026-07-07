-- +goose Up
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
