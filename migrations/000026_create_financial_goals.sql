-- +goose Up
CREATE TABLE financial_goals (
    id UUID PRIMARY KEY,
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    target_amount NUMERIC(38, 18) NOT NULL CHECK (target_amount > 0),
    currency VARCHAR(3) NOT NULL CHECK (currency = upper(currency)),
    target_date DATE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX financial_goals_owner_status_idx ON financial_goals (owner_user_id, status, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS financial_goals;
