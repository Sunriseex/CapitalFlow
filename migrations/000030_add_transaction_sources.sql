-- +goose Up
ALTER TABLE transactions
    ADD COLUMN source_type TEXT NOT NULL DEFAULT 'manual',
    ADD COLUMN source_ref_id UUID,
    ADD COLUMN source_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD CONSTRAINT transactions_source_type_check CHECK (source_type IN (
        'manual',
        'csv_import',
        'transfer',
        'deposit_interest',
        'savings_allocation',
        'subscription',
        'reconciliation_adjustment',
        'automation_rule',
        'llm_draft',
        'system'
    )),
    ADD CONSTRAINT transactions_source_metadata_object_check CHECK (jsonb_typeof(source_metadata) = 'object');

UPDATE transactions
SET source_type = 'transfer', source_ref_id = transfer_id
WHERE transfer_id IS NOT NULL;

UPDATE transactions AS transaction
SET source_type = 'deposit_interest', source_ref_id = accrual.id
FROM interest_accruals AS accrual
WHERE accrual.transaction_id = transaction.id;

SET CONSTRAINTS ALL IMMEDIATE;

CREATE INDEX transactions_source_idx
    ON transactions (source_type, source_ref_id)
    WHERE source_ref_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS transactions_source_idx;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_source_metadata_object_check,
    DROP CONSTRAINT IF EXISTS transactions_source_type_check,
    DROP COLUMN IF EXISTS source_metadata,
    DROP COLUMN IF EXISTS source_ref_id,
    DROP COLUMN IF EXISTS source_type;
