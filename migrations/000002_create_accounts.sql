-- +goose Up
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    legacy_id TEXT UNIQUE,
    name TEXT NOT NULL,
    bank TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL CHECK (type IN ('cash', 'card', 'savings', 'term_deposit', 'broker', 'other')),
    currency CHAR(3) NOT NULL DEFAULT 'RUB' CHECK (currency ~ '^[A-Z]{3}$'),
    is_active BOOLEAN NOT NULL DEFAULT true,
    opened_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS accounts;
