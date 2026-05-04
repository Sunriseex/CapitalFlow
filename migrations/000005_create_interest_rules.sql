-- +goose Up
CREATE TABLE interest_rules (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    annual_rate_bps BIGINT NOT NULL CHECK (annual_rate_bps > 0),
    promo_rate_bps BIGINT CHECK (promo_rate_bps IS NULL OR promo_rate_bps > 0),
    promo_end_date DATE,
    accrual_frequency TEXT NOT NULL CHECK (accrual_frequency IN ('daily', 'monthly', 'end_of_term')),
    capitalization_frequency TEXT NOT NULL CHECK (capitalization_frequency IN ('daily', 'monthly', 'end_of_term', 'none')),
    day_count_convention TEXT NOT NULL CHECK (day_count_convention IN ('actual_365', 'actual_366', 'actual_actual')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    start_date DATE NOT NULL,
    end_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date IS NULL OR end_date >= start_date),
    CHECK (promo_end_date IS NULL OR promo_end_date >= start_date)
);

CREATE INDEX interest_rules_account_id_is_active_idx ON interest_rules (account_id, is_active);

-- +goose Down
DROP TABLE IF EXISTS interest_rules;
