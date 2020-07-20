-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS withdrawal_history_new
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

INSERT INTO withdrawal_history_new SELECT * FROM withdrawal_history;
DROP TABLE withdrawal_history;
ALTER TABLE withdrawal_history_new RENAME TO withdrawal_history;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
CREATE TABLE IF NOT EXISTS "withdrawal_history_new"
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

INSERT INTO withdrawal_history_new SELECT * FROM withdrawal_history;
DROP TABLE withdrawal_history;
ALTER TABLE withdrawal_history_new RENAME TO withdrawal_history;
