-- +goose Up
-- Direct ALTER/RENAME migration for maintenance-window deployments.
ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_amount_minor_bounds_check;

ALTER TABLE transactions
    RENAME COLUMN amount_minor TO amount;

ALTER TABLE transactions
    ALTER COLUMN amount TYPE NUMERIC(38,18) USING (amount::NUMERIC(38,18) / 100);

ALTER TABLE transfers
    RENAME COLUMN from_amount_minor TO from_amount;

ALTER TABLE transfers
    RENAME COLUMN to_amount_minor TO to_amount;

ALTER TABLE transfers
    ALTER COLUMN from_amount TYPE NUMERIC(38,18) USING (from_amount::NUMERIC(38,18) / 100),
    ALTER COLUMN to_amount TYPE NUMERIC(38,18) USING (to_amount::NUMERIC(38,18) / 100);

ALTER TABLE interest_accruals
    RENAME COLUMN amount_minor TO amount;

ALTER TABLE interest_accruals
    RENAME COLUMN balance_minor TO balance;

ALTER TABLE interest_accruals
    ALTER COLUMN amount TYPE NUMERIC(38,18) USING (amount::NUMERIC(38,18) / 100),
    ALTER COLUMN balance TYPE NUMERIC(38,18) USING (balance::NUMERIC(38,18) / 100);

ALTER TABLE balance_snapshots
    RENAME COLUMN balance_minor TO balance;

ALTER TABLE balance_snapshots
    ALTER COLUMN balance TYPE NUMERIC(38,18) USING (balance::NUMERIC(38,18) / 100);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_transfer_integrity(p_transfer_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    invalid_count INTEGER;
BEGIN
    IF p_transfer_id IS NULL THEN
        RETURN;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM transfers WHERE id = p_transfer_id) THEN
        RETURN;
    END IF;

    SELECT COUNT(*)
    INTO invalid_count
    FROM transfers tr
    WHERE tr.id = p_transfer_id
      AND (
          (SELECT COUNT(*) FROM transactions tx WHERE tx.transfer_id = tr.id) <> 2
          OR NOT EXISTS (
              SELECT 1
              FROM transactions out_tx
              JOIN transactions in_tx ON in_tx.id = tr.to_transaction_id
              WHERE out_tx.id = tr.from_transaction_id
                AND out_tx.transfer_id = tr.id
                AND in_tx.transfer_id = tr.id
                AND out_tx.type = 'transfer_out'
                AND in_tx.type = 'transfer_in'
                AND out_tx.account_id = tr.from_account_id
                AND in_tx.account_id = tr.to_account_id
                AND out_tx.related_account_id = tr.to_account_id
                AND in_tx.related_account_id = tr.from_account_id
                AND out_tx.amount = tr.from_amount
                AND in_tx.amount = tr.to_amount
          )
      );

    IF invalid_count > 0 THEN
        RAISE EXCEPTION 'invalid transfer invariant for transfer %', p_transfer_id
            USING ERRCODE = '23514';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- Down migration is intentionally blocked if fractional NUMERIC values cannot be represented
-- by the previous BIGINT minor-unit schema after multiplying by 100.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM transactions WHERE amount * 100 <> trunc(amount * 100)) THEN
        RAISE EXCEPTION 'cannot downgrade transactions.amount with fractional minor-unit values';
    END IF;
    IF EXISTS (SELECT 1 FROM transfers WHERE from_amount * 100 <> trunc(from_amount * 100) OR to_amount * 100 <> trunc(to_amount * 100)) THEN
        RAISE EXCEPTION 'cannot downgrade transfers money columns with fractional minor-unit values';
    END IF;
    IF EXISTS (SELECT 1 FROM interest_accruals WHERE amount * 100 <> trunc(amount * 100) OR balance * 100 <> trunc(balance * 100)) THEN
        RAISE EXCEPTION 'cannot downgrade interest_accruals money columns with fractional minor-unit values';
    END IF;
    IF EXISTS (SELECT 1 FROM balance_snapshots WHERE balance * 100 <> trunc(balance * 100)) THEN
        RAISE EXCEPTION 'cannot downgrade balance_snapshots.balance with fractional minor-unit values';
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE transactions
    ALTER COLUMN amount TYPE BIGINT USING (amount * 100)::BIGINT;

ALTER TABLE transactions
    RENAME COLUMN amount TO amount_minor;

ALTER TABLE transactions
    ADD CONSTRAINT transactions_amount_minor_bounds_check
    CHECK (amount_minor BETWEEN -100000000000000 AND 100000000000000);

ALTER TABLE transfers
    ALTER COLUMN from_amount TYPE BIGINT USING (from_amount * 100)::BIGINT,
    ALTER COLUMN to_amount TYPE BIGINT USING (to_amount * 100)::BIGINT;

ALTER TABLE transfers
    RENAME COLUMN from_amount TO from_amount_minor;

ALTER TABLE transfers
    RENAME COLUMN to_amount TO to_amount_minor;

ALTER TABLE interest_accruals
    ALTER COLUMN amount TYPE BIGINT USING (amount * 100)::BIGINT,
    ALTER COLUMN balance TYPE BIGINT USING (balance * 100)::BIGINT;

ALTER TABLE interest_accruals
    RENAME COLUMN amount TO amount_minor;

ALTER TABLE interest_accruals
    RENAME COLUMN balance TO balance_minor;

ALTER TABLE balance_snapshots
    ALTER COLUMN balance TYPE BIGINT USING (balance * 100)::BIGINT;

ALTER TABLE balance_snapshots
    RENAME COLUMN balance TO balance_minor;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_transfer_integrity(p_transfer_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    invalid_count INTEGER;
BEGIN
    IF p_transfer_id IS NULL THEN
        RETURN;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM transfers WHERE id = p_transfer_id) THEN
        RETURN;
    END IF;

    SELECT COUNT(*)
    INTO invalid_count
    FROM transfers tr
    WHERE tr.id = p_transfer_id
      AND (
          (SELECT COUNT(*) FROM transactions tx WHERE tx.transfer_id = tr.id) <> 2
          OR NOT EXISTS (
              SELECT 1
              FROM transactions out_tx
              JOIN transactions in_tx ON in_tx.id = tr.to_transaction_id
              WHERE out_tx.id = tr.from_transaction_id
                AND out_tx.transfer_id = tr.id
                AND in_tx.transfer_id = tr.id
                AND out_tx.type = 'transfer_out'
                AND in_tx.type = 'transfer_in'
                AND out_tx.account_id = tr.from_account_id
                AND in_tx.account_id = tr.to_account_id
                AND out_tx.related_account_id = tr.to_account_id
                AND in_tx.related_account_id = tr.from_account_id
                AND out_tx.amount_minor = tr.from_amount_minor
                AND in_tx.amount_minor = tr.to_amount_minor
          )
      );

    IF invalid_count > 0 THEN
        RAISE EXCEPTION 'invalid transfer invariant for transfer %', p_transfer_id
            USING ERRCODE = '23514';
    END IF;
END;
$$;
-- +goose StatementEnd
