-- +goose Up
ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX idx_users_email
ON users(email)
WHERE email <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_users_email;

ALTER TABLE users DROP COLUMN email;
