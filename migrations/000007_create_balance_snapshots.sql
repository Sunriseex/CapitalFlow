-- +goose Up
CREATE TABLE balance_snapshots (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    snapshot_date DATE NOT NULL,
    balance_minor BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (account_id, snapshot_date)
);

-- +goose Down
DROP TABLE IF EXISTS balance_snapshots;
