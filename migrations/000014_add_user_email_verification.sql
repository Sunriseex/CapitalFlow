-- +goose Up
ALTER TABLE users
    ADD COLUMN email_verified_at TIMESTAMPTZ,
    ADD COLUMN email_verification_token_hash TEXT,
    ADD COLUMN email_verification_sent_at TIMESTAMPTZ;

CREATE INDEX users_email_verification_token_hash_idx
    ON users (email_verification_token_hash)
    WHERE email_verification_token_hash IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS users_email_verification_token_hash_idx;

ALTER TABLE users
    DROP COLUMN IF EXISTS email_verification_sent_at,
    DROP COLUMN IF EXISTS email_verification_token_hash,
    DROP COLUMN IF EXISTS email_verified_at;
