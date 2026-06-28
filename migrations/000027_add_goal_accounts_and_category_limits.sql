-- +goose Up
ALTER TABLE financial_goals
    ADD COLUMN account_id UUID REFERENCES accounts(id) ON DELETE SET NULL;

CREATE INDEX financial_goals_account_id_idx ON financial_goals (account_id);

CREATE TABLE category_limits (
    id UUID PRIMARY KEY,
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    amount NUMERIC(38, 18) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_user_id, category_id, currency)
);

CREATE INDEX category_limits_owner_active_idx
    ON category_limits (owner_user_id, is_active, updated_at DESC);

ALTER TABLE transactions
    DROP CONSTRAINT transactions_transfer_id_fkey,
    ADD CONSTRAINT transactions_transfer_id_fkey
        FOREIGN KEY (transfer_id) REFERENCES transfers(id) ON DELETE NO ACTION
        DEFERRABLE INITIALLY DEFERRED;

-- +goose Down
ALTER TABLE transactions
    DROP CONSTRAINT transactions_transfer_id_fkey,
    ADD CONSTRAINT transactions_transfer_id_fkey
        FOREIGN KEY (transfer_id) REFERENCES transfers(id) ON DELETE RESTRICT;
DROP TABLE IF EXISTS category_limits;
DROP INDEX IF EXISTS financial_goals_account_id_idx;
ALTER TABLE financial_goals DROP COLUMN IF EXISTS account_id;
