-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE "script_event" (
    id	        integer not null primary key,
    script_id    BLOB not null,
    script_name text NULL,
    script_path text NULL,
    script_hash text NULL,
    execution_type text NOT NULL,
    execution_time  timestamp not null default CURRENT_TIMESTAMP,
    execution_status text NOT NULL

);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE "script_event";
