-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS script_event
(
    id bigserial PRIMARY KEY NOT NULL,
    script_name       varchar(255)  NOT NULL,
    script_id UUID  NOT NULL,
    execution_type     varchar(255)    NOT NULL,
    execution_time TIMESTAMP NOT NULL DEFAULT (now() at time zone 'utc'),
    execution_status varchar(255)  NOT NULL,
    script_hash    bytea          NULL
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE script_event;