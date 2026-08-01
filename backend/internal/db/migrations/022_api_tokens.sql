-- +goose Up
CREATE TABLE api_tokens (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT    NOT NULL,
    token_prefix TEXT    NOT NULL,
    token_hash   TEXT    NOT NULL,
    scopes       TEXT    NOT NULL DEFAULT '',
    created_at   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TEXT    NOT NULL DEFAULT '',
    expires_at   TEXT    NOT NULL DEFAULT '',
    revoked_at   TEXT    NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX idx_api_tokens_token_hash ON api_tokens(token_hash);
CREATE INDEX idx_api_tokens_user_id ON api_tokens(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_api_tokens_user_id;
DROP INDEX IF EXISTS idx_api_tokens_token_hash;
DROP TABLE api_tokens;
