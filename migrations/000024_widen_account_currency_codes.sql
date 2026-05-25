-- +goose Up
ALTER TABLE accounts
    DROP CONSTRAINT IF EXISTS accounts_currency_check,
    ALTER COLUMN currency TYPE TEXT USING btrim(currency::text),
    ADD CONSTRAINT accounts_currency_code_check CHECK (currency ~ '^[A-Z][A-Z0-9]{1,11}$');

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_primary_currency_check,
    ALTER COLUMN primary_currency TYPE TEXT USING btrim(primary_currency::text),
    ADD CONSTRAINT users_primary_currency_code_check CHECK (primary_currency ~ '^[A-Z][A-Z0-9]{1,11}$');

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM accounts WHERE currency !~ '^[A-Z]{3}$')
        OR EXISTS (SELECT 1 FROM users WHERE primary_currency !~ '^[A-Z]{3}$') THEN
        RAISE EXCEPTION 'cannot downgrade currency columns to CHAR(3) while non-ISO asset codes exist';
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE accounts
    DROP CONSTRAINT IF EXISTS accounts_currency_code_check,
    ALTER COLUMN currency TYPE CHAR(3) USING currency::CHAR(3),
    ADD CONSTRAINT accounts_currency_check CHECK (currency ~ '^[A-Z]{3}$');

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_primary_currency_code_check,
    ALTER COLUMN primary_currency TYPE CHAR(3) USING primary_currency::CHAR(3),
    ADD CONSTRAINT users_primary_currency_check CHECK (primary_currency ~ '^[A-Z]{3}$');
