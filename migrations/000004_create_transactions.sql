-- +goose Up
CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    related_account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    type TEXT NOT NULL CHECK (type IN ('initial_balance', 'income', 'expense', 'transfer_in', 'transfer_out', 'interest_income', 'adjustment')),
    amount_minor BIGINT NOT NULL,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    description TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transactions_amount_sign_check CHECK (
        (type = 'adjustment' AND amount_minor <> 0)
        OR
        (type <> 'adjustment' AND amount_minor > 0)
    )
);

CREATE INDEX transactions_account_id_occurred_at_idx ON transactions (account_id, occurred_at);
CREATE INDEX transactions_type_idx ON transactions (type);

-- +goose Down
DROP TABLE IF EXISTS transactions;
