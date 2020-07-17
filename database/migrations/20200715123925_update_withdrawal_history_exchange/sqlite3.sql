-- +goose Up
-- SQL in this section is executed when the migration is applied.
DROP TABLE IF EXISTS  withdrawal_history;
CREATE TABLE IF NOT EXISTS "withdrawal_history"
(
    id text PRIMARY KEY NOT NULL,
    exchange_name_id         text NOT NULL,
    exchange_id              text NOT NULL,
    status                   text  NOT NULL,
    currency                 text NOT NULL,
    amount                   real NOT NULL,
    description              text  NULL,
    withdraw_type            integer  NOT NULL,
    created_at               timestamp not null default CURRENT_TIMESTAMP,
    updated_at               timestamp not null default CURRENT_TIMESTAMP,
    FOREIGN KEY(exchange_name_id) REFERENCES exchange(id) ON DELETE RESTRICT
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS  withdrawal_history;
CREATE TABLE IF NOT EXISTS "withdrawal_history"
(
    id text PRIMARY KEY NOT NULL,
    exchange         text NOT NULL,
    exchange_id      text NOT NULL,
    status           text  NOT NULL,
    currency         text NOT NULL,
    amount           real NOT NULL,
    description      text  NULL,
    withdraw_type    integer  NOT NULL,
    created_at       timestamp not null default CURRENT_TIMESTAMP,
    updated_at       timestamp not null default CURRENT_TIMESTAMP
);
