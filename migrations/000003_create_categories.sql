-- +goose Up
CREATE TABLE categories (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    parent_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX categories_name_parent_idx ON categories (lower(name), COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid));

-- +goose Down
DROP TABLE IF EXISTS categories;
