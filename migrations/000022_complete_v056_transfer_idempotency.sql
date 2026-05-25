-- +goose Up
ALTER TABLE transfers
    ADD COLUMN exchange_rate_scale INTEGER NOT NULL DEFAULT 18,
    ADD COLUMN fee_transaction_id UUID,
    ADD COLUMN fee_amount NUMERIC(38,18) NOT NULL DEFAULT 0 CHECK (fee_amount >= 0),
    ADD COLUMN fee_currency TEXT,
    ADD COLUMN status TEXT NOT NULL DEFAULT 'completed',
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD CONSTRAINT transfers_fee_transaction_fk FOREIGN KEY (fee_transaction_id) REFERENCES transactions(id) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    ADD CONSTRAINT transfers_status_check CHECK (status IN ('completed')),
    ADD CONSTRAINT transfers_fee_currency_check CHECK (
        (fee_amount = 0 AND fee_currency IS NULL AND fee_transaction_id IS NULL)
        OR
        (fee_amount > 0 AND fee_currency = from_currency AND fee_transaction_id IS NOT NULL)
    );

CREATE UNIQUE INDEX transfers_fee_transaction_id_idx
    ON transfers (fee_transaction_id)
    WHERE fee_transaction_id IS NOT NULL;

ALTER TABLE idempotency_keys
    ADD COLUMN id UUID,
    ADD COLUMN endpoint TEXT,
    ADD COLUMN status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN locked_until TIMESTAMPTZ,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD CONSTRAINT idempotency_keys_status_check CHECK (status IN ('pending', 'completed'));

UPDATE idempotency_keys
SET id = gen_random_uuid(),
    endpoint = method || ' ' || path,
    status = CASE WHEN status_code IS NULL THEN 'pending' ELSE 'completed' END,
    updated_at = created_at
WHERE id IS NULL;

ALTER TABLE idempotency_keys
    ALTER COLUMN id SET NOT NULL,
    ALTER COLUMN endpoint SET NOT NULL;

CREATE UNIQUE INDEX idempotency_keys_id_idx ON idempotency_keys (id);
CREATE INDEX idempotency_keys_status_locked_until_idx ON idempotency_keys (status, locked_until);

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
              LEFT JOIN transactions fee_tx ON fee_tx.id = tr.fee_transaction_id
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
                AND (
                    (tr.fee_amount = 0 AND tr.fee_transaction_id IS NULL)
                    OR
                    (
                        tr.fee_amount > 0
                        AND fee_tx.id = tr.fee_transaction_id
                        AND fee_tx.transfer_id IS NULL
                        AND fee_tx.type = 'expense'
                        AND fee_tx.account_id = tr.from_account_id
                        AND fee_tx.amount = tr.fee_amount
                        AND tr.fee_currency = tr.from_currency
                    )
                )
          )
      );

    IF invalid_count > 0 THEN
        RAISE EXCEPTION 'invalid transfer invariant for transfer %', p_transfer_id
            USING ERRCODE = '23514';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_transfer_integrity_from_transaction()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM validate_transfer_integrity(OLD.transfer_id);
        PERFORM validate_transfer_integrity(tr.id)
        FROM transfers tr
        WHERE tr.fee_transaction_id = OLD.id;
        RETURN OLD;
    END IF;

    PERFORM validate_transfer_integrity(NEW.transfer_id);
    PERFORM validate_transfer_integrity(tr.id)
    FROM transfers tr
    WHERE tr.fee_transaction_id = NEW.id;
    IF TG_OP = 'UPDATE' AND OLD.transfer_id IS DISTINCT FROM NEW.transfer_id THEN
        PERFORM validate_transfer_integrity(OLD.transfer_id);
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose Down
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

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_transfer_integrity_from_transaction()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM validate_transfer_integrity(OLD.transfer_id);
        RETURN OLD;
    END IF;

    PERFORM validate_transfer_integrity(NEW.transfer_id);
    IF TG_OP = 'UPDATE' AND OLD.transfer_id IS DISTINCT FROM NEW.transfer_id THEN
        PERFORM validate_transfer_integrity(OLD.transfer_id);
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

DROP INDEX IF EXISTS idempotency_keys_status_locked_until_idx;
DROP INDEX IF EXISTS idempotency_keys_id_idx;

ALTER TABLE idempotency_keys
    DROP CONSTRAINT IF EXISTS idempotency_keys_status_check,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS locked_until,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS endpoint,
    DROP COLUMN IF EXISTS id;

DROP INDEX IF EXISTS transfers_fee_transaction_id_idx;

ALTER TABLE transfers
    DROP CONSTRAINT IF EXISTS transfers_fee_currency_check,
    DROP CONSTRAINT IF EXISTS transfers_status_check,
    DROP CONSTRAINT IF EXISTS transfers_fee_transaction_fk,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS fee_currency,
    DROP COLUMN IF EXISTS fee_amount,
    DROP COLUMN IF EXISTS fee_transaction_id,
    DROP COLUMN IF EXISTS exchange_rate_scale;
