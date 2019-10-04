-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS audit_event
(
    id bigserial PRIMARY KEY NOT NULL,
    type       varchar(255)  NOT NULL,
    identifier varchar(255)  NOT NULL,
    message    text          NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT (now() at time zone 'utc')
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE audit_event;