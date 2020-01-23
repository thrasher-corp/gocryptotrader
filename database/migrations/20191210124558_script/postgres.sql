-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS script
(
    id uuid   PRIMARY KEY DEFAULT gen_random_uuid(),
    script_id text not null,
    script_name varchar not null,
    script_path varchar not null,
    script_data bytea null,
    last_executed_at TIMESTAMP DEFAULT (now() at time zone 'utc'),
    created_at TIMESTAMP DEFAULT (now() at time zone 'utc'),
    CONSTRAINT script_event_uniq UNIQUE (script_id)
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE script;