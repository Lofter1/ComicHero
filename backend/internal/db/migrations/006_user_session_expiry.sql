-- +goose Up
ALTER TABLE user_sessions ADD COLUMN expires_at TEXT NOT NULL DEFAULT '';

UPDATE user_sessions
SET expires_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+30 days')
WHERE expires_at = '';

CREATE INDEX idx_user_sessions_expires_at
ON user_sessions(expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_user_sessions_expires_at;
ALTER TABLE user_sessions DROP COLUMN expires_at;
