-- +goose Up
-- SQL in this section is executed when the migration is applied.
INSERT INTO exchange(id, name)
SELECT
    lower(hex( randomblob(4)) || '-' || hex( randomblob(2)) || '-' || '4' || substr( hex( randomblob(2)), 2) || '-'
         || substr('AB89', 1 + (abs(random()) % 4) , 1)  ||
         substr(hex(randomblob(2)), 2) || '-' || hex(randomblob(6))), exchange
         from withdrawal_history;

ALTER TABLE withdrawal_history ADD COLUMN exchange_name_id;

UPDATE withdrawal_history
SET
    exchange_name_id = (SELECT id FROM exchange WHERE withdrawal_history.exchange = lower(name))
WHERE
    EXISTS(
        SELECT *
        FROM exchange
        WHERE lower("exchange".name) = withdrawal_history.exchange
        );

CREATE TABLE IF NOT EXISTS withdrawal_history_new
(
    id                            text                  PRIMARY KEY NOT NULL,
    exchange_name_id              text,
    exchange_id                   text                  NOT NULL,
    status                        text                  NOT NULL,
    currency                      text                  NOT NULL,
    amount                        real                  NOT NULL,
    description                   text,
    withdraw_type                 integer               NOT NULL,
    created_at                    timestamp             NOT NULL default CURRENT_TIMESTAMP,
    updated_at                    timestamp             NOT NULL default CURRENT_TIMESTAMP,
    FOREIGN KEY(exchange_name_id) REFERENCES exchange(id) ON DELETE RESTRICT
);
INSERT INTO
    withdrawal_history_new (id, exchange_name_id, exchange_id, status, currency, amount, description, withdraw_type, created_at, updated_at)
SELECT
    id, exchange_name_id, exchange_id, status, currency, amount, description, withdraw_type, created_at, updated_at
FROM
    withdrawal_history;

DROP TABLE withdrawal_history;

ALTER TABLE withdrawal_history_new RENAME TO withdrawal_history;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE withdrawal_history ADD COLUMN exchange;

UPDATE withdrawal_history
SET
    exchange = (SELECT name FROM exchange WHERE withdrawal_history.exchange_name_id = id)
WHERE
    EXISTS(
        SELECT *
        FROM exchange
        WHERE exchange.id = withdrawal_history.exchange_name_id
        );

CREATE TABLE IF NOT EXISTS withdrawal_history_new
(
    id text          PRIMARY KEY NOT NULL,
    exchange         text        NOT NULL,
    exchange_id      text        NOT NULL,
    status           text        NOT NULL,
    currency         text        NOT NULL,
    amount           real        NOT NULL,
    description      text,
    withdraw_type    integer     NOT NULL,
    created_at       timestamp   NOT NULL default CURRENT_TIMESTAMP,
    updated_at       timestamp   NOT NULL default CURRENT_TIMESTAMP
);

INSERT INTO
    withdrawal_history_new (id, exchange, exchange_id, status, currency, amount, description, withdraw_type, created_at, updated_at)
SELECT
    id, exchange, exchange_id, status, currency, amount, description, withdraw_type, created_at, updated_at
FROM
    withdrawal_history;

DROP TABLE withdrawal_history;

ALTER TABLE withdrawal_history_new RENAME TO withdrawal_history;
