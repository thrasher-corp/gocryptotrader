-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS withdrawal_history
(
    id uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange         text NOT NULL,
    exchange_id      text NOT NULL,
    status           varchar(255)  NOT NULL,
    currency         text NOT NULL,
    amount           DOUBLE PRECISION NOT NULL,
    description      text   NULL,
    withdraw_type    integer  NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT (now() at time zone 'utc'),
    updated_at       TIMESTAMP NOT NULL DEFAULT (now() at time zone 'utc')
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS  withdrawal_history;