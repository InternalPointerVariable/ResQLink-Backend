-- +goose Up
-- +goose StatementBegin
ALTER TABLE reporters
ADD CONSTRAINT reporters_user_id_unique UNIQUE (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE reporters
DROP CONSTRAINT reporters_user_id_unique;
-- +goose StatementEnd
