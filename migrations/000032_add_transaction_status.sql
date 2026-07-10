-- +goose Up
ALTER TABLE transactions
    ADD COLUMN status TEXT NOT NULL DEFAULT 'confirmed',
    ADD CONSTRAINT transactions_status_check CHECK (status IN (
        'pending',
        'confirmed',
        'cancelled',
        'reversed',
        'soft_deleted'
    ));

CREATE INDEX transactions_account_status_occurred_at_idx
    ON transactions (account_id, status, occurred_at DESC, created_at DESC);

-- +goose Down
-- Transaction lifecycle states affect balances. Removing this column would make
-- cancelled and soft-deleted transactions visible to older application code.
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'migration 000032 is irreversible: transaction lifecycle status cannot be removed safely';
END;
$$;
-- +goose StatementEnd
