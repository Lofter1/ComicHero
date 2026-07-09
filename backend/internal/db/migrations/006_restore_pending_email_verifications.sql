-- +goose Up
UPDATE users
SET email_verified_at = ''
WHERE id IN (
    SELECT user_id
    FROM user_email_verifications
    WHERE used_at = ''
);

-- +goose Down
-- No-op: this migration restores pending verification state after an earlier
-- overly broad backfill. There is no safe inverse.
