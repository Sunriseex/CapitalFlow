-- +goose Up
CREATE TABLE interest_accruals (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    rule_id UUID NOT NULL REFERENCES interest_rules(id) ON DELETE CASCADE,
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    accrual_date DATE NOT NULL,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    balance_minor BIGINT NOT NULL CHECK (balance_minor > 0),
    annual_rate_bps BIGINT NOT NULL CHECK (annual_rate_bps > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (account_id, accrual_date, rule_id)
);

CREATE INDEX interest_accruals_account_id_accrual_date_idx ON interest_accruals (account_id, accrual_date);

-- +goose Down
DROP TABLE IF EXISTS interest_accruals;
