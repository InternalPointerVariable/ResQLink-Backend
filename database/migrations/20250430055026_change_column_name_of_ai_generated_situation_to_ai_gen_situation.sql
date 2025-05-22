-- +goose Up
-- +goose StatementBegin
ALTER TABLE disaster_reports
RENAME COLUMN ai_generated_situation TO ai_gen_situation;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE disaster_reports
RENAME COLUMN ai_gen_situation TO ai_generated_situation;
-- +goose StatementEnd
