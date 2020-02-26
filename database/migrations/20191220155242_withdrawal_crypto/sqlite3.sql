-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS withdrawal_crypto
(
    id	        integer not null primary key,
    address                   text NOT NULL,
    address_tag               text NULL,
    fee                       real NOT NULL,
    withdrawal_history_id  text NOT NULL,
    FOREIGN KEY(withdrawal_history_id) REFERENCES withdrawal_history(id) ON DELETE RESTRICT
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS  withdrawal_crypto;