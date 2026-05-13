-- +goose Up
ALTER TABLE refresh_tokens
ADD COLUMN revoked_reason TEXT;
UPDATE refresh_tokens
SET revoked_reason = 'rotated'
WHERE revoked_at IS NOT NULL
    AND revoked_reason IS NULL;
-- +goose Down
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS revoked_reason;