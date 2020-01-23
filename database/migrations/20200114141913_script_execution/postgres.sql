-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS script_execution
(
    id uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    script_id uuid REFERENCES script(id) ON DELETE CASCADE,
    execution_type varchar NOT NULL,
    execution_status varchar NOT NULL,
    execution_time TIMESTAMP NOT NULL DEFAULT (now() at time zone 'utc')
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE script_execution;