-- +goose Up
-- +goose StatementBegin
ALTER TABLE disaster_reports
DROP COLUMN photo_evidence_url;

CREATE TABLE IF NOT EXISTS disaster_photos (
    disaster_photo_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    photo_url TEXT NOT NULL,
    disaster_report_id UUID NOT NULL,

    FOREIGN KEY(disaster_report_id) REFERENCES disaster_reports(disaster_report_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE disaster_photos;

ALTER TABLE disaster_reports
ADD COLUMN photo_evidence_url TEXT;
-- +goose StatementEnd
