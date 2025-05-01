-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS reporters (
    reporter_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),

    created_at timestamptz NOT NULL DEFAULT now(),
    name text NOT NULL,
    user_id uuid,

    FOREIGN KEY(user_id) REFERENCES users(user_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE reporters;
-- +goose StatementEnd
