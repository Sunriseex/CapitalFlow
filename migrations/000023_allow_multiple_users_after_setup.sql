-- +goose Up
DROP INDEX IF EXISTS users_single_setup_user_idx;

-- +goose Down
CREATE UNIQUE INDEX users_single_setup_user_idx ON users ((true));
