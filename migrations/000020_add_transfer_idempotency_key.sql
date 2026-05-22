-- +goose Up
ALTER TABLE transfers
    ADD COLUMN idempotency_key TEXT;

CREATE UNIQUE INDEX transfers_user_id_idempotency_key_idx
    ON transfers (user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';

-- +goose Down
DROP INDEX IF EXISTS transfers_user_id_idempotency_key_idx;

ALTER TABLE transfers
    DROP COLUMN IF EXISTS idempotency_key;
