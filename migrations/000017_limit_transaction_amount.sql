-- +goose Up
ALTER TABLE transactions
ADD CONSTRAINT transactions_amount_minor_bounds_check
CHECK (amount_minor BETWEEN -100000000000000 AND 100000000000000);

-- +goose Down
ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS transactions_amount_minor_bounds_check;
