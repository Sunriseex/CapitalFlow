-- +goose Up
CREATE UNIQUE INDEX interest_rules_active_effective_date_idx
    ON interest_rules (account_id, accrual_frequency, start_date)
    WHERE is_active;

-- +goose Down
DROP INDEX IF EXISTS interest_rules_active_effective_date_idx;
