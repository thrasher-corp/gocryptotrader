-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS "script_execution"
(
    id	        integer not null primary key,
    script_id text not null,
    execution_type text NOT NULL,
    execution_status text NOT NULL,
    execution_time timestamp not null default CURRENT_TIMESTAMP,
    FOREIGN KEY(script_id) REFERENCES script(id)
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE script_execution;