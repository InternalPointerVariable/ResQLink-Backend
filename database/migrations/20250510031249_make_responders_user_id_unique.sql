-- +goose Up
-- +goose StatementBegin
ALTER TABLE responders
ADD CONSTRAINT responders_user_id_unique UNIQUE (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE responders
DROP CONSTRAINT responders_user_id_unique;
-- +goose StatementEnd
