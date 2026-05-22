-- +goose Up
ALTER TABLE transactions
    ALTER COLUMN amount_minor TYPE NUMERIC(38,18) USING amount_minor::NUMERIC(38,18);

ALTER TABLE transfers
    ALTER COLUMN from_amount_minor TYPE NUMERIC(38,18) USING from_amount_minor::NUMERIC(38,18),
    ALTER COLUMN to_amount_minor TYPE NUMERIC(38,18) USING to_amount_minor::NUMERIC(38,18);

ALTER TABLE interest_accruals
    ALTER COLUMN amount_minor TYPE NUMERIC(38,18) USING amount_minor::NUMERIC(38,18),
    ALTER COLUMN balance_minor TYPE NUMERIC(38,18) USING balance_minor::NUMERIC(38,18);

ALTER TABLE balance_snapshots
    ALTER COLUMN balance_minor TYPE NUMERIC(38,18) USING balance_minor::NUMERIC(38,18);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM transactions WHERE amount_minor <> trunc(amount_minor)) THEN
        RAISE EXCEPTION 'cannot downgrade transactions.amount_minor with fractional NUMERIC values';
    END IF;
    IF EXISTS (SELECT 1 FROM transfers WHERE from_amount_minor <> trunc(from_amount_minor) OR to_amount_minor <> trunc(to_amount_minor)) THEN
        RAISE EXCEPTION 'cannot downgrade transfers money columns with fractional NUMERIC values';
    END IF;
    IF EXISTS (SELECT 1 FROM interest_accruals WHERE amount_minor <> trunc(amount_minor) OR balance_minor <> trunc(balance_minor)) THEN
        RAISE EXCEPTION 'cannot downgrade interest_accruals money columns with fractional NUMERIC values';
    END IF;
    IF EXISTS (SELECT 1 FROM balance_snapshots WHERE balance_minor <> trunc(balance_minor)) THEN
        RAISE EXCEPTION 'cannot downgrade balance_snapshots.balance_minor with fractional NUMERIC values';
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE transactions
    ALTER COLUMN amount_minor TYPE BIGINT USING amount_minor::BIGINT;

ALTER TABLE transfers
    ALTER COLUMN from_amount_minor TYPE BIGINT USING from_amount_minor::BIGINT,
    ALTER COLUMN to_amount_minor TYPE BIGINT USING to_amount_minor::BIGINT;

ALTER TABLE interest_accruals
    ALTER COLUMN amount_minor TYPE BIGINT USING amount_minor::BIGINT,
    ALTER COLUMN balance_minor TYPE BIGINT USING balance_minor::BIGINT;

ALTER TABLE balance_snapshots
    ALTER COLUMN balance_minor TYPE BIGINT USING balance_minor::BIGINT;
