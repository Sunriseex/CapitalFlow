-- +goose Up
DROP INDEX IF EXISTS users_single_setup_user_idx;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF (SELECT count(*) FROM users) <= 1 THEN
        CREATE UNIQUE INDEX users_single_setup_user_idx ON users ((true));
    END IF;
END $$;
-- +goose StatementEnd
