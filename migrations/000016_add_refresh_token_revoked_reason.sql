-- +goose Up
ALTER TABLE refresh_tokens
ADD COLUMN revoked_reason text;

-- +goose Down
ALTER TABLE refresh_tokens
DROP COLUMN IF EXISTS revoked_reason;
