-- +goose Up
CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

DROP INDEX IF EXISTS interest_rules_active_effective_date_idx;

ALTER TABLE interest_rules
    ADD CONSTRAINT interest_rules_active_period_excl
    EXCLUDE USING gist (
        account_id WITH =,
        daterange(start_date, end_date, '[]') WITH &&
    ) WHERE (is_active);

ALTER TABLE financial_goals ADD COLUMN version BIGINT NOT NULL DEFAULT 0 CHECK (version >= 0);
ALTER TABLE category_limits ADD COLUMN version BIGINT NOT NULL DEFAULT 0 CHECK (version >= 0);
ALTER TABLE interest_rules ADD COLUMN version BIGINT NOT NULL DEFAULT 0 CHECK (version >= 0);

ALTER TABLE accounts
    ADD CONSTRAINT accounts_id_owner_unique UNIQUE (id, owner_user_id);
ALTER TABLE interest_rules
    ADD CONSTRAINT interest_rules_id_account_unique UNIQUE (id, account_id);
ALTER TABLE transactions
    ADD CONSTRAINT transactions_id_account_unique UNIQUE (id, account_id);

ALTER TABLE transfers
    ADD CONSTRAINT transfers_from_account_owner_fk
        FOREIGN KEY (from_account_id, user_id) REFERENCES accounts (id, owner_user_id) ON DELETE RESTRICT,
    ADD CONSTRAINT transfers_to_account_owner_fk
        FOREIGN KEY (to_account_id, user_id) REFERENCES accounts (id, owner_user_id) ON DELETE RESTRICT;

ALTER TABLE financial_goals
    ADD CONSTRAINT financial_goals_account_owner_fk
        FOREIGN KEY (account_id, owner_user_id) REFERENCES accounts (id, owner_user_id) ON DELETE SET NULL (account_id);

ALTER TABLE interest_accruals
    ADD CONSTRAINT interest_accruals_rule_account_fk
        FOREIGN KEY (rule_id, account_id) REFERENCES interest_rules (id, account_id) ON DELETE CASCADE,
    ADD CONSTRAINT interest_accruals_transaction_account_fk
        FOREIGN KEY (transaction_id, account_id) REFERENCES transactions (id, account_id) ON DELETE CASCADE;

-- +goose StatementBegin
CREATE FUNCTION validate_related_account_owner()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    account_owner UUID;
    related_owner UUID;
BEGIN
    IF NEW.related_account_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT owner_user_id INTO account_owner FROM accounts WHERE id = NEW.account_id;
    SELECT owner_user_id INTO related_owner FROM accounts WHERE id = NEW.related_account_id;
    IF account_owner IS DISTINCT FROM related_owner THEN
        RAISE EXCEPTION 'transaction accounts must have the same owner'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER transactions_related_account_owner_check
AFTER INSERT OR UPDATE OF account_id, related_account_id ON transactions
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION validate_related_account_owner();

-- +goose StatementBegin
CREATE FUNCTION validate_interest_transaction()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM interest_accruals accrual
        JOIN transactions transaction ON transaction.id = accrual.transaction_id
        WHERE transaction.id = NEW.id
          AND (
              transaction.type <> 'interest_income'
              OR transaction.source_type <> 'deposit_interest'
              OR transaction.source_ref_id IS DISTINCT FROM accrual.id
          )
    ) THEN
        RAISE EXCEPTION 'invalid interest transaction %', NEW.id
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER transactions_interest_integrity_check
AFTER INSERT OR UPDATE OF type, source_type, source_ref_id ON transactions
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION validate_interest_transaction();

-- +goose StatementBegin
CREATE FUNCTION validate_interest_accrual()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    transaction_row transactions%ROWTYPE;
BEGIN
    SELECT * INTO transaction_row FROM transactions WHERE id = NEW.transaction_id;
    IF transaction_row.type <> 'interest_income'
       OR transaction_row.source_type <> 'deposit_interest'
       OR transaction_row.source_ref_id IS DISTINCT FROM NEW.id THEN
        RAISE EXCEPTION 'invalid transaction for interest accrual %', NEW.id
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER interest_accruals_transaction_integrity_check
AFTER INSERT OR UPDATE OF transaction_id, account_id, rule_id ON interest_accruals
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION validate_interest_accrual();

CREATE INDEX transactions_description_trgm_idx
    ON transactions USING gin (lower(description) gin_trgm_ops);
CREATE INDEX accounts_name_trgm_idx
    ON accounts USING gin (lower(name) gin_trgm_ops);
CREATE INDEX accounts_bank_trgm_idx
    ON accounts USING gin (lower(bank) gin_trgm_ops);
CREATE INDEX categories_name_trgm_idx
    ON categories USING gin (lower(name) gin_trgm_ops);
CREATE INDEX categories_slug_trgm_idx
    ON categories USING gin (lower(slug) gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS categories_slug_trgm_idx;
DROP INDEX IF EXISTS categories_name_trgm_idx;
DROP INDEX IF EXISTS accounts_bank_trgm_idx;
DROP INDEX IF EXISTS accounts_name_trgm_idx;
DROP INDEX IF EXISTS transactions_description_trgm_idx;

DROP TRIGGER IF EXISTS interest_accruals_transaction_integrity_check ON interest_accruals;
DROP TRIGGER IF EXISTS transactions_interest_integrity_check ON transactions;
DROP TRIGGER IF EXISTS transactions_related_account_owner_check ON transactions;
DROP FUNCTION IF EXISTS validate_interest_accrual();
DROP FUNCTION IF EXISTS validate_interest_transaction();
DROP FUNCTION IF EXISTS validate_related_account_owner();

ALTER TABLE interest_accruals
    DROP CONSTRAINT IF EXISTS interest_accruals_transaction_account_fk,
    DROP CONSTRAINT IF EXISTS interest_accruals_rule_account_fk;
ALTER TABLE financial_goals
    DROP CONSTRAINT IF EXISTS financial_goals_account_owner_fk;
ALTER TABLE transfers
    DROP CONSTRAINT IF EXISTS transfers_to_account_owner_fk,
    DROP CONSTRAINT IF EXISTS transfers_from_account_owner_fk;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_id_account_unique;
ALTER TABLE interest_rules DROP CONSTRAINT IF EXISTS interest_rules_id_account_unique;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_id_owner_unique;

ALTER TABLE interest_rules DROP COLUMN IF EXISTS version;
ALTER TABLE category_limits DROP COLUMN IF EXISTS version;
ALTER TABLE financial_goals DROP COLUMN IF EXISTS version;

ALTER TABLE interest_rules DROP CONSTRAINT IF EXISTS interest_rules_active_period_excl;
CREATE UNIQUE INDEX interest_rules_active_effective_date_idx
    ON interest_rules (account_id, accrual_frequency, start_date)
    WHERE is_active;
