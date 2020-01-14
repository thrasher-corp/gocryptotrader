-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS script_event
(
    id uuid   PRIMARY KEY DEFAULT gen_random_uuid(),
    script_id text not null,
    script_name varchar not null,
    script_path varchar not null,
    script_hash text null,
    created_at TIMESTAMP DEFAULT (now() at time zone 'utc')
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE script_event;