-- +goose Up
-- +goose StatementBegin
CREATE TYPE user_role AS ENUM('citizen', 'responder');
CREATE TYPE citizen_status AS ENUM('safe', 'at_risk', 'in_danger');

CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    first_name TEXT NOT NULL,
    middle_name TEXT,
    birth_date DATE NOT NULL,
    role user_role NOT NULL,
    status_update_frequency INTERVAL NOT NULL, -- Usually in minutes
    is_location_shared BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS disaster_reports (
    disaster_report_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    status citizen_status NOT NULL,
    raw_situation TEXT NOT NULL,
    photo_evidence_url TEXT,
    ai_generated_situation TEXT,
    responded_at TIMESTAMPTZ,

    user_id UUID NOT NULL,

    FOREIGN KEY(user_id) REFERENCES users(user_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE user_role;
DROP TYPE citizen_status;
DROP TABLE users;
DROP TABLE disaster_reports;
-- +goose StatementEnd
