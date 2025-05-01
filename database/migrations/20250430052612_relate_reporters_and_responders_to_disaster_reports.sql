-- +goose Up
-- +goose StatementBegin
ALTER TABLE disaster_reports
DROP COLUMN responded_at, 
DROP COLUMN user_id;

ALTER TABLE disaster_reports
ADD COLUMN reporter_id uuid NOT NULL, 
ADD COLUMN responder_id uuid;

ALTER TABLE disaster_reports
ADD CONSTRAINT fk_reporters_disaster_reports 
FOREIGN KEY (reporter_id)
REFERENCES reporters(reporter_id);

ALTER TABLE disaster_reports
ADD CONSTRAINT fk_responders_disaster_reports 
FOREIGN KEY (responder_id)
REFERENCES responders(responder_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE disaster_reports
DROP CONSTRAINT fk_responders_disaster_reports;

ALTER TABLE disaster_reports
DROP CONSTRAINT fk_reporters_disaster_reports;

ALTER TABLE disaster_reports
DROP COLUMN responder_id,
DROP COLUMN reporter_id;

ALTER TABLE disaster_reports
ADD COLUMN responded_at timestamptz NOT NULL,
ADD COLUMN user_id uuid NOT NULL;

ALTER TABLE disaster_reports
ADD CONSTRAINT fk_users_disaster_reports 
FOREIGN KEY (user_id)
REFERENCES users(user_id);
-- +goose StatementEnd
